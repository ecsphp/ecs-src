package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/ArrayNotation/ArrayListItemNewlineFixer.php
//
// ArrayListItemNewline puts each item of an associative array literal (one that
// contains a "=>") on its own line, with "[" and "]" on their own lines. Plain
// list arrays (no "=>") are left as they are. array_indentation then aligns the
// result, so this fixer only needs to establish the one-item-per-line structure.
type ArrayListItemNewline struct{}

func (ArrayListItemNewline) Name() string {
	return `Symplify\CodingStandard\Fixer\ArrayNotation\ArrayListItemNewlineFixer`
}

func (ArrayListItemNewline) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/ArrayNotation/ArrayListItemNewlineFixer.php"
}

func (ArrayListItemNewline) Fix(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "[" {
			continue
		}
		if !isArrayLiteralOpen(s, open) {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue // empty []
		}
		if arrayTopLevelMultiline(s, open, closeIdx) {
			continue // items already on their own lines: array_indentation aligns it
		}
		if !arrayHasTopLevelArrow(s, open, closeIdx) {
			continue // plain list array: leave inline
		}
		// a multiline array carries a trailing comma; add it before reflowing so
		// the last item ends up on its own line with the comma
		if last := sigPrev(s, closeIdx); last > open && s.At(last).Value != "," {
			s.InsertAt(last+1, token.Token{Kind: token.Punct, Value: ","})
			closeIdx++
			changed = true
		}
		if reflowParen(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// arrayTopLevelMultiline reports whether the array already has a newline at its
// own nesting level (an item on its own line). Newlines only inside a nested
// element (e.g. a match block) do not count.
func arrayTopLevelMultiline(s *tokens.Stream, open, closeIdx int) bool {
	depth := 0
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
			}
		}
		if depth == 0 && t.Kind == token.Whitespace && hasNewline(t.Value) {
			return true
		}
	}
	return false
}

// arrayHasTopLevelArrow reports whether a "=>" appears at the array's own nesting
// level (an associative key or an arrow-function element).
func arrayHasTopLevelArrow(s *tokens.Stream, open, closeIdx int) bool {
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case "=>":
			if depth == 0 {
				return true
			}
		}
	}
	return false
}
