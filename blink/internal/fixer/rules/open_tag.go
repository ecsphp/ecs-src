package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/OpenTagSniff.php
//
// OpenTag puts code that follows the opening "<?php" of a PHP-only file on its
// own line: "<?php echo 1;" -> "<?php\necho 1;".
type OpenTag struct{}

func (OpenTag) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Files\OpenTagSniff`
}

func (OpenTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/OpenTagSniff.php"
}

func (OpenTag) Fix(s *tokens.Stream) bool {
	if s.Len() == 0 || s.At(0).Kind != token.OpenTag || s.At(0).Value == "<?=" {
		return false
	}
	next := nextSignificantIndex(s, 0)
	if next < 0 || next > 1 && strings.ContainsAny(s.At(1).Value, "\r\n") {
		return false
	}
	for i := range s.Len() {
		if s.At(i).Kind == token.InlineHTML {
			return false
		}
	}
	eol := phpcsEolChar(s)
	if next == 1 {
		s.InsertAt(1, token.Token{Kind: token.Whitespace, Value: eol})
	} else {
		s.SetValue(1, eol)
	}
	return true
}
