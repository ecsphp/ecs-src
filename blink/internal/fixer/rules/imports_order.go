package rules

import (
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

type importStmt struct {
	start, semi int
	kind        string // "class", "function" or "const"
	pathKey     string // lowercased imported path, for alphabetical ordering
}

// importSortKey returns the lowercased imported path of a use statement (after an
// optional "function"/"const", up to ";" or " as "), used as the alphabetical
// sort key by ECS's default OrderedImports (sort_algorithm: alpha).
func importSortKey(s *tokens.Stream, useIdx, semi int) string {
	j := skipWhitespace(s, useIdx+1)
	if j < s.Len() && s.At(j).Kind == token.Keyword {
		if lw := strings.ToLower(s.At(j).Value); lw == "function" || lw == "const" {
			j = skipWhitespace(s, j+1)
		}
	}
	var b strings.Builder
	for k := j; k < semi; k++ {
		t := s.At(k)
		if t.Kind == token.Keyword && strings.ToLower(t.Value) == "as" {
			break
		}
		if t.Kind == token.Whitespace {
			continue
		}
		b.WriteString(t.Value)
	}
	// compare segment by segment: "\" must sort before any other character so
	// "Rector\Php\X" precedes "Rector\Php71\Y" (ECS's alpha comparator)
	return strings.ReplaceAll(strings.ToLower(b.String()), `\`, "\x00")
}

// importKindRank orders use statements by kind for ECS's default imports_order
// (['class', 'function', 'const']).
func importKindRank(kind string) int {
	switch kind {
	case "function":
		return 1
	case "const":
		return 2
	default: // class
		return 0
	}
}

func useKind(s *tokens.Stream, useIdx int) string {
	j := skipWhitespace(s, useIdx+1)
	if j < s.Len() && s.At(j).Kind == token.Keyword {
		switch strings.ToLower(s.At(j).Value) {
		case "function":
			return "function"
		case "const":
			return "const"
		}
	}
	return "class"
}

// collectImportRun gathers a maximal run of consecutive top-level use statements
// starting at start, separated only by whitespace. Group imports and closure use
// stop the run.
func collectImportRun(s *tokens.Stream, start int) []importStmt {
	var stmts []importStmt
	k := start
	for k < s.Len() {
		t := s.At(k)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" || inClassLikeBody(s, k) {
			break
		}
		j := skipWhitespace(s, k+1)
		if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			break // closure use
		}
		semi, group := -1, false
		for m := k + 1; m < s.Len(); m++ {
			if s.At(m).Kind != token.Punct {
				continue
			}
			if s.At(m).Value == "{" {
				group = true
				break
			}
			if s.At(m).Value == ";" {
				semi = m
				break
			}
		}
		if group || semi < 0 {
			break
		}
		stmts = append(stmts, importStmt{start: k, semi: semi, kind: useKind(s, k), pathKey: importSortKey(s, k, semi)})
		n := skipWhitespace(s, semi+1)
		if n < s.Len() && s.At(n).Kind == token.Keyword && strings.ToLower(s.At(n).Value) == "use" {
			k = n
			continue
		}
		break
	}
	return stmts
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/OrderedImportsFixer.php
//
// OrderedImports sorts a run of use statements by kind (class, then function,
// then const) and alphabetically by imported path within each kind
// (case-insensitive), matching ECS's psr12 config (sort_algorithm: alpha,
// imports_order: ['class', 'function', 'const']).
type OrderedImports struct{}

func (OrderedImports) Name() string {
	return `PhpCsFixer\Fixer\Import\OrderedImportsFixer`
}

func (OrderedImports) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/OrderedImportsFixer.php"
}

func (OrderedImports) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" || inClassLikeBody(s, i) {
			i++
			continue
		}
		run := collectImportRun(s, i)
		if len(run) < 2 {
			i++
			continue
		}
		newEnd, c := reorderImports(s, run)
		if c {
			changed = true
		}
		i = newEnd
	}
	return changed
}

func reorderImports(s *tokens.Stream, run []importStmt) (int, bool) {
	first := run[0].start
	last := run[len(run)-1].semi
	indent := indentBefore(s, first)

	ordered := append([]importStmt(nil), run...)
	sort.SliceStable(ordered, func(a, b int) bool {
		// ECS default imports_order: class, then function, then const; alpha within
		if ra, rb := importKindRank(ordered[a].kind), importKindRank(ordered[b].kind); ra != rb {
			return ra < rb
		}
		return ordered[a].pathKey < ordered[b].pathKey
	})

	var repl []token.Token
	for p, st := range ordered {
		if p > 0 {
			repl = append(repl, token.Token{Kind: token.Whitespace, Value: "\n" + indent})
		}
		for k := st.start; k <= st.semi; k++ {
			repl = append(repl, s.At(k))
		}
	}

	orig := make([]token.Token, 0, last-first+1)
	for k := first; k <= last; k++ {
		orig = append(orig, s.At(k))
	}
	changed := !tokensEqual(orig, repl)
	s.ReplaceRange(first, last, repl)
	return first + len(repl), changed
}

func tokensEqual(a, b []token.Token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind || a[i].Value != b[i].Value {
			return false
		}
	}
	return true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBetweenImportGroupsFixer.php
//
// BlankLineBetweenImportGroups puts a blank line between class, function and
// const import groups.
type BlankLineBetweenImportGroups struct{}

func (BlankLineBetweenImportGroups) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\BlankLineBetweenImportGroupsFixer`
}

func (BlankLineBetweenImportGroups) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBetweenImportGroupsFixer.php"
}

func (BlankLineBetweenImportGroups) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" || inClassLikeBody(s, i) {
			i++
			continue
		}
		run := collectImportRun(s, i)
		if len(run) >= 2 {
			for p := 1; p < len(run); p++ {
				if run[p].kind == run[p-1].kind {
					continue
				}
				ws := run[p].start - 1
				if ws >= 0 && s.At(ws).Kind == token.Whitespace &&
					hasNewline(s.At(ws).Value) && s.At(ws).Value != "\n\n" {
					s.SetValue(ws, "\n\n")
					changed = true
				}
			}
		}
		if len(run) > 0 {
			i = run[len(run)-1].semi + 1
		} else {
			i++
		}
	}
	return changed
}
