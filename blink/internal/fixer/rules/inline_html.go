package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/InlineHTMLSniff.php
//
// InlineHTML reports the first inline HTML outside the PHP tags, allowing a
// lone byte order mark and a shebang line. It fixes nothing.
type InlineHTML struct{}

func (InlineHTML) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\InlineHTMLSniff`
}

func (InlineHTML) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/InlineHTMLSniff.php"
}

func (InlineHTML) Fix(*tokens.Stream) bool { return false }

func (h InlineHTML) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	if len(f.toks) == 0 {
		return nil
	}
	for _, m := range byteOrderMarks {
		if f.toks[0].orig == m.bom {
			return nil
		}
	}
	for _, t := range f.toks {
		if t.kind != token.InlineHTML || strings.HasPrefix(t.orig, "#!") {
			continue
		}
		return []fixer.CodingStandardError{{Line: t.line, Message: "PHP files must only contain PHP code", Checker: h.Name() + ".Found"}}
	}
	return nil
}
