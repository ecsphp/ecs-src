package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ObjectOperatorSpacingSniff.php
//
// ObjectOperatorSpacing removes whitespace around "->", "?->" and "::":
// "$a -> b" -> "$a->b". With IgnoreNewlines a line break on either side is kept.
type ObjectOperatorSpacing struct {
	IgnoreNewlines bool
}

func (ObjectOperatorSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\ObjectOperatorSpacingSniff`
}

func (ObjectOperatorSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/ObjectOperatorSpacingSniff.php"
}

func (f ObjectOperatorSpacing) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

func (f ObjectOperatorSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || (t.Value != "->" && t.Value != "?->" && t.Value != "::") {
			continue
		}
		// before: a line comment's newline is part of the comment for PHPCS, so
		// whitespace after one starts a new line
		if w := i - 1; phpcsWhitespaceAt(s, w) {
			newline := strings.ContainsAny(s.At(w).Value, "\r\n") || (w > 0 && isLineComment(s.At(w-1)))
			if !newline || !f.IgnoreNewlines {
				if dropPhpcsWhitespace(s, w) {
					i--
				}
				changed = true
			}
		}
		if i+2 >= s.Len() {
			continue
		}
		if w := i + 1; s.At(w).Kind == token.Whitespace {
			if !strings.ContainsAny(s.At(w).Value, "\r\n") || !f.IgnoreNewlines {
				s.RemoveAt(w)
				changed = true
			}
		}
	}
	return changed
}
