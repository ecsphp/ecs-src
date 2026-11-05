package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// fqDocReservedIdentifiers is the reserved-identifier scan of applyFix:
// /\*\h*@(?:psalm-|phpstan-)?(?:template(?:-covariant|-contravariant)?|(?:import-)?type)\h+(identifier)(?!\S)/i
func fqDocReservedIdentifiers(content string) []string {
	m := docTypeMatcher{src: content, utf: docUTF(content)}
	var out []string
	for i := 0; i < len(content); i++ {
		if content[i] != '*' {
			continue
		}
		p := m.skipH(i + 1)
		if m.at(p) != '@' {
			continue
		}
		p++
		for _, pfx := range []string{"psalm-", "phpstan-"} {
			if q := m.litFold(p, pfx); q >= 0 {
				p = q
				break
			}
		}
		q := -1
		if t := m.litFold(p, "template"); t >= 0 {
			q = t
			for _, sfx := range []string{"-covariant", "-contravariant"} {
				if r := m.litFold(t, sfx); r >= 0 {
					q = r
					break
				}
			}
		} else {
			if r := m.litFold(p, "import-"); r >= 0 {
				p = r
			}
			q = m.litFold(p, "type")
		}
		if q < 0 {
			continue
		}
		h := m.skipH(q)
		if h == q {
			continue
		}
		e := m.identifier(h)
		if e < 0 || (e < len(content) && docS.at(content, e, m.utf) == 0) {
			continue
		}
		out = append(out, content[h:e])
		i = e - 1
	}
	return out
}

// fixPhpDoc shortens the types of the phpdoc_tags tags of a docblock.
func (c *fqCtx) fixPhpDoc(index int) {
	if len(c.tags) == 0 {
		return
	}
	content := c.s.At(index).Value
	newContent := fqDocReplaceTypes(content, c.tags, c.fixPhpDocType)
	if c.tags["see"] {
		newContent = fqDocReplaceSeeMembers(newContent, c.fixPhpDocType)
	}
	if newContent != content && !c.discovering {
		c.addEdit(fqEdit{start: index, end: index, toks: []token.Token{{Kind: token.DocComment, Value: newContent}}})
	}
}

// fqDocLookaheadOK is (?!(?!\})\S) at p: the end, a "}" or a whitespace.
func fqDocLookaheadOK(content string, p int, utf bool) bool {
	return p >= len(content) || content[p] == '}' || docS.at(content, p, utf) > 0
}

// fqDocReplaceTypes is
// preg_replace_callback('/([*{]\h*@)(\S+)(\h+)('.REGEX_TYPES.')(?!(?!\})\S)/', ...)
// with fix applied to the types of the allowed tags.
func fqDocReplaceTypes(content string, tags map[string]bool, fix func(string) string) string {
	m := docTypeMatcher{src: content, utf: docUTF(content)}
	var b strings.Builder
	last := 0
	for p := 0; p < len(content); p++ {
		if content[p] != '*' && content[p] != '{' {
			continue
		}
		q := m.skipH(p + 1)
		if m.at(q) != '@' {
			continue
		}
		tagEnd := q + 1
		for tagEnd < len(content) && docS.at(content, tagEnd, m.utf) == 0 {
			tagEnd++
		}
		if tagEnd == q+1 {
			continue
		}
		h := m.skipH(tagEnd)
		if h == tagEnd {
			continue
		}
		for _, e := range m.types(h) {
			if !fqDocLookaheadOK(content, e, m.utf) {
				continue
			}
			if tags[content[q+1:tagEnd]] {
				b.WriteString(content[last:h])
				b.WriteString(fix(content[h:e]))
				last = e
			}
			p = e - 1
			break
		}
	}
	if last == 0 {
		return content
	}
	b.WriteString(content[last:])
	return b.String()
}

