package rules

import (
	"slices"
	"sort"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderMethodOrderFixer.php
//
// PhpUnitDataProviderMethodOrder places each data provider method right after
// (default) or before the test that uses it, reordering class elements through
// the OrderedClassElements port.
type PhpUnitDataProviderMethodOrder struct {
	before bool
}

func (PhpUnitDataProviderMethodOrder) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDataProviderMethodOrderFixer`
}

func (PhpUnitDataProviderMethodOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderMethodOrderFixer.php"
}

func (f PhpUnitDataProviderMethodOrder) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["placement"].(string); ok {
		f.before = v == "before"
	}
	return f
}

func (f PhpUnitDataProviderMethodOrder) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if f.orderClass(s, cls[0]) {
			changed = true
		}
	}
	return changed
}

// methodOrderPair is one `@dataProvider` link: the test that uses a provider.
type methodOrderPair struct {
	testName string
	provider string
}

func (f PhpUnitDataProviderMethodOrder) orderClass(s *tokens.Stream, open int) bool {
	elements := oceElements(s, open)
	if len(elements) == 0 {
		return false
	}
	endIndex := elements[len(elements)-1].end

	pairs := methodOrderPairs(s, open, endIndex)
	if len(pairs) == 0 {
		return false
	}

	sorted := slices.Clone(elements)
	placed := map[string]bool{}
	if f.before {
		for _, p := range pairs {
			if placed[p.provider] {
				continue
			}
			placed[p.provider] = true
			sorted = moveMethodElement(sorted, p.testName, p.provider, false)
		}
	} else {
		sameUsage, sameProvider, haveSame := "", "", false
		for _, p := range pairs {
			if placed[p.provider] {
				continue
			}
			placed[p.provider] = true
			keep := p.testName
			if haveSame && p.testName == sameUsage {
				keep = sameProvider
			}
			before := sorted
			sorted = moveMethodElement(sorted, keep, p.provider, true)
			sameUsage, sameProvider, haveSame = p.testName, p.provider, true
			if !sameElementOrder(before, sorted) {
				delete(placed, p.provider)
			}
		}
	}

	if sameElementOrder(elements, sorted) {
		return false
	}
	var repl []token.Token
	for _, e := range sorted {
		repl = append(repl, s.Tokens()[e.start:e.end+1]...)
	}
	s.ReplaceRange(open+1, endIndex, repl)
	return true
}

// methodOrderPairs is getOrigUsageDataProviderOrderPairs: the (test, provider)
// links ordered by test-name position, then by usage offset.
func methodOrderPairs(s *tokens.Stream, start, endIndex int) []methodOrderPair {
	type flatUsage struct {
		testNameIdx int
		testName    string
		offset      int
		provider    string
	}
	var flat []flatUsage
	for _, dp := range puDataProviders(s, start, endIndex) {
		seen := map[int]int{} // testNameIdx -> position in flat (last wins)
		for _, u := range dp.usages {
			nameIdx := puNextMethodNameIndex(s, u.index, endIndex)
			if nameIdx < 0 {
				continue
			}
			fu := flatUsage{testNameIdx: nameIdx, testName: s.At(nameIdx).Value, offset: u.offset, provider: dp.name}
			if pos, ok := seen[nameIdx]; ok {
				flat[pos] = fu
				continue
			}
			seen[nameIdx] = len(flat)
			flat = append(flat, fu)
		}
	}
	sort.SliceStable(flat, func(a, b int) bool {
		if flat[a].testNameIdx != flat[b].testNameIdx {
			return flat[a].testNameIdx < flat[b].testNameIdx
		}
		return flat[a].offset < flat[b].offset
	})
	pairs := make([]methodOrderPair, 0, len(flat))
	for _, fu := range flat {
		pairs = append(pairs, methodOrderPair{testName: fu.testName, provider: fu.provider})
	}
	return pairs
}

// moveMethodElement mirrors the fixer's own helper: move the method named
// nameToMove to right after (or before) the method named nameKeep.
func moveMethodElement(elements []oceElement, nameKeep, nameToMove string, after bool) []oceElement {
	iKeep, iToMove := -1, -1
	for i, e := range elements {
		if e.typ != "method" {
			continue
		}
		switch e.name {
		case nameKeep:
			iKeep = i
		case nameToMove:
			iToMove = i
		}
	}
	if iKeep < 0 || iToMove < 0 {
		return elements
	}
	delta := -1
	if after {
		delta = 1
	}
	if iToMove == iKeep+delta {
		return elements
	}

	toMove := elements[iToMove]
	rest := slices.Delete(slices.Clone(elements), iToMove, iToMove+1)

	c := iKeep
	if after {
		c++
	}
	if iToMove < iKeep {
		c--
	}
	return slices.Insert(rest, c, toMove)
}

func sameElementOrder(a, b []oceElement) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].start != b[i].start {
			return false
		}
	}
	return true
}
