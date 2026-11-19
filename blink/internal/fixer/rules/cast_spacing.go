package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/CastSpacingSniff.php
//
// CastSpacing removes the spaces and tabs inside a cast: "( int )" -> "(int)".
type CastSpacing struct{}

func (CastSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\CastSpacingSniff`
}

func (CastSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/CastSpacingSniff.php"
}

func (CastSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		closeIdx, _, ok := phpCastAt(s, i)
		if !ok {
			continue
		}
		for j := closeIdx - 1; j > i; j-- {
			if isCastPadding(s.At(j)) {
				s.RemoveAt(j)
				closeIdx--
				changed = true
			}
		}
		i = closeIdx
	}
	return changed
}
