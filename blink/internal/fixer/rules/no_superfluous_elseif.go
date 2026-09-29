package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoSuperfluousElseifFixer.php
//
// NoSuperfluousElseif replaces `elseif` with `if` when every preceding branch of
// the if-chain always exits (return/throw/break/continue/exit/goto), so the
// elseif is not actually reachable as an else. It ports the flow analysis from
// AbstractNoUselessElseFixer.
type NoSuperfluousElseif struct{}

func (NoSuperfluousElseif) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoSuperfluousElseifFixer`
}

func (NoSuperfluousElseif) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoSuperfluousElseifFixer.php"
}

var nseTrailingIndentRe = regexp.MustCompile(`(?:\r\n|\r|\n)[^\r\n]*$`)

func (NoSuperfluousElseif) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if nseIsElseif(s, i) && nseIsSuperfluousElse(s, i) {
			nseConvertElseifToIf(s, i)
			changed = true
		}
	}
	return changed
}

func nseIsElseif(s *tokens.Stream, i int) bool {
	if kwIs(s.At(i), "elseif") {
		return true
	}
	if kwIs(s.At(i), "else") {
		n := sigNext(s, i)
		return n >= 0 && kwIs(s.At(n), "if")
	}
	return false
}

func nseConvertElseifToIf(s *tokens.Stream, index int) {
	if kwIs(s.At(index), "else") {
		// clearTokenAndMergeSurroundingWhitespace: merge the whitespace on both
		// sides of the removed "else" into one; index then lands on the "if".
		before, after := index-1, index+1
		if before >= 0 && s.At(before).Kind == token.Whitespace &&
			after < s.Len() && s.At(after).Kind == token.Whitespace {
			s.SetValue(before, s.At(before).Value+s.At(after).Value)
			s.RemoveAt(after)
		}
		s.RemoveAt(index)
	} else {
		s.Set(index, token.Token{Kind: token.Keyword, Value: "if"})
	}

	whitespace := ""
	for previous := index - 1; previous > 0; previous-- {
		t := s.At(previous)
		if t.Kind == token.Whitespace {
			if m := nseTrailingIndentRe.FindString(t.Value); m != "" {
				whitespace = m
				break
			}
		}
	}
	if whitespace == "" {
		return
	}

	if index-1 < 0 {
		return
	}
	prev := s.At(index - 1)
	if prev.Kind != token.Whitespace {
		s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: whitespace})
	} else if !strings.ContainsAny(prev.Value, "\r\n") {
		s.SetValue(index-1, whitespace)
	}
}

func nseIsSuperfluousElse(s *tokens.Stream, index int) bool {
	previousBlockStart := index
	for {
		var previousBlockEnd int
		previousBlockStart, previousBlockEnd = nseGetPreviousBlock(s, previousBlockStart)
		if previousBlockStart < 0 || previousBlockEnd < 0 {
			return false
		}

		previous := previousBlockEnd
		if isPunctVal(s, previous, "}") {
			previous = sigPrev(s, previous)
		}
		if previous < 0 {
			return false
		}
		// 'if' block doesn't end with ';' -> keep; empty '{ }' block -> keep
		if !isPunctVal(s, previous, ";") || isPunctVal(s, sigPrev(s, previous), "{") {
			return false
		}

		candidateIndex := nsePrevTokenOfKind(s, previous, nseTerminators)
		if candidateIndex < 0 {
			return false
		}
		cand := s.At(candidateIndex)
		if isPunctVal(s, candidateIndex, ";") || cand.Kind == token.CloseTag || kwIs(cand, "if") {
			return false
		}

		if kwIs(cand, "throw") {
			pi := sigPrev(s, candidateIndex)
			if !isPunctVal(s, pi, ";") && !isPunctVal(s, pi, "{") {
				return false
			}
		}

		if nseIsInConditional(s, candidateIndex, previousBlockStart) ||
			nseIsInConditionWithoutBraces(s, candidateIndex, previousBlockStart) {
			return false
		}

		if kwIs(s.At(previousBlockStart), "if") {
			break
		}
	}
	return true
}

func nseGetPreviousBlock(s *tokens.Stream, index int) (int, int) {
	closeIdx := sigPrev(s, index)
	previous := closeIdx
	if closeIdx >= 0 && isPunctVal(s, closeIdx, "}") {
		previous = s.MatchBackward(closeIdx)
	}
	if previous < 0 {
		return -1, closeIdx
	}
	open := nsePrevTokenOfKind(s, previous, nseIfElseElseif)
	if open < 0 {
		return -1, closeIdx
	}
	if kwIs(s.At(open), "if") {
		elseCandidate := sigPrev(s, open)
		if elseCandidate >= 0 && kwIs(s.At(elseCandidate), "else") {
			open = elseCandidate
		}
	}
	return open, closeIdx
}

func nseIsInConditional(s *tokens.Stream, index, lowerLimit int) bool {
	candidateIndex := nsePrevTokenOfKind(s, index, nseParenSemiColon)
	if candidateIndex < 0 {
		return false
	}
	if isPunctVal(s, candidateIndex, ":") {
		return true
	}
	if !isPunctVal(s, candidateIndex, ")") {
		return false
	}
	open := s.MatchBackward(candidateIndex)
	if open < 0 {
		return false
	}
	return sigPrev(s, open) > lowerLimit
}

func nseIsInConditionWithoutBraces(s *tokens.Stream, index, lowerLimit int) bool {
	for index > lowerLimit {
		if k := s.At(index).Kind; k == token.Comment || k == token.DocComment || k == token.Whitespace {
			index = sigPrev(s, index)
			if index < 0 {
				return false
			}
		}
		t := s.At(index)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "if", "elseif", "else":
				return true
			}
		}
		if isPunctVal(s, index, ";") {
			return false
		}
		if isPunctVal(s, index, "{") {
			index = sigPrev(s, index)
			if index >= 0 && kwIs(s.At(index), "do") {
				index--
				continue
			}
			if !isPunctVal(s, index, ")") {
				return false
			}
			index = s.MatchBackward(index)
			if index < 0 {
				return false
			}
			index = sigPrev(s, index)
			if index >= 0 && (kwIs(s.At(index), "if") || kwIs(s.At(index), "elseif")) {
				return false
			}
		} else if isPunctVal(s, index, ")") {
			index = s.MatchBackward(index)
			if index < 0 {
				return false
			}
			index = sigPrev(s, index)
		} else {
			index--
		}
	}
	return false
}

// token-kind specs for nsePrevTokenOfKind
type nseKind struct {
	kw    string // keyword name (lowercase), or ""
	punct string // punct value, or ""
	close bool   // matches token.CloseTag
}

var nseTerminators = []nseKind{
	{punct: ";"}, {kw: "break"}, {close: true}, {kw: "continue"},
	{kw: "exit"}, {kw: "die"}, {kw: "goto"}, {kw: "if"}, {kw: "return"}, {kw: "throw"},
}
var nseIfElseElseif = []nseKind{{kw: "if"}, {kw: "else"}, {kw: "elseif"}}
var nseParenSemiColon = []nseKind{{punct: ")"}, {punct: ";"}, {punct: ":"}}

func nseMatchKind(t token.Token, k nseKind) bool {
	if k.close {
		return t.Kind == token.CloseTag
	}
	if k.kw != "" {
		return t.Kind == token.Keyword && strings.EqualFold(t.Value, k.kw)
	}
	return t.Kind == token.Punct && t.Value == k.punct
}

// nsePrevTokenOfKind mirrors Tokens::getPrevTokenOfKind: scan back from before
// idx for the first token matching any spec.
func nsePrevTokenOfKind(s *tokens.Stream, idx int, kinds []nseKind) int {
	for j := idx - 1; j >= 0; j-- {
		t := s.At(j)
		for _, k := range kinds {
			if nseMatchKind(t, k) {
				return j
			}
		}
	}
	return -1
}
