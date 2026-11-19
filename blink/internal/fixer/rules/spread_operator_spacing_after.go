package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/SpreadOperatorSpacingAfterSniff.php
//
// SpreadOperatorSpacingAfter puts exactly Spacing spaces (default 0) after "...":
// "... $args" -> "...$args". First-class callables ("foo(...)") are left alone.
type SpreadOperatorSpacingAfter struct {
	Spacing        int
	IgnoreNewlines bool
}

func (SpreadOperatorSpacingAfter) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\SpreadOperatorSpacingAfterSniff`
}

func (SpreadOperatorSpacingAfter) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/SpreadOperatorSpacingAfterSniff.php"
}

func (f SpreadOperatorSpacingAfter) WithConfig(config map[string]any) fixer.Fixer {
	f.Spacing = configInt(config, "spacing", f.Spacing)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

func (f SpreadOperatorSpacingAfter) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "..." {
			continue
		}
		if next := phpcsNextNonEmpty(s, i+1); next < s.Len() && s.At(next).Kind == token.Punct && s.At(next).Value == ")" {
			continue
		}
		if phpcsSpaceAfter(s, i, f.Spacing, f.IgnoreNewlines) {
			changed = true
		}
	}
	return changed
}
