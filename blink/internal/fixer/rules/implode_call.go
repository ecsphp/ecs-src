package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/ImplodeCallFixer.php
//
// ImplodeCall makes global implode() calls pass the separator first:
// "implode($pieces)" gets an explicit empty-string separator, and
// "implode($pieces, ',')" has its arguments swapped when the second one is a
// plain string literal.
// Risky: implode may be overridden.
type ImplodeCall struct{}

func (ImplodeCall) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\ImplodeCallFixer`
}

func (ImplodeCall) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/ImplodeCallFixer.php"
}

func (ImplodeCall) Fix(s *tokens.Stream) bool {
	// Upstream throws "Must not add empty token to collection" when it moves an
	// argument holding a token it already cleared, and ECS keeps the file as is.
	original := slices.Clone(s.Tokens())
	var cleared []int
	insert := func(at int, ts []token.Token) {
		for k, c := range cleared {
			if c >= at {
				cleared[k] = c + len(ts)
			}
		}
		s.InsertSliceAt(at, ts)
	}
	clearAt := func(i int) {
		clearTokenAt(s, i)
		cleared = append(cleared, i)
	}
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		t := s.At(index)
		if t.Kind != token.Ident || !strings.EqualFold(t.Value, "implode") {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue
		}
		args := implodeArgumentIndices(s, index)
		switch len(args) {
		case 1:
			insert(args[0][0], []token.Token{
				{Kind: token.String, Value: "''"},
				{Kind: token.Punct, Value: ","},
				{Kind: token.Whitespace, Value: " "},
			})
			changed = true
		case 2:
			first, second := args[0], args[1]
			if _, ok := constantStringAt(s, first[0]); ok {
				continue
			}
			// the swap needs a string literal to start the second argument
			secondEnd, ok := constantStringAt(s, second[0])
			if !ok {
				continue
			}
			if slices.ContainsFunc(cleared, func(c int) bool { return c >= first[0] && c <= first[1] }) {
				s.ReplaceRange(0, s.Len()-1, original)
				return false
			}
			moved := slices.Clone(s.Tokens()[first[0] : first[1]+1])
			for i := first[0]; i <= first[1]; i++ {
				clearAt(i)
			}
			// only the string token(s) move to the front, as upstream does
			str := slices.Clone(s.Tokens()[second[0] : secondEnd+1])
			insert(first[0], str)
			at := second[0] + len(str)
			for i := range str {
				clearAt(at + i)
			}
			insert(at, moved)
			changed = true
		}
	}
	return changed
}

// implodeArgumentIndices returns, for each argument of the call named at
// index, its first and last meaningful token (ImplodeCallFixer::getArgumentIndices).
func implodeArgumentIndices(s *tokens.Stream, index int) [][2]int {
	open := -1
	for j := index + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, "(") {
			open = j
			break
		}
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var out [][2]int
	for _, a := range aliasArguments(s, open, closeIdx) {
		out = append(out, [2]int{nextMeaningfulIndex(s, a[0]-1), prevMeaningfulIndex(s, a[1]+1)})
	}
	return out
}
