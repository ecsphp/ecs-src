package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

const (
	groupTraitUse = iota
	groupConst
	groupProperty
	groupMethod
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedClassElementsFixer.php
//
// OrderedClassElements sorts direct class members into groups - trait use,
// constants, properties, then methods - keeping the original order within each
// group. It is deliberately conservative: any comment, doc comment or attribute
// in the body makes it skip the whole class so leading trivia is never detached.
type OrderedClassElements struct{}

func (OrderedClassElements) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedClassElementsFixer`
}

func (OrderedClassElements) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedClassElementsFixer.php"
}

func (OrderedClassElements) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == "{" {
			if kind, _ := classifyBrace(s, i); kind == braceClassLike {
				if reorderClassBody(s, i) {
					changed = true
					i = 0 // members moved; restart - each class sorts once so this ends
					continue
				}
			}
		}
		i++
	}
	return changed
}

func reorderClassBody(s *tokens.Stream, open int) bool {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	// Any comment/attribute in the body means trivia we cannot safely move.
	for k := open + 1; k < closeIdx; k++ {
		switch s.At(k).Kind {
		case token.Comment, token.DocComment:
			return false
		}
	}
	starts := classMemberStarts(s, open)
	if len(starts) < 2 {
		return false
	}
	ends := make([]int, len(starts))
	groups := make([]int, len(starts))
	for idx, m := range starts {
		end := memberSpanEnd(s, m)
		if end < 0 || end >= closeIdx {
			return false
		}
		g := classifyMemberGroup(s, m, end)
		if g < 0 {
			return false
		}
		ends[idx] = end
		groups[idx] = g
	}
	// separators between members must be whitespace only
	for idx := 0; idx+1 < len(starts); idx++ {
		for k := ends[idx] + 1; k < starts[idx+1]; k++ {
			if s.At(k).Kind != token.Whitespace {
				return false
			}
		}
	}
	order := make([]int, 0, len(starts))
	for g := groupTraitUse; g <= groupMethod; g++ {
		for idx, gg := range groups {
			if gg == g {
				order = append(order, idx)
			}
		}
	}
	if isIdentityOrder(order) {
		return false
	}
	// reordered member spans joined by the original positional separators
	var repl []token.Token
	for pos, memberIdx := range order {
		for k := starts[memberIdx]; k <= ends[memberIdx]; k++ {
			repl = append(repl, s.At(k))
		}
		if pos+1 < len(order) {
			for k := ends[pos] + 1; k < starts[pos+1]; k++ {
				repl = append(repl, s.At(k))
			}
		}
	}
	s.ReplaceRange(starts[0], ends[len(ends)-1], repl)
	return true
}

// memberSpanEnd returns the last token index of the member starting at m: the
// terminating ";" for properties/consts/trait-use, or the matching "}" of a
// method body. Nested (), [] and {} groups are skipped. Returns -1 on trouble.
func memberSpanEnd(s *tokens.Stream, m int) int {
	for k := m; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			mm := s.MatchForward(k)
			if mm < 0 {
				return -1
			}
			k = mm
		case "{":
			return s.MatchForward(k)
		case ";":
			return k
		case "}":
			return -1
		}
	}
	return -1
}

// classifyMemberGroup returns the ordering group of the member in [m, end], or
// -1 when it is anything unexpected (enum case, unknown token) so the caller
// can skip the whole class.
func classifyMemberGroup(s *tokens.Stream, m, end int) int {
	k := m
	for k <= end && s.At(k).Kind == token.Keyword && memberModifiers[strings.ToLower(s.At(k).Value)] {
		k = skipWhitespace(s, k+1)
	}
	if k > end {
		return -1
	}
	if s.At(k).Kind == token.Keyword {
		switch strings.ToLower(s.At(k).Value) {
		case "use":
			return groupTraitUse
		case "const":
			return groupConst
		case "function":
			return groupMethod
		default:
			return -1
		}
	}
	// property: a variable at the top level before the terminating ";"
	for x := k; x <= end; x++ {
		t := s.At(x)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				if mm := s.MatchForward(x); mm > 0 {
					x = mm
					continue
				}
			}
		}
		if t.Kind == token.Variable {
			return groupProperty
		}
	}
	return -1
}

func isIdentityOrder(order []int) bool {
	for i, v := range order {
		if i != v {
			return false
		}
	}
	return true
}
