package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Files/ClosingTagSniff.php
//
// PSR2ClosingTag removes the closing "?>" at the end of a PHP-only file.
// A file with any non-blank inline HTML is left alone.
type PSR2ClosingTag struct{}

func (PSR2ClosingTag) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Files\ClosingTagSniff`
}

func (PSR2ClosingTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Files/ClosingTagSniff.php"
}

func (PSR2ClosingTag) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, false) {
		return false
	}
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.InlineHTML && strings.Trim(t.Value, " \t\n\r\x00\x0B") != "" {
			return false
		}
	}
	return phpcsRemoveFinalCloseTag(s)
}
