package rules

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The parts of PHP-CS-Fixer's DocBlock model that GlobalNamespaceImportFixer
// needs: Tag::getName, Annotation::getTypesContent, TypeExpression::getTypes
// and Annotation::setTypes. TypeExpression::REGEX_TYPES is recursive, so it is
// ported as a recursive-descent matcher that tries alternatives in the regex's
// order. Preg matches with /u, or in byte mode when the subject is not valid
// UTF-8 (see docClass).

// phpdocTypedTags is Annotation::TAGS_WITH_TYPES.
var phpdocTypedTags = map[string]bool{
	"extends": true, "implements": true, "method": true, "param": true, "param-out": true,
	"phpstan-import-type": true, "phpstan-type": true, "phpstan-var": true, "property": true,
	"property-read": true, "property-write": true, "psalm-import-type": true, "psalm-type": true,
	"psalm-var": true, "return": true, "throws": true, "type": true, "var": true,
}

// isDocH is PCRE's \h in byte mode.
func isDocH(c byte) bool { return c == ' ' || c == '\t' || c == 0xa0 }

// isDocV is PCRE's \v in byte mode.
func isDocV(c byte) bool {
	return c == '\n' || c == '\r' || c == 0x0b || c == 0x0c || c == 0x85
}

// isDocS is PCRE's \s in byte mode.
func isDocS(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == 0x0b || c == 0x0c
}

// docClass is a PCRE class in both of Preg's modes: Unicode with /u (utf), or
// the byte-mode one when the subject is not valid UTF-8.
type docClass struct {
	uni   func(rune) bool
	bytes func(byte) bool
}

var (
	docH = docClass{pcreIsHSpace, isDocH}
	docV = docClass{pcreIsVSpace, isDocV}
	docS = docClass{pcreIsSpace, isDocS}
	docW = docClass{pcreIsWord, isIdentByte}
	docD = docClass{unicode.IsDigit, isDigitByte}
)

// docUTF reports whether Preg matches s in UTF mode.
func docUTF(s string) bool { return utf8.ValidString(s) }

// at returns the byte width of the character at p when it is in the class,
// else 0. ASCII characters are in both modes' classes alike.
func (c docClass) at(s string, p int, utf bool) int {
	if p < 0 || p >= len(s) {
		return 0
	}
	if b := s[p]; !utf || b < utf8.RuneSelf {
		if c.bytes(b) {
			return 1
		}
		return 0
	}
	r, n := utf8.DecodeRuneInString(s[p:])
	if c.uni(r) {
		return n
	}
	return 0
}

// skip skips the characters of the class from p.
func (c docClass) skip(s string, p int, utf bool) int {
	for {
		n := c.at(s, p, utf)
		if n == 0 {
			return p
		}
		p += n
	}
}

// before reports whether the character ending at p is in the class (a
// lookbehind).
func (c docClass) before(s string, p int, utf bool) bool {
	if p <= 0 || p > len(s) {
		return false
	}
	if b := s[p-1]; !utf || b < utf8.RuneSelf {
		return c.bytes(b)
	}
	r, _ := utf8.DecodeLastRuneInString(s[:p])
	return c.uni(r)
}

// docFoldAt matches the ASCII literal lit case-insensitively at p and returns
// its end, or -1. With /u, PCRE folds per Unicode, so "k" also matches the
// Kelvin sign and "s" the long s.
func docFoldAt(s string, p int, lit string, utf bool) int {
	for i := 0; i < len(lit); i++ {
		if p >= len(s) {
			return -1
		}
		n := 1
		if utf {
			_, n = utf8.DecodeRuneInString(s[p:])
		}
		if !strings.EqualFold(s[p:p+n], lit[i:i+1]) {
			return -1
		}
		p += n
	}
	return p
}

