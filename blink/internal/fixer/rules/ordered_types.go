package rules

import (
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedTypesFixer.php
//
// OrderedTypes sorts the members of a union or intersection type declaration
// using the fixer defaults: case-insensitive alphabetical order with "null"
// always first ("B|A" -> "A|B", "string|int|null" -> "null|int|string"). It acts
// only in confirmed type positions (parameter, return, typed property) on a flat
// single-operator run; nullable "?" prefixes, DNF "()" types, mixed "|"/"&" runs
// and types spanning multiple lines are left untouched.
type OrderedTypes struct{}

func (OrderedTypes) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedTypesFixer`
}

func (OrderedTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedTypesFixer.php"
}

// typeMember is one member of a union/intersection, its tokens and its sort key.
type typeMember struct {
	toks []token.Token
	key  string
}

func (OrderedTypes) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "|" && t.Value != "&") {
			continue
		}
		if !isTypeUnionOperator(s, i) {
			continue
		}
		start := typeRunBoundaryPrev(s, i)
		end := typeRunBoundaryNext(s, i)
		if end < 0 {
			continue
		}
		// the run's first..last significant token (excludes surrounding whitespace)
		runStart := start + 1
		for runStart < end && s.At(runStart).Kind == token.Whitespace {
			runStart++
		}
		runEnd := end - 1
		for runEnd > runStart && s.At(runEnd).Kind == token.Whitespace {
			runEnd--
		}
		// process each run once, at its leftmost operator
		first := -1
		for j := runStart; j <= runEnd; j++ {
			if v := s.At(j); v.Kind == token.Punct && (v.Value == "|" || v.Value == "&") {
				first = j
				break
			}
		}
		if first != i {
			continue
		}

		op := t.Value
		members, ok := collectTypeMembers(s, runStart, runEnd+1, op)
		if !ok || len(members) < 2 {
			continue
		}

		sorted := make([]typeMember, len(members))
		copy(sorted, members)
		sort.SliceStable(sorted, func(a, b int) bool {
			an, bn := sorted[a].key == "null", sorted[b].key == "null"
			if an != bn {
				return an // null first
			}
			return sorted[a].key < sorted[b].key
		})
		if sameMemberOrder(members, sorted) {
			continue // already ordered; nothing to rewrite
		}

		repl := make([]token.Token, 0, len(sorted)*2)
		for idx, m := range sorted {
			if idx > 0 {
				repl = append(repl, token.Token{Kind: token.Punct, Value: op})
			}
			repl = append(repl, m.toks...)
		}
		s.ReplaceRange(runStart, runEnd, repl)
		changed = true
		i = runStart + len(repl) - 1
	}
	return changed
}

// collectTypeMembers splits the significant tokens in [from, to) into members
// separated by the single operator op. It fails (ok=false) when the run mixes
// operators, contains a "?"/"("/")" or spans multiple lines - cases the fixer
// must not reorder.
func collectTypeMembers(s *tokens.Stream, from, to int, op string) (members []typeMember, ok bool) {
	var cur typeMember
	flush := func() {
		cur.key = strings.ToLower(cur.key)
		members = append(members, cur)
		cur = typeMember{}
	}
	for j := from; j < to; j++ {
		tk := s.At(j)
		switch tk.Kind {
		case token.Whitespace:
			if hasNewline(tk.Value) {
				return nil, false // multi-line type, leave alone
			}
			continue
		case token.Comment, token.DocComment:
			return nil, false
		case token.Punct:
			switch tk.Value {
			case op:
				if len(cur.toks) == 0 {
					return nil, false
				}
				flush()
				continue
			case `\`:
				cur.toks = append(cur.toks, tk)
				cur.key += tk.Value
				continue
			default:
				return nil, false // "?", "(", ")", "&"/"|" mismatch
			}
		default:
			cur.toks = append(cur.toks, tk)
			cur.key += tk.Value
		}
	}
	if len(cur.toks) == 0 {
		return nil, false
	}
	flush()
	return members, true
}

// sameMemberOrder reports whether a and b list members in the same order.
func sameMemberOrder(a, b []typeMember) bool {
	for i := range a {
		if a[i].key != b[i].key {
			return false
		}
	}
	return true
}
