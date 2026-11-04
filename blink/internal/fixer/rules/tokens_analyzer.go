package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
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

// taKw reports whether the token at i is the keyword name (case-insensitive).
func taKw(s *tokens.Stream, i int, name string) bool {
	return i >= 0 && i < s.Len() && kwIs(s.At(i), name)
}

// taIsNamedArgument reports whether the token at i is a named-argument label
// ("foo(class: $x)"), which PHP tokenizes as T_STRING rather than a keyword.
func taIsNamedArgument(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	if n < 0 || !isPunctVal(s, n, ":") {
		return false
	}
	p := sigPrev(s, i)
	return p >= 0 && (isPunctVal(s, p, "(") || isPunctVal(s, p, ","))
}

// taIsClassKw reports whether the token at i is a T_CLASS.
func taIsClassKw(s *tokens.Stream, i int) bool {
	return taKw(s, i, "class") && !taIsNamedArgument(s, i)
}

// taIsLambda mirrors TokensAnalyzer::isLambda for the "function"/"fn" at i: the
// next meaningful token, after an optional "&", is "(".
func taIsLambda(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	if isPunctVal(s, n, "&") {
		n = sigNext(s, n)
	}
	return isPunctVal(s, n, "(")
}

// taIsModifier reports whether the token at i is a method modifier keyword.
func taIsModifier(s *tokens.Stream, i int) bool {
	if s.At(i).Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(s.At(i).Value) {
	case "public", "protected", "private", "final", "abstract", "static":
		return true
	}
	return false
}

// taIsAnonymousClass mirrors TokensAnalyzer::isAnonymousClass for the classy
// token at i: a T_CLASS preceded (past "readonly" and attributes) by "new".
func taIsAnonymousClass(s *tokens.Stream, i int) bool {
	if !taIsClassKw(s, i) {
		return false
	}
	if p := sigPrev(s, i); taKw(s, p, "readonly") {
		i = p
	}
	return anonClassNewIndex(s, i) >= 0
}

// taIsConstruct reports whether the token at i is the language construct name
// (exit, die, eval) that blink lexes as Ident, outside a name position.
func taIsConstruct(s *tokens.Stream, i int, name string) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	return (t.Kind == token.Ident || t.Kind == token.Keyword) && strings.EqualFold(t.Value, name) && !taInNamePosition(s, i)
}

// taIsClassy mirrors Token::isClassy: T_CLASS, T_INTERFACE, T_TRAIT or T_ENUM.
// "::class" is an Ident in blink (PHP's CT::T_CLASS_CONSTANT), and "enum" is
// only T_ENUM when an enum name follows.
func taIsClassy(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() || s.At(i).Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(s.At(i).Value) {
	case "class", "interface", "trait":
		return !taIsNamedArgument(s, i)
	case "enum":
		n := sigNext(s, i)
		return n >= 0 && s.At(n).Kind == token.Ident
	}
	return false
}

// taArrowFunctionEnd mirrors TokensAnalyzer::getLastTokenIndexOfArrowFunction
// for the "fn" at i: the last meaningful token of its expression body.
func taArrowFunctionEnd(s *tokens.Stream, i int) int {
	index := nextPunctOfKind(s, i, "=>")
	if index < 0 {
		return -1
	}
	for {
		index = sigNext(s, index)
		if index < 0 {
			return sigPrev(s, s.Len())
		}
		t := s.At(index)
		if t.Kind == token.CloseTag || t.Kind == token.AttributeEnd {
			break
		}
		switch t.Kind {
		case token.AttributeStart:
			index = s.AttributeEnd(index)
		case token.Punct:
			switch t.Value {
			case ")", "]", "}", ",", ";":
				return sigPrev(s, index)
			case "(", "[", "{":
				index = s.MatchForward(index)
			}
		}
		if index < 0 {
			return sigPrev(s, s.Len())
		}
	}
	return sigPrev(s, index)
}

func taSeqVariable(name string) taSeqItem {
	return func(s *tokens.Stream, i int) bool {
		return s.At(i).Kind == token.Variable && s.At(i).Value == name
	}
}

func taSeqKeyword(name string) taSeqItem {
	return func(s *tokens.Stream, i int) bool { return kwIs(s.At(i), name) }
}
