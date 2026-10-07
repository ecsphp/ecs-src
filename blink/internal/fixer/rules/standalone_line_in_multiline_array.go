package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/ArrayNotation/StandaloneLineInMultilineArrayFixer.php
//
// StandaloneLineInMultilineArray puts each item of an associative array literal
// on its own line. Unlike ArrayListItemNewline it also reflows arrays that are
// already spread across lines, and it does not add a trailing comma. Matching
// the upstream fix(), the first array that must be skipped (a non-associative
// array, or a lone non-array item not introduced by "=>") stops the whole pass.
type StandaloneLineInMultilineArray struct{}

func (StandaloneLineInMultilineArray) Name() string {
	return `Symplify\CodingStandard\Fixer\ArrayNotation\StandaloneLineInMultilineArrayFixer`
}

func (StandaloneLineInMultilineArray) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/ArrayNotation/StandaloneLineInMultilineArrayFixer.php"
}

func (StandaloneLineInMultilineArray) Fix(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "[" {
			continue
		}
		if !isArrayLiteralOpen(s, open) {
			continue
		}
		if isDestructuringAssignOpen(s, open) {
			continue // destructuring target, not an array literal
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue // empty []
		}
		if standaloneArrayShouldSkip(s, open, closeIdx) {
			return changed // upstream fix() returns on the first skipped array
		}
		if reflowParen(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// standaloneArrayShouldSkip mirrors shouldSkipNestedArrayValue.
func standaloneArrayShouldSkip(s *tokens.Stream, open, closeIdx int) bool {
	if !arrayHasTopLevelArrow(s, open, closeIdx) {
		return true // not an associative array
	}
	count, firstIsArray := arrayItemInfo(s, open, closeIdx)
	if count == 1 && !firstIsArray {
		prev := sigPrev(s, open)
		if prev < 0 {
			return false
		}
		return s.At(prev).Value != "=>"
	}
	return false
}

// arrayItemInfo returns the top-level item count and whether the first item's
// value is itself an array.
func arrayItemInfo(s *tokens.Stream, open, closeIdx int) (count int, firstIsArray bool) {
	count = 1
	depth := 0
	firstArrowChecked := false
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
				continue
			case ")", "]", "}":
				depth--
				continue
			case ",":
				if depth == 0 && sigNext(s, j) != closeIdx {
					count++
				}
				continue
			case "=>":
				if depth == 0 && !firstArrowChecked {
					firstArrowChecked = true
					if v := sigNext(s, j); v >= 0 && s.At(v).Kind == token.Punct && s.At(v).Value == "[" {
						firstIsArray = true
					}
				}
				continue
			}
		}
	}
	return count, firstIsArray
}
