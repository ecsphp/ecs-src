package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Helpers mirroring PHP-CS-Fixer's Tokens and TokensAnalyzer.

// taInNamePosition reports whether the token at i follows "->", "?->", "::",
// "function" or "const", or neighbours a namespace separator: the positions
// where PHP's TOKEN_PARSE tokenizer turns a reserved word into a T_STRING.
func taInNamePosition(s *tokens.Stream, i int) bool {
	if i+1 < s.Len() && isPunctVal(s, i+1, `\`) {
		return true
	}
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	if t.Kind == token.Punct {
		return t.Value == "->" || t.Value == "?->" || t.Value == "::" || t.Value == `\`
	}
	return kwIs(t, "function") || kwIs(t, "const")
}

// taConstructIdents are language constructs blink lexes as Ident but PHP gives
// a dedicated token (T_EXIT, T_EVAL, magic constants, T_HALT_COMPILER).
var taConstructIdents = map[string]bool{
	"exit": true, "die": true, "eval": true, "__halt_compiler": true,
	"__class__": true, "__dir__": true, "__file__": true, "__function__": true,
	"__line__": true, "__method__": true, "__namespace__": true, "__trait__": true,
	"__property__": true,
}

// taIsTString reports whether the token at i is what PHP-CS-Fixer sees as a
// T_STRING.
func taIsTString(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	if t.Kind != token.Ident {
		return false
	}
	return !taConstructIdents[strings.ToLower(t.Value)] || taInNamePosition(s, i)
}

// taSeqItem is one element of a Tokens::findSequence pattern.
type taSeqItem func(s *tokens.Stream, i int) bool

func taSeqPunct(v string) taSeqItem {
	return func(s *tokens.Stream, i int) bool { return isPunctVal(s, i, v) }
}

// taSeqTString matches a T_STRING, with the given content when name is not
// empty (case-insensitively when ci is set).
func taSeqTString(name string, ci bool) taSeqItem {
	return func(s *tokens.Stream, i int) bool {
		if !taIsTString(s, i) {
			return false
		}
		if name == "" {
			return true
		}
		if ci {
			return strings.EqualFold(s.At(i).Value, name)
		}
		return s.At(i).Value == name
	}
}

// taFindSequence mirrors Tokens::findSequence: the indices of the first run of
// meaningful tokens matching seq that starts within [start, end].
func taFindSequence(s *tokens.Stream, seq []taSeqItem, start, end int) []int {
	if end > s.Len()-1 {
		end = s.Len() - 1
	}
	if start < 0 || start+len(seq)-1 > end {
		return nil
	}
outer:
	for index := start; index <= end; index++ {
		if !seq[0](s, index) {
			continue
		}
		result := []int{index}
		cur := index
		for _, item := range seq[1:] {
			cur = sigNext(s, cur)
			if cur < 0 || cur > end || !item(s, cur) {
				continue outer
			}
			result = append(result, cur)
		}
		return result
	}
	return nil
}
