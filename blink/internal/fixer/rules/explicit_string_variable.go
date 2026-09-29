package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/ExplicitStringVariableFixer.php
//
// ExplicitStringVariable wraps simple variable interpolation in double-quoted
// strings and heredocs in the explicit "{$...}" form: "$a" -> "{$a}",
// "$a->b" -> "{$a->b}", "$a[key]" -> "{$a['key']}". Already-explicit ("{$a}",
// "${a}"), single-quoted strings and nowdocs are left untouched.
type ExplicitStringVariable struct{}

func (ExplicitStringVariable) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\ExplicitStringVariableFixer`
}

func (ExplicitStringVariable) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/ExplicitStringVariableFixer.php"
}

func (ExplicitStringVariable) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind != token.String {
			continue
		}
		v := s.At(i).Value
		start, ok := interpolatedBody(v)
		if !ok {
			continue
		}
		end := len(v) - suffixLen(v, start)
		if end < start {
			continue
		}
		body, hit := wrapInterpolations(v[start:end])
		if hit {
			s.SetValue(i, v[:start]+body+v[end:])
			changed = true
		}
	}
	return changed
}

// interpolatedBody returns the offset where the interpolated body begins for a
// double-quoted string or a heredoc (not a nowdoc or single-quoted string), and
// whether the token is one that interpolates at all.
func interpolatedBody(v string) (int, bool) {
	if len(v) > 0 && v[0] == '"' {
		return 1, true
	}
	if len(v) >= 3 && v[0] == '<' && v[1] == '<' && v[2] == '<' {
		p := 3
		for p < len(v) && (v[p] == ' ' || v[p] == '\t') {
			p++
		}
		if p < len(v) && v[p] == '\'' {
			return 0, false // nowdoc
		}
		// heredoc: body starts after the opener line
		for p < len(v) && v[p] != '\n' {
			p++
		}
		if p < len(v) {
			return p + 1, true
		}
	}
	return 0, false
}

// suffixLen is the length of the trailing delimiter after the body: 1 for the
// closing quote of a double-quoted string, or the closing heredoc label line.
func suffixLen(v string, bodyStart int) int {
	if v[0] == '"' {
		return 1
	}
	// heredoc: the closing label sits on the last line; the body ends at the
	// last newline before it.
	last := -1
	for i := bodyStart; i < len(v); i++ {
		if v[i] == '\n' {
			last = i
		}
	}
	if last < 0 {
		return 0
	}
	return len(v) - last // include the newline and the closing label
}

func wrapInterpolations(b string) (string, bool) {
	out := make([]byte, 0, len(b)+8)
	changed := false
	i := 0
	for i < len(b) {
		c := b[i]
		if c == '\\' && i+1 < len(b) {
			out = append(out, c, b[i+1])
			i += 2
			continue
		}
		// already-explicit "{$...}" - copy the balanced block verbatim
		if c == '{' && i+1 < len(b) && b[i+1] == '$' {
			depth := 0
			j := i
			for j < len(b) {
				if b[j] == '{' {
					depth++
				} else if b[j] == '}' {
					depth--
					if depth == 0 {
						j++
						break
					}
				}
				j++
			}
			out = append(out, b[i:j]...)
			i = j
			continue
		}
		if c == '$' && i+1 < len(b) {
			// "${...}" complex syntax - leave as is
			if b[i+1] == '{' {
				out = append(out, '$', '{')
				i += 2
				continue
			}
			// "$$x" - first $ was literal; do not start an interpolation here
			if i > 0 && b[i-1] == '$' {
				out = append(out, c)
				i++
				continue
			}
			if isNameStart(b[i+1]) {
				expr, end := parseSimpleVar(b, i)
				out = append(out, '{')
				out = append(out, expr...)
				out = append(out, '}')
				i = end
				changed = true
				continue
			}
		}
		out = append(out, c)
		i++
	}
	return string(out), changed
}

// parseSimpleVar reads a PHP simple-syntax interpolation starting at the "$" at
// index i ("$name", "$name->prop", "$name[index]") and returns the expression
// to place inside braces, quoting a bareword array key, plus the end offset.
func parseSimpleVar(b string, i int) ([]byte, int) {
	j := i + 1
	for j < len(b) && isNameByte(b[j]) {
		j++
	}
	expr := make([]byte, 0, j-i+4)
	expr = append(expr, b[i:j]...) // "$name"

	switch {
	case j+2 < len(b) && b[j] == '-' && b[j+1] == '>' && isNameStart(b[j+2]):
		k := j + 2
		for k < len(b) && isNameByte(b[k]) {
			k++
		}
		expr = append(expr, b[j:k]...) // "->prop"
		return expr, k
	case j < len(b) && b[j] == '[':
		k := j + 1
		for k < len(b) && b[k] != ']' {
			k++
		}
		if k < len(b) && b[k] == ']' {
			idx := b[j+1 : k]
			expr = append(expr, '[')
			expr = append(expr, quoteIndex(idx)...)
			expr = append(expr, ']')
			return expr, k + 1
		}
	}
	return expr, j
}

// quoteIndex mirrors PHP's simple-syntax array access inside interpolation: a
// numeric or variable index is kept, a bareword key is single-quoted.
func quoteIndex(idx string) []byte {
	if len(idx) == 0 {
		return []byte(idx)
	}
	if idx[0] == '$' {
		return []byte(idx)
	}
	numeric := true
	s := idx
	if s[0] == '-' {
		s = s[1:]
	}
	if len(s) == 0 {
		numeric = false
	}
	for k := 0; k < len(s); k++ {
		if s[k] < '0' || s[k] > '9' {
			numeric = false
			break
		}
	}
	if numeric {
		return []byte(idx)
	}
	out := make([]byte, 0, len(idx)+2)
	out = append(out, '\'')
	out = append(out, idx...)
	out = append(out, '\'')
	return out
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isNameByte(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}