// docTagName is Tag::getName: the first "@name" followed by whitespace or the
// end of the line, or "other".
func docTagName(line string) string {
	utf := docUTF(line)
	for i := 0; i < len(line); i++ {
		if line[i] != '@' {
			continue
		}
		j := i + 1
		for j < len(line) && (isIdentByte(line[j]) || line[j] == '-') {
			j++
		}
		if j > i+1 && (j == len(line) || docS.at(line, j, utf) > 0) {
			return line[i+1 : j]
		}
	}
	return "other"
}

// docTypesRange returns the byte range of an annotation's types
// (Annotation::getTypesContent), or ok=false when the regex does not match.
func docTypesRange(content, tag string) (start, end int, ok bool) {
	utf := docUTF(content)
	p := 0
	q := docH.skip(content, 0, utf)
	switch {
	case q < len(content) && content[q] == '*':
		p = q + 1
	case strings.HasPrefix(content, "/**"):
		p = 3
	default:
		return 0, 0, false
	}
	for p < len(content) {
		if content[p] == '*' {
			p++
		} else if n := docH.at(content, p, utf); n > 0 {
			p += n
		} else {
			break
		}
	}
	if p >= len(content) || content[p] != '@' {
		return 0, 0, false
	}
	if p = docFoldAt(content, p+1, tag, utf); p < 0 {
		return 0, 0, false
	}
	h := p
	p = docH.skip(content, p, utf)
	if p == h {
		return 0, 0, false
	}
	m := docTypeMatcher{src: content, utf: utf}
	for _, e := range m.types(p) {
		if docTypesTrailerOK(content, e, utf) {
			return p, e, true
		}
	}
	return 0, 0, false
}

// docTypesTrailerOK checks what may follow the types:
// (?:(?:[*\h\v]|\&?[\.\$\s]).*)?\r?$ with the s flag.
func docTypesTrailerOK(content string, e int, utf bool) bool {
	rest := content[e:]
	switch rest {
	case "", "\n", "\r", "\r\n":
		return true
	}
	if c := rest[0]; c == '*' || c == '.' || c == '$' || docH.at(rest, 0, utf) > 0 || docV.at(rest, 0, utf) > 0 {
		return true
	}
	return rest[0] == '&' && len(rest) > 1 && (rest[1] == '.' || rest[1] == '$' || docS.at(rest, 1, utf) > 0)
}

// docTypeMembers is TypeExpression::getTypes plus getTypesGlue for a types
// string: the top-level union (or intersection) members, or the whole value.
func docTypeMembers(value string) (members []string, glue string) {
	m := docTypeMatcher{src: value, utf: docUTF(value)}
	type inner struct {
		value, glue, glueRaw string
	}
	var parts []inner
	idx := 0
	seen := map[byte]bool{}
	for {
		matched := false
		for _, e := range m.typ(idx) {
			// (?<glue_raw>\h*(?<glue>[|&])\h*(?!$)|$)
			g := docH.skip(value, e, m.utf)
			if g < len(value) && (value[g] == '|' || value[g] == '&') {
				gl := value[g]
				g = docH.skip(value, g+1, m.utf)
				if !docAtEnd(value, g) {
					parts = append(parts, inner{value: value[idx:e], glue: string(gl), glueRaw: value[e:g]})
					seen[gl] = true
					idx = g
					matched = true
					break
				}
			}
			if docAtEnd(value, e) {
				if len(parts) == 0 {
					return []string{value}, ""
				}
				parts = append(parts, inner{value: value[idx:e]})
				idx = len(value)
				matched = true
				break
			}
		}
		if !matched {
			return []string{value}, ""
		}
		if idx >= len(value) {
			break
		}
	}
	if len(parts) == 0 {
		return []string{value}, ""
	}
	glue = "&"
	if seen['|'] {
		glue = "|"
	}
	if len(seen) == 1 {
		for _, p := range parts {
			members = append(members, p.value)
		}
		return members, glue
	}
	for i := 0; i < len(parts); i++ {
		var v strings.Builder
		for {
			v.WriteString(parts[i].value)
			if parts[i].glue == "" || parts[i].glue == glue {
				break
			}
			v.WriteString(parts[i].glueRaw)
			i++
			if i >= len(parts) {
				break
			}
		}
		members = append(members, v.String())
	}
	return members, glue
}

