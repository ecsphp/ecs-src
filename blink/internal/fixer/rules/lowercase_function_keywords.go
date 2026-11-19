package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/LowercaseFunctionKeywordsSniff.php
//
// LowercaseFunctionKeywords lowercases "function", "fn" and the method
// modifiers: "Final STATIC PUBLIC Function" -> "final static public function".
type LowercaseFunctionKeywords struct{}

func (LowercaseFunctionKeywords) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\LowercaseFunctionKeywordsSniff`
}

func (LowercaseFunctionKeywords) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/LowercaseFunctionKeywordsSniff.php"
}

var functionKeywords = map[string]bool{
	"private": true, "public": true, "protected": true, "abstract": true,
	"static": true, "final": true, "function": true, "fn": true,
}

func (LowercaseFunctionKeywords) Fix(s *tokens.Stream) bool {
	return lowercaseKeywordsIn(s, functionKeywords)
}

// lowercaseKeywordsIn lowercases every keyword token whose lowercase form is in words.
func lowercaseKeywordsIn(s *tokens.Stream, words map[string]bool) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		lower := lowerASCII(t.Value)
		if lower == t.Value || !words[lower] {
			continue
		}
		// "use function" / "use const" imports are not the declaration keywords
		if prev, ok := prevSignificant(s, i); ok && (lower == "function" || lower == "const") && strings.EqualFold(prev.Value, "use") {
			continue
		}
		s.SetValue(i, lower)
		changed = true
	}
	return changed
}
