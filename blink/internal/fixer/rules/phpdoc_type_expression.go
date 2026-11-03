package rules

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A parser for PHP-CS-Fixer's TypeExpression grammar (REGEX_TYPE and
// REGEX_TYPES in src/DocBlock/TypeExpression.php), used to sort phpdoc types
// like TypeExpression::sortTypes. Callers leave a type that does not parse
// untouched.

type phpTypeSpan struct{ start, end int }

// phpType is one REGEX_TYPE match: its end, the length of its
// covariant/contravariant prefix and of its nullable part, where its array
// suffix starts, and the spans of its inner type expressions in the order
// TypeExpression::parse registers them.
type phpType struct {
	end         int
	prefixLen   int
	nullableLen int
	arrayStart  int
	inners      []phpTypeSpan
}

// phpTypes is one REGEX_TYPES match: members joined by glues.
type phpTypes struct {
	end     int
	members []phpTypeSpan
	glues   []byte
}

// skipPHPHSpace skips \h*.
func skipPHPHSpace(s string, i int) int { return pcreSkip(s, i, pcreIsHSpace) }

// skipPHPHVStar skips [\h\v*]*, which generics and array shapes allow after
// their opening bracket, after a comma and before their closing bracket: a
// type may continue on the next docblock line there.
func skipPHPHVStar(s string, i int) int {
	return pcreSkip(s, i, func(r rune) bool { return r == '*' || pcreIsSpace(r) })
}