// docAtEnd is PCRE's $ (end of subject or before a final newline).
func docAtEnd(s string, p int) bool {
	return p == len(s) || (p == len(s)-1 && s[p] == '\n')
}

// docTypeRuns calls fn for each match of /[\\\w]+(?![\\\w:])/ in a type,
// right to left, replacing the match with what fn returns.
func docTypeRuns(typ string, fn func(string) string) string {
	type run struct{ start, end int }
	var runs []run
	utf := docUTF(typ)
	nameChar := func(p int) int {
		if p < len(typ) && typ[p] == '\\' {
			return 1
		}
		return docW.at(typ, p, utf)
	}
	for i := 0; i < len(typ); {
		if nameChar(i) == 0 {
			i++
			continue
		}
		j := i
		for n := nameChar(j); n > 0; n = nameChar(j) {
			j += n
		}
		if j >= len(typ) || typ[j] != ':' {
			runs = append(runs, run{i, j})
		}
		i = j
	}
	for _, r := range slices.Backward(runs) {
		repl := fn(typ[r.start:r.end])
		typ = typ[:r.start] + repl + typ[r.end:]
	}
	return typ
}

// traverseDocTypes mirrors GlobalNamespaceImportFixer::traverseDocBlockTypes:
// fn maps every name in every typed annotation; it returns the new docblock
// and whether it changed.
func traverseDocTypes(content string, fn func(string) string) (string, bool) {
	lines := splitDocLines(content)
	changed := false
	for _, a := range docAnnotations(lines) {
		tag := docTagName(lines[a.start])
		if !phpdocTypedTags[tag] {
			continue
		}
		annotation := strings.Join(lines[a.start:a.end+1], "")
		s, e, ok := docTypesRange(annotation, tag)
		if !ok {
			continue
		}
		typesContent := annotation[s:e]
		members, glue := docTypeMembers(typesContent)
		mapped := make([]string, len(members))
		same := true
		for i, t := range members {
			mapped[i] = docTypeRuns(t, fn)
			if mapped[i] != t {
				same = false
			}
		}
		if same {
			continue
		}
		if glue == "" {
			glue = "|"
		}
		newContent := strings.Join(mapped, glue)
		if newContent == typesContent {
			continue
		}
		origLines, newLines := splitDocLines(typesContent), splitDocLines(newContent)
		if len(origLines) != len(newLines) {
			continue
		}
		for k, nl := range newLines {
			li := a.start + k
			if li > a.end {
				break
			}
			if pos := strings.Index(lines[li], origLines[k]); pos >= 0 {
				lines[li] = lines[li][:pos] + pregReplacement(nl, origLines[k]) + lines[li][pos+len(origLines[k]):]
			}
		}
		changed = true
	}
	if !changed {
		return content, false
	}
	return strings.Join(lines, ""), true
}

