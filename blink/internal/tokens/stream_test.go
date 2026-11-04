package tokens

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

func mk(vals ...string) *Stream {
	var toks []token.Token
	for _, v := range vals {
		toks = append(toks, token.Token{Kind: token.Punct, Value: v})
	}
	return New(toks)
}

func TestInsertRemoveRender(t *testing.T) {
	s := mk("a", "b", "d")
	s.InsertAt(2, token.Token{Value: "c"})
	if got := s.Render(); got != "abcd" {
		t.Fatalf("insert: got %q", got)
	}
	s.RemoveAt(0)
	if got := s.Render(); got != "bcd" {
		t.Fatalf("remove: got %q", got)
	}
}

func TestInsertAtEnd(t *testing.T) {
	s := mk("a", "b")
	s.InsertAt(s.Len(), token.Token{Value: "c"})
	if got := s.Render(); got != "abc" {
		t.Fatalf("append: got %q", got)
	}
}

func TestSetValue(t *testing.T) {
	s := mk("a", "b")
	s.SetValue(1, "B")
	if got := s.Render(); got != "aB" {
		t.Fatalf("setvalue: got %q", got)
	}
}

func TestMatchForward(t *testing.T) {
	// ( a [ b ] c )  -> outer "(" at 0 matches ")" at 6
	s := mk("(", "a", "[", "b", "]", "c", ")")
	if got := s.MatchForward(0); got != 6 {
		t.Fatalf("outer match: got %d, want 6", got)
	}
	if got := s.MatchForward(2); got != 4 {
		t.Fatalf("inner match: got %d, want 4", got)
	}
	if got := s.MatchForward(1); got != -1 {
		t.Fatalf("non-opener should be -1, got %d", got)
	}
}

func TestReplaceRange(t *testing.T) {
	s := mk("a", "b", "c", "d")
	s.ReplaceRange(1, 2, []token.Token{{Kind: token.Punct, Value: "X"}})
	if got := s.Render(); got != "aXd" {
		t.Fatalf("replace: got %q", got)
	}
}

func TestClearEmptyTokens(t *testing.T) {
	s := mk("", "a", "", "", "b", "")
	s.ClearEmptyTokens()
	if s.Len() != 2 || s.At(0).Value != "a" || s.At(1).Value != "b" {
		t.Fatalf("got %v", s.Tokens())
	}
}

// Every write that can add an empty token makes the next ClearEmptyTokens scan.
func TestClearEmptyTokensAfterEachWrite(t *testing.T) {
	writes := map[string]func(*Stream){
		"Set":           func(s *Stream) { s.Set(1, token.Token{}) },
		"SetValue":      func(s *Stream) { s.SetValue(1, "") },
		"InsertAt":      func(s *Stream) { s.InsertAt(1, token.Token{}) },
		"InsertSliceAt": func(s *Stream) { s.InsertSliceAt(1, []token.Token{{Value: "x"}, {}}) },
		"ReplaceRange":  func(s *Stream) { s.ReplaceRange(1, 1, []token.Token{{}, {Value: "b"}}) },
	}
	for name, write := range writes {
		s := mk("a", "b", "c")
		s.ClearEmptyTokens()
		write(s)
		s.ClearEmptyTokens()
		for i := range s.Len() {
			if s.At(i).Value == "" {
				t.Errorf("%s: empty token survived at %d", name, i)
			}
		}
	}
}

func TestVersionMovesOnWritesOnly(t *testing.T) {
	s := mk("a", "b", "c")
	writes := []struct {
		name  string
		write func()
	}{
		{"Set", func() { s.Set(0, token.Token{Kind: token.Punct, Value: "A"}) }},
		{"SetValue", func() { s.SetValue(1, "B") }},
		{"InsertAt", func() { s.InsertAt(0, token.Token{Value: "x"}) }},
		{"InsertSliceAt", func() { s.InsertSliceAt(0, []token.Token{{Value: "y"}}) }},
		{"RemoveAt", func() { s.RemoveAt(0) }},
		{"ReplaceRange", func() { s.ReplaceRange(0, 0, []token.Token{{Value: ""}}) }},
		{"ClearEmptyTokens", s.ClearEmptyTokens},
	}
	for _, w := range writes {
		v := s.Version()
		w.write()
		if s.Version() == v {
			t.Errorf("%s: version did not move", w.name)
		}
	}

	v := s.Version()
	s.SetValue(0, s.At(0).Value)
	s.Set(1, s.At(1))
	s.ClearEmptyTokens()
	if s.Version() != v {
		t.Errorf("no-op writes moved the version")
	}
}

func TestMemo(t *testing.T) {
	s := mk("a", "b")
	builds := 0
	build := func() any { builds++; return s.Render() }
	type key struct{}
	if got := s.Memo(key{}, build); got != "ab" || builds != 1 {
		t.Fatalf("first: got %v after %d builds", got, builds)
	}
	if got := s.Memo(key{}, build); got != "ab" || builds != 1 {
		t.Fatalf("unchanged: got %v after %d builds", got, builds)
	}
	s.SetValue(0, "A")
	if got := s.Memo(key{}, build); got != "Ab" || builds != 2 {
		t.Fatalf("after a write: got %v after %d builds", got, builds)
	}
}
