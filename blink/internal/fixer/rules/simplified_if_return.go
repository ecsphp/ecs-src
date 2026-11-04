package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SimplifiedIfReturnFixer.php
//
// SimplifiedIfReturn collapses an "if" whose only job is to return the boolean
// of its condition: "if ($foo) { return true; } return false;" becomes
// "return (bool) ($foo);" (or "return ! ($foo);" when the booleans are flipped).
type SimplifiedIfReturn struct{}

func (SimplifiedIfReturn) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SimplifiedIfReturnFixer`
}

func (SimplifiedIfReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SimplifiedIfReturnFixer.php"
}

func (SimplifiedIfReturn) Fix(s *tokens.Stream) bool {
	changed := false
	for ifIndex := s.Len() - 1; ifIndex >= 0; ifIndex-- {
		t := s.At(ifIndex)
		if t.Kind != token.Keyword {
			continue
		}
		if lv := strings.ToLower(t.Value); lv != "if" && lv != "elseif" {
			continue
		}

		// skip an "if" that is the body of a braceless loop
		if p := aliasPrevMeaningful(s, ifIndex); p >= 0 && isPunctVal(s, p, ")") {
			continue
		}

		startParen := nextPunctOfKind(s, ifIndex, "(")
		if startParen < 0 {
			continue
		}
		endParen := s.MatchForward(startParen)
		if endParen < 0 {
			continue
		}
		firstCandidate := aliasNextMeaningful(s, endParen)
		if firstCandidate < 0 {
			continue
		}

		indices, isNegative, ok := sirMatchReturnSequence(s, firstCandidate)
		if !ok {
			continue
		}

		lastSemicolon := indices[len(indices)-1]
		toClear := slices.Clone(indices[:len(indices)-1])
		slices.Sort(toClear)
		slices.Reverse(toClear)
		for _, idx := range toClear {
			clearTokenAndMergeWS(s, idx)
		}
		removeWhitespaceSafely(s, lastSemicolon, -1)

		var newTokens []token.Token
		if isNegative {
			newTokens = []token.Token{
				{Kind: token.Keyword, Value: "return"},
				{Kind: token.Whitespace, Value: " "},
				{Kind: token.Punct, Value: "!"},
			}
		} else {
			newTokens = []token.Token{
				{Kind: token.Keyword, Value: "return"},
				{Kind: token.Whitespace, Value: " "},
				{Kind: token.Punct, Value: "("},
				{Kind: token.Ident, Value: "bool"},
				{Kind: token.Punct, Value: ")"},
			}
		}
		s.ReplaceRange(ifIndex, ifIndex, newTokens)
		changed = true
	}
	return changed
}

// sirMatchReturnSequence ports SimplifiedIfReturnFixer::matchReturnSequence: it
// recognizes an optional "{ ... }" wrapped "return true; } return false;" pair
// after the if condition. On success it returns the token indices making up the
// sequence (the last being the final semicolon to preserve), and whether the
// booleans are flipped ("false" first).
func sirMatchReturnSequence(s *tokens.Stream, start int) ([]int, bool, bool) {
	var indices []int

	if isPunctVal(s, start, "{") {
		indices = append(indices, start)
		start = aliasNextMeaningful(s, start)
		if start < 0 {
			return nil, false, false
		}
	}

	// first return
	if !kwIs(s.At(start), "return") {
		return nil, false, false
	}
	indices = append(indices, start)

	bool1 := aliasNextMeaningful(s, start)
	if bool1 < 0 || !sirIsBool(s, bool1) {
		return nil, false, false
	}
	value1 := strings.ToLower(s.At(bool1).Value)

	semi1 := aliasNextMeaningful(s, bool1)
	if semi1 < 0 || !isPunctVal(s, semi1, ";") {
		return nil, false, false
	}
	indices = append(indices, bool1, semi1)

	next := aliasNextMeaningful(s, semi1)
	if next < 0 {
		return nil, false, false
	}
	if isPunctVal(s, next, "}") {
		indices = append(indices, next)
		next = aliasNextMeaningful(s, next)
		if next < 0 {
			return nil, false, false
		}
	}

	// second return
	isNegative := value1 == "false"
	if !kwIs(s.At(next), "return") {
		return nil, false, false
	}
	indices = append(indices, next)

	bool2 := aliasNextMeaningful(s, next)
	want := "false"
	if isNegative {
		want = "true"
	}
	if bool2 < 0 || s.At(bool2).Kind != token.Ident || strings.ToLower(s.At(bool2).Value) != want {
		return nil, false, false
	}

	semi2 := aliasNextMeaningful(s, bool2)
	if semi2 < 0 || !isPunctVal(s, semi2, ";") {
		return nil, false, false
	}
	indices = append(indices, bool2, semi2)

	return indices, isNegative, true
}

func sirIsBool(s *tokens.Stream, i int) bool {
	if s.At(i).Kind != token.Ident {
		return false
	}
	v := strings.ToLower(s.At(i).Value)
	return v == "true" || v == "false"
}
