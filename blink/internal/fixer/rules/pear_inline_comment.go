package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/InlineCommentSniff.php
//
// PEARInlineComment rewrites Perl-style comments to "//": "# note" -> "// note".
type PEARInlineComment struct{}

func (PEARInlineComment) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Commenting\InlineCommentSniff`
}

func (PEARInlineComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/InlineCommentSniff.php"
}

func (PEARInlineComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "#") {
			continue
		}
		s.SetValue(i, "// "+strings.TrimLeft(t.Value, "# "))
		changed = true
	}
	return changed
}
