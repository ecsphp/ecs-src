package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Default phpdoc_tags whose type expression is shortened.
var fqPhpdocTags = map[string]bool{
	"param": true, "phpstan-param": true, "phpstan-property": true,
	"phpstan-property-read": true, "phpstan-property-write": true,
	"phpstan-return": true, "phpstan-var": true, "property": true,
	"property-read": true, "property-write": true, "psalm-param": true,
	"psalm-property": true, "psalm-property-read": true, "psalm-property-write": true,
	"psalm-return": true, "psalm-var": true, "return": true, "see": true,
	"throws": true, "var": true,
}

var fqDocKeywords = map[string]bool{
	"min": true, "max": true, "class-string": true, "int": true, "positive-int": true,
	"negative-int": true, "non-empty-string": true, "numeric-string": true,
	"array-key": true, "scalar": true, "non-empty-array": true, "non-empty-list": true,
	"key-of": true, "value-of": true, "literal-string": true, "callable-string": true,
	"double": true, "boolean": true, "integer": true, "this": true, "$this": true,
	"lowercase-string": true, "non-falsy-string": true, "truthy-string": true,
}

func fqHSpace(c byte) bool { return c == ' ' || c == '\t' }
func fqWS(c byte) bool {
	return c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}
func fqTagChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}
func fqNameStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80
}
func fqNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c >= 0x80
}

// fqDocTemplateNames collects @template/@type declared identifiers (reserved).
func fqDocTemplateNames(content string) []string {
	var out []string
	b := []byte(content)
	n := len(b)
	prefixes := [][]byte{[]byte("psalm-"), []byte("phpstan-")}
	kws := [][]byte{
		[]byte("template-covariant"), []byte("template-contravariant"),
		[]byte("template"), []byte("import-type"), []byte("type"),
	}
	for i := range n {
		if b[i] != '*' {
			continue
		}
		j := i + 1
		for j < n && fqHSpace(b[j]) {
			j++
		}
		if j >= n || b[j] != '@' {
			continue
		}
		rest := b[j+1:]
		lower := []byte(strings.ToLower(string(rest)))
		off := 0
		for _, pfx := range prefixes {
			if fqHasPrefix(lower, pfx) {
				off = len(pfx)
				break
			}
		}
		after := lower[off:]
		var matched []byte
		for _, kw := range kws {
			if fqHasPrefix(after, kw) {
				matched = kw
				break
			}
		}
		if matched == nil {
			continue
		}
		p := j + 1 + off + len(matched)
		hs := false
		for p < n && fqHSpace(b[p]) {
			p++
			hs = true
		}
		if hs && p < n && fqNameStart(b[p]) {
			st := p
			p++
			for p < n && fqNameChar(b[p]) {
				p++
			}
			out = append(out, string(b[st:p]))
		}
	}
	return out
}

func fqHasPrefix(b, pfx []byte) bool {
	if len(b) < len(pfx) {
		return false
	}
	for i := range pfx {
		if b[i] != pfx[i] {
			return false
		}
	}
	return true
}

// fqPhpDoc shortens class names inside each allowed docblock tag's type.
func fqPhpDoc(s *tokens.Stream, idx int, uses *fqUses, ns string, reserved map[string]bool) (fqReplace, bool) {
	content := s.At(idx).Value
	newContent := fqPhpDocContent(content, uses, ns, reserved)
	if newContent == content {
		return fqReplace{}, false
	}
	tok := token.Token{Kind: token.DocComment, Value: newContent}
	return fqReplace{start: idx, end: idx, raw: &tok}, true
}

