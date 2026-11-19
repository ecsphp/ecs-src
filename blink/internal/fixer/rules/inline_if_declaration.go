package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/InlineIfDeclarationSniff.php
//
// InlineIfDeclaration puts one space around the "?" and ":" of a single-line
// ternary ("($a)?'x':'y'" -> "($a) ? 'x' : 'y'"), and none between them in a
// short ternary ("? :" -> "?:"). A ternary spanning lines is left alone.
type InlineIfDeclaration struct{}

func (InlineIfDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\InlineIfDeclarationSniff`
}

func (InlineIfDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/InlineIfDeclarationSniff.php"
}

func (InlineIfDeclaration) Fix(s *tokens.Stream) bool {
	return phpcsRunUntilStable(s, inlineIfDeclarationPass)
}

func inlineIfDeclarationPass(s *tokens.Stream) bool {
	changed := false
	ctx := newPhpcsOperatorContext(s)
	lines := phpcsTokenLines(s)
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "?" || phpcsIsNullable(s, i) {
			continue
		}
		if inlineIfFix(s, i, ctx, lines) {
			changed = true
			ctx = newPhpcsOperatorContext(s)
			lines = phpcsTokenLines(s)
		}
	}
	return changed
}

// inlineIfFix applies the sniff's fixes to the ternary whose "?" is at q, right
// to left so the indices it computed stay valid.
func inlineIfFix(s *tokens.Stream, q int, ctx phpcsOperatorContext, lines []int) bool {
	closeBracket := -1
	if open := phpcsInnermostParen(s, q); open >= 0 {
		closeBracket = s.MatchForward(open)
	}
	firstElse := -1
	for j := q + 1; j < s.Len(); j++ {
		if ctx.inlineElse[j] {
			firstElse = j
			break
		}
	}
	if firstElse < 0 {
		return false
	}
	end := s.Len()
	if closeBracket >= 0 {
		end = closeBracket
	}
	statementEnd := -1
	for j := firstElse + 1; j < end; j++ {
		if v := s.At(j); v.Kind == token.Punct && (v.Value == ";" || v.Value == ",") {
			statementEnd = j
			break
		}
	}
	if statementEnd < 0 {
		if closeBracket < 0 {
			return false
		}
		statementEnd = closeBracket - 1
		for statementEnd >= 0 && s.At(statementEnd).Kind == token.Whitespace {
			statementEnd--
		}
	}
	if statementEnd < 0 || lines[statementEnd] != lines[q] {
		return false
	}

	var elseIdx int
	elvis := false
	if next := inlineIfNextNonWhitespace(s, q+1); next < s.Len() && ctx.inlineElse[next] {
		elseIdx, elvis = next, true
	} else {
		elseIdx = -1
		for j := q + 1; j < statementEnd; j++ {
			if ctx.inlineElse[j] {
				elseIdx = j
				break
			}
		}
		if elseIdx < 0 {
			return false
		}
	}

	// after the ":" first, so the earlier indices stay valid
	changed := inlineIfSetSpaceAfter(s, elseIdx)
	if elvis {
		if elseIdx != q+1 {
			s.RemoveAt(q + 1)
			changed = true
		}
	} else {
		if inlineIfSetSpaceBefore(s, elseIdx) {
			changed = true
		}
		if inlineIfSetSpaceAfter(s, q) {
			changed = true
		}
	}
	if inlineIfSetSpaceBefore(s, q) {
		changed = true
	}
	return changed
}

// inlineIfSetSpaceAfter makes the space after the operator at i one column
// wide, measured to the next non-whitespace token as PHP_CodeSniffer does.
func inlineIfSetSpaceAfter(s *tokens.Stream, i int) bool {
	w := i + 1
	if w >= s.Len() {
		return false
	}
	if s.At(w).Kind != token.Whitespace {
		s.InsertAt(w, token.Token{Kind: token.Whitespace, Value: " "})
		return true
	}
	if strings.ContainsAny(s.At(w).Value, "\r\n") {
		s.SetValue(w, " "+s.At(w).Value[phpcsFirstPieceEnd(s.At(w).Value):])
		return true
	}
	if phpcsTabWidth(s, w) == 1 {
		return false
	}
	s.SetValue(w, " ")
	return true
}

// inlineIfSetSpaceBefore makes the space before the operator at i one column
// wide; across a line break only the last line's piece of the whitespace is
// replaced, PHP_CodeSniffer's whitespace token directly before the operator.
func inlineIfSetSpaceBefore(s *tokens.Stream, i int) bool {
	w := i - 1
	if w < 0 {
		return false
	}
	if s.At(w).Kind != token.Whitespace {
		s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
		return true
	}
	v := s.At(w).Value
	if !strings.ContainsAny(v, "\r\n") {
		if phpcsTabWidth(s, w) == 1 {
			return false
		}
		s.SetValue(w, " ")
		return true
	}
	fixed := v[:phpcsLastPieceStart(v)] + " "
	if fixed == v {
		return false
	}
	s.SetValue(w, fixed)
	return true
}

func inlineIfNextNonWhitespace(s *tokens.Stream, i int) int {
	for i < s.Len() && s.At(i).Kind == token.Whitespace {
		i++
	}
	return i
}

// phpcsFirstPieceEnd is where PHP_CodeSniffer's first whitespace token of v
// ends: just after its first line break.
func phpcsFirstPieceEnd(v string) int {
	return len(phpcsThroughFirstBreak(v))
}

// phpcsLastPieceStart is where PHP_CodeSniffer's last whitespace token of v
// starts: after the last line break, or at the start of the last line when v
// ends with a break.
func phpcsLastPieceStart(v string) int {
	if tail := phpcsAfterLastBreak(v); tail != "" {
		return len(v) - len(tail)
	}
	body := strings.TrimSuffix(strings.TrimSuffix(v, "\n"), "\r")
	if strings.HasSuffix(v, "\r\n") {
		body = v[:len(v)-2]
	}
	return len(body) - len(phpcsAfterLastBreak(body))
}

// phpcsTokenLines is the 1-based line each token starts on.
func phpcsTokenLines(s *tokens.Stream) []int {
	lines := make([]int, s.Len())
	line := 1
	for i := range s.Len() {
		lines[i] = line
		line += phpcsLineBreaks(s.At(i).Value)
	}
	return lines
}
