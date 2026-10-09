package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

type importInfo struct {
	start, semi int
	shortLower  string
	kind        string // "class", "function" or "const"
	fullLower   string // full imported path, lowercased, no leading "\"
	aliased     bool
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnusedImportsFixer.php
//
// NoUnusedImports removes a top-level "use" import that is never referenced. It
// matches usage by import kind (class/function/const), treats a name in a comment
// as used (word-boundary), and removes imports of the current namespace.
type NoUnusedImports struct{}

func (NoUnusedImports) Name() string {
	return `PhpCsFixer\Fixer\Import\NoUnusedImportsFixer`
}

func (NoUnusedImports) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnusedImportsFixer.php"
}

func (NoUnusedImports) Fix(s *tokens.Stream) bool {
	var imports []importInfo
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" || inClassLikeBody(s, i) {
			continue
		}
		j := skipWhitespace(s, i+1)
		if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue // closure use
		}
		kind := "class"
		if j < s.Len() && s.At(j).Kind == token.Keyword {
			switch strings.ToLower(s.At(j).Value) {
			case "function":
				kind = "function"
			case "const":
				kind = "const"
			}
		}

		semi, group := -1, false
		for k := i + 1; k < s.Len(); k++ {
			if s.At(k).Kind != token.Punct {
				continue
			}
			if s.At(k).Value == "{" {
				group = true
				break
			}
			if s.At(k).Value == ";" {
				semi = k
				break
			}
		}
		if group || semi < 0 {
			continue
		}

		short, full, aliased := importParts(s, i, semi, kind)
		if short == "" {
			continue
		}
		imports = append(imports, importInfo{
			start: i, semi: semi, shortLower: strings.ToLower(short),
			kind: kind, fullLower: strings.ToLower(full), aliased: aliased,
		})
		i = semi
	}
	if len(imports) == 0 {
		return false
	}

	regions := namespaceRegions(s)

	inImport := func(idx int) bool {
		for _, im := range imports {
			if idx >= im.start && idx <= im.semi {
				return true
			}
		}
		return false
	}

	regionOf := func(idx int) nsRegion {
		for _, r := range regions {
			if idx >= r.start && idx <= r.end {
				return r
			}
		}
		return nsRegion{start: 0, end: s.Len() - 1}
	}

	// Decide first, remove after: usage detection ignores import token ranges,
	// so removals never change another import's used-ness, and deciding on the
	// pristine stream avoids stale indices from earlier removals.
	var toRemove []importInfo
	for _, im := range imports {
		r := regionOf(im.start)
		nsLower := r.nameLower
		// an import of the current namespace ("namespace\Name") is redundant
		redundant := !im.aliased && nsLower != "" &&
			strings.HasPrefix(im.fullLower, nsLower+`\`) &&
			!strings.Contains(im.fullLower[len(nsLower)+1:], `\`)
		if !redundant && importIsUsed(s, im, r.start, r.end, inImport) {
			continue
		}
		toRemove = append(toRemove, im)
	}
	if len(toRemove) == 0 {
		return false
	}
	for _, im := range slices.Backward(toRemove) {
		removeImport(s, im)
	}
	return true
}

// importIsUsed reports whether the import's short name appears as a matching-kind
// reference or in a comment (word-boundary) within its namespace region [lo, hi].
func importIsUsed(s *tokens.Stream, im importInfo, lo, hi int, inImport func(int) bool) bool {
	if lo < 0 {
		lo = 0
	}
	if hi >= s.Len() {
		hi = s.Len() - 1
	}
	for k := lo; k <= hi; k++ {
		t := s.At(k)
		if t.Kind == token.Ident && !inImport(k) && strings.ToLower(t.Value) == im.shortLower {
			if usageMatchesKind(s, k, im.kind) {
				return true
			}
			continue
		}
		// a class name colliding with a PHP keyword (enum, list, ...) is lexed as a
		// keyword; count it when used in a class-reference position
		if t.Kind == token.Keyword && im.kind == "class" && !inImport(k) &&
			strings.ToLower(t.Value) == im.shortLower && keywordIsClassRef(s, k) {
			return true
		}
		if (t.Kind == token.Comment || t.Kind == token.DocComment) && commentReferences(t.Value, im.shortLower) {
			return true
		}
	}
	return false
}

// usageMatchesKind reports whether the identifier at k is a reference of the
// given import kind.
func usageMatchesKind(s *tokens.Stream, k int, kind string) bool {
	prev, hasPrev := prevSignificant(s, k)
	if hasPrev {
		if prev.Kind == token.Punct {
			switch prev.Value {
			case `\`, "->", "?->", "::":
				return false
			}
		}
		if prev.Kind == token.Keyword {
			switch strings.ToLower(prev.Value) {
			case "namespace", "function":
				return false
			case "const":
				if nextSignificantValue(s, k) == "=" {
					return false
				}
			}
		}
	}
	prevNew := hasPrev && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new")
	fnCall := nextSignificantValue(s, k) == "(" && !prevNew
	if kind == "function" {
		return fnCall
	}
	// class and const references are any matching identifier that is not a call
	return !fnCall
}

// commentReferences reports whether short appears in v as a whole word, not
// preceded by an identifier char, "$" or "\".
func commentReferences(v, short string) bool {
	lv := strings.ToLower(v)
	from := 0
	for {
		idx := strings.Index(lv[from:], short)
		if idx < 0 {
			return false
		}
		p := from + idx
		before := byte(' ')
		if p > 0 {
			before = lv[p-1]
		}
		after := byte(' ')
		if p+len(short) < len(lv) {
			after = lv[p+len(short)]
		}
		if !isIdentByte(before) && before != '$' && before != '\\' && !isIdentByte(after) {
			return true
		}
		from = p + 1
	}
}

func keywordIsClassRef(s *tokens.Stream, k int) bool {
	if prev, ok := prevSignificant(s, k); ok {
		if prev.Kind == token.Keyword {
			switch strings.ToLower(prev.Value) {
			case "extends", "implements", "new", "instanceof":
				return true
			}
		}
		if prev.Kind == token.Punct {
			switch prev.Value {
			case "(", ",", ":", "|", "&", "?":
				return true
			}
		}
	}
	if n := nextSignificantIndex(s, k); n >= 0 {
		if s.At(n).Kind == token.Punct && s.At(n).Value == "::" {
			return true
		}
		if s.At(n).Kind == token.Variable {
			return true
		}
	}
	return false
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// importParts returns the imported short name, full path and whether it is
// aliased, for the use statement spanning (useIdx, semi).
func importParts(s *tokens.Stream, useIdx, semi int, kind string) (short, full string, aliased bool) {
	j := skipWhitespace(s, useIdx+1)
	if kind != "class" {
		j = skipWhitespace(s, j+1) // skip "function"/"const"
	}
	var path strings.Builder
	alias := ""
	inAlias := false
	for k := j; k < semi; k++ {
		t := s.At(k)
		if t.Kind == token.Keyword && strings.ToLower(t.Value) == "as" {
			inAlias = true
			aliased = true
			continue
		}
		if t.Kind == token.Whitespace {
			continue
		}
		if inAlias {
			if t.Kind == token.Ident {
				alias = t.Value
			}
			continue
		}
		path.WriteString(t.Value)
	}
	full = strings.TrimPrefix(path.String(), `\`)
	if alias != "" {
		short = alias
	} else if i := strings.LastIndex(full, `\`); i >= 0 {
		short = full[i+1:]
	} else {
		short = full
	}
	return short, full, aliased
}

// nsRegion is a namespace scope: the token index range [start, end] that a use
// statement belongs to and whose usages count, plus the lowercased namespace name.
type nsRegion struct {
	start, end int
	nameLower  string
}

// namespaceRegions partitions the stream into namespace scopes, mirroring PHP-CS-Fixer's
// per-namespace handling. A file with no namespace is one global region; brace-style
// `namespace X { ... }` blocks each scope to their braces (including a global `namespace { ... }`);
// semicolon-style `namespace X;` scopes to the next namespace declaration or end of file.
func namespaceRegions(s *tokens.Stream) []nsRegion {
	var starts []int
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.ToLower(s.At(i).Value) == "namespace" && !memberPrev(s, i) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return []nsRegion{{start: 0, end: s.Len() - 1}}
	}

	var regions []nsRegion
	for idx, ni := range starts {
		var name strings.Builder
		delim, brace := -1, false
		for k := ni + 1; k < s.Len(); k++ {
			t := s.At(k)
			if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{") {
				delim, brace = k, t.Value == "{"
				break
			}
			if t.Kind == token.Whitespace {
				continue
			}
			name.WriteString(t.Value)
		}
		if delim < 0 {
			continue
		}
		nameLower := strings.ToLower(strings.TrimPrefix(name.String(), `\`))
		if brace {
			regions = append(regions, nsRegion{start: delim, end: matchCloseBrace(s, delim), nameLower: nameLower})
			continue
		}
		end := s.Len() - 1
		if idx+1 < len(starts) {
			end = starts[idx+1] - 1
		}
		regions = append(regions, nsRegion{start: ni, end: end, nameLower: nameLower})
	}
	return regions
}

// matchCloseBrace returns the index of the "}" matching the "{" at open.
func matchCloseBrace(s *tokens.Stream, open int) int {
	depth := 0
	for k := open; k < s.Len(); k++ {
		if s.At(k).Kind != token.Punct {
			continue
		}
		switch s.At(k).Value {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return s.Len() - 1
}

func removeImport(s *tokens.Stream, im importInfo) {
	s.ReplaceRange(im.start, im.semi, nil)
	// drop just the statement's own line break, keeping any blank line
	if im.start < s.Len() && s.At(im.start).Kind == token.Whitespace {
		if v := s.At(im.start).Value; strings.HasPrefix(v, "\n") {
			s.SetValue(im.start, v[1:])
		}
	}
}
