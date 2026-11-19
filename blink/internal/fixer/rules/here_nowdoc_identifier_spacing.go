package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/HereNowdocIdentifierSpacingSniff.php
//
// HereNowdocIdentifierSpacing removes the spaces and tabs between "<<<" and the
// heredoc/nowdoc identifier: "<<< 'END'" -> "<<<'END'".
type HereNowdocIdentifierSpacing struct{}

func (HereNowdocIdentifierSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\HereNowdocIdentifierSpacingSniff`
}

func (HereNowdocIdentifierSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/HereNowdocIdentifierSpacingSniff.php"
}

func (HereNowdocIdentifierSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || (t.Part != token.Whole && t.Part != token.Head) {
			continue
		}
		prefix, rest := token.SplitBinaryPrefix(t.Value)
		after, ok := strings.CutPrefix(rest, "<<<")
		if !ok {
			continue
		}
		if trimmed := strings.TrimLeft(after, " \t"); trimmed != after {
			s.SetValue(i, prefix+"<<<"+trimmed)
			changed = true
		}
	}
	return changed
}
