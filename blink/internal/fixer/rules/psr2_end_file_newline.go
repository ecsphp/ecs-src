package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Files/EndFileNewlineSniff.php
//
// PSR2EndFileNewline ends a PHP-only file with exactly one newline: a missing
// one is added, extra blank lines at the end are removed.
type PSR2EndFileNewline struct{}

func (PSR2EndFileNewline) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Files\EndFileNewlineSniff`
}

func (PSR2EndFileNewline) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Files/EndFileNewlineSniff.php"
}

func (PSR2EndFileNewline) Fix(s *tokens.Stream) bool {
	if phpcsInlineHTMLAfterOpenTag(s) {
		return false
	}
	last := phpcsLastToken(s)
	if last < 0 {
		return false
	}
	eol := phpcsEolChar(s)
	if v := s.At(last).Value; !strings.HasSuffix(v, "\n") {
		s.SetValue(last, v+eol)
		return true
	}
	// the last non-whitespace character and the newlines after it
	code, cut := -1, 0
	for i := last; i >= 0 && code < 0; i-- {
		v := s.At(i).Value
		if j := strings.LastIndexFunc(v, func(r rune) bool { return !strings.ContainsRune(" \t\r\n", r) }); j >= 0 {
			code, cut = i, j+1
		}
	}
	if code < 0 {
		return false
	}
	newlines := strings.Count(s.At(code).Value[cut:], "\n")
	for i := code + 1; i <= last; i++ {
		newlines += strings.Count(s.At(i).Value, "\n")
	}
	if newlines <= 1 {
		return false
	}
	s.SetValue(code, s.At(code).Value[:cut]+eol)
	for i := last; i > code; i-- {
		s.RemoveAt(i)
	}
	return true
}

// phpcsInlineHTMLAfterOpenTag is false when the file has no open tag, as the
// sniff then never runs; the newline PHP folds into a "?>" is not inline HTML.
func phpcsInlineHTMLAfterOpenTag(s *tokens.Stream) bool {
	open := -1
	for i := range s.Len() {
		if s.At(i).Kind == token.OpenTag {
			open = i
			break
		}
	}
	if open < 0 {
		return true
	}
	for i := open + 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.InlineHTML {
			continue
		}
		v := t.Value
		if s.At(i-1).Kind == token.CloseTag {
			if rest, ok := strings.CutPrefix(v, "\r\n"); ok {
				v = rest
			} else {
				v = strings.TrimPrefix(v, "\n")
			}
		}
		if v != "" {
			return true
		}
	}
	return false
}
