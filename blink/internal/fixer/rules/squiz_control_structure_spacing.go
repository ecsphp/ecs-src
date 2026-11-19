package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ControlStructureSpacingSniff.php
//
// SquizControlStructureSpacing removes the spaces inside a control structure's
// parentheses and the blank lines at the start and end of its body, and wants
// one blank line after it.
type SquizControlStructureSpacing struct{}

func (SquizControlStructureSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\ControlStructureSpacingSniff`
}

func (SquizControlStructureSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ControlStructureSpacingSniff.php"
}

var squizControlStructures = phpcsSet("if", "while", "foreach", "for", "switch", "do", "else", "elseif", "try",
	"catch", "finally", "match")

func (SquizControlStructureSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if f.toks[i].kind == token.Keyword && squizControlStructures[sc.code[i]] {
				squizControlStructureSpacing(sc, i)
			}
		}
	})
}

// squizIgnoredStructure tells the structures whose spacing other sniffs check:
// named functions, classes, interfaces, traits, enums and docblocks.
func squizIgnoredStructure(sc *phpcsScope, i int) bool {
	f := sc.f
	switch sc.code[i] {
	case "function":
		return squizFunctionCode(sc, i) == "function"
	case "class":
		return !phpcsIsAnonClassKeyword(f, i)
	case "interface", "trait", "enum":
		return true
	}
	return f.toks[i].kind == token.DocComment && f.toks[i].doc == slvDocOpen
}

// squizIsComment is T_COMMENT or a phpcs: annotation.
func squizIsComment(f *slvFile, i int) bool {
	return f.toks[i].kind == token.Comment || phpcsIsAnnotation(f, i)
}

func squizControlStructureSpacing(sc *phpcsScope, stackPtr int) {
	f := sc.f
	n := len(f.toks)
	if pc := sc.pCloser[stackPtr]; pc >= 0 && f.toks[pc].closer >= 0 {
		po := f.toks[pc].closer
		if f.toks[po+1].kind == token.Whitespace {
			f.replaceToken(po+1, "")
		}
		if f.toks[po].line == f.toks[pc].line && f.toks[pc-1].kind == token.Whitespace {
			f.replaceToken(pc-1, "")
		}
	}
	scopeCloser := squizScopeCloser(sc, stackPtr)
	if scopeCloser < 0 {
		return
	}
	scopeOpener := sc.opener[stackPtr]
	firstContent := scopeOpener + 1
	for ; firstContent < n; firstContent++ {
		t := f.toks[firstContent]
		if t.kind == token.Whitespace || t.kind == token.InlineHTML && strings.TrimSpace(t.content) == "" {
			continue
		}
		if t.line == f.toks[scopeOpener].line && (f.isEmpty(firstContent) || t.kind == token.CloseTag) {
			continue
		}
		break
	}
	if firstContent >= n {
		return
	}
	if !squizIgnoredStructure(sc, firstContent) && f.toks[firstContent].line >= f.toks[scopeOpener].line+2 {
		f.beginChangeset()
		for i := scopeOpener + 1; f.toks[i].line != f.toks[firstContent].line; i++ {
			if f.toks[i].line != f.toks[scopeOpener].line {
				f.replaceToken(i, "")
			}
		}
		f.endChangeset()
	}
	if firstContent != scopeCloser {
		lastContent := max(f.prevNonWhitespace(scopeCloser-1), 0)
		lastNonEmpty := max(f.prevEffective(scopeCloser-1), 0)
		checkToken := lastContent
		if sc.cond[lastNonEmpty] >= 0 {
			checkToken = sc.cond[lastNonEmpty]
		}
		if !squizIgnoredStructure(sc, checkToken) && f.toks[lastContent].line <= f.toks[scopeCloser].line-2 {
			f.beginChangeset()
			for i := scopeCloser - 1; i > lastContent; i-- {
				if f.toks[i].line == f.toks[scopeCloser].line {
					continue
				}
				if f.toks[i].line == f.toks[lastContent].line {
					break
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
	}
	if sc.code[stackPtr] == "match" {
		if next := f.nextEffective(scopeCloser+1, n); next >= 0 && (f.is(next, ";") || f.is(next, ",")) {
			scopeCloser = next
		}
	}
	trailing := max(f.nextNonWhitespace(scopeCloser+1, n), 0)
	if squizIsComment(f, trailing) {
		nextCode := max(f.nextEffective(scopeCloser+1, n), 0)
		if f.isKeyword(nextCode, "else") || f.isKeyword(nextCode, "elseif") ||
			f.toks[trailing].line == f.toks[scopeCloser].line {
			trailing = nextCode
		}
	}
	code := sc.code[stackPtr]
	switch {
	case f.isKeyword(trailing, "else") && code == "if",
		f.isKeyword(trailing, "while") && code == "do",
		f.toks[trailing].kind == token.CloseTag:
		return
	}
	if c := sc.cond[trailing]; c >= 0 && c != trailing && sc.opener[trailing] >= 0 && sc.opener[trailing] != trailing {
		switch squizFunctionCode(sc, c) {
		case "function":
			return
		case "closure":
			if squizInFunction(sc, stackPtr) || sc.nested[stackPtr] >= 0 {
				return
			}
		}
		if f.toks[trailing].line != f.toks[scopeCloser].line+1 {
			f.beginChangeset()
			for i := scopeCloser + 1; f.toks[i].line != f.toks[trailing].line; i++ {
				f.replaceToken(i, "")
			}
			f.addContent(scopeCloser, f.eol)
			f.endChangeset()
		}
		return
	}
	switch sc.code[trailing] {
	case "else", "elseif", "catch", "finally":
		return
	}
	if f.toks[trailing].line != f.toks[scopeCloser].line+1 {
		return
	}
	trailing = max(f.nextNonWhitespace(scopeCloser+1, n), 0)
	if squizIsComment(f, trailing) && f.toks[trailing].line == f.toks[scopeCloser].line {
		f.addContent(trailing, f.eol)
	} else {
		f.addContent(scopeCloser, f.eol)
	}
}

// squizFunctionCode is "function" for PHP_CodeSniffer's T_FUNCTION, "closure"
// for T_CLOSURE, else "": the tokenizer only retags a "function (" with a scope
// opener as T_CLOSURE.
func squizFunctionCode(sc *phpcsScope, i int) string {
	f := sc.f
	switch {
	case !f.isKeyword(i, "function"):
		return ""
	case f.isClosure(i):
		if squizScopeCloser(sc, i) >= 0 {
			return "closure"
		}
		return "function"
	case phpcsIsNamedFunction(f, i):
		return "function"
	}
	return ""
}

// squizScopeCloser is the scope_closer of the scope owner at i, or -1.
func squizScopeCloser(sc *phpcsScope, i int) int {
	if sc.cond[i] != i || sc.opener[i] < 0 || sc.closer[i] < 0 {
		return -1
	}
	return sc.closer[i]
}

// squizInFunction is File::hasCondition(i, [T_FUNCTION, T_CLOSURE]).
func squizInFunction(sc *phpcsScope, i int) bool {
	for _, c := range sc.conditions(i) {
		if sc.code[c] == "function" {
			return true
		}
	}
	return false
}