// pregReplacement expands a preg_replace replacement string for a pattern
// without groups: \0, $0 and ${0} are the whole match, other group references
// are empty, and "\\" is one backslash.
func pregReplacement(repl, whole string) string {
	if !strings.ContainsAny(repl, `\$`) {
		return repl
	}
	var b strings.Builder
	for i := 0; i < len(repl); i++ {
		c := repl[i]
		if c == '\\' && i+1 < len(repl) && repl[i+1] == '\\' {
			b.WriteByte('\\')
			i++
			continue
		}
		if (c == '\\' || c == '$') && i+1 < len(repl) {
			j := i + 1
			braced := c == '$' && repl[j] == '{'
			if braced {
				j++
			}
			k := j
			for k < len(repl) && k < j+2 && repl[k] >= '0' && repl[k] <= '9' {
				k++
			}
			if k > j && (!braced || (k < len(repl) && repl[k] == '}')) {
				if repl[j:k] == "0" || repl[j:k] == "00" {
					b.WriteString(whole)
				}
				if braced {
					k++
				}
				i = k - 1
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// docTypeMatcher matches TypeExpression's REGEX_TYPE grammar at byte offsets.
// Each method returns the possible end offsets in PCRE's preference order
// (empty when it does not match); atomic and possessive parts return at most one.
type docTypeMatcher struct {
	src string
	utf bool // Preg's /u mode (docClass)
}

func (m docTypeMatcher) at(p int) byte {
	if p < len(m.src) {
		return m.src[p]
	}
	return 0
}

func (m docTypeMatcher) skipH(p int) int { return docH.skip(m.src, p, m.utf) }

// isH reports whether a \h starts at p.
func (m docTypeMatcher) isH(p int) bool { return docH.at(m.src, p, m.utf) > 0 }

// skipHVStar skips [\h\v*]*.
func (m docTypeMatcher) skipHVStar(p int) int {
	for p < len(m.src) {
		if m.src[p] == '*' {
			p++
		} else if n := max(docH.at(m.src, p, m.utf), docV.at(m.src, p, m.utf)); n > 0 {
			p += n
		} else {
			break
		}
	}
	return p
}

func (m docTypeMatcher) lit(p int, s string) int {
	if strings.HasPrefix(m.src[min(p, len(m.src)):], s) {
		return p + len(s)
	}
	return -1
}

func (m docTypeMatcher) litFold(p int, s string) int { return docFoldAt(m.src, p, s, m.utf) }

func isDocIdentChar(c byte) bool {
	switch {
	case c <= 0x2f, c >= 0x3a && c <= 0x40, c >= 0x5b && c <= 0x5e, c == 0x60, c >= 0x7b && c <= 0x7f:
		return false
	}
	return true
}

// identifier: (?:(?!(?<!\*)\d)[^\x00-\x2f\x3a-\x40\x5b-\x5e\x60\x7b-\x7f]++)
func (m docTypeMatcher) identifier(p int) int {
	if !isDocIdentChar(m.at(p)) {
		return -1
	}
	if docD.at(m.src, p, m.utf) > 0 && (p <= 0 || m.src[p-1] != '*') {
		return -1
	}
	for isDocIdentChar(m.at(p)) {
		p++
	}
	return p
}

// name: \\?+ identifier (?:[\\\-] identifier)*+
func (m docTypeMatcher) name(p int) int {
	if m.at(p) == '\\' {
		p++
	}
	p = m.identifier(p)
	if p < 0 {
		return -1
	}
	for {
		c := m.at(p)
		if c != '\\' && c != '-' {
			return p
		}
		q := m.identifier(p + 1)
		if q < 0 {
			return p
		}
		p = q
	}
}

// typesInner: (?>(?&type)(?:\h*[|&]\h*(?&type))*+)
func (m docTypeMatcher) typesInner(p int) int {
	ends := m.typ(p)
	if len(ends) == 0 {
		return -1
	}
	return m.glueRest(ends[0])
}

// glueRest is the possessive (?:\h*[|&]\h*(?&type))*+ tail.
func (m docTypeMatcher) glueRest(p int) int {
	for {
		q := m.skipH(p)
		if c := m.at(q); c != '|' && c != '&' {
			return p
		}
		ends := m.typ(m.skipH(q + 1))
		if len(ends) == 0 {
			return p
		}
		p = ends[0]
	}
}

// types is REGEX_TYPES: type (?:\h*[|&]\h*(?&type))*+, ends in preference order.
func (m docTypeMatcher) types(p int) []int {
	var out []int
	for _, e := range m.typ(p) {
		out = append(out, m.glueRest(e))
	}
	return out
}

// typ is REGEX_TYPE.
func (m docTypeMatcher) typ(p int) []int {
	var out []int
	starts := []int{}
	if q := m.lit(p, "contravariant"); q >= 0 && m.isH(q) {
		starts = append(starts, m.skipH(q))
	} else if q := m.lit(p, "covariant"); q >= 0 && m.isH(q) {
		starts = append(starts, m.skipH(q))
	}
	starts = append(starts, p)
	for _, s := range starts {
		// nullable: \??\h*
		q := s
		if m.at(q) == '?' {
			q++
		}
		q = m.skipH(q)
		for _, e := range m.alternatives(q) {
			out = append(out, m.arraySuffixes(e)...)
		}
	}
	return out
}

// arraySuffixes is the greedy (\h*\[\h*\])* suffix: the longest first.
func (m docTypeMatcher) arraySuffixes(p int) []int {
	ends := []int{p}
	for {
		q := m.skipH(p)
		if m.at(q) != '[' {
			break
		}
		q = m.skipH(q + 1)
		if m.at(q) != ']' {
			break
		}
		p = q + 1
		ends = append(ends, p)
	}
	slices.Reverse(ends)
	return ends
}

func (m docTypeMatcher) alternatives(p int) []int {
	var out []int
	add := func(e int) {
		if e >= 0 {
			out = append(out, e)
		}
	}
	add(m.arrayShape(p))
	out = append(out, m.callable(p)...)
	add(m.generic(p))
	add(m.classConstant(p))
	add(m.constant(p))
	add(m.this(p))
	add(m.name(p))
	add(m.parenthesized(p))
	return out
}

func (m docTypeMatcher) arrayShape(p int) int {
	q := -1
	for _, kw := range []string{"array", "list", "object"} {
		if q = m.litFold(p, kw); q >= 0 {
			break
		}
	}
	if q < 0 {
		return -1
	}
	q = m.skipH(q)
	if m.at(q) != '{' {
		return -1
	}
	q = m.skipHVStar(q + 1)
	// inners (optional)
	if r := m.shapeInners(q); r >= 0 {
		q = r
	}
	// unsealed (optional)
	if r := m.shapeUnsealed(q); r >= 0 {
		q = r
	}
	q = m.skipHVStar(q)
	if m.at(q) != '}' {
		return -1
	}
	return q + 1
}

func (m docTypeMatcher) shapeInners(p int) int {
	q := m.shapeInner(p)
	if q < 0 {
		return -1
	}
	for {
		r := m.skipH(q)
		if m.at(r) != ',' {
			break
		}
		r = m.skipHVStar(r + 1)
		s := m.shapeInner(r)
		if s < 0 {
			break
		}
		q = s
	}
	if r := m.skipH(q); m.at(r) == ',' {
		return r + 1
	}
	if r := m.skipH(q); m.lit(r, "...") >= 0 {
		return -1
	}
	return q
}

func (m docTypeMatcher) shapeInner(p int) int {
	for _, key := range []int{m.constant(p), m.identifier(p), m.name(p)} {
		if key < 0 {
			continue
		}
		q := m.skipH(key)
		if m.at(q) == '?' {
			q++
		}
		q = m.skipH(q)
		if m.at(q) != ':' {
			continue
		}
		if v := m.typesInner(m.skipH(q + 1)); v >= 0 {
			return v
		}
	}
	return m.typesInner(p)
}

func (m docTypeMatcher) shapeUnsealed(p int) int {
	q := m.lit(m.skipH(p), "...")
	if q < 0 {
		return -1
	}
	r := m.skipH(q)
	if m.at(r) != '<' {
		return q
	}
	a := m.typesInner(m.skipH(r + 1))
	if a < 0 {
		return q
	}
	if c := m.skipH(a); m.at(c) == ',' {
		if b := m.typesInner(m.skipH(c + 1)); b >= 0 {
			a = b
		}
	}
	e := m.skipH(a)
	if m.at(e) != '>' {
		return q
	}
	return e + 1
}

func (m docTypeMatcher) callable(p int) []int {
	q := m.name(p)
	if q < 0 {
		return nil
	}
	if t := m.callableTemplate(q); t >= 0 {
		q = t
	}
	q = m.skipH(q)
	if m.at(q) != '(' {
		return nil
	}
	q = m.skipH(q + 1)
	if a := m.callableArgs(q); a >= 0 {
		q = a
	}
	q = m.skipH(q)
	if m.at(q) != ')' {
		return nil
	}
	q++
	var out []int
	if r := m.skipH(q); m.at(r) == ':' {
		out = append(out, m.typ(m.skipH(r+1))...)
	}
	return append(out, q)
}

func (m docTypeMatcher) callableTemplate(p int) int {
	q := m.skipH(p)
	if m.at(q) != '<' {
		return -1
	}
	q = m.callableTemplateInner(m.skipH(q + 1))
	if q < 0 {
		return -1
	}
	for {
		r := m.skipH(q)
		if m.at(r) != ',' {
			break
		}
		s := m.callableTemplateInner(m.skipH(r + 1))
		if s < 0 {
			break
		}
		q = s
	}
	q = m.skipH(q)
	if m.at(q) != '>' {
		return -1
	}
	q++
	if r := m.skipH(q); m.at(r) != '(' {
		return -1
	}
	return q
}

func (m docTypeMatcher) callableTemplateInner(p int) int {
	q := m.identifier(p)
	if q < 0 {
		return -1
	}
	if h := q; m.isH(h) {
		r := m.skipH(h)
		for _, kw := range []string{"of", "as"} {
			if k := m.litFold(r, kw); k >= 0 && m.isH(k) {
				if t := m.typesInner(m.skipH(k)); t >= 0 {
					q = t
					break
				}
			}
		}
	}
	if r := m.skipH(q); m.at(r) == '=' {
		if t := m.typesInner(m.skipH(r + 1)); t >= 0 {
			q = t
		}
	}
	return q
}

func (m docTypeMatcher) callableArgs(p int) int {
	q := m.callableArg(p)
	if q < 0 {
		return -1
	}
	for {
		r := m.skipH(q)
		if m.at(r) != ',' {
			break
		}
		s := m.callableArg(m.skipH(r + 1))
		if s < 0 {
			break
		}
		q = s
	}
	if r := m.skipH(q); m.at(r) == ',' {
		q = r + 1
	}
	return q
}

func (m docTypeMatcher) callableArg(p int) int {
	q := m.typesInner(p)
	if q < 0 {
		return -1
	}
	if r := m.skipH(q); m.at(r) == '&' {
		q = r + 1
	}
	if r := m.lit(m.skipH(q), "..."); r >= 0 {
		q = r
	}
	if r := m.skipH(q); m.at(r) == '$' {
		if e := m.identifier(r + 1); e >= 0 {
			q = e
		}
	}
	if r := m.skipH(q); m.at(r) == '=' {
		q = r + 1
	}
	return q
}

func (m docTypeMatcher) generic(p int) int {
	q := m.name(p)
	if q < 0 {
		return -1
	}
	q = m.skipH(q)
	if m.at(q) != '<' {
		return -1
	}
	q = m.typesInner(m.skipHVStar(q + 1))
	if q < 0 {
		return -1
	}
	for {
		r := m.skipH(q)
		if m.at(r) != ',' {
			break
		}
		s := m.typesInner(m.skipHVStar(r + 1))
		if s < 0 {
			break
		}
		q = s
	}
	if r := m.skipH(q); m.at(r) == ',' {
		q = r + 1
	}
	q = m.skipHVStar(q)
	if m.at(q) != '>' {
		return -1
	}
	return q + 1
}

func (m docTypeMatcher) classConstant(p int) int {
	q := m.name(p)
	if q < 0 {
		return -1
	}
	if q = m.lit(q, "::"); q < 0 {
		return -1
	}
	if m.at(q) == '*' {
		q++
	}
	for {
		r := m.identifier(q)
		if r < 0 {
			return q
		}
		q = r
		if m.at(q) == '*' {
			q++
		}
	}
}

// The digit classes of the constant numbers; only \d is not ASCII with /u.
var (
	docBinDigit = docClass{func(r rune) bool { return r == '0' || r == '1' }, func(c byte) bool { return c == '0' || c == '1' }}
	docOctDigit = docClass{func(r rune) bool { return r >= '0' && r <= '7' }, func(c byte) bool { return c >= '0' && c <= '7' }}
	docHexDigit = docClass{
		func(r rune) bool { return unicode.IsDigit(r) || (r|0x20 >= 'a' && r|0x20 <= 'f') },
		func(c byte) bool { return isDigitByte(c) || (c|0x20 >= 'a' && c|0x20 <= 'f') },
	}
)

// digitsSep matches \d++(?:_\d++)*+ (or any class with separators).
func (m docTypeMatcher) digitsSep(p int, cls docClass) int {
	q := cls.skip(m.src, p, m.utf)
	if q == p {
		return -1
	}
	for m.at(q) == '_' && cls.at(m.src, q+1, m.utf) > 0 {
		q = cls.skip(m.src, q+1, m.utf)
	}
	return q
}

func (m docTypeMatcher) constant(p int) int {
	switch m.at(p) {
	case '\'', '"':
		q := m.src[p]
		for i := p + 1; i < len(m.src); i++ {
			if m.src[i] == '\\' && i+1 < len(m.src) {
				i++
				continue
			}
			if m.src[i] == q {
				return i + 1
			}
		}
		return -1
	}
	q := p
	if c := m.at(q); c == '+' || c == '-' {
		q++
	}
	lower := func(i int) byte { return m.at(i) | 0x20 }
	if m.at(q) == '0' {
		switch lower(q + 1) {
		case 'b':
			if e := m.digitsSep(q+2, docBinDigit); e >= 0 {
				return e
			}
		case 'o':
			if e := m.digitsSep(q+2, docOctDigit); e >= 0 {
				return e
			}
		case 'x':
			if e := m.digitsSep(q+2, docHexDigit); e >= 0 {
				return e
			}
		}
	}
	e := m.digitsSep(q, docD)
	if e < 0 {
		if m.at(q) != '.' || docD.at(m.src, q+1, m.utf) <= 0 {
			return -1
		}
		e = q
	}
	if m.at(e) == '.' {
		if f := m.digitsSep(e+1, docD); f >= 0 {
			e = f
		} else if docD.before(m.src, e, m.utf) {
			e++
		}
	}
	if lower(e) == 'e' {
		f := e + 1
		if c := m.at(f); c == '+' || c == '-' {
			f++
		}
		if g := m.digitsSep(f, docD); g >= 0 {
			e = g
		}
	}
	return e
}

func (m docTypeMatcher) this(p int) int {
	if c := m.at(p); c != '@' && c != '$' {
		return -1
	}
	for _, w := range []string{"this", "self", "static"} {
		if q := m.litFold(p+1, w); q >= 0 {
			return q
		}
	}
	return -1
}

func (m docTypeMatcher) parenthesized(p int) int {
	if m.at(p) != '(' {
		return -1
	}
	q := m.skipH(p + 1)
	if t := m.typesInner(q); t >= 0 {
		if r := m.skipH(t); m.at(r) == ')' {
			return r + 1
		}
	}
	if c := m.conditional(q); c >= 0 {
		if r := m.skipH(c); m.at(r) == ')' {
			return r + 1
		}
	}
	return -1
}

func (m docTypeMatcher) conditional(p int) int {
	var q int
	if m.at(p) == '$' {
		q = m.identifier(p + 1)
	} else {
		q = m.typesInner(p)
	}
	if q < 0 || !m.isH(q) {
		return -1
	}
	q = m.skipH(q)
	if q = m.litFold(q, "is"); q < 0 || !m.isH(q) {
		return -1
	}
	q = m.skipH(q)
	if r := m.litFold(q, "not"); r >= 0 && m.isH(r) {
		q = m.skipH(r)
	}
	if q = m.typesInner(q); q < 0 {
		return -1
	}
	if q = m.skipH(q); m.at(q) != '?' {
		return -1
	}
	if q = m.typesInner(m.skipH(q + 1)); q < 0 {
		return -1
	}
	if q = m.skipH(q); m.at(q) != ':' {
		return -1
	}
	return m.typesInner(m.skipH(q + 1))
}
