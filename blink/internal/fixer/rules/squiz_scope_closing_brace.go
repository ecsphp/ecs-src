package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ScopeClosingBraceSniff.php
//
// SquizScopeClosingBrace puts a scope's closing brace on its own line, in the
// column of the first content on the line that opens the scope.
type SquizScopeClosingBrace struct{}

func (SquizScopeClosingBrace) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\ScopeClosingBraceSniff`
}

func (SquizScopeClosingBrace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ScopeClosingBraceSniff.php"
}

func (r SquizScopeClosingBrace) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if phpcsScopeOpenerCodes[sc.code[i]] {
				r.process(sc, i)
			}
		}
	})
}

func (SquizScopeClosingBrace) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	scopeEnd := sc.closer[stackPtr]
	if scopeEnd < 0 {
		return
	}
	scopeStart := sc.opener[stackPtr]
	lineStart := squizFirstOnLine(sc, stackPtr)
	for squizConstantString(f, lineStart) && squizConstantString(f, lineStart-1) {
		lineStart = squizFirstOnLine(sc, lineStart-1)
	}
	startColumn := f.toks[lineStart].column

	lastContent := 0
	for i := scopeEnd - 1; i >= scopeStart && i >= 0; i-- {
		if c := sc.code[i]; c != "inline_html" && c != "whitespace" && c != "open_tag" {
			lastContent = i
			break
		}
	}
	lineStart = scopeEnd
	for lineStart > 0 && f.toks[lineStart].column > 1 {
		lineStart--
	}
	if f.toks[lastContent].line == f.toks[scopeEnd].line ||
		sc.code[lineStart] == "inline_html" && strings.TrimSpace(f.toks[lineStart].content) != "" {
		at := lineStart + 1
		if f.toks[lastContent].line == f.toks[scopeEnd].line {
			at = scopeEnd
		}
		f.replaceToken(at, f.eol+f.tokenContent(at))
		return
	}

	lineStart = squizFirstOnLine(sc, scopeEnd)
	braceIndent := f.toks[lineStart].column
	if c := sc.code[stackPtr]; c == "default" || c == "case" || braceIndent == startColumn {
		return
	}
	if diff := startColumn - braceIndent; diff > 0 {
		f.replaceToken(lineStart, strings.Repeat(" ", diff)+f.tokenContent(lineStart))
	} else if lineStart > 0 {
		v := f.tokenContent(lineStart - 1)
		f.replaceToken(lineStart-1, v[:max(0, len(v)+diff)])
	}
}

// squizFirstOnLine is File::findFirstOnLine([T_WHITESPACE, T_INLINE_HTML],
// start, true): the first token on start's line that is neither, or 0 for
// none (PHP reads $tokens[false] as $tokens[0]).
func squizFirstOnLine(sc *phpcsScope, start int) int {
	found := 0
	line := sc.f.toks[start].line
	for i := start; i >= 0 && sc.f.toks[i].line == line; i-- {
		if c := sc.code[i]; c != "whitespace" && c != "inline_html" {
			found = i
		}
	}
	return found
}

// squizConstantString reports a T_CONSTANT_ENCAPSED_STRING token.
func squizConstantString(f *slvFile, i int) bool {
	if i < 0 || i >= len(f.toks) {
		return false
	}
	t := f.toks[i]
	return t.kind == token.String && !t.heredoc && !t.interp
}
