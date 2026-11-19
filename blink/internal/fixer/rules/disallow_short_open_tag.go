package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowShortOpenTagSniff.php
//
// DisallowShortOpenTag rewrites "<?=" to "<?php echo". A bare "<?" is left
// alone: PHP under ECS runs with short_open_tag off, so it is inline HTML.
type DisallowShortOpenTag struct{}

func (DisallowShortOpenTag) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\DisallowShortOpenTagSniff`
}

func (DisallowShortOpenTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowShortOpenTagSniff.php"
}

func (DisallowShortOpenTag) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.OpenTag || t.Value != "<?=" {
			continue
		}
		next := phpcsNextNonEmpty(s, i+1)
		if next >= s.Len() || s.At(next).Kind == token.CloseTag {
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.SetValue(i, "<?php echo")
		} else {
			s.SetValue(i, "<?php echo ")
		}
		changed = true
	}
	return changed
}
