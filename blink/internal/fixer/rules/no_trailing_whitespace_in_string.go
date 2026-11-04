package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/NoTrailingWhitespaceInStringFixer.php
//
// NoTrailingWhitespaceInString removes horizontal whitespace at the end of lines
// in string literals, heredoc/nowdoc bodies and inline HTML. For the very last
// inline-HTML token it also trims whitespace at the end of the file (risky: it
// may change string comparisons and output).
type NoTrailingWhitespaceInString struct{}

func (NoTrailingWhitespaceInString) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\NoTrailingWhitespaceInStringFixer`
}

func (NoTrailingWhitespaceInString) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/NoTrailingWhitespaceInStringFixer.php"
}

func (NoTrailingWhitespaceInString) Fix(s *tokens.Stream) bool {
	changed := false
	last := s.Len() - 1
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.InlineHTML {
			continue
		}
		// only the stream's last inline-HTML token trims a trailing run at the
		// very end (PHP's "/\h+(?=\R|$)/" vs "/\h+(?=\R)/")
		if v := stripTrailingWSBeforeBreak(t.Value, i == last); v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	eachStringChunk(s, func(i int, parts []encPart) {
		v := s.At(i).Value
		repl := map[int]string{}
		for k, p := range parts {
			if p.kind != encConstString && p.kind != encText {
				continue // quotes, markers and interpolation are left intact
			}
			content := v[p.start:p.end]
			if r := stripTrailingWSBeforeBreak(content, false); r != content {
				repl[k] = r
			}
		}
		if len(repl) > 0 {
			s.SetValue(i, encRebuild(v, parts, repl))
			changed = true
		}
	})
	return changed
}

// stripTrailingWSBeforeBreak drops every maximal run of horizontal whitespace
// that is immediately followed by a line break, and (when atEnd) a run at the
// very end of v. It mirrors PHP-CS-Fixer's "/\h+(?=\R)/" / "/\h+(?=\R|$)/".
func stripTrailingWSBeforeBreak(v string, atEnd bool) string {
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); {
		if !isHorizontalWS(v[i]) {
			b.WriteByte(v[i])
			i++
			continue
		}
		j := i
		for j < len(v) && isHorizontalWS(v[j]) {
			j++
		}
		if (j < len(v) && isLineBreakByte(v[j])) || (atEnd && j == len(v)) {
			i = j // drop the run
			continue
		}
		b.WriteString(v[i:j])
		i = j
	}
	return b.String()
}

// isHorizontalWS reports whether c is matched by PCRE's "\h" in non-UTF mode:
// tab, space and the no-break space byte.
func isHorizontalWS(c byte) bool {
	return c == '\t' || c == ' ' || c == 0xA0
}

// isLineBreakByte reports whether c starts a line break matched by PCRE's "\R"
// in non-UTF mode: LF, CR, vertical tab, form feed or NEL.
func isLineBreakByte(c byte) bool {
	return c == '\n' || c == '\r' || c == '\v' || c == '\f' || c == 0x85
}
