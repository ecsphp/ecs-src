package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/EndFileNewlineSniff.php
//
// EndFileNewline appends the file's line ending when a PHP file does not end with one.
type EndFileNewline struct{}

func (EndFileNewline) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\EndFileNewlineSniff`
}

func (EndFileNewline) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/EndFileNewlineSniff.php"
}

func (EndFileNewline) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, true) {
		return false
	}
	last := phpcsLastToken(s)
	if last < 0 {
		return false
	}
	eol := phpcsEolChar(s)
	if v := s.At(last).Value; !strings.HasSuffix(v, eol) {
		s.SetValue(last, v+eol)
		return true
	}
	return false
}
