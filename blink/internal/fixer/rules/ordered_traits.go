package rules

import (
	"slices"
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedTraitsFixer.php
//
// OrderedTraits sorts trait `use` statements alphabetically (case-insensitive by
// default), and sorts multiple traits within one `use A, B;` statement. Only the
// name portions move; surrounding whitespace and punctuation stay in place.
type OrderedTraits struct{}

func (OrderedTraits) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedTraitsFixer`
}

func (OrderedTraits) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedTraitsFixer.php"
}

func (OrderedTraits) Fix(s *tokens.Stream) bool {
	// gather all trait-use statements as (start, end) ranges
	type stmt struct {
		start, end int
	}
	var groups [][]stmt
	var cur []stmt

	i := 0
	for i < s.Len() {
		t := s.At(i)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			i++
			continue
		}
		if !kwIs(t, "use") || !isTraitUse(s, i) {
			if len(cur) > 0 {
				groups = append(groups, cur)
				cur = nil
			}
			i++
			continue
		}
		// statement end: next ";" or "{...}" block
		end := nextPunctOfKind(s, i, ";", "{")
		if end < 0 {
			break
		}
		if isPunctVal(s, end, "{") {
			end = s.MatchForward(end)
			if end < 0 {
				break
			}
		}
		cur = append(cur, stmt{start: i, end: end})
		i = end + 1
	}
	if len(cur) > 0 {
		groups = append(groups, cur)
	}

	changed := false
	// process groups in reverse so earlier indices stay valid across ReplaceRange
	for _, g := range slices.Backward(groups) {
		// build each statement's token slice, with its multiple traits sorted
		stmtToks := make([][]token.Token, len(g))
		names := make([]string, len(g))
		for k, st := range g {
			toks := make([]token.Token, 0, st.end-st.start+1)
			for j := st.start; j <= st.end; j++ {
				toks = append(toks, s.At(j))
			}
			toks = sortTraitsInStatement(toks)
			stmtToks[k] = toks
			names[k] = traitName(toks)
		}
		// sort statement slices by trait name (case-insensitive, stable)
		order := make([]int, len(g))
		for k := range order {
			order[k] = k
		}
		sort.SliceStable(order, func(a, b int) bool {
			return strings.ToLower(names[order[a]]) < strings.ToLower(names[order[b]])
		})
		// assign sorted slices to original positions; detect change
		reordered := false
		for k := range order {
			if order[k] != k {
				reordered = true
				break
			}
		}
		if !reordered {
			// even if statement order is unchanged, within-statement sort may have changed a slice
			within := false
			for k, st := range g {
				if !sameTokens(stmtToks[k], s, st.start, st.end) {
					within = true
					break
				}
			}
			if !within {
				continue
			}
		}
		// apply position -> sorted slice, in reverse index order
		for k, gk := range slices.Backward(g) {
			s.ReplaceRange(gk.start, gk.end, stmtToks[order[k]])
		}
		changed = true
	}
	return changed
}

func sameTokens(toks []token.Token, s *tokens.Stream, start, end int) bool {
	if len(toks) != end-start+1 {
		return false
	}
	for k, t := range toks {
		o := s.At(start + k)
		if o.Kind != t.Kind || o.Value != t.Value {
			return false
		}
	}
	return true
}

// isTraitUse reports whether the "use" keyword at i is a trait use (inside a
// class/trait/enum body), not a namespace import or a closure "use".
func isTraitUse(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	if n < 0 {
		return false
	}
	// closure use is "use (" ; imports/traits start with a name or "\"
	if s.At(n).Kind != token.Ident && !isPunctVal(s, n, `\`) {
		return false
	}
	// enclosing "{" must belong to a class/trait/enum
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth == 0 {
				return braceIsClassy(s, j)
			}
			depth--
		}
	}
	return false
}

// braceIsClassy reports whether the "{" at index opens a class/trait/enum body,
// by walking back over the class header to its keyword.
func braceIsClassy(s *tokens.Stream, brace int) bool {
	for j := sigPrev(s, brace); j >= 0; j = sigPrev(s, j) {
		t := s.At(j)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "class", "trait", "enum":
				return true
			case "extends", "implements", "abstract", "final", "readonly":
				continue
			}
			return false
		}
		switch t.Kind {
		case token.Ident:
			continue
		case token.Punct:
			if t.Value == "," || t.Value == `\` {
				continue
			}
			return false
		default:
			return false
		}
	}
	return false
}

// traitName returns the first trait's fully-qualified name with the leading "\"
// trimmed, matching the fixer's toTraitName sort key.
func traitName(toks []token.Token) string {
	var b strings.Builder
	for _, t := range toks {
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{" || t.Value == ",") {
			break
		}
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
			b.WriteString(t.Value)
		}
	}
	return strings.TrimLeft(b.String(), `\`)
}

// sortTraitsInStatement sorts multiple traits within a single "use A, B;"
// statement, swapping only the name token runs and keeping everything else.
func sortTraitsInStatement(toks []token.Token) []token.Token {
	// find name runs (T_STRING / "\") separated by "," ; stop at ";" or "{"
	type run struct{ start, end int }
	var runs []run
	rs := -1
	for idx, t := range toks {
		isName := t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`)
		if isName {
			if rs < 0 {
				rs = idx
			}
			continue
		}
		if t.Kind == token.Punct && (t.Value == "," || t.Value == ";" || t.Value == "{") {
			if rs >= 0 {
				runs = append(runs, run{rs, idx - 1})
				rs = -1
			}
			if t.Value == "{" {
				break
			}
		}
	}
	if len(runs) <= 1 {
		return toks
	}
	keys := make([]string, len(runs))
	for k, r := range runs {
		var b strings.Builder
		for j := r.start; j <= r.end; j++ {
			b.WriteString(toks[j].Value)
		}
		keys[k] = strings.TrimLeft(b.String(), `\`)
	}
	order := make([]int, len(runs))
	for k := range order {
		order[k] = k
	}
	sort.SliceStable(order, func(a, b int) bool {
		return strings.ToLower(keys[order[a]]) < strings.ToLower(keys[order[b]])
	})
	same := true
	for k := range order {
		if order[k] != k {
			same = false
			break
		}
	}
	if same {
		return toks
	}
	// rebuild sequentially: at each run's slot, place the sorted run's tokens
	// (runs may differ in length); everything between runs is kept verbatim
	var res []token.Token
	prev := 0
	for k, r := range runs {
		res = append(res, toks[prev:r.start]...)
		src := runs[order[k]]
		res = append(res, toks[src.start:src.end+1]...)
		prev = r.end + 1
	}
	res = append(res, toks[prev:]...)
	return res
}
