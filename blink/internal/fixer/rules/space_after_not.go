package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceAfterNotSniff.php
//
// SpaceAfterNot (registered with Spacing 1, the sniff's default) puts exactly
// Spacing spaces after "!": "!$a" -> "! $a".
type SpaceAfterNot struct {
	Spacing        int
	IgnoreNewlines bool
}

func (SpaceAfterNot) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Formatting\SpaceAfterNotSniff`
}

func (SpaceAfterNot) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceAfterNotSniff.php"
}

func (f SpaceAfterNot) WithConfig(config map[string]any) fixer.Fixer {
	f.Spacing = configInt(config, "spacing", f.Spacing)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

func (f SpaceAfterNot) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "!" {
			continue
		}
		if phpcsSpaceAfter(s, i, f.Spacing, f.IgnoreNewlines) {
			changed = true
		}
	}
	return changed
}
