package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/ArrayNotation/ArrayOpenerAndCloserNewlineFixer.php
//
// ArrayOpenerAndCloserNewline puts the "[" and "]" of an associative array (one
// with a top-level "=>") on their own lines, leaving the items untouched. It
// skips an array whose first element is itself an array opener.
type ArrayOpenerAndCloserNewline struct{}

func (ArrayOpenerAndCloserNewline) Name() string {
	return `Symplify\CodingStandard\Fixer\ArrayNotation\ArrayOpenerAndCloserNewlineFixer`
}

func (ArrayOpenerAndCloserNewline) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/ArrayNotation/ArrayOpenerAndCloserNewlineFixer.php"
}

func (ArrayOpenerAndCloserNewline) Fix(s *tokens.Stream) bool {
	changed := false
	// high-to-low so inserts inside inner arrays don't shift outer indices we
	// have yet to reach
	for open := s.Len() - 1; open >= 0; open-- {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "[" {
			continue
		}
		if !isArrayLiteralOpen(s, open) {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		first := sigNext(s, open)
		if first < 0 || first == closeIdx {
			continue // no items
		}
		// first element is itself an array opener: leave it
		if s.At(first).Kind == token.Punct && s.At(first).Value == "[" && isArrayLiteralOpen(s, first) {
			continue
		}
		if !arrayHasTopLevelArrow(s, open, closeIdx) {
			continue // not associative
		}
		// closer before opener, so the opener insert does not shift the closer.
		// only act when the border is not already on its own line.
		closerOnOwnLine := closeIdx > 0 && s.At(closeIdx-1).Kind == token.Whitespace && hasNewline(s.At(closeIdx-1).Value)
		if !closerOnOwnLine && editSlotBefore(s, closeIdx, "\n") {
			changed = true
		}
		openerOnOwnLine := open+1 < s.Len() && s.At(open+1).Kind == token.Whitespace && hasNewline(s.At(open+1).Value)
		if !openerOnOwnLine && editSlotAfter(s, open, "\n") {
			changed = true
		}
	}
	return changed
}
