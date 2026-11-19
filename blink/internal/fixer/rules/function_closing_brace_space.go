package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionClosingBraceSpaceSniff.php
//
// FunctionClosingBraceSpace puts exactly one blank line before the closing
// brace of a function, and none before the brace of a nested function or a
// closure passed as an argument.
type FunctionClosingBraceSpace struct{}

func (FunctionClosingBraceSpace) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\FunctionClosingBraceSpaceSniff`
}

func (FunctionClosingBraceSpace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionClosingBraceSpaceSniff.php"
}

func (FunctionClosingBraceSpace) Fix(s *tokens.Stream) bool {
	return phpcsRunUntilStable(s, functionClosingBraceSpacePass)
}

func functionClosingBraceSpacePass(s *tokens.Stream) bool {
	type body struct{ keyword, open, close int }
	var bodies []body
	for i := range s.Len() {
		if open, closeIdx, ok := phpcsFunctionBody(s, i); ok {
			bodies = append(bodies, body{i, open, closeIdx})
		}
	}
	nested := map[int]bool{}
	for _, b := range bodies {
		if phpcsInnermostParen(s, b.keyword) >= 0 {
			nested[b.keyword] = true
			continue
		}
		for _, o := range bodies {
			if o.open < b.keyword && b.keyword < o.close {
				nested[b.keyword] = true
				break
			}
		}
	}
	eol := phpcsEolChar(s)
	changed := false
	// last to first, so the keywords still to visit keep their indices
	for _, b := range slices.Backward(bodies) {
		_, closeIdx, ok := phpcsFunctionBody(s, b.keyword)
		if ok && functionClosingBraceFix(s, closeIdx, nested[b.keyword], eol) {
			changed = true
		}
	}
	return changed
}

func functionClosingBraceFix(s *tokens.Stream, closeIdx int, nested bool, eol string) bool {
	prev := closeIdx - 1
	for prev >= 0 && s.At(prev).Kind == token.Whitespace {
		prev--
	}
	if prev < 0 {
		return false
	}
	lines := phpcsTokenLines(s)
	found := lines[closeIdx] - lines[prev] - 1
	w := -1
	if prev+1 < closeIdx {
		w = prev + 1
	}
	if nested {
		switch {
		case found < 0:
			functionClosingBraceNewlineBefore(s, closeIdx, w, eol)
			return true
		case found > 0 && w >= 0:
			keep := phpcsCommentNewline(s, w)
			v := s.At(w).Value
			pieces := phpcsWhitespacePieces(v[keep:])
			line := lines[prev] + phpcsLineBreaks(s.At(prev).Value+v[:keep])
			drop := 0
			for _, p := range pieces {
				if line == lines[closeIdx]-1 {
					break
				}
				drop++
				line += phpcsLineBreaks(p)
			}
			if drop == 0 {
				drop = 1
			}
			s.SetValue(w, v[:keep]+strings.Join(pieces[drop:], ""))
			return true
		}
		return false
	}
	if found == 1 {
		return false
	}
	if found > 1 {
		if w < 0 {
			return false
		}
		s.SetValue(w, s.At(w).Value[:phpcsCommentNewline(s, w)]+eol)
		return true
	}
	if phpcsWhitespaceAt(s, w) {
		v := s.At(w).Value
		p := phpcsLastPieceStart(v)
		s.SetValue(w, v[:p]+eol+v[p:])
		return true
	}
	functionClosingBraceNewlineBefore(s, closeIdx, w, eol)
	return true
}

// functionClosingBraceNewlineBefore adds a newline right before the brace.
func functionClosingBraceNewlineBefore(s *tokens.Stream, closeIdx, w int, eol string) {
	if w >= 0 {
		s.SetValue(w, s.At(w).Value+eol)
		return
	}
	s.InsertAt(closeIdx, token.Token{Kind: token.Whitespace, Value: eol})
}

// phpcsWhitespacePieces splits whitespace into PHP_CodeSniffer's per-line tokens.
func phpcsWhitespacePieces(v string) []string {
	var pieces []string
	for v != "" {
		p := phpcsThroughFirstBreak(v)
		pieces = append(pieces, p)
		v = v[len(p):]
	}
	return pieces
}
