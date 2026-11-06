package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/OrderedAttributesFixer.php
//
// OrderedAttributes sorts attributes with the configured algorithm. The default
// (and only one blink implements) is "alpha": within each "#[A, B]" group the
// attributes are sorted by their written name, and consecutive "#[...]" groups
// are then sorted by their first attribute's name. The "custom" algorithm needs
// the FullyQualifiedNameAnalyzer to resolve every name to its FQN against the
// surrounding namespace and imports, which blink does not model, so it is left
// as a no-op (never wrong output).
type OrderedAttributes struct {
	// custom reports sort_algorithm=custom, which blink defers to a no-op.
	custom bool
}

func (OrderedAttributes) Name() string {
	return `PhpCsFixer\Fixer\AttributeNotation\OrderedAttributesFixer`
}

func (OrderedAttributes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/OrderedAttributesFixer.php"
}

func (f OrderedAttributes) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["sort_algorithm"].(string); ok && v == "custom" {
		f.custom = true
	}
	return f
}

func (f OrderedAttributes) Fix(s *tokens.Stream) bool {
	if f.custom {
		return false
	}
	changed := false
	index := -1
	for {
		gi := attrNextStart(s, index)
		if gi < 0 {
			break
		}
		groups := collectAttrGroupRun(s, gi)
		endIndex := groups[len(groups)-1].end
		for k := range groups {
			if orderedSortWithinGroup(s, &groups[k]) {
				changed = true
			}
		}
		if len(groups) > 1 && orderedSortGroups(s, groups) {
			changed = true
		}
		index = endIndex
	}
	return changed
}

// orderedAlphaKey ports getAttributeName for the alpha algorithm: the name with
// its leading backslash trimmed.
func orderedAlphaKey(name string) string {
	return strings.TrimLeft(name, `\`)
}

// orderedSortWithinGroup sorts the attributes inside g, sets g.key to the first
// attribute's key, and reports whether it reordered tokens.
func orderedSortWithinGroup(s *tokens.Stream, g *attrGroup) bool {
	keys := make([]string, len(g.attrs))
	for i, a := range g.attrs {
		keys[i] = orderedAlphaKey(a.name)
	}
	order := identityOrder(len(g.attrs))
	slices.SortStableFunc(order, func(a, b int) int { return strings.Compare(keys[a], keys[b]) })
	g.key = keys[order[0]]
	if len(g.attrs) == 1 || isIdentityOrder(order) {
		return false
	}
	lastEnd := g.attrs[len(g.attrs)-1].end
	var repl []token.Token
	for pos, oi := range order {
		a := g.attrs[oi]
		for j := a.start; j <= a.end; j++ {
			repl = append(repl, s.At(j))
		}
		if pos != len(order)-1 {
			repl = append(repl, token.Token{Kind: token.Punct, Value: ","})
		}
	}
	s.ReplaceRange(g.opening+1, lastEnd, repl)
	return true
}

// orderedSortGroups sorts the groups of a run by their key, without a delimiter.
func orderedSortGroups(s *tokens.Stream, groups []attrGroup) bool {
	order := identityOrder(len(groups))
	slices.SortStableFunc(order, func(a, b int) int { return strings.Compare(groups[a].key, groups[b].key) })
	if isIdentityOrder(order) {
		return false
	}
	var repl []token.Token
	for _, oi := range order {
		g := groups[oi]
		for j := g.start; j <= g.end; j++ {
			repl = append(repl, s.At(j))
		}
	}
	s.ReplaceRange(groups[0].start, groups[len(groups)-1].end, repl)
	return true
}

func identityOrder(n int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	return order
}

func isIdentityOrder(order []int) bool {
	for i, v := range order {
		if i != v {
			return false
		}
	}
	return true
}
