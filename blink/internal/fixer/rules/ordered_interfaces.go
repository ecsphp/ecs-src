package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedInterfacesFixer.php
//
// OrderedInterfaces sorts the interface list of an "implements" or
// "interface extends" clause. The zero value is the default config: alpha
// order, ascending, case-insensitive. Names are compared with "\" read as a
// space, as upstream does: strcasecmp (ASCII case folding) or, case-sensitive,
// a byte comparison; the `length` order compares byte lengths. The sort is
// stable, so equal keys keep their source order in both directions.
type OrderedInterfaces struct {
	Length        bool // order: length (default alpha)
	Descend       bool // direction: descend (default ascend)
	CaseSensitive bool // case_sensitive
}

func (OrderedInterfaces) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedInterfacesFixer`
}

func (OrderedInterfaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedInterfacesFixer.php"
}

// WithConfig reads `order` (alpha, length), `direction` (ascend, descend) and
// `case_sensitive`.
func (f OrderedInterfaces) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["order"].(string); ok {
		f.Length = v == "length"
	}
	if v, ok := config["direction"].(string); ok {
		f.Descend = v == "descend"
	}
	if v, ok := config["case_sensitive"].(bool); ok {
		f.CaseSensitive = v
	}
	return f
}

func (f OrderedInterfaces) Fix(s *tokens.Stream) bool {
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
		slices.SortStableFunc(items, func(a, b keyed) int {
			var score int
			if f.Length {
				score = len(a.key) - len(b.key)
			} else {
				score = phpNameCompare(a.key, b.key, f.CaseSensitive)
			}
			if f.Descend {
				score = -score
			}
			return score
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

// phpNameCompare compares two names the way the ordering fixers do: PHP's
// strcasecmp (which folds ASCII letters only) or, case-sensitive, `<=>` (a byte
// comparison, as names are never numeric strings).
func phpNameCompare(a, b string, caseSensitive bool) int {
	if !caseSensitive {
		a, b = asciiLower(a), asciiLower(b)
	}
	return strings.Compare(a, b)
}