// fqPhpDocContent mirrors ECS's `([*{]\h*@)(tag)(\h+)(type)` replace over the
// allowed tags, using an explicit scanner (identical in go and rust).
func fqPhpDocContent(content string, uses *fqUses, ns string, reserved map[string]bool) string {
	b := []byte(content)
	n := len(b)
	var out []byte
	i := 0
	for i < n {
		if b[i] == '*' || b[i] == '{' {
			j := i + 1
			for j < n && fqHSpace(b[j]) {
				j++
			}
			if j < n && b[j] == '@' {
				g1End := j + 1
				t := g1End
				for t < n && fqTagChar(b[t]) {
					t++
				}
				if t > g1End {
					tag := b[g1End:t]
					h := t
					for h < n && fqHSpace(b[h]) {
						h++
					}
					if h > t && h < n && !fqWS(b[h]) && b[h] != '*' {
						typeStart := h
						te := h + 1
						for te < n && !fqWS(b[te]) {
							te++
						}
						out = append(out, b[i:g1End]...)
						out = append(out, tag...)
						out = append(out, b[t:h]...)
						if fqPhpdocTags[strings.ToLower(string(tag))] {
							out = append(out, []byte(fqShortenDocType(string(b[typeStart:te]), uses, ns, reserved))...)
						} else {
							out = append(out, b[typeStart:te]...)
						}
						i = te
						continue
					}
				}
			}
		}
		out = append(out, b[i])
		i++
	}
	return string(out)
}

// fqShortenDocType shortens each class-name atom in a type expression, skipping
// variables ($x), string keys, array-shape keys and `::` members.
func fqShortenDocType(typeStr string, uses *fqUses, ns string, reserved map[string]bool) string {
	// a wildcard type argument (`*`, e.g. `Foo<*>` or `Rel<T, *, *>`) makes the
	// whole type expression unparseable for ECS's TypeExpression, which then
	// leaves it entirely fully qualified.
	if strings.Contains(typeStr, "*") {
		return typeStr
	}
	b := []byte(typeStr)
	n := len(b)
	var out []byte
	i := 0
	inString := byte(0)
	for i < n {
		c := b[i]
		if inString != 0 {
			out = append(out, c)
			if c == inString {
				inString = 0
			}
			i++
			continue
		}
		if c == '\'' || c == '"' {
			inString = c
			out = append(out, c)
			i++
			continue
		}
		// try to read an atom at i
		j := i
		if b[j] == '\\' && j+1 < n && fqNameStart(b[j+1]) {
			j++
		}
		if j < n && fqNameStart(b[j]) {
			start := i
			j++
			for j < n && fqNameChar(b[j]) {
				j++
			}
			for j < n && b[j] == '\\' && j+1 < n && fqNameStart(b[j+1]) {
				j++
				for j < n && fqNameChar(b[j]) {
					j++
				}
			}
			atom := string(b[start:j])
			if fqDocAtomShortenable(b, start, j, n) {
				out = append(out, []byte(fqMapDocAtom(atom, uses, ns, reserved))...)
			} else {
				out = append(out, b[start:j]...)
			}
			i = j
			continue
		}
		out = append(out, c)
		i++
	}
	return string(out)
}

// fqDocAtomShortenable guards against non-type atoms.
func fqDocAtomShortenable(b []byte, start, end, n int) bool {
	// preceded by `$` (variable) or `:` (::member or value separator artefact)
	if start > 0 {
		p := b[start-1]
		if p == '$' || p == ':' {
			return false
		}
	}
	// array-shape / object-shape key: atom immediately followed by a single `:`
	if end < n && b[end] == ':' && (end+1 >= n || b[end+1] != ':') {
		return false
	}
	// generic base (`Foo<...>`): ECS shortens some but keeps others (wildcards,
	// multi-arg spaced generics its parser rejects); leaving every generic base
	// fully qualified is byte-safe and never over-shortens. (Valid callable
	// bases `Closure(...)` are not guarded - ECS shortens those.)
	if end < n && b[end] == '<' {
		return false
	}
	return true
}

func fqMapDocAtom(atom string, uses *fqUses, ns string, reserved map[string]bool) string {
	if !strings.Contains(atom, "\\") {
		lower := strings.ToLower(atom)
		if fqReservedTypes[lower] || fqDocKeywords[lower] {
			return atom
		}
	}
	repl, changed := fqDetermineShort(atom, uses, ns, reserved)
	if !changed {
		return atom
	}
	return repl
}
