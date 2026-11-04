package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/CombineConsecutiveIssetsFixer.php
//
// CombineConsecutiveIssets merges "isset($a) && isset($b)" into "isset($a, $b)".
type CombineConsecutiveIssets struct{}

func (CombineConsecutiveIssets) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\CombineConsecutiveIssetsFixer`
}

func (CombineConsecutiveIssets) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/CombineConsecutiveIssetsFixer.php"
}

func (CombineConsecutiveIssets) Fix(s *tokens.Stream) bool {
	before := s.Render()
	tokenCount := s.Len()
	for index := 1; index < tokenCount; index++ {
		if !kwIs(s.At(index), "isset") {
			continue
		}
		prev := sigPrev(s, index)
		if !combineIssetPrevAllowed(s, prev) {
			continue
		}

		issetInfo := combineIssetInfo(s, index)
		issetCloseBraceIndex := issetInfo[len(issetInfo)-1]
		insertLocation := issetInfo[len(issetInfo)-2] + 1

		booleanAndTokenIndex := sigNext(s, issetCloseBraceIndex)
		for booleanAndTokenIndex >= 0 && isPunctVal(s, booleanAndTokenIndex, "&&") {
			issetIndex := sigNext(s, booleanAndTokenIndex)
			if issetIndex < 0 || !kwIs(s.At(issetIndex), "isset") {
				index = issetIndex
				break
			}

			nextIssetInfo := combineIssetInfo(s, issetIndex)
			nextMeaningful := sigNext(s, nextIssetInfo[len(nextIssetInfo)-1])
			if !combineIssetNextAllowed(s, nextMeaningful) {
				index = nextMeaningful
				break
			}

			// clone the isset arguments (excluding its "(" and ")")
			inner := nextIssetInfo[1 : len(nextIssetInfo)-1]
			clones := make([]token.Token, 0, len(inner)+2)
			clones = append(clones,
				token.Token{Kind: token.Punct, Value: ","},
				token.Token{Kind: token.Whitespace, Value: " "})
			for _, ci := range inner {
				clones = append(clones, s.At(ci))
			}

			// clear the merged isset: its tokens, the isset keyword and the "&&"
			for _, ci := range nextIssetInfo {
				clearTokenAndMergeWS(s, ci)
			}
			clearTokenAndMergeWS(s, issetIndex)
			clearTokenAndMergeWS(s, booleanAndTokenIndex)

			s.InsertSliceAt(insertLocation, clones)

			inserted := len(clones)
			tokenCount += inserted
			issetCloseBraceIndex += inserted
			insertLocation += inserted
			booleanAndTokenIndex = sigNext(s, issetCloseBraceIndex)
		}
	}
	s.ClearEmptyTokens()
	return s.Render() != before
}

// combineIssetPrevAllowed reports whether the token at i lets the isset after it
// start a mergeable statement: "(", "{", ";", "=", an open tag, "&&" or "||".
func combineIssetPrevAllowed(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	switch t.Kind {
	case token.OpenTag:
		return true
	case token.Punct:
		switch t.Value {
		case "(", "{", ";", "=", "&&", "||":
			return true
		}
	}
	return false
}

// combineIssetNextAllowed reports whether the token after a merged isset may
// follow it: ")", "}", ";", a close tag, "&&" or "||".
func combineIssetNextAllowed(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	switch t.Kind {
	case token.CloseTag:
		return true
	case token.Punct:
		switch t.Value {
		case ")", "}", ";", "&&", "||":
			return true
		}
	}
	return false
}

// combineIssetInfo returns the meaningful token indices of an isset call, from
// its "(" through the matching ")", ports getIssetInfo.
func combineIssetInfo(s *tokens.Stream, index int) []int {
	open := sigNext(s, index)
	indices := []int{open}
	depth := 1
	for i := open + 1; i < s.Len(); i++ {
		if t := s.At(i); t.Kind == token.Whitespace || isComment(t) {
			continue
		}
		indices = append(indices, i)
		switch {
		case isPunctVal(s, i, ")"):
			depth--
			if depth == 0 {
				return indices
			}
		case isPunctVal(s, i, "("):
			depth++
		}
	}
	return indices
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/CombineConsecutiveUnsetsFixer.php
//
// CombineConsecutiveUnsets merges "unset($a); unset($b);" into "unset($a, $b);".
type CombineConsecutiveUnsets struct{}

func (CombineConsecutiveUnsets) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\CombineConsecutiveUnsetsFixer`
}

