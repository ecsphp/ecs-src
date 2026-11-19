package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ScopeClosingBraceSniff.php
//
// PearScopeClosingBrace puts a scope's closing brace on its own line, lined up
// with the start of the line that opens the scope; the terminating statement of
// a CASE or DEFAULT goes Indent spaces further in.
type PearScopeClosingBrace struct {
	Indent int
}

func (PearScopeClosingBrace) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\WhiteSpace\ScopeClosingBraceSniff`
}

func (PearScopeClosingBrace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ScopeClosingBraceSniff.php"
}

func (r PearScopeClosingBrace) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r PearScopeClosingBrace) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if phpcsScopeOpenerCodes[sc.code[i]] {
				r.process(sc, i)
			}
		}
	})
}

func (r PearScopeClosingBrace) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	scopeStart, scopeEnd := sc.opener[stackPtr], sc.closer[stackPtr]
	if scopeEnd < 0 || sc.cond[scopeEnd] != stackPtr {
		return
	}
	startColumn := phpcsLineStartColumn(sc, phpcsLineStart(f, stackPtr))
	lastContent := scopeEnd - 1
	for ; lastContent > scopeStart; lastContent-- {
		c := sc.code[lastContent]
		if c == "whitespace" || c == "open_tag" || c == "inline_html" && phpcsLtrim(f.toks[lastContent].content) == "" {
			continue
		}
		break
	}
	if f.toks[lastContent].line == f.toks[scopeEnd].line {
		f.replaceToken(scopeEnd, f.eol+f.tokenContent(scopeEnd))
		return
	}
	lineStart := phpcsLineStart(f, scopeEnd)
	braceIndent := phpcsLineStartColumn(sc, lineStart) - 1
	expectedIndent := max(0, startColumn-1)
	if sc.code[stackPtr] == "case" || sc.code[stackPtr] == "default" {
		expectedIndent = startColumn + r.Indent - 1
	}
	if braceIndent == expectedIndent {
		return
	}
	spaces := strings.Repeat(" ", expectedIndent)
	if braceIndent != 0 {
		f.replaceToken(lineStart, phpcsLtrim(f.toks[lineStart].content))
	}
	f.replaceToken(lineStart, spaces+f.tokenContent(lineStart))
}

// phpcsLineStart returns the token after the last one before i holding a
// newline.
func phpcsLineStart(f *slvFile, i int) int {
	lineStart := i - 1
	for ; lineStart > 0; lineStart-- {
		if strings.Contains(f.toks[lineStart].content, f.eol) {
			break
		}
	}
	return lineStart + 1
}

// phpcsLineStartColumn is the column the content of the line starting at
// lineStart begins in, as ScopeClosingBraceSniff reads it (one short after an
// inline HTML indent).
func phpcsLineStartColumn(sc *phpcsScope, lineStart int) int {
	f := sc.f
	switch sc.code[lineStart] {
	case "whitespace":
		return f.toks[lineStart+1].column
	case "inline_html":
		content := f.toks[lineStart].content
		trimmed := phpcsLtrim(content)
		if trimmed == "" {
			return f.toks[lineStart+1].column
		}
		return len(content) - len(trimmed)
	}
	return 1
}

func phpcsLtrim(v string) string {
	return strings.TrimLeft(v, " \t\n\r\x00\x0B")
}
