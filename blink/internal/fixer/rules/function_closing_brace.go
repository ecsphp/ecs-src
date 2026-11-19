package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/FunctionClosingBraceSniff.php
//
// FunctionClosingBrace removes blank lines before a function's closing brace,
// keeping the brace's indentation. A brace sharing a line with code is left alone.
type FunctionClosingBrace struct{}

func (FunctionClosingBrace) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Methods\FunctionClosingBraceSniff`
}

func (FunctionClosingBrace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/FunctionClosingBraceSniff.php"
}

func (FunctionClosingBrace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		_, closeIdx, ok := phpcsFunctionBody(s, i)
		if !ok {
			continue
		}
		ws := closeIdx - 1
		if s.At(ws).Kind != token.Whitespace || phpcsLineBreaks(s.At(ws).Value) < 2 {
			continue
		}
		v := s.At(ws).Value
		s.SetValue(ws, phpcsThroughFirstBreak(v)+phpcsAfterLastBreak(v))
		changed = true
	}
	return changed
}
