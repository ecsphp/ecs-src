package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Zend/Sniffs/Files/ClosingTagSniff.php
//
// ZendClosingTag replaces a closing tag ending the file with a line ending, and
// adds the ";" the tag stood in for: "<?php foo() ?>" -> "<?php foo();\n".
type ZendClosingTag struct{}

func (ZendClosingTag) Name() string {
	return `PHP_CodeSniffer\Standards\Zend\Sniffs\Files\ClosingTagSniff`
}

func (ZendClosingTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Zend/Sniffs/Files/ClosingTagSniff.php"
}

func (ZendClosingTag) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, false) {
		return false
	}
	last := s.Len() - 1
	for ; last > 0; last-- {
		if strings.Trim(s.At(last).Value, " \t\n\r\x00\x0B") != "" {
			break
		}
	}
	if last < 0 || s.At(last).Kind != token.CloseTag {
		return false
	}
	eol := phpcsEolChar(s)
	// PHP's close tag token takes one newline right after "?>" along with it
	if last+1 < s.Len() {
		next := s.At(last + 1).Value
		if rest, ok := strings.CutPrefix(next, "\r\n"); ok {
			s.SetValue(last+1, rest)
		} else if rest, ok := strings.CutPrefix(next, "\n"); ok {
			s.SetValue(last+1, rest)
		}
	}
	s.SetValue(last, eol)
	prev := last - 1
	for prev >= 0 && isPhpcsEmpty(s.At(prev)) {
		prev--
	}
	if prev < 0 {
		return true
	}
	if p := s.At(prev); p.Value != ";" && p.Value != "}" && (p.Kind != token.OpenTag || strings.HasPrefix(p.Value, "<?=")) {
		s.SetValue(prev, p.Value+";")
	}
	return true
}
