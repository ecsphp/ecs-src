package rules

import (
	"regexp"
	"sort"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoSuperfluousPhpdocTagsFixer.php
//
// NoSuperfluousPhpdocTags removes a @param/@return tag that only repeats the
// function's native type declaration and carries no description. A tag whose
// phpdoc type is more specific than the native type (generics, array shapes,
// callable signatures) or that has a description is kept.
//
// The zero value matches blink's built-in behavior (equivalent to allow_mixed and
// allow_unused_params). WithConfig honors allow_mixed, allow_unused_params,
// allow_hidden_params and remove_inheritdoc.
type NoSuperfluousPhpdocTags struct {
	mixedIsSuperfluous      bool // allow_mixed=false: @param/@return mixed on an untyped element is superfluous
	unusedParamsSuperfluous bool // allow_unused_params=false: a @param not in the signature is superfluous
	removeInheritdoc        bool // remove_inheritdoc=true: drop standalone @inheritDoc lines
	allowHiddenParams       bool // keep a @param named inside a signature comment
}

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

	nspInheritDocRe = regexp.MustCompile(`(?i)^\{?@inheritdocs?\}?$`)
)

func (f NoSuperfluousPhpdocTags) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["allow_mixed"].(bool); ok {
		f.mixedIsSuperfluous = !v
	}
	if v, ok := config["allow_unused_params"].(bool); ok {
		f.unusedParamsSuperfluous = !v
	}
	if v, ok := config["remove_inheritdoc"].(bool); ok {
		f.removeInheritdoc = v
	}
	if v, ok := config["allow_hidden_params"].(bool); ok {
		f.allowHiddenParams = v
	}
	return f
}

func (f NoSuperfluousPhpdocTags) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok || d.single {
			continue
		}
		docChanged := false
		if f.removeInheritdoc && nspRemoveInheritDoc(&d) {
			docChanged = true
		}
		if sig, ok := signatureAfter(s, i); ok && f.filterSuperfluous(&d, sig) {
			docChanged = true
		}
		if docChanged {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

// nspRemoveInheritDoc drops inner lines that are a standalone @inheritDoc /
// @inheritDocs tag (bare or inline braces), reporting whether anything changed.
// It mirrors PHP-CS-Fixer's removeSuperfluousInheritDoc regex: the tag is only
// dropped when it is bounded by the comment start or a tag before it and by a
// tag or the comment end after it (blank lines aside), so an @inheritDoc mixed
// with a real description is left in place.
func nspRemoveInheritDoc(d *docblock) bool {
	kept := d.inner[:0:0]
	removed := false
	for i, l := range d.inner {
		if nspInheritDocRe.MatchString(strings.TrimSpace(l.content)) &&
			nspInheritDocBounded(d.inner, i, -1) && nspInheritDocBounded(d.inner, i, 1) {
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

// nspInheritDocBounded reports whether the inner line before (step -1) or after
// (step +1) idx is absent (comment start/end) or a tag, skipping blank lines.
func nspInheritDocBounded(inner []docLine, idx, step int) bool {
	for j := idx + step; j >= 0 && j < len(inner); j += step {
		content := strings.TrimSpace(inner[j].content)
		if content == "" {
			continue
		}
		return strings.Contains(content, "@")
	}
	return true
}

type funcSig struct {
	params  map[string]string // $var name (without $) -> normalized native type ("" if none)
	hidden  map[string]bool   // param names appearing only in signature comments
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
	sig := funcSig{params: map[string]string{}, hidden: map[string]bool{}}
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
			if t.Kind == token.Comment || t.Kind == token.DocComment {
				for _, name := range nspCommentParamNames(t.Value) {
					sig.hidden[name] = true
				}
			}
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
func (f NoSuperfluousPhpdocTags) filterSuperfluous(d *docblock, sig funcSig) bool {
	kept := make([]docLine, 0, len(d.inner))
	removed := false
	for idx := 0; idx < len(d.inner); idx++ {
		l := d.inner[idx]
		content := strings.TrimLeft(l.content, " ")
		if drop, ok := f.superfluousTag(content, sig); ok && drop && !hasContinuation(d.inner, idx) {
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
func (f NoSuperfluousPhpdocTags) superfluousTag(content string, sig funcSig) (bool, bool) {
	if m := nspParamRe.FindStringSubmatch(content); m != nil {
		phpType, name, desc := m[1], m[2], strings.TrimSpace(m[3])
		if desc != "" {
			return false, true
		}
		varName := name[strings.IndexByte(name, '$')+1:]
		native, ok := sig.params[varName]
		if !ok {
			// param not in signature
			if !f.unusedParamsSuperfluous {
				return false, true
			}
			if f.allowHiddenParams && sig.hidden[varName] {
				return false, true
			}
			return true, true
		}
		return f.typeIsSuperfluous(phpType, native), true
	}
	if m := nspReturnRe.FindStringSubmatch(content); m != nil {
		phpType, desc := m[1], strings.TrimSpace(m[2])
		if desc != "" {
			return false, true
		}
		return f.typeIsSuperfluous(phpType, sig.ret), true
	}
	if sig.hasVar {
		if m := nspVarRe.FindStringSubmatch(content); m != nil {
			phpType, desc := m[1], strings.TrimSpace(m[2])
			if desc != "" {
				return false, true
			}
			return f.typeIsSuperfluous(phpType, sig.varType), true
		}
	}
	return false, false
}

// typeIsSuperfluous reports whether a phpdoc type adds nothing over the native
// type: it normalizes to exactly the native type and is not more specific (no
// generics/shapes/callable signatures). On an untyped element only a "mixed" tag
// can be superfluous, and only when allow_mixed is off (mixedIsSuperfluous).
func (f NoSuperfluousPhpdocTags) typeIsSuperfluous(phpType, native string) bool {
	if strings.ContainsAny(phpType, "<{(") {
		return false
	}
	if native == "" {
		return f.mixedIsSuperfluous && normalizeType(phpType) == "mixed"
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

var nspVarNameRe = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*`)

// nspCommentParamNames extracts variable names from a comment inside the parameter
// list, mirroring the "hidden params" the fixer virtualises for allow_hidden_params.
func nspCommentParamNames(v string) []string {
	matches := nspVarNameRe.FindAllString(v, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1:])
	}
	return out
}
