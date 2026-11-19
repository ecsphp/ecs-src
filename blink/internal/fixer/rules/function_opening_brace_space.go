package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionOpeningBraceSpaceSniff.php
//
// FunctionOpeningBraceSpace removes blank lines after a function's opening brace.
// An empty body is left alone.
type FunctionOpeningBraceSpace struct{}

func (FunctionOpeningBraceSpace) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\FunctionOpeningBraceSpaceSniff`
}

func (FunctionOpeningBraceSpace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionOpeningBraceSpaceSniff.php"
}

func (FunctionOpeningBraceSpace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		open, closeIdx, ok := phpcsFunctionBody(s, i)
		if !ok {
			continue
		}
		ws := open + 1
		if ws+1 >= closeIdx || s.At(ws).Kind != token.Whitespace {
			continue
		}
		if phpcsLineBreaks(s.At(ws).Value) < 2 {
			continue
		}
		s.SetValue(ws, phpcsEolChar(s)+phpcsAfterLastBreak(s.At(ws).Value))
		changed = true
	}
	return changed
}
