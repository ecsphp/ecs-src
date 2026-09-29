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

	nsLower := strings.ToLower(fileNamespace(s))

	inImport := func(idx int) bool {
		for _, im := range imports {
			if idx >= im.start && idx <= im.semi {
				return true
			}
		}
		return false
	}

	changed := false
	for _, im := range slices.Backward(imports) {
		// an import of the current namespace ("namespace\Name") is redundant
		redundant := !im.aliased && nsLower != "" &&
			strings.HasPrefix(im.fullLower, nsLower+`\`) &&
			!strings.Contains(im.fullLower[len(nsLower)+1:], `\`)
		if !redundant && importIsUsed(s, im, inImport) {
			continue
		}
		removeImport(s, im)
		changed = true
	}
	return changed
}

// importIsUsed reports whether the import's short name appears as a matching-kind
// reference or in a comment (word-boundary).
func importIsUsed(s *tokens.Stream, im importInfo, inImport func(int) bool) bool {
	for k := 0; k < s.Len(); k++ {
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

// fileNamespace returns the first top-level namespace name, or "".
func fileNamespace(s *tokens.Stream) string {
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || strings.ToLower(s.At(i).Value) != "namespace" || memberPrev(s, i) {
			continue
		}
		var b strings.Builder
		for k := i + 1; k < s.Len(); k++ {
			t := s.At(k)
			if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{") {
				break
			}
			if t.Kind == token.Whitespace {
				continue
			}
			b.WriteString(t.Value)
		}
		return strings.TrimPrefix(b.String(), `\`)
	}
	return ""
}

func removeImport(s *tokens.Stream, im importInfo) {
	for r := im.semi; r >= im.start; r-- {
		s.RemoveAt(r)
	}
	// drop just the statement's own line break, keeping any blank line
	if im.start < s.Len() && s.At(im.start).Kind == token.Whitespace {
		if v := s.At(im.start).Value; strings.HasPrefix(v, "\n") {
			s.SetValue(im.start, v[1:])
		}
	}
}
