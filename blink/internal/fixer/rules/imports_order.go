package rules

import (
	"sort"
	"strings"

	"blink/internal/fixer"
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
//
// The fields mirror the fixer options; their zero values reproduce today's
// behavior (alpha sort, grouped class/function/const, case-insensitive):
//   - sortAlgorithm: "" == "alpha" (also "length" or "none")
//   - orderKinds: nil == default [class, function, const]
//   - noGrouping: "imports_order" explicitly null (sort without grouping)
//   - caseSensitive: "case_sensitive"
type OrderedImports struct {
	sortAlgorithm string
	orderKinds    []string
	noGrouping    bool
	caseSensitive bool
}

func (f OrderedImports) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["sort_algorithm"].(string); ok {
		switch v {
		case "length":
			f.sortAlgorithm = "length"
		case "none":
			f.sortAlgorithm = "none"
		default:
			f.sortAlgorithm = ""
		}
	}
	if raw, exists := config["imports_order"]; exists {
		if raw == nil {
			f.noGrouping = true
		} else if list := orderedImportsConfigStrings(raw); list != nil {
			f.orderKinds = list
		}
	}
	if v, ok := config["case_sensitive"].(bool); ok {
		f.caseSensitive = v
	}
	return f
}

// orderedImportsConfigStrings converts a config list ([]any or []string) to []string.
func orderedImportsConfigStrings(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if sv, ok := e.(string); ok {
				out = append(out, sv)
			}
		}
		return out
	}
	return nil
}

func (OrderedImports) Name() string {
	return `PhpCsFixer\Fixer\Import\OrderedImportsFixer`
}

func (OrderedImports) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/OrderedImportsFixer.php"
}

func (f OrderedImports) Fix(s *tokens.Stream) bool {
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
		newEnd, c := f.reorderImports(s, run)
		if c {
			changed = true
		}
		i = newEnd
	}
	return changed
}

func (f OrderedImports) reorderImports(s *tokens.Stream, run []importStmt) (int, bool) {
	first := run[0].start
	last := run[len(run)-1].semi
	indent := indentBefore(s, first)

	ordered := append([]importStmt(nil), run...)
	sort.SliceStable(ordered, func(a, b int) bool {
		if !f.noGrouping {
			if ra, rb := f.orderedImportsRank(ordered[a].kind), f.orderedImportsRank(ordered[b].kind); ra != rb {
				return ra < rb
			}
		}
		switch f.sortAlgorithm {
		case "none":
			return false // keep original order within a group
		case "length":
			if la, lb := f.orderedImportsLen(s, ordered[a]), f.orderedImportsLen(s, ordered[b]); la != lb {
				return la < lb
			}
			return f.orderedImportsAlphaKey(s, ordered[a]) < f.orderedImportsAlphaKey(s, ordered[b])
		default: // alpha
			return f.orderedImportsAlphaKey(s, ordered[a]) < f.orderedImportsAlphaKey(s, ordered[b])
		}
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

// orderedImportsRank returns the group index of a kind per the configured order,
// or the default class/function/const order when none is configured.
func (f OrderedImports) orderedImportsRank(kind string) int {
	kinds := f.orderKinds
	if kinds == nil {
		return importKindRank(kind)
	}
	for i, k := range kinds {
		if k == kind {
			return i
		}
	}
	return len(kinds)
}

// orderedImportsAlphaKey returns the alphabetical sort key for a statement,
// honoring case sensitivity. The default (case-insensitive) reuses the key
// precomputed in collectImportRun.
func (f OrderedImports) orderedImportsAlphaKey(s *tokens.Stream, st importStmt) string {
	if !f.caseSensitive {
		return st.pathKey
	}
	return strings.ReplaceAll(orderedImportsPath(s, st.start, st.semi), `\`, "\x00")
}

// orderedImportsLen returns the length used by the (deprecated) length algorithm:
// the imported path plus a "function "/"const " prefix for non-class imports.
func (f OrderedImports) orderedImportsLen(s *tokens.Stream, st importStmt) int {
	n := len(orderedImportsPath(s, st.start, st.semi))
	if st.kind != "class" {
		n += len(st.kind) + 1
	}
	return n
}

// orderedImportsPath returns the imported path (case preserved) after an optional
// "function"/"const", up to ";" or " as ".
func orderedImportsPath(s *tokens.Stream, useIdx, semi int) string {
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
	return b.String()
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
