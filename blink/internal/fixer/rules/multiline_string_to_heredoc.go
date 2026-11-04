package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/MultilineStringToHeredocFixer.php
//
// MultilineStringToHeredoc converts a multiline single- or double-quoted string
// into a nowdoc or heredoc with the "EOD" marker (adding "_" to avoid colliding
// with the body). Single-line strings and backtick/heredoc literals are left
// alone.
type MultilineStringToHeredoc struct{}

func (MultilineStringToHeredoc) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\MultilineStringToHeredocFixer`
}

func (MultilineStringToHeredoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/MultilineStringToHeredocFixer.php"
}

func (MultilineStringToHeredoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.IsLiteral() {
			if _, rest := token.SplitBinaryPrefix(t.Value); len(rest) >= 2 && (rest[0] == '\'' || rest[0] == '"') {
				if multilineConvertWhole(s, i) {
					changed = true
				}
			}
			continue
		}
		// a double-quoted interpolated string opens with its Head; skip to its Tail
		// so the tokens inside (including nested strings) are left untouched, like
		// PHP-CS-Fixer's stateful loop does
		if t.Part == token.Head {
			if _, rest := token.SplitBinaryPrefix(t.Value); len(rest) > 0 && rest[0] == '"' {
				chunks := literalChunks(s, i)
				if chunks != nil {
					if multilineConvertComplex(s, chunks) {
						changed = true
					}
					i = chunks[len(chunks)-1]
				}
			}
		}
	}
	return changed
}

// multilineConvertWhole converts a single whole single-/double-quoted string
// token at i into a nowdoc/heredoc, when its content spans more than one line.
func multilineConvertWhole(s *tokens.Stream, i int) bool {
	_, rest := token.SplitBinaryPrefix(s.At(i).Value)
	quote := rest[0]
	if rest[len(rest)-1] != quote {
		return false
	}
	content := rest[1 : len(rest)-1]
	isSingle := quote == '\''
	if isSingle {
		content = unescapeSingleQuotedBody(content)
	} else {
		content = unescapeDoubleQuotedBody(content)
	}
	if !strings.ContainsAny(content, "\n\r") {
		return false
	}
	marker := multilineClosingMarker(content)
	quoting := ""
	if isSingle {
		quoting = "'"
	}
	s.SetValue(i, "<<<"+quoting+marker+quoting+"\n"+content+"\n"+marker)
	return true
}

// multilineConvertComplex converts a double-quoted interpolated string, given
// its chunk indices (Head, Mids, Tail), into a heredoc.
func multilineConvertComplex(s *tokens.Stream, chunks []int) bool {
	start, end := chunks[0], chunks[len(chunks)-1]
	_, headRest := token.SplitBinaryPrefix(s.At(start).Value)
	tailVal := s.At(end).Value
	if len(headRest) < 1 || headRest[0] != '"' || len(tailVal) < 1 || tailVal[len(tailVal)-1] != '"' {
		return false
	}
	headBody := headRest[1:]
	tailBody := tailVal[:len(tailVal)-1]

	var body strings.Builder
	body.WriteString(headBody)
	for j := start + 1; j < end; j++ {
		body.WriteString(s.At(j).Value)
	}
	body.WriteString(tailBody)
	if !strings.ContainsAny(body.String(), "\n\r") {
		return false
	}
	marker := multilineClosingMarker(body.String())

	s.SetValue(start, "<<<"+marker+"\n"+unescapeDoubleQuotedBody(headBody))
	for _, j := range chunks[1 : len(chunks)-1] {
		s.SetValue(j, unescapeDoubleQuotedBody(s.At(j).Value))
	}
	// a "\n" always separates the last body content from the end marker
	s.SetValue(end, unescapeDoubleQuotedBody(tailBody)+"\n"+marker)
	return true
}

// multilineClosingMarker returns "EOD" with as many "_" appended as needed so it
// does not collide with a line of the body.
func multilineClosingMarker(body string) string {
	marker := "EOD"
	for heredocMarkerCollides(body, marker) {
		marker += "_"
	}
	return marker
}

// unescapeSingleQuotedBody applies Preg::replace('~\\([\\\'])~', '$1', ...):
// "\\" -> "\" and "\'" -> "'".
func unescapeSingleQuotedBody(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); {
		if v[i] == '\\' && i+1 < len(v) && (v[i+1] == '\\' || v[i+1] == '\'') {
			b.WriteByte(v[i+1])
			i += 2
			continue
		}
		b.WriteByte(v[i])
		i++
	}
	return b.String()
}

// unescapeDoubleQuotedBody applies Preg::replace('~(\\\\)|\\(")~', '$1$2', ...):
// "\\" is kept and "\"" -> "\"".
func unescapeDoubleQuotedBody(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); {
		if v[i] == '\\' && i+1 < len(v) {
			if v[i+1] == '\\' {
				b.WriteString(`\\`)
				i += 2
				continue
			}
			if v[i+1] == '"' {
				b.WriteByte('"')
				i += 2
				continue
			}
		}
		b.WriteByte(v[i])
		i++
	}
	return b.String()
}
