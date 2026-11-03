package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

type importInfo struct {
	start, semi int
	shortLower  string
	kind        string // "class", "function" or "const"
	full        string // full imported path as written, no leading "\"
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
		// comments are skipped too: "use /* x */ function Foo\bar;" imports a function
		j := nextMeaningfulIndex(s, i)
		if j >= 0 && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue // closure use
		}
		kind := "class"
		if j >= 0 && s.At(j).Kind == token.Keyword {
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
			kind: kind, full: full, aliased: aliased,
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

	regionOf := func(idx int) int {
		for ri, r := range regions {
			if idx >= r.start && idx <= r.end {
				return ri
			}
		}
		return -1
	}

	// Decide first, remove after: usage detection ignores import token ranges,
	// so removals never change another import's used-ness. The removals then
	// replay PHP-CS-Fixer's order, namespace by namespace: first the unused
	// imports, then the imports of the current namespace ("namespace\Name"),
	// which PHP removes a second time when they were also unused.
	var removals []importInfo
	var unused, redundant []importInfo
	flush := func() {
		removals = append(removals, unused...)
		removals = append(removals, redundant...)
		unused, redundant = nil, nil
	}
	curRegion := -2
	for _, im := range imports {
		ri := regionOf(im.start)
		if ri != curRegion {
			flush()
			curRegion = ri
		}
		r := nsRegion{start: 0, end: s.Len() - 1}
		if ri >= 0 {
			r = regions[ri]
		}
		if !importIsUsed(s, im, r.start, r.end, inImport) {
			unused = append(unused, im)
		}
		// PHP-CS-Fixer compares the paths case-sensitively (strncmp), so a path
		// that differs from the namespace only by case is another name
		if !im.aliased && r.name != "" && strings.HasPrefix(im.full, r.name+`\`) &&
			!strings.Contains(im.full[len(r.name)+1:], `\`) {
			redundant = append(redundant, im)
		}
	}
	flush()
	if len(removals) == 0 {
		return false
	}
	ts := newClearableTokens(s)
	for _, im := range removals {
		ts.removeUseDeclaration(ts.at[im.start], ts.at[im.semi])
	}
	ts.writeBack(s)
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

// attributeNameAt reports whether the identifier at k is an attribute name - the
// first segment right after "#[", or after a top-level comma in a grouped attribute
// "#[A, B]". Such a name is a class reference regardless of a following "(".
func attributeNameAt(s *tokens.Stream, k int) bool {
	p := prevSignificantIndex(s, k)
	if p < 0 {
		return false
	}
	if s.At(p).Kind == token.AttributeStart {
		return true
	}
	if s.At(p).Kind != token.Punct || s.At(p).Value != "," {
		return false
	}
	depth := 0
	for j := p - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.AttributeStart && depth == 0 {
			return true
		}
		if t.Kind == token.AttributeEnd {
			return false
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case ")", "]", "}":
				depth++
			case "(", "[", "{":
				if depth == 0 {
					return false
				}
				depth--
			}
		}
	}
	return false
}

// usageMatchesKind reports whether the identifier at k is a reference of the
// given import kind.
func usageMatchesKind(s *tokens.Stream, k int, kind string) bool {
	if attributeNameAt(s, k) {
		return kind == "class" // an attribute is always a class reference, even with "(" after
	}
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
	j := nextMeaningfulIndex(s, useIdx)
	if kind != "class" {
		j++ // past "function"/"const"; the loop skips whitespace and comments
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
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
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
// statement belongs to and whose usages count, plus the namespace name as written.
type nsRegion struct {
	start, end int
	name       string
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
		nsName := strings.TrimPrefix(name.String(), `\`)
		if brace {
			regions = append(regions, nsRegion{start: delim, end: matchCloseBrace(s, delim), name: nsName})
			continue
		}
		end := s.Len() - 1
		if idx+1 < len(starts) {
			end = starts[idx+1] - 1
		}
		regions = append(regions, nsRegion{start: ni, end: end, name: nsName})
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

// clearableTokens mirrors PHP-CS-Fixer's Tokens for the import removal: a
// removed token is cleared in place (an empty token) instead of being cut out,
// so indices stay stable across successive removals and the cleanup helpers
// see cleared tokens as PHP does (as empty whitespace). The open tag carries
// its first whitespace character ("<?php " or "<?php\n"), as T_OPEN_TAG does,
// so the clean-up never joins "<?php" to the next token; at maps a stream
// index to its index here.
type clearableTokens struct {
	toks    []token.Token
	cleared []bool
	at      []int
}

func newClearableTokens(s *tokens.Stream) *clearableTokens {
	src := s.Tokens()
	c := &clearableTokens{toks: make([]token.Token, 0, len(src)), at: make([]int, len(src))}
	for i := 0; i < len(src); i++ {
		t := src[i]
		c.at[i] = len(c.toks)
		if t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=") && i+1 < len(src) && src[i+1].Kind == token.Whitespace {
			n := src[i+1]
			head := n.Value[:1]
			if strings.HasPrefix(n.Value, "\r\n") {
				head = "\r\n"
			}
			t.Value += head
			c.toks = append(c.toks, t)
			i++
			c.at[i] = len(c.toks) - 1
			if rest := n.Value[len(head):]; rest != "" {
				c.at[i] = len(c.toks)
				c.toks = append(c.toks, token.Token{Kind: token.Whitespace, Value: rest, Pos: n.Pos + len(head)})
			}
			continue
		}
		c.toks = append(c.toks, t)
	}
	c.cleared = make([]bool, len(c.toks))
	return c
}

// writeBack replaces the stream's tokens with the non-cleared ones, splitting
// the open tag back into blink's shape and merging the whitespace runs that
// clearing left side by side.
func (c *clearableTokens) writeBack(s *tokens.Stream) {
	out := make([]token.Token, 0, len(c.toks))
	push := func(t token.Token) {
		if t.Value == "" {
			return
		}
		if l := len(out) - 1; l >= 0 && t.Kind == token.Whitespace && out[l].Kind == token.Whitespace {
			out[l].Value += t.Value
			return
		}
		out = append(out, t)
	}
	for i, t := range c.toks {
		if c.cleared[i] {
			continue
		}
		if tag := strings.TrimRight(t.Value, " \t\r\n"); t.Kind == token.OpenTag && tag != t.Value {
			push(token.Token{Kind: token.OpenTag, Value: tag, Pos: t.Pos})
			push(token.Token{Kind: token.Whitespace, Value: t.Value[len(tag):], Pos: t.Pos + len(tag)})
			continue
		}
		push(t)
	}
	s.ReplaceRange(0, s.Len()-1, out)
}

func (c *clearableTokens) clear(i int) { c.cleared[i] = true }

// isWhitespace is Token::isWhitespace(): a whitespace token or a cleared one.
func (c *clearableTokens) isWhitespace(i int) bool {
	return c.cleared[i] || c.toks[i].Kind == token.Whitespace
}

func (c *clearableTokens) isComment(i int) bool {
	return !c.cleared[i] && (c.toks[i].Kind == token.Comment || c.toks[i].Kind == token.DocComment)
}

func (c *clearableTokens) value(i int) string {
	if c.cleared[i] {
		return ""
	}
	return c.toks[i].Value
}

// setWhitespace is Tokens::ensureWhitespaceAtIndex($i, 0, $v) on a token that
// is (or was cleared to) whitespace: an empty value clears it.
func (c *clearableTokens) setWhitespace(i int, v string) {
	if v == "" {
		c.clear(i)
		return
	}
	c.toks[i] = token.Token{Kind: token.Whitespace, Value: v, Pos: c.toks[i].Pos}
	c.cleared[i] = false
}

// nonEmptySibling is Tokens::getNonEmptySibling(): the next index in direction
// dir that is not cleared, or -1.
func (c *clearableTokens) nonEmptySibling(i, dir int) int {
	for i += dir; i >= 0 && i < len(c.toks); i += dir {
		if !c.cleared[i] {
			return i
		}
	}
	return -1
}

// prevNonWhitespace is Tokens::getPrevNonWhitespace(), or -1.
func (c *clearableTokens) prevNonWhitespace(i int) int {
	for i--; i >= 0; i-- {
		if !c.isWhitespace(i) {
			return i
		}
	}
	return -1
}

// clearAndMergeSurroundingWhitespace is Tokens::clearTokenAndMergeSurroundingWhitespace().
func (c *clearableTokens) clearAndMergeSurroundingWhitespace(i int) {
	c.clear(i)
	if i == len(c.toks)-1 {
		return
	}
	next := c.nonEmptySibling(i, 1)
	if next < 0 || !c.isWhitespace(next) {
		return
	}
	prev := c.nonEmptySibling(i, -1)
	switch {
	case prev >= 0 && c.isWhitespace(prev):
		c.setWhitespace(prev, c.value(prev)+c.value(next))
	case prev+1 < len(c.toks) && c.cleared[prev+1]:
		c.setWhitespace(prev+1, c.value(next))
	}
	c.clear(next)
}

// removeUseDeclaration ports NoUnusedImportsFixer::removeUseDeclaration() for a
// single (non-group) import spanning [start, semi]: comments inside it are
// kept, everything else is cleared, then the surrounding line breaks are
// tidied by cleanUpSurroundingNewLines.
func (c *clearableTokens) removeUseDeclaration(start, semi int) {
	for i := semi - 1; i >= start; i-- {
		if c.isComment(i) {
			continue
		}
		if !c.isWhitespace(i) || !strings.Contains(c.value(i), "\n") {
			c.clear(i)
			continue
		}
		// a multi-line whitespace keeps its line feed (and indent) after a comment
		if p := c.prevNonWhitespace(i); p >= 0 && c.isComment(p) {
			v := c.value(i)
			c.setWhitespace(i, v[strings.LastIndex(v, "\n"):])
		} else {
			c.clearAndMergeSurroundingWhitespace(i)
		}
	}
	if !c.cleared[semi] && c.toks[semi].Kind == token.Punct && c.toks[semi].Value == ";" {
		c.clear(semi)
	}
	c.cleanUpSurroundingNewLines(start, semi)
}

// cleanUpSurroundingNewLines ports NoUnusedImportsFixer::cleanUpSurroundingNewLines().
func (c *clearableTokens) cleanUpSurroundingNewLines(start, end int) {
	prev := start - 1
	if prev >= 0 && c.isWhitespace(prev) {
		c.setWhitespace(prev, strings.TrimRight(c.value(prev), " \t"))
	}
	if end+1 >= len(c.toks) {
		return
	}
	next := c.nonEmptySibling(end, 1)
	if next < 0 {
		return
	}
	if c.isWhitespace(next) {
		v := strings.TrimLeft(c.value(next), " \t")
		if strings.HasPrefix(v, "\r\n") {
			v = v[2:]
		} else if strings.HasPrefix(v, "\n") {
			v = v[1:]
		}
		c.setWhitespace(next, v)
	}
	if prev >= 0 && c.isWhitespace(prev) && c.isWhitespace(next) {
		c.setWhitespace(next, c.value(prev)+c.value(next))
		c.clear(prev)
	}
}
