package rules

import (
	"regexp"
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoSuperfluousPhpdocTagsFixer.php
//
// NoSuperfluousPhpdocTags removes a @param/@return tag that only repeats the
// function's native type declaration and carries no description. A tag whose
// phpdoc type is more specific than the native type (generics, array shapes,
// callable signatures) or that has a description is kept.
type NoSuperfluousPhpdocTags struct{}

func (NoSuperfluousPhpdocTags) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\NoSuperfluousPhpdocTagsFixer`
}

func (NoSuperfluousPhpdocTags) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoSuperfluousPhpdocTagsFixer.php"
}

var (
	nspParamRe  = regexp.MustCompile(`(?i)^@param\s+(\S+)\s+(&?\.{0,3}\$[A-Za-z_][A-Za-z0-9_]*)\s*(.*)$`)
	nspReturnRe = regexp.MustCompile(`(?i)^@return\s+(\S+)\s*(.*)$`)
	nspVarRe    = regexp.MustCompile(`(?i)^@var\s+(\S+)(?:\s+\$[A-Za-z_][A-Za-z0-9_]*)?\s*(.*)$`)
)

func (NoSuperfluousPhpdocTags) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		sig, ok := signatureAfter(s, i)
		if !ok {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok || d.single {
			continue
		}
		if filterSuperfluous(&d, sig) {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

type funcSig struct {
	params  map[string]string // $var name (without $) -> normalized native type ("" if none)
	ret     string            // normalized native return type ("" if none)
	hasRet  bool
	varType string // normalized native type of a documented property
	hasVar  bool   // the docblock documents a property
}

// signatureAfter finds the function declaration a docblock at index i documents
// and extracts its parameter and return types.
func signatureAfter(s *tokens.Stream, doc int) (funcSig, bool) {
	// Walk forward across whitespace, attributes (#[...] Comment) and modifier
	// keywords to the "function" keyword.
	j := doc + 1
	for j < s.Len() {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment:
			j++
			continue
		case token.Keyword:
			lv := strings.ToLower(t.Value)
			if lv == "function" {
				return parseSignature(s, j)
			}
			if lv == "const" {
				return funcSig{}, false
			}
			if lv == "var" || isFuncModifier(lv) || lv == "readonly" {
				j++
				continue
			}
			// a typed property whose type is a reserved word (e.g. "array")
			return parseProperty(s, j)
		case token.Ident, token.Punct, token.Variable:
			// a property: "Foo $x", "?Foo $x", "\Foo\Bar $x", or untyped "$x"
			return parseProperty(s, j)
		default:
			return funcSig{}, false
		}
	}
	return funcSig{}, false
}

// parseProperty reads the native type of a property declaration starting at the
// first type (or variable) token, for validating a documenting @var tag.
func parseProperty(s *tokens.Stream, start int) (funcSig, bool) {
	var typeToks []string
	for k := start; k < s.Len(); k++ {
		t := s.At(k)
		switch t.Kind {
		case token.Whitespace, token.Comment:
			continue
		case token.Variable:
			return funcSig{hasVar: true, varType: normalizeType(strings.Join(typeToks, ""))}, true
		case token.Ident, token.Keyword:
			typeToks = append(typeToks, t.Value)
		case token.Punct:
			if t.Value == "?" || t.Value == "|" || t.Value == "&" || t.Value == `\` {
				typeToks = append(typeToks, t.Value)
				continue
			}
			return funcSig{}, false
		default:
			return funcSig{}, false
		}
	}
	return funcSig{}, false
}

func isFuncModifier(v string) bool {
	switch v {
	case "public", "private", "protected", "static", "final", "abstract":
		return true
	}
	return false
}

// parseSignature parses "(...)" params and an optional ": returnType" starting at
// the "function" keyword index.
func parseSignature(s *tokens.Stream, fn int) (funcSig, bool) {
	open := nextSignificantIndex(s, fn)
	// skip an optional function name and the "&" of a by-ref function
	for open >= 0 && s.At(open).Kind != token.Punct {
		open = nextSignificantIndex(s, open)
	}
	if open < 0 || s.At(open).Value == "&" {
		open = nextSignificantIndex(s, open)
	}
	if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return funcSig{}, false
	}
	close := s.MatchForward(open)
	if close < 0 {
		return funcSig{}, false
	}
	sig := funcSig{params: map[string]string{}}
	parseParams(s, open, close, &sig)

	// return type: ": type ..." up to "{" or ";"
	c := nextSignificantIndex(s, close)
	if c >= 0 && s.At(c).Kind == token.Punct && s.At(c).Value == ":" {
		var parts []string
		for k := c + 1; k < s.Len(); k++ {
			t := s.At(k)
			if t.Kind == token.Whitespace {
				continue
			}
			if t.Kind == token.Punct && (t.Value == "{" || t.Value == ";") {
				break
			}
			parts = append(parts, t.Value)
		}
		sig.ret = normalizeType(strings.Join(parts, ""))
		sig.hasRet = true
	}
	return sig, true
}

// parseParams collects each parameter's variable name and native type between the
// parameter-list parentheses.
func parseParams(s *tokens.Stream, open, close int, sig *funcSig) {
	var typeToks []string
	var varName string
	depth := 0
	inDefault := false
	flush := func() {
		if varName != "" {
			sig.params[varName] = normalizeType(strings.Join(typeToks, ""))
		}
		typeToks = nil
		varName = ""
		inDefault = false
	}
	for k := open + 1; k < close; k++ {
		t := s.At(k)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case ",":
				if depth == 0 {
					flush()
					continue
				}
			case "=":
				if depth == 0 {
					inDefault = true
					continue
				}
			}
			if !inDefault && depth == 0 && (t.Value == "?" || t.Value == "|" || t.Value == "&" || t.Value == `\`) {
				typeToks = append(typeToks, t.Value)
			}
			continue
		}
		if inDefault {
			continue
		}
		switch t.Kind {
		case token.Variable:
			if depth == 0 {
				varName = strings.TrimPrefix(t.Value, "$")
			}
		case token.Ident, token.Keyword:
			if depth == 0 && !isParamModifier(strings.ToLower(t.Value)) {
				typeToks = append(typeToks, t.Value)
			}
		}
	}
	flush()
}

func isParamModifier(v string) bool {
	switch v {
	case "public", "private", "protected", "readonly":
		return true
	}
	return false
}

// filterSuperfluous drops @param/@return lines made redundant by sig. Reports
// whether anything was removed.
func filterSuperfluous(d *docblock, sig funcSig) bool {
	kept := make([]docLine, 0, len(d.inner))
	removed := false
	for idx := 0; idx < len(d.inner); idx++ {
		l := d.inner[idx]
		content := strings.TrimLeft(l.content, " ")
		if drop, ok := superfluousTag(content, sig); ok && drop && !hasContinuation(d.inner, idx) {
			removed = true
			continue
		}
		kept = append(kept, l)
	}
	if removed {
		d.inner = kept
	}
	return removed
}

// hasContinuation reports whether the tag line at idx is followed by a wrapped
// description line (a non-blank, non-tag inner line), which means it carries a
// description and must be kept.
func hasContinuation(inner []docLine, idx int) bool {
	if idx+1 >= len(inner) {
		return false
	}
	next := strings.TrimSpace(inner[idx+1].content)
	return next != "" && !strings.HasPrefix(next, "@")
}

// superfluousTag reports whether a @param/@return line is redundant given sig.
func superfluousTag(content string, sig funcSig) (bool, bool) {
	if m := nspParamRe.FindStringSubmatch(content); m != nil {
		phpType, name, desc := m[1], m[2], strings.TrimSpace(m[3])
		if desc != "" {
			return false, true
		}
		varName := name[strings.IndexByte(name, '$')+1:]
		native, ok := sig.params[varName]
		if !ok {
			return false, true
		}
		return typeIsSuperfluous(phpType, native), true
	}
	if m := nspReturnRe.FindStringSubmatch(content); m != nil {
		phpType, desc := m[1], strings.TrimSpace(m[2])
		if desc != "" {
			return false, true
		}
		return typeIsSuperfluous(phpType, sig.ret), true
	}
	if sig.hasVar {
		if m := nspVarRe.FindStringSubmatch(content); m != nil {
			phpType, desc := m[1], strings.TrimSpace(m[2])
			if desc != "" {
				return false, true
			}
			return typeIsSuperfluous(phpType, sig.varType), true
		}
	}
	return false, false
}

// typeIsSuperfluous reports whether a phpdoc type adds nothing over the native
// type: it normalizes to exactly the native type and is not more specific (no
// generics/shapes/callable signatures). A "mixed" tag is superfluous only when
// the native type is also mixed; on an untyped param/return "@param mixed" adds
// information and is kept, matching ECS (allow_mixed).
func typeIsSuperfluous(phpType, native string) bool {
	if strings.ContainsAny(phpType, "<{(") {
		return false
	}
	if native == "" {
		return false
	}
	return normalizeType(phpType) == native
}

// normalizeType canonicalizes a type for comparison: lowercase, each union member
// reduced to its short name, "?T" expanded to the "null|t" set, members sorted
// and de-duplicated.
func normalizeType(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	nullable := strings.HasPrefix(t, "?")
	t = strings.TrimPrefix(t, "?")
	seps := func(r rune) bool { return r == '|' || r == '&' }
	members := strings.FieldsFunc(t, seps)
	set := map[string]struct{}{}
	for _, m := range members {
		m = strings.ToLower(strings.TrimSpace(m))
		if i := strings.LastIndex(m, `\`); i >= 0 {
			m = m[i+1:]
		}
		if m != "" {
			set[m] = struct{}{}
		}
	}
	if nullable {
		set["null"] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return strings.Join(out, "|")
}
