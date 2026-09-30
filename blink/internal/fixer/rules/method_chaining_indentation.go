package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/MethodChainingIndentationFixer.php
//
// MethodChainingIndentation aligns a chained "->"/"?->" that already sits at the
// start of a line to one indentation level (four spaces) past the line the chain
// starts on. It only adjusts the indentation of an existing line break, so it
// never joins or splits calls and is a no-op once the chain is aligned.
type MethodChainingIndentation struct{}

func (MethodChainingIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\MethodChainingIndentationFixer`
}

func (MethodChainingIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/MethodChainingIndentationFixer.php"
}

func (MethodChainingIndentation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "->" && t.Value != "?->") {
			continue
		}
		ws := s.At(i - 1)
		if ws.Kind != token.Whitespace || !strings.Contains(ws.Value, "\n") {
			continue // only operators that already begin a line
		}
		want := chainExpectedIndent(s, i)
		nl := strings.LastIndexByte(ws.Value, '\n')
		got := ws.Value[nl+1:]
		if got == want {
			continue
		}
		s.SetValue(i-1, ws.Value[:nl+1]+want)
		changed = true

		// Shift the nested lines inside this chained call's "(...)" by the same
		// delta, so multiline arguments keep their alignment relative to the "->"
		// line that just moved (mirrors php-cs-fixer's MethodChainingIndentation).
		if chainIndentShiftNested(s, i, got, want) {
			changed = true
		}
	}
	return changed
}

// chainExpectedIndent computes the indent a chained "->" line should have,
// mirroring php-cs-fixer's getExpectedIndentAt: walk back from the token before
// the operator, jumping over "(...)" groups, to the first line-starting token,
// and add one level when that line requires it.
func chainExpectedIndent(s *tokens.Stream, opIdx int) string {
	end := prevMeaningfulIndex(s, opIdx)
	if end < 0 {
		return "    "
	}
	for i := end; i >= 0; i-- {
		if s.At(i).Kind == token.Punct && s.At(i).Value == ")" {
			if o := s.MatchBackward(i); o >= 0 {
				i = o
			}
		}
		ind, ok := chainIndentAt(s, i)
		if !ok {
			continue
		}
		if chainRequiresExtraIndent(s, i, end) {
			return ind + "    "
		}
		return ind
	}
	return "    "
}

// prevMeaningfulIndex skips whitespace and comments, mirroring php-cs-fixer's
// getPrevMeaningfulToken.
func prevMeaningfulIndex(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if k := s.At(j).Kind; k != token.Whitespace && k != token.Comment && k != token.DocComment {
			return j
		}
	}
	return -1
}

// chainIndentAt returns the indent carried by the whitespace token at i, or ok
// false when i is not a newline-bearing whitespace token.
func chainIndentAt(s *tokens.Stream, i int) (string, bool) {
	t := s.At(i)
	if t.Kind != token.Whitespace || !strings.Contains(t.Value, "\n") {
		return "", false
	}
	return t.Value[strings.LastIndexByte(t.Value, '\n')+1:], true
}

// chainRequiresExtraIndent mirrors php-cs-fixer's currentLineRequiresExtraIndentLevel.
func chainRequiresExtraIndent(s *tokens.Stream, start, end int) bool {
	first := nextMeaningfulIndex(s, start)
	if first < 0 {
		return false
	}
	if s.At(first).Kind == token.Punct && (s.At(first).Value == "->" || s.At(first).Value == "?->") {
		m := nextMeaningfulIndex(s, first)
		if m < 0 {
			return false
		}
		third := nextMeaningfulIndex(s, m)
		if third < 0 || s.At(third).Value != "(" {
			return false
		}
		return s.MatchForward(third) > end
	}
	if !(s.At(end).Kind == token.Punct && s.At(end).Value == ")") {
		return true
	}
	return s.MatchBackward(end) >= start
}

// chainIndentShiftNested reparents every newline-started line inside the call
// opened right after the "->" at opIdx: a line whose indent begins with the old
// indent gets that prefix rewritten to the new indent.
func chainIndentShiftNested(s *tokens.Stream, opIdx int, oldIndent, newIndent string) bool {
	name := nextSignificantIndex(s, opIdx)
	if name < 0 {
		return false
	}
	open := nextSignificantIndex(s, name)
	if open < 0 || s.At(open).Value != "(" {
		return false
	}
	close := s.MatchForward(open)
	if close < 0 {
		return false
	}
	changed := false
	for j := open + 1; j < close; j++ {
		t := s.At(j)
		if t.Kind != token.Whitespace || !strings.Contains(t.Value, "\n") {
			continue
		}
		k := strings.LastIndexByte(t.Value, '\n')
		tail := t.Value[k+1:]
		if !strings.HasPrefix(tail, oldIndent) {
			continue
		}
		newTail := newIndent + tail[len(oldIndent):]
		if newTail == tail {
			continue
		}
		s.SetValue(j, t.Value[:k+1]+newTail)
		changed = true
	}
	return changed
}
