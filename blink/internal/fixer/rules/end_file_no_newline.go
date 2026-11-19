package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/EndFileNoNewlineSniff.php
//
// EndFileNoNewline strips the trailing line endings of a PHP file that ends with
// one. Trailing spaces before them stay.
type EndFileNoNewline struct{}

func (EndFileNoNewline) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\EndFileNoNewlineSniff`
}

func (EndFileNoNewline) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/EndFileNoNewlineSniff.php"
}

func (EndFileNoNewline) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, true) {
		return false
	}
	last := phpcsLastToken(s)
	eol := phpcsEolChar(s)
	if last < 0 || !strings.HasSuffix(s.At(last).Value, eol) {
		return false
	}
	// like upstream's rtrim($content, $eolChar), eol is a character set here
	for i := last; i > 0; i-- {
		v := strings.TrimRight(s.At(i).Value, eol)
		s.SetValue(i, v)
		if v != "" {
			break
		}
	}
	return true
}
