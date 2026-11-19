package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ScopeKeywordSpacingSniff.php
//
// ScopeKeywordSpacing puts exactly one space after a scope or method modifier:
// "public  static\nfunction" -> "public static function".
type ScopeKeywordSpacing struct{}

func (ScopeKeywordSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\ScopeKeywordSpacingSniff`
}

func (ScopeKeywordSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ScopeKeywordSpacingSniff.php"
}

func (ScopeKeywordSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isScopeOrMethodModifier, func(i int) int {
			f.scopeKeywordSpacing(i)
			return 0
		})
	})
}

// isScopeOrMethodModifier matches Tokens::METHOD_MODIFIERS + SCOPE_MODIFIERS + T_READONLY.
func (f *slvFile) isScopeOrMethodModifier(i int) bool {
	t := f.toks[i]
	if t.kind != token.Keyword {
		return false
	}
	switch v := strings.ToLower(t.orig); v {
	case "public", "protected", "private", "static", "abstract", "final", "readonly":
		return true
	default:
		return strings.HasSuffix(v, "(set)")
	}
}

func (f *slvFile) scopeKeywordSpacing(i int) {
	n := len(f.toks)
	if f.nextNonWhitespace(i+1, n) < 0 {
		return
	}
	prev := f.prevEffective(i - 1)
	next := f.nextEffective(i+1, n)
	if f.isKeyword(i, "static") {
		if next < 0 || f.is(next, "::") || f.isKeyword(prev, "new") || f.isKeyword(prev, "instanceof") {
			return
		}
		if f.is(prev, "|") || f.is(prev, "?") {
			return
		}
		if f.is(prev, ":") && f.isReturnColon(prev) {
			return
		}
	}
	if f.isKeyword(prev, "as") {
		return
	}
	if next >= 0 && f.toks[next].kind == token.Variable && !f.inFunctionDeclarationParens(i) {
		end := -1
		for j := next + 1; j < n; j++ {
			if f.is(j, ";") {
				end = j
				break
			}
		}
		if end < 0 {
			return
		}
		multi := false
		for j := next + 1; j < end; j++ {
			if f.toks[j].kind == token.Variable {
				multi = true
				break
			}
		}
		if multi && f.toks[i].line != f.toks[next].line && f.toks[next].line != f.toks[end].line {
			return
		}
	}
	spacing := 0
	newline := false
	if i+1 < n && f.toks[i+1].kind == token.Whitespace {
		if i+2 < n && f.toks[i+2].line != f.toks[i].line {
			newline = true
		} else {
			spacing = f.toks[i+1].length
		}
	}
	if !newline && spacing == 1 {
		return
	}
	if !newline && spacing == 0 {
		f.replaceToken(i, f.tokenContent(i)+" ")
		return
	}
	f.beginChangeset()
	for j := i + 2; j < n && f.toks[j].kind == token.Whitespace; j++ {
		f.replaceToken(j, "")
	}
	f.replaceToken(i+1, " ")
	f.endChangeset()
}

// inFunctionDeclarationParens reports whether the innermost parentheses holding
// i are a named function's parameter list (constructor property promotion).
func (f *slvFile) inFunctionDeclarationParens(i int) bool {
	open := -1
	for j := i - 1; j >= 0; j-- {
		if f.is(j, "(") && f.toks[j].closer > i {
			open = j
			break
		}
	}
	if open < 0 {
		return false
	}
	name := f.prevEffective(open - 1)
	if name < 0 || f.toks[name].kind != token.Ident && f.toks[name].kind != token.Keyword {
		return false
	}
	fn := f.prevEffective(name - 1)
	if f.is(fn, "&") {
		fn = f.prevEffective(fn - 1)
	}
	return f.isKeyword(fn, "function")
}

// isReturnColon tells a return type's ":" (T_COLON after a function's, closure's
// or arrow function's parameters or closure use) from a ternary's.
func (f *slvFile) isReturnColon(colon int) bool {
	closeIdx := f.prevEffective(colon - 1)
	if !f.is(closeIdx, ")") || f.toks[closeIdx].closer < 0 {
		return false
	}
	p := f.prevEffective(f.toks[closeIdx].closer - 1)
	if p >= 0 && f.toks[p].kind == token.Ident {
		p = f.prevEffective(p - 1)
	}
	if f.is(p, "&") {
		p = f.prevEffective(p - 1)
	}
	return f.isKeyword(p, "function") || f.isKeyword(p, "fn") || f.isKeyword(p, "use")
}