// fqDocReplaceSeeMembers is the @see pass:
// /([*{]\h*)@see(\h+)(REGEX_CLASS)(::(?:\$\w+|\w+\(\)))(?!(?!\})\S)/
func fqDocReplaceSeeMembers(content string, fix func(string) string) string {
	m := docTypeMatcher{src: content, utf: docUTF(content)}
	var b strings.Builder
	last := 0
	for p := 0; p < len(content); p++ {
		if content[p] != '*' && content[p] != '{' {
			continue
		}
		q := m.lit(m.skipH(p+1), "@see")
		if q < 0 {
			continue
		}
		h := m.skipH(q)
		if h == q {
			continue
		}
		classEnd := fqRegexClassAt(m, h)
		if classEnd < 0 || m.lit(classEnd, "::") < 0 {
			continue
		}
		r := classEnd + 2
		end := -1
		if m.at(r) == '$' {
			// \$\w+ backtracks until the lookahead holds
			var ends []int
			for w := r + 1; ; {
				n := docW.at(content, w, m.utf)
				if n == 0 {
					break
				}
				w += n
				ends = append(ends, w)
			}
			for k := len(ends) - 1; k >= 0; k-- {
				if fqDocLookaheadOK(content, ends[k], m.utf) {
					end = ends[k]
					break
				}
			}
		} else {
			w := docW.skip(content, r, m.utf)
			if w > r && m.lit(w, "()") >= 0 && fqDocLookaheadOK(content, w+2, m.utf) {
				end = w + 2
			}
		}
		if end < 0 {
			continue
		}
		b.WriteString(content[last:h])
		b.WriteString(fix(content[h:classEnd]))
		last = classEnd
		p = end - 1
	}
	if last == 0 {
		return content
	}
	b.WriteString(content[last:])
	return b.String()
}

// fixPhpDocType is fixPhpDocType: TypeExpression::mapTypes over the type,
// shortening every type that is a bare class name.
func (c *fqCtx) fixPhpDocType(typ string) string {
	v, ok := fqMapTypes(typ, c.docTypeCallback)
	if !ok {
		c.fail = true
		return typ
	}
	return v
}

func (c *fqCtx) docTypeCallback(v string) string {
	ts, ok := parsePHPTypeValue(v)
	if !ok {
		c.fail = true
		return v
	}
	if _, glue := ts.composite(); glue != 0 {
		return v
	}
	if !fqIsRegexClass(v) || v == "min" || v == "max" {
		return v
	}
	if short, changed := c.determineShortType(v); changed {
		return short
	}
	return v
}

// fqIsRegexClass is preg_match('/^'.REGEX_CLASS.'$/', v).
func fqIsRegexClass(v string) bool {
	m := docTypeMatcher{src: v, utf: docUTF(v)}
	e := fqRegexClassAt(m, 0)
	return e >= 0 && (e == len(v) || (e == len(v)-1 && v[e] == '\n'))
}

// fqMapTypes ports TypeExpression::mapTypes: the inner type expressions are
// mapped first (in the order parse registers them), then the type itself.
// ok is false when the value does not parse, or would make PHP's
// TypeExpression misbehave.
func fqMapTypes(s string, cb func(string) string) (string, bool) {
	ts, ok := parsePHPTypeValue(s)
	if !ok {
		return s, false
	}
	var inners []phpTypeSpan
	if spans, glue := ts.composite(); glue != 0 {
		inners = spans
	} else {
		t, _ := parsePHPType(s, 0)
		switch {
		case t.prefixLen > 0 && t.nullableLen > 0:
			// PHP cuts the nullable part's length off the front, prefix included
			return s, false
		case t.nullableLen > 0:
			inners = []phpTypeSpan{{t.nullableLen, len(s)}}
		case t.arrayStart < t.end:
			inners = []phpTypeSpan{{0, t.arrayStart}}
		case t.prefixLen > 0:
			// the generic, callable, shape and parenthesized branches only
			// register inner types when they start the value; a class
			// constant registers its class at offset 0, over the prefix
			if nameEnd, ok := parsePHPTypeName(s, t.prefixLen); ok && strings.HasPrefix(s[nameEnd:], "::") {
				return s, false
			}
		case s[0] == '(':
			inners = t.inners
		default:
			// a generic, callable, array shape or class constant registers
			// its name first
			if nameEnd, ok := parsePHPTypeName(s, 0); ok && nameEnd < len(s) {
				inners = append([]phpTypeSpan{{0, nameEnd}}, t.inners...)
			}
		}
	}
	value, offset := s, 0
	for _, sp := range inners {
		orig := s[sp.start:sp.end]
		mapped, ok := fqMapTypes(orig, cb)
		if !ok {
			return s, false
		}
		if mapped != orig {
			at := sp.start + offset
			value = value[:at] + mapped + value[at+len(orig):]
			offset += len(mapped) - len(orig)
		}
	}
	return cb(value), true
}
