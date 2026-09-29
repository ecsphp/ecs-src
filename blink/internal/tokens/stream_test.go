package tokens

import (
	"testing"

	"blink/internal/token"
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
