package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Arrays/ArrayBracketSpacingSniff.php
//
// ArrayBracketSpacing removes whitespace before an index bracket and inside it:
// "$a [ 0 ]" -> "$a[0]". Short arrays and brackets inside strings are left alone.
type ArrayBracketSpacing struct{}

func (ArrayBracketSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Arrays\ArrayBracketSpacingSniff`
}

func (ArrayBracketSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Arrays/ArrayBracketSpacingSniff.php"
}

func (ArrayBracketSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	inString := 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.String {
			switch t.Part {
			case token.Head:
				inString++
			case token.Tail:
				inString--
			}
			continue
		}
		if inString > 0 || t.Kind != token.Punct || t.Value != "[" || phpcsShortArrayOpen(s, i) {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		if closeIdx > 0 && phpcsWhitespaceAt(s, closeIdx-1) {
			dropPhpcsWhitespace(s, closeIdx-1)
			changed = true
		}
		if s.At(i+1).Kind == token.Whitespace {
			s.RemoveAt(i + 1)
			changed = true
		}
		if phpcsWhitespaceAt(s, i-1) {
			if dropPhpcsWhitespace(s, i-1) {
				i--
			}
			changed = true
		}
	}
	return changed
}