func (CombineConsecutiveUnsets) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/CombineConsecutiveUnsetsFixer.php"
}

func (CombineConsecutiveUnsets) Fix(s *tokens.Stream) bool {
	before := s.Render()
	for index := s.Len() - 1; index >= 0; index-- {
		if !kwIs(s.At(index), "unset") {
			continue
		}

		prevUnset, braceEnd, redirect, found := combineUnsetPreviousCall(s, index)
		if !found {
			index = redirect
			continue
		}

		nextContentStart := nextPunctOfKind(s, index, "(")
		nextContentEnd := s.MatchForward(nextContentStart)
		addCount := combineUnsetMoveTokens(s, nextContentStart, nextContentEnd, braceEnd-1)

		if s.At(braceEnd).Kind != token.Whitespace {
			s.InsertAt(braceEnd, token.Token{Kind: token.Whitespace, Value: " "})
			addCount++
		}

		tokenBefore := sigPrev(s, braceEnd)
		if !isPunctVal(s, tokenBefore, ",") {
			s.InsertAt(braceEnd, token.Token{Kind: token.Punct, Value: ","})
			addCount++
		} else if tokenBefore+1 < s.Len() && s.At(tokenBefore+1).Kind == token.Whitespace {
			// kept trailing comma from previous unset - drop the now doubled space
			clearTokenAndMergeWS(s, tokenBefore+1)
		}

		for _, ci := range []int{index, nextContentStart, nextContentEnd} {
			clearTokenAndMergeWS(s, ci+addCount)
		}

		if nextSemicolon := sigNext(s, nextContentEnd); isPunctVal(s, nextSemicolon, ";") {
			clearTokenAndMergeWS(s, nextSemicolon)
		}

		index = prevUnset + 1
	}
	s.ClearEmptyTokens()
	return s.Render() != before
}

// combineUnsetPreviousCall finds an "unset(...)" statement directly before the
// unset at index. It returns that unset's index and its ")" index when found;
// otherwise redirect is the index the scan reached, ported from
// getPreviousUnsetCall.
func combineUnsetPreviousCall(s *tokens.Stream, index int) (prevUnset, braceEnd, redirect int, found bool) {
	semicolon := sigPrev(s, index)
	if semicolon < 0 {
		return 0, 0, index, false
	}
	if !isPunctVal(s, semicolon, ";") {
		return 0, 0, semicolon, false
	}
	braceEnd = sigPrev(s, semicolon)
	if braceEnd < 0 {
		return 0, 0, index, false
	}
	if !isPunctVal(s, braceEnd, ")") {
		return 0, 0, braceEnd, false
	}
	braceStart := s.MatchBackward(braceEnd)
	prevUnset = sigPrev(s, braceStart)
	if prevUnset < 0 {
		return 0, 0, index, false
	}
	if !kwIs(s.At(prevUnset), "unset") {
		return 0, 0, prevUnset, false
	}
	return prevUnset, braceEnd, 0, true
}

// combineUnsetMoveTokens moves the tokens between the parentheses [start, end]
// into the previous unset's argument list, just before "to"+1, and clears the
// moved originals. It returns how many tokens were inserted, ported from
// moveTokens.
func combineUnsetMoveTokens(s *tokens.Stream, start, end, to int) int {
	added := 0
	for i := start + 1; i < end; i += 2 {
		if s.At(i).Kind == token.Whitespace && s.At(to+1).Kind == token.Whitespace {
			s.SetValue(to+1, s.At(to+1).Value+s.At(i).Value)
		} else {
			s.InsertAt(to+1, s.At(i))
			to++
			end++
			added++
		}
		clearTokenAt(s, i+1)
	}
	return added
}
