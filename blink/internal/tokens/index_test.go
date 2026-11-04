package tokens

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

func TestIndexKindsAndWords(t *testing.T) {
	s := New([]token.Token{
		{Kind: token.OpenTag, Value: "<?php "},
		{Kind: token.Keyword, Value: "SWITCH"},
		{Kind: token.Whitespace, Value: " "},
		{Kind: token.Ident, Value: "SprintF"},
		{Kind: token.Variable, Value: "$implode"},
		{Kind: token.String, Value: "'join'"},
		{Kind: token.Comment, Value: "// pos"},
	})
	x := s.Index()
	for _, k := range []token.Kind{token.OpenTag, token.Keyword, token.Whitespace, token.Ident, token.Variable, token.String, token.Comment} {
		if !x.HasKind(k) {
			t.Errorf("HasKind(%v) = false", k)
		}
	}
	for _, k := range []token.Kind{token.DocComment, token.CloseTag, token.InlineHTML, token.Punct} {
		if x.HasKind(k) {
			t.Errorf("HasKind(%v) = true", k)
		}
	}
	// keywords and names, whatever their case
	if !x.HasWord("switch") || !x.HasWord("sprintf") || !x.HasWord("SPRINTF") {
		t.Error("a word in the stream is missing")
	}
	// variables, strings and comments are not words
	if x.HasWord("implode") || x.HasWord("join") || x.HasWord("pos") {
		t.Error("a variable, string or comment counted as a word")
	}
	if !x.HasAnyWord("implode", "sprintf") || x.HasAnyWord("implode", "join") {
		t.Error("HasAnyWord")
	}
}

// A word with a non-ASCII byte may fold to an ASCII one (strings.EqualFold
// matches the Kelvin sign with "k"), so the index can then rule nothing out.
func TestIndexNonASCIIWord(t *testing.T) {
	s := New([]token.Token{{Kind: token.Ident, Value: "Key"}})
	if !s.Index().HasWord("key") || !s.Index().HasWord("anything") {
		t.Fatal("a non-ASCII word must make every word possible")
	}
}

// The index follows the stream: a write moves the version, and the next
// Index call sees the new tokens.
func TestIndexRebuiltAfterWrite(t *testing.T) {
	s := New([]token.Token{{Kind: token.Ident, Value: "foo"}})
	if s.Index().HasWord("sprintf") {
		t.Fatal("sprintf before the write")
	}
	s.InsertAt(1, token.Token{Kind: token.Ident, Value: "sprintf"})
	if !s.Index().HasWord("sprintf") || !s.Index().HasWord("foo") {
		t.Fatal("index not rebuilt after InsertAt")
	}
	s.SetValue(0, "bar")
	if !s.Index().HasWord("bar") {
		t.Fatal("index not rebuilt after SetValue")
	}
	s.Set(1, token.Token{Kind: token.DocComment, Value: "/** */"})
	if !s.Index().HasKind(token.DocComment) {
		t.Fatal("index not rebuilt after Set")
	}
}

// Many distinct words: every one of them must still be found.
func TestIndexNoFalseNegatives(t *testing.T) {
	var toks []token.Token
	var words []string
	for i := range 20000 {
		w := "w" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676%26))
		words = append(words, w)
		toks = append(toks, token.Token{Kind: token.Ident, Value: w})
	}
	x := New(toks).Index()
	for _, w := range words {
		if !x.HasWord(w) {
			t.Fatalf("%s missing", w)
		}
	}
}

func TestHasNonASCII(t *testing.T) {
	s := New([]token.Token{{Kind: token.Ident, Value: "foo"}, {Kind: token.Comment, Value: "// plain"}})
	if s.HasNonASCII() {
		t.Fatal("an ASCII stream has no non-ASCII byte")
	}
	s.InsertAt(1, token.Token{Kind: token.String, Value: "'a b'"})
	if !s.HasNonASCII() {
		t.Fatal("a no-break space written after the first call was missed")
	}
}
