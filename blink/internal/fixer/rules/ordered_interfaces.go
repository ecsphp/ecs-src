package rules

import (
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedInterfacesFixer.php
//
// OrderedInterfaces sorts the interface list of an "implements" or
// "interface extends" clause. Default config: alpha order, ascending,
// case-insensitive.
type OrderedInterfaces struct{}

func (OrderedInterfaces) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedInterfacesFixer`
}

func (OrderedInterfaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedInterfacesFixer.php"
}

func (OrderedInterfaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if kwIs(t, "implements") {
			// ok
		} else if kwIs(t, "extends") {
			name := sigPrev(s, i)
			if name < 0 {
				continue
			}
			iface := sigPrev(s, name)
			if iface < 0 || !kwIs(s.At(iface), "interface") {
				continue
			}
		} else {
			continue
		}

		start := i + 1
		brace := nextPunctOfKind(s, start, "{")
		if brace < 0 {
			continue
		}
		end := sigPrev(s, brace)
		if end < start {
			continue
		}

		groups := interfaceGroups(s, start, end)
		if len(groups) <= 1 {
			continue
		}

		type keyed struct {
			toks []token.Token
			key  string
			orig int
		}
		items := make([]keyed, len(groups))
		for gi, g := range groups {
			items[gi] = keyed{toks: g, key: interfaceSortKey(g), orig: gi}
		}
		sort.SliceStable(items, func(a, b int) bool {
			return strings.ToLower(items[a].key) < strings.ToLower(items[b].key)
		})

		reordered := false
		for gi := range items {
			if items[gi].orig != gi {
				reordered = true
				break
			}
		}
		if !reordered {
			continue
		}

		var repl []token.Token
		for gi, it := range items {
			if gi > 0 {
				repl = append(repl, token.Token{Kind: token.Punct, Value: ","})
			}
			repl = append(repl, it.toks...)
		}
		s.ReplaceRange(start, end, repl)
		changed = true
		i = start + len(repl)
	}
	return changed
}

// interfaceGroups splits the inclusive range [start, end] on top-level commas;
// each group keeps its own leading whitespace, mirroring getInterfaces.
func interfaceGroups(s *tokens.Stream, start, end int) [][]token.Token {
	var groups [][]token.Token
	cur := []token.Token{}
	for i := start; i <= end && i < s.Len(); i++ {
		if isPunctVal(s, i, ",") {
			groups = append(groups, cur)
			cur = []token.Token{}
			continue
		}
		cur = append(cur, s.At(i))
	}
	groups = append(groups, cur)
	return groups
}

// interfaceSortKey mirrors the fixer's normalized name: from the first
// meaningful token, concatenate token content (with "\" replaced by a space)
// until whitespace or a comment ends the name.
func interfaceSortKey(group []token.Token) string {
	var b strings.Builder
	started := false
	for _, t := range group {
		if !started {
			if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
				continue
			}
			started = true
		}
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			break
		}
		b.WriteString(strings.ReplaceAll(t.Value, `\`, " "))
	}
	return b.String()
}
