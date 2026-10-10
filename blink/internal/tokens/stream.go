// Package tokens holds a flat, mutable, index-addressable token stream. This is
// the ECS/PHP-CS-Fixer working model: fixers read tokens by index and insert,
// replace or remove them in place, then Render rebuilds the source.
package tokens

import (
	"slices"
	"strings"

	"blink/internal/token"
)

// Stream is a flat, mutable, index-addressable token sequence. Fixers read
// tokens by index and splice them in place; Render rebuilds the source.
type Stream struct {
	toks []token.Token
	path string
}

func New(toks []token.Token) *Stream {
	return &Stream{toks: toks}
}

// SetPath records the source file path, for the few fixers (PsrAutoloading) that
// derive output from the file location rather than the tokens alone.
func (s *Stream) SetPath(p string) { s.path = p }

// Path returns the source file path set by the runner, or "" when the stream was
// built from a string with no backing file.
func (s *Stream) Path() string { return s.path }

func (s *Stream) Len() int { return len(s.toks) }

func (s *Stream) At(i int) token.Token { return s.toks[i] }

// Kind returns just the kind of the token at i, avoiding a full Token copy in
// the hot index-scanning helpers.
func (s *Stream) Kind(i int) token.Kind { return s.toks[i].Kind }

func (s *Stream) Set(i int, t token.Token) { s.toks[i] = t }

// SetValue replaces only the Value of the token at i, keeping its kind.
func (s *Stream) SetValue(i int, v string) { s.toks[i].Value = v }

func (s *Stream) RemoveAt(i int) {
	s.toks = slices.Delete(s.toks, i, i+1)
}

func (s *Stream) InsertAt(i int, t token.Token) {
	s.toks = slices.Insert(s.toks, i, t)
}

// InsertSliceAt inserts ts at index i in one splice, so the tail shifts once
// instead of once per element.
func (s *Stream) InsertSliceAt(i int, ts []token.Token) {
	s.toks = slices.Insert(s.toks, i, ts...)
}

// MatchForward returns the index of the delimiter matching the opener at i
// ("(", "{" or "["), or -1 if there is none. Strings and comments are single
// tokens, so scanning punctuation is safe.
func (s *Stream) MatchForward(i int) int {
	if i < 0 || i >= len(s.toks) || s.toks[i].Kind != token.Punct {
		return -1
	}
	open := s.toks[i].Value
	var closer string
	switch open {
	case "(":
		closer = ")"
	case "{":
		closer = "}"
	case "[":
		closer = "]"
	default:
		return -1
	}
	depth := 0
	for j := i; j < len(s.toks); j++ {
		if s.toks[j].Kind != token.Punct {
			continue
		}
		switch s.toks[j].Value {
		case open:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// MatchBackward returns the index of the delimiter matching the closer at i
// (")", "}" or "]"), or -1 if there is none.
func (s *Stream) MatchBackward(i int) int {
	if i < 0 || i >= len(s.toks) || s.toks[i].Kind != token.Punct {
		return -1
	}
	closer := s.toks[i].Value
	var opener string
	switch closer {
	case ")":
		opener = "("
	case "}":
		opener = "{"
	case "]":
		opener = "["
	default:
		return -1
	}
	depth := 0
	for j := i; j >= 0; j-- {
		if s.toks[j].Kind != token.Punct {
			continue
		}
		switch s.toks[j].Value {
		case closer:
			depth++
		case opener:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// ReplaceRange swaps tokens [start, end] (inclusive) for repl. The splice is
// done in place: when len(repl) matches the range the tail never moves, and
// otherwise only the tail shifts, so no fresh tail copy is allocated.
func (s *Stream) ReplaceRange(start, end int, repl []token.Token) {
	s.toks = slices.Replace(s.toks, start, end+1, repl...)
}

// Render concatenates every token value back into source.
func (s *Stream) Render() string {
	n := 0
	for i := range s.toks {
		n += len(s.toks[i].Value)
	}
	var b strings.Builder
	b.Grow(n)
	for i := range s.toks {
		b.WriteString(s.toks[i].Value)
	}
	return b.String()
}

// Tokens returns the underlying slice (read-only use).
func (s *Stream) Tokens() []token.Token { return s.toks }
