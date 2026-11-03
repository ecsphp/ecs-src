package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// literalChunks returns the indices of the String chunks of the interpolated
// literal whose Head is at head, through its Tail, leaving out those of nested
// literals; nil when the Tail is missing.
func literalChunks(s *tokens.Stream, head int) []int {
	chunks := []int{head}
	depth := 0
	for j := head + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.String {
			continue
		}
		switch t.Part {
		case token.Head:
			depth++
		case token.Mid:
			if depth == 0 {
				chunks = append(chunks, j)
			}
		case token.Tail:
			if depth == 0 {
				return append(chunks, j)
			}
			depth--
		}
	}
	return nil
}

// chunkEdges tells which PHP tokens a literal chunk holds at its edges.
type chunkEdges struct {
	// hasStringPart: the chunk holds a T_ENCAPSED_AND_WHITESPACE, T_START_HEREDOC
	// or the opening/closing quote (PHP-CS-Fixer's "string part" tokens)
	hasStringPart bool
	// endsWithStringPart: its last token is one of those
	endsWithStringPart bool
	// opensExpr: it ends with "{" (T_CURLY_OPEN), "${" or "${name" (followed by
	// "["), so an expression follows, closed by the "}" the next chunk starts with
	opensExpr bool
}

// scanChunk reports the edges of the chunk value v: the literal's Head when head
// is set, else a chunk that follows interpolated code, starting with the "}"
// that closes an expression when afterExpr is set.
func scanChunk(v string, head, afterExpr bool) chunkEdges {
	var e chunkEdges
	p := 0
	switch {
	case head:
		if len(v) > 1 && (v[0] == 'b' || v[0] == 'B') {
			p++
		}
		if strings.HasPrefix(v[p:], "<<<") {
			p += strings.IndexByte(v[p:], '\n') + 1
		} else {
			p++ // the opening quote
		}
		e.hasStringPart, e.endsWithStringPart = true, true
	case afterExpr:
		p = 1 // the "}" closing the expression
	}
	for p < len(v) {
		switch c := v[p]; {
		case c == '$' && p+1 < len(v) && v[p+1] == '{':
			q := p + 2
			for q < len(v) && isNameByte(v[q]) {
				q++
			}
			if q < len(v) && v[q] == '}' { // "${name}"
				p = q + 1
				e.endsWithStringPart = false
				continue
			}
			e.opensExpr, e.endsWithStringPart = true, false
			return e
		case c == '{' && p == len(v)-1:
			// a "{" right before interpolated code is T_CURLY_OPEN (a "{"
			// before a simple "$name" would have opened "{$")
			e.opensExpr, e.endsWithStringPart = true, false
			return e
		case c == '\\':
			p++
		}
		p++
		e.hasStringPart, e.endsWithStringPart = true, true
	}
	return e
}

// encapsedHasNewline reports whether the String token t holds a
// T_ENCAPSED_AND_WHITESPACE with a newline: text of an interpolated literal or
// a backtick one, or the body of a heredoc or nowdoc (a plain
// T_CONSTANT_ENCAPSED_STRING holds none).
func encapsedHasNewline(t token.Token) bool {
	if t.Kind != token.String {
		return false
	}
	_, v := token.SplitBinaryPrefix(t.Value)
	if strings.HasPrefix(v, "<<<") && (t.Part == token.Whole || t.Part == token.Head) {
		v = v[strings.IndexByte(v, '\n')+1:]
	} else if t.Part == token.Whole && !t.IsShellExec() && !strings.Contains(v, "${") {
		return false
	}
	return strings.Contains(v, "\n")
}

// lineIndentOutsideString returns the indentation of the line holding token
// idx, found as PHP-CS-Fixer's MethodArgumentSpaceFixer finds it: the text after
// the last newline of the closest whitespace, inline HTML or string text
// before idx that spans one. ok is false when that is not whitespace (idx sits
// on a line inside a heredoc or a multiline string).
func lineIndentOutsideString(s *tokens.Stream, idx int) (indent string, ok bool) {
	for i := idx - 1; i >= 0; i-- {
		t := s.At(i)
		switch {
		case t.Kind == token.Whitespace && strings.Contains(t.Value, "\n"):
			return t.Value[strings.LastIndexByte(t.Value, '\n')+1:], true
		case t.Kind == token.InlineHTML && strings.Contains(t.Value, "\n"), encapsedHasNewline(t):
			return "", false
		}
	}
	return "", true
}