// isPHPTypeIdentByte is REGEX_IDENTIFIER's [^\x00-\x2f\x3a-\x40\x5b-\x5e\x60\x7b-\x7f].
func isPHPTypeIdentByte(c byte) bool {
	return c >= 0x80 || c == '_' || ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

// parsePHPTypeIdentifier matches an identifier at i: (?!(?<!\*)\d), so not
// starting with a digit unless it follows a "*".
func parsePHPTypeIdentifier(s string, i int) (int, bool) {
	if i >= len(s) || !isPHPTypeIdentByte(s[i]) {
		return i, false
	}
	if r, _ := pcreRuneAt(s, i); unicode.IsDigit(r) && (i == 0 || s[i-1] != '*') {
		return i, false
	}
	for i < len(s) && isPHPTypeIdentByte(s[i]) {
		i++
	}
	return i, true
}

// parsePHPTypeName matches `\\?+identifier(?:[\\-]identifier)*+` at i.
func parsePHPTypeName(s string, i int) (int, bool) {
	if i < len(s) && s[i] == '\\' {
		i++
	}
	end, ok := parsePHPTypeIdentifier(s, i)
	if !ok {
		return i, false
	}
	for end < len(s) && (s[end] == '\\' || s[end] == '-') {
		next, ok := parsePHPTypeIdentifier(s, end+1)
		if !ok {
			break
		}
		end = next
	}
	return end, true
}

// parsePHPTypeConstant matches the constant alternative at i: a quoted string
// or a number.
func parsePHPTypeConstant(s string, i int) (int, bool) {
	if i >= len(s) {
		return i, false
	}
	if q := s[i]; q == '\'' || q == '"' {
		for j := i + 1; j < len(s); j++ {
			switch s[j] {
			case '\\':
				j++
			case q:
				return j + 1, true
			}
		}
		return i, false
	}
	return parsePHPTypeNumber(s, i)
}

// parsePHPTypeNumber matches the constant pattern's numbers, case-insensitively:
//
//	[+-]?(?: 0b[01]++(?:_[01]++)*+ | 0o[0-7]++(?:_[0-7]++)*+ | 0x[\da-f]++(?:_[\da-f]++)*+
//	  | (?:(?<digits>\d++(?:_\d++)*+)|(?=\.\d)) (?:\.(?&digits)|(?<=\d)\.)?+ (?:e[+-]?(?&digits))?+ )
func parsePHPTypeNumber(s string, i int) (int, bool) {
	j := i
	if j < len(s) && (s[j] == '+' || s[j] == '-') {
		j++
	}
	if j+1 < len(s) && s[j] == '0' {
		var digit func(r rune) bool
		switch s[j+1] | 0x20 {
		case 'b':
			digit = func(r rune) bool { return r == '0' || r == '1' }
		case 'o':
			digit = func(r rune) bool { return '0' <= r && r <= '7' }
		case 'x':
			digit = func(r rune) bool { return unicode.IsDigit(r) || ('a' <= r|0x20 && r|0x20 <= 'f') }
		}
		if digit != nil {
			if end, ok := phpDigitGroups(s, j+2, digit); ok {
				return end, true
			}
		}
	}
	end, hasDigits := phpDigitGroups(s, j, unicode.IsDigit)
	if !hasDigits {
		if r, _ := pcreRuneAt(s, j+1); j >= len(s) || s[j] != '.' || !unicode.IsDigit(r) {
			return i, false
		}
		end = j
	}
	if end < len(s) && s[end] == '.' {
		if e, ok := phpDigitGroups(s, end+1, unicode.IsDigit); ok {
			end = e
		} else if hasDigits {
			end++
		}
	}
	if end < len(s) && s[end]|0x20 == 'e' {
		e := end + 1
		if e < len(s) && (s[e] == '+' || s[e] == '-') {
			e++
		}
		if e, ok := phpDigitGroups(s, e, unicode.IsDigit); ok {
			end = e
		}
	}
	return end, true
}

// phpDigitGroups matches D++(?:_D++)*+ at i for the digit class D.
func phpDigitGroups(s string, i int, digit func(r rune) bool) (int, bool) {
	end := pcreSkip(s, i, digit)
	if end == i {
		return i, false
	}
	for end < len(s) && s[end] == '_' {
		next := pcreSkip(s, end+1, digit)
		if next == end+1 {
			break
		}
		end = next
	}
	return end, true
}

// parsePHPAfterSpace runs parse at the end of the spaces skip matches from i.
// If parse fails there, it retries at each earlier character in turn, as
// PCRE backtracks into a greedy quantifier: the non-ASCII spaces are also
// identifier characters, so a type can start with one. It returns where the
// match starts.
func parsePHPAfterSpace[T any](s string, i int, skip func(string, int) int, parse func(string, int) (T, bool)) (int, T, bool) {
	for p := skip(s, i); ; {
		if v, ok := parse(s, p); ok {
			return p, v, true
		}
		if p <= i {
			var zero T
			return i, zero, false
		}
		_, n := utf8.DecodeLastRuneInString(s[i:p])
		p -= n
	}
}

// parsePHPAfterHSpaces is parsePHPAfterSpace for \h+ (at least one space).
func parsePHPAfterHSpaces[T any](s string, i int, parse func(string, int) (T, bool)) (int, T, bool) {
	if r, n := pcreRuneAt(s, i); r >= 0 && pcreIsHSpace(r) {
		return parsePHPAfterSpace(s, i+n, skipPHPHSpace, parse)
	}
	var zero T
	return i, zero, false
}

// parsePHPTypes matches REGEX_TYPES (or types_inner) at i: a type, then
// possessively any `\h*[|&]\h*` + type.
func parsePHPTypes(s string, i int) (phpTypes, bool) {
	first, ok := parsePHPType(s, i)
	if !ok {
		return phpTypes{}, false
	}
	out := phpTypes{end: first.end, members: []phpTypeSpan{{i, first.end}}}
	for {
		q := skipPHPHSpace(s, out.end)
		if q >= len(s) || (s[q] != '|' && s[q] != '&') {
			break
		}
		r, next, ok := parsePHPAfterSpace(s, q+1, skipPHPHSpace, parsePHPType)
		if !ok {
			break
		}
		out.glues = append(out.glues, s[q])
		out.members = append(out.members, phpTypeSpan{r, next.end})
		out.end = next.end
	}
	return out, true
}

// parsePHPType matches REGEX_TYPE at i: an optional (?:co(?:ntra)?variant\h+)
// prefix, which the regex gives up when the rest does not match with it.
func parsePHPType(s string, i int) (phpType, bool) {
	if p := phpTypeVariancePrefix(s, i); p > i {
		if t, ok := parsePHPTypeBody(s, p); ok {
			t.prefixLen = p - i
			return t, true
		}
	}
	return parsePHPTypeBody(s, i)
}

// phpTypeVariancePrefix returns the end of a "covariant " or "contravariant "
// prefix at i, or i.
func phpTypeVariancePrefix(s string, i int) int {
	for _, kw := range []string{"covariant", "contravariant"} {
		if strings.HasPrefix(s[i:], kw) {
			if p := skipPHPHSpace(s, i+len(kw)); p > i+len(kw) {
				return p
			}
		}
	}
	return i
}

// parsePHPTypeBody matches REGEX_TYPE after its prefix, at i: the nullable
// part (\??\h*), the type and its array suffix.
func parsePHPTypeBody(s string, i int) (phpType, bool) {
	if i < len(s) && s[i] == '?' {
		q, t, ok := parsePHPAfterSpace(s, i+1, skipPHPHSpace, parsePHPTypeCore)
		t.nullableLen = q - i
		return t, ok
	}
	return parsePHPTypeCore(s, i)
}

// parsePHPTypeCore matches the type alternatives and the array suffix at i.
func parsePHPTypeCore(s string, i int) (phpType, bool) {
	t := phpType{}
	q := i
	if q >= len(s) {
		return t, false
	}
	var end int
	r, _ := pcreRuneAt(s, q)
	switch c := s[q]; {
	case c >= utf8.RuneSelf && unicode.IsDigit(r):
		e, ok := parsePHPTypeNumber(s, q)
		if !ok {
			return t, false
		}
		end = e
	case c == '(':
		_, e, ok := parsePHPAfterSpace(s, q+1, skipPHPHSpace, func(s string, r int) (int, bool) {
			if inner, ok := parsePHPTypes(s, r); ok {
				if e := skipPHPHSpace(s, inner.end); e < len(s) && s[e] == ')' {
					t.inners = []phpTypeSpan{{r, inner.end}}
					return e + 1, true
				}
			}
			return parsePHPConditional(s, r, &t)
		})
		if !ok {
			return t, false
		}
		end = e
	case c == '$' || c == '@':
		for _, kw := range []string{"this", "self", "static"} {
			if n := len(kw); len(s) >= q+1+n && strings.EqualFold(s[q+1:q+1+n], kw) {
				end = q + 1 + n
				break
			}
		}
		if end == 0 {
			return t, false
		}
	case c == '\'' || c == '"' || c == '+' || c == '-' || c == '.' || ('0' <= c && c <= '9'):
		e, ok := parsePHPTypeConstant(s, q)
		if !ok {
			return t, false
		}
		end = e
	default:
		nameEnd, ok := parsePHPTypeName(s, q)
		if !ok {
			return t, false
		}
		e, ok := parsePHPTypeAfterName(s, q, nameEnd, &t)
		if !ok {
			return t, false
		}
		end = e
	}
	t.arrayStart = end
	for {
		r := skipPHPHSpace(s, end)
		if r >= len(s) || s[r] != '[' {
			break
		}
		r = skipPHPHSpace(s, r+1)
		if r >= len(s) || s[r] != ']' {
			break
		}
		end = r + 1
	}
	t.end = end
	return t, true
}

// parsePHPTypeAfterName handles the alternatives that start with a name:
// array shape, callable, generic, class constant, or the bare name. As in the
// regex, a shape, callable or generic that does not parse falls back to the
// bare name (and the text left over then usually fails the caller).
func parsePHPTypeAfterName(s string, start, nameEnd int, t *phpType) (int, bool) {
	if end, ok := parsePHPTypeNameSuffix(s, start, nameEnd, t); ok {
		return end, true
	}
	t.inners = nil
	return nameEnd, true
}

func parsePHPTypeNameSuffix(s string, start, nameEnd int, t *phpType) (int, bool) {
	q := skipPHPHSpace(s, nameEnd)
	if q >= len(s) {
		return nameEnd, true
	}
	switch name := strings.ToLower(s[start:nameEnd]); {
	case s[q] == '{' && (name == "array" || name == "list" || name == "object"):
		return parsePHPArrayShape(s, q+1, t)
	case s[q] == '(':
		return parsePHPCallable(s, q+1, t)
	case s[q] == '<':
		// the callable alternative, with a template, comes before the generic one
		var callable phpType
		if end, ok := parsePHPCallableAfterName(s, nameEnd, &callable); ok {
			t.inners = callable.inners
			return end, true
		}
		// \h*<[\h\v*]* types (\h*,[\h\v*]* types)*+ (\h*,)? [\h\v*]*>
		r, inner, ok := parsePHPAfterSpace(s, q+1, skipPHPHVStar, parsePHPTypes)
		for {
			if !ok {
				return 0, false
			}
			t.inners = append(t.inners, phpTypeSpan{r, inner.end})
			closeFrom := inner.end
			if c := skipPHPHSpace(s, inner.end); c < len(s) && s[c] == ',' {
				if r, inner, ok = parsePHPAfterSpace(s, c+1, skipPHPHVStar, parsePHPTypes); ok {
					continue
				}
				closeFrom = c + 1
			}
			c := skipPHPHVStar(s, closeFrom)
			if c >= len(s) || s[c] != '>' {
				return 0, false
			}
			return c + 1, true
		}
	case nameEnd+1 < len(s) && s[nameEnd] == ':' && s[nameEnd+1] == ':':
		e := nameEnd + 2
		if e < len(s) && s[e] == '*' {
			e++
		}
		for {
			next, ok := parsePHPTypeIdentifier(s, e)
			if !ok {
				break
			}
			e = next
			if e < len(s) && s[e] == '*' {
				e++
			}
		}
		return e, true
	}
	return nameEnd, true
}

// parsePHPConditional matches a conditional type's body (from just past "("
// and its spaces) and the closing ")":
// ($var|types) is [not] types ? types : types.
func parsePHPConditional(s string, i int, t *phpType) (int, bool) {
	var inners []phpTypeSpan
	e := i
	if i < len(s) && s[i] == '$' {
		next, ok := parsePHPTypeIdentifier(s, i+1)
		if !ok {
			return 0, false
		}
		e = next
	} else {
		left, ok := parsePHPTypes(s, i)
		if !ok {
			return 0, false
		}
		inners = append(inners, phpTypeSpan{i, left.end})
		e = left.end
	}
	// \h+is(?:\h+not)?\h+, case-insensitively
	p := skipPHPHSpace(s, e)
	if p == e || len(s) < p+2 || !strings.EqualFold(s[p:p+2], "is") {
		return 0, false
	}
	p += 2
	right, ts, ok := 0, phpTypes{}, false
	if n := skipPHPHSpace(s, p); n > p && len(s) >= n+3 && strings.EqualFold(s[n:n+3], "not") {
		right, ts, ok = parsePHPAfterHSpaces(s, n+3, parsePHPTypes)
	}
	if !ok {
		right, ts, ok = parsePHPAfterHSpaces(s, p, parsePHPTypes)
	}
	for k, sep := range []byte{'?', ':', ')'} {
		if !ok {
			return 0, false
		}
		inners = append(inners, phpTypeSpan{right, ts.end})
		c := skipPHPHSpace(s, ts.end)
		if c >= len(s) || s[c] != sep {
			return 0, false
		}
		if k == 2 {
			t.inners = inners
			return c + 1, true
		}
		right, ts, ok = parsePHPAfterSpace(s, c+1, skipPHPHSpace, parsePHPTypes)
	}
	return 0, false
}

// parsePHPCallableAfterName matches the callable alternative after its name:
// an optional template, then the arguments and return type.
func parsePHPCallableAfterName(s string, nameEnd int, t *phpType) (int, bool) {
	q := skipPHPHSpace(s, nameEnd)
	if q < len(s) && s[q] == '<' {
		end, ok := parsePHPCallableTemplate(s, q+1, t)
		if !ok {
			return 0, false
		}
		q = skipPHPHSpace(s, end)
	}
	if q >= len(s) || s[q] != '(' {
		return 0, false
	}
	return parsePHPCallable(s, q+1, t)
}

// parsePHPCallableTemplate matches a callable template from just past "<":
// \h* inner (\h*,\h* inner)*+ \h*> (?=\h*\(), each inner an identifier with an
// optional bound (\h+(?i)(?:of|as)\h+ types) and default (\h*=\h* types).
func parsePHPCallableTemplate(s string, i int, t *phpType) (int, bool) {
	_, e, ok := parsePHPAfterSpace(s, i, skipPHPHSpace, parsePHPTypeIdentifier)
	for {
		if !ok {
			return 0, false
		}
		if p := skipPHPHSpace(s, e); p > e && len(s) >= p+2 && (strings.EqualFold(s[p:p+2], "of") || strings.EqualFold(s[p:p+2], "as")) {
			if b, bound, ok := parsePHPAfterHSpaces(s, p+2, parsePHPTypes); ok {
				t.inners = append(t.inners, phpTypeSpan{b, bound.end})
				e = bound.end
			}
		}
		if p := skipPHPHSpace(s, e); p < len(s) && s[p] == '=' {
			if d, def, ok := parsePHPAfterSpace(s, p+1, skipPHPHSpace, parsePHPTypes); ok {
				t.inners = append(t.inners, phpTypeSpan{d, def.end})
				e = def.end
			}
		}
		c := skipPHPHSpace(s, e)
		if c < len(s) && s[c] == ',' {
			_, e, ok = parsePHPAfterSpace(s, c+1, skipPHPHSpace, parsePHPTypeIdentifier)
			continue
		}
		if c >= len(s) || s[c] != '>' {
			return 0, false
		}
		if p := skipPHPHSpace(s, c+1); p >= len(s) || s[p] != '(' {
			return 0, false
		}
		return c + 1, true
	}
}

// parsePHPCallable matches a callable's arguments (from just past "(") and
// optional return type.
func parsePHPCallable(s string, i int, t *phpType) (int, bool) {
	r := skipPHPHSpace(s, i)
	start, arg, ok := parsePHPAfterSpace(s, i, skipPHPHSpace, parsePHPTypes)
	for ok {
		t.inners = append(t.inners, phpTypeSpan{start, arg.end})
		e := arg.end
		if p := skipPHPHSpace(s, e); p < len(s) && s[p] == '&' {
			e = p + 1
		}
		if p := skipPHPHSpace(s, e); strings.HasPrefix(s[p:], "...") {
			e = p + 3
		}
		if p := skipPHPHSpace(s, e); p < len(s) && s[p] == '$' {
			if next, ok := parsePHPTypeIdentifier(s, p+1); ok {
				e = next
			}
		}
		if p := skipPHPHSpace(s, e); p < len(s) && s[p] == '=' {
			e = p + 1
		}
		r = skipPHPHSpace(s, e)
		if r >= len(s) || s[r] != ',' {
			break
		}
		// another argument, else a trailing comma
		if start, arg, ok = parsePHPAfterSpace(s, r+1, skipPHPHSpace, parsePHPTypes); !ok {
			r = skipPHPHSpace(s, r+1)
		}
	}
	if r >= len(s) || s[r] != ')' {
		return 0, false
	}
	end := r + 1
	if c := skipPHPHSpace(s, end); c < len(s) && s[c] == ':' {
		if rt, ret, ok := parsePHPAfterSpace(s, c+1, skipPHPHSpace, parsePHPType); ok {
			t.inners = append(t.inners, phpTypeSpan{rt, ret.end})
			end = ret.end
		}
	}
	return end, true
}

// parsePHPArrayShape matches an array shape's entries (from just past "{"),
// its optional unsealed part and the closing "}".
func parsePHPArrayShape(s string, i int, t *phpType) (int, bool) {
	entry := func(s string, p int) (int, bool) {
		// a key and its colon, else (the key branch failing) none
		if k, ok := parsePHPShapeKey(s, p); ok {
			c := skipPHPHSpace(s, k)
			if c < len(s) && s[c] == '?' {
				c = skipPHPHSpace(s, c+1)
			}
			if c < len(s) && s[c] == ':' {
				if value, v, ok := parsePHPAfterSpace(s, c+1, skipPHPHSpace, parsePHPTypes); ok {
					t.inners = append(t.inners, phpTypeSpan{value, v.end})
					return v.end, true
				}
			}
		}
		v, ok := parsePHPTypes(s, p)
		if !ok {
			return 0, false
		}
		t.inners = append(t.inners, phpTypeSpan{p, v.end})
		return v.end, true
	}
	r := skipPHPHVStar(s, i)
	if _, e, ok := parsePHPAfterSpace(s, i, skipPHPHVStar, entry); ok {
		end, trailingComma := e, false
		for {
			c := skipPHPHSpace(s, end)
			if c >= len(s) || s[c] != ',' {
				break
			}
			_, e, ok := parsePHPAfterSpace(s, c+1, skipPHPHVStar, entry)
			if !ok {
				end, trailingComma = c+1, true
				break
			}
			end = e
		}
		// entries end with a comma or are not followed by "..."
		if p := skipPHPHSpace(s, end); !trailingComma && strings.HasPrefix(s[p:], "...") {
			t.inners = nil
		} else {
			r = end
		}
	}
	if p := skipPHPHSpace(s, r); strings.HasPrefix(s[p:], "...") {
		r = p + 3
		if p := skipPHPHSpace(s, r); p < len(s) && s[p] == '<' {
			a, ta, ok := parsePHPAfterSpace(s, p+1, skipPHPHSpace, parsePHPTypes)
			if !ok {
				return 0, false
			}
			t.inners = append(t.inners, phpTypeSpan{a, ta.end})
			c := skipPHPHSpace(s, ta.end)
			if c < len(s) && s[c] == ',' {
				b, tb, ok := parsePHPAfterSpace(s, c+1, skipPHPHSpace, parsePHPTypes)
				if !ok {
					return 0, false
				}
				t.inners = append(t.inners, phpTypeSpan{b, tb.end})
				c = skipPHPHSpace(s, tb.end)
			}
			if c >= len(s) || s[c] != '>' {
				return 0, false
			}
			r = c + 1
		}
	}
	r = skipPHPHVStar(s, r)
	if r >= len(s) || s[r] != '}' {
		return 0, false
	}
	return r + 1, true
}

// parsePHPShapeKey matches an array shape key: a constant, identifier or name.
func parsePHPShapeKey(s string, i int) (int, bool) {
	if e, ok := parsePHPTypeConstant(s, i); ok {
		return e, true
	}
	return parsePHPTypeName(s, i)
}

// parsePHPTypeValue parses a whole TypeExpression value, as
// TypeExpression::parse requires: the types must span all of it.
func parsePHPTypeValue(s string) (phpTypes, bool) {
	ts, ok := parsePHPTypes(s, 0)
	if !ok || ts.end != len(s) {
		return phpTypes{}, false
	}
	return ts, true
}

// composite ports how TypeExpression::parse splits a union or intersection:
// its members and its glue, or a zero glue for a single type. When both glues
// appear at the top level, the glue is "|" and each run of "&"-joined members
// (with the spaces around their "&") is one member.
func (ts phpTypes) composite() ([]phpTypeSpan, byte) {
	if len(ts.members) < 2 {
		return nil, 0
	}
	glue := ts.glues[0]
	for _, g := range ts.glues {
		if g != glue {
			glue = '|'
		}
	}
	var out []phpTypeSpan
	member := ts.members[0]
	for k, g := range ts.glues {
		if g == glue {
			out = append(out, member)
			member = ts.members[k+1]
		} else {
			member.end = ts.members[k+1].end
		}
	}
	return append(out, member), glue
}

// sortPHPTypeValue ports TypeExpression::sortTypes(cmp)->toString(): inner
// expressions are sorted first (mapTypes is bottom-up), then the members of a
// union or intersection are stable-sorted, and re-joined with a bare glue
// only if their order changed.
func sortPHPTypeValue(s string, cmp func(a, b string) int) (string, bool) {
	ts, ok := parsePHPTypeValue(s)
	if !ok {
		return s, false
	}
	spans, glue := ts.composite()
	if glue == 0 {
		t, _ := parsePHPType(s, 0)
		var inners []phpTypeSpan
		switch {
		case t.prefixLen > 0 && t.nullableLen > 0:
			// PHP cuts the nullable mark's length off the front, prefix
			// included, and fails to parse what is left
			return s, false
		case t.nullableLen > 0:
			inners = []phpTypeSpan{{t.nullableLen, len(s)}}
		case t.arrayStart < t.end:
			inners = []phpTypeSpan{{0, t.arrayStart}}
		case t.prefixLen > 0:
			// the generic, callable, shape and parenthesized branches only
			// register inner types when they start the value
		default:
			inners = t.inners
		}
		return replacePHPTypeSpans(s, inners, cmp)
	}
	value, ok := replacePHPTypeSpans(s, spans, cmp)
	if !ok {
		return s, false
	}
	if value != s {
		ts, ok := parsePHPTypeValue(value)
		if !ok {
			return s, false
		}
		if spans, glue = ts.composite(); glue == 0 {
			return s, false
		}
	}
	members := make([]string, len(spans))
	for k, m := range spans {
		members[k] = value[m.start:m.end]
	}
	order := make([]int, len(members))
	for k := range order {
		order[k] = k
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp(members[a], members[b]) })
	moved := false
	sorted := make([]string, len(members))
	for k, idx := range order {
		sorted[k] = members[idx]
		moved = moved || idx != k
	}
	if !moved {
		return value, true
	}
	return strings.Join(sorted, string(glue)), true
}

// replacePHPTypeSpans sorts each inner span of s and splices the results in.
func replacePHPTypeSpans(s string, spans []phpTypeSpan, cmp func(a, b string) int) (string, bool) {
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		inner, ok := sortPHPTypeValue(s[sp.start:sp.end], cmp)
		if !ok {
			return s, false
		}
		b.WriteString(s[last:sp.start])
		b.WriteString(inner)
		last = sp.end
	}
	b.WriteString(s[last:])
	return b.String(), true
}

// phpTypeMembers returns TypeExpression::getTypes() and getTypesGlue() of a
// value: its members (or the value itself) and its glue (0 when not composite).
func phpTypeMembers(s string) ([]string, byte) {
	ts, ok := parsePHPTypeValue(s)
	if !ok {
		return []string{s}, 0
	}
	spans, glue := ts.composite()
	if glue == 0 {
		return []string{s}, 0
	}
	out := make([]string, len(spans))
	for k, m := range spans {
		out[k] = s[m.start:m.end]
	}
	return out, glue
}
