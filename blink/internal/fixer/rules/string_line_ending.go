package rules

import (
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringLineEndingFixer.php
//
// StringLineEnding turns every line break in string literals and inline HTML
// into "\n". A line break is what PCRE's \R matches.
type StringLineEnding struct{}

func (StringLineEnding) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\StringLineEndingFixer`
}

func (StringLineEnding) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringLineEndingFixer.php"
}

func (StringLineEnding) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.InlineHTML || !stringLineEndingCandidate(t.Value) {
			continue
		}
		// PHP's T_CLOSE_TAG takes the line break right after "?>"
		lead := 0
		if i > 0 && s.At(i-1).Kind == token.CloseTag {
			switch {
			case strings.HasPrefix(t.Value, "\r\n"):
				lead = 2
			case strings.HasPrefix(t.Value, "\n") || strings.HasPrefix(t.Value, "\r"):
				lead = 1
			}
		}
		if v := t.Value[:lead] + stringLineEndingReplace(t.Value[lead:]); v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	eachStringChunk(s, func(i int, parts []encPart) {
		v := s.At(i).Value
		if !stringLineEndingCandidate(v) {
			return
		}
		repl := map[int]string{}
		for k, p := range parts {
			if p.kind != encConstString && p.kind != encText {
				continue
			}
			content := v[p.start:p.end]
			if r := stringLineEndingReplace(content); r != content {
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

// stringLineEndingCandidate is a quick filter: v holds a byte that can be part
// of a line break other than LF.
func stringLineEndingCandidate(v string) bool {
	return strings.ContainsAny(v, "\r\v\f") || strings.IndexByte(v, 0x85) >= 0 ||
		strings.Contains(v, " ") || strings.Contains(v, " ")
}

// stringLineEndingReplace is Preg::replace('#\R#u', "\n", $v): \R with the u
// modifier on valid UTF-8, and byte-wise (the retry without u) otherwise.
func stringLineEndingReplace(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	if utf8.ValidString(v) {
		for i, r := range v {
			switch r {
			case '\r':
				if i+1 < len(v) && v[i+1] == '\n' {
					continue // the LF that follows writes the break
				}
				b.WriteByte('\n')
			case '\v', '\f', '\u0085', ' ', ' ':
				b.WriteByte('\n')
			default:
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	for i := 0; i < len(v); i++ {
		switch c := v[i]; c {
		case '\r':
			if i+1 < len(v) && v[i+1] == '\n' {
				continue
			}
			b.WriteByte('\n')
		case '\v', '\f', 0x85:
			b.WriteByte('\n')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
