package lexer

import (
	"strings"
	"testing"

	"blink/internal/token"
)

// lossless is the core invariant: rendering all token values equals the source.
func TestLexLossless(t *testing.T) {
	cases := []string{
		"",
		"<?php $x = 1;\n",
		"plain html only",
		"<html><?php echo $a ; ?><div>after</div>",
		"<?php\n// line comment\n$foo = 'bar';\n/** doc */\nfunction f() {}\n",
		"<?php $s = \"a ; b\"; $t = 'c ; d';\n",
		"text <?= $v ?> more",
	}
	for _, src := range cases {
		var got strings.Builder
		for _, tk := range Lex(src) {
			got.WriteString(tk.Value)
		}
		if got.String() != src {
			t.Errorf("not lossless\n src: %q\n got: %q", src, got.String())
		}
	}
}

func TestLexClassifies(t *testing.T) {
	toks := Lex("<?php $x = 1;")
	want := []token.Kind{
		token.OpenTag,    // <?php
		token.Whitespace, // space
		token.Variable,   // $x
		token.Whitespace,
		token.Punct, // =
		token.Whitespace,
		token.Number, // 1
		token.Punct,  // ;
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, k := range want {
		if toks[i].Kind != k {
			t.Errorf("token %d: got %s, want %s (value %q)", i, toks[i].Kind, k, toks[i].Value)
		}
	}
}

func TestDocCommentVsComment(t *testing.T) {
	toks := Lex("<?php /** doc */ /* plain */ /**/")
	var kinds []token.Kind
	for _, tk := range toks {
		if tk.Kind == token.DocComment || tk.Kind == token.Comment {
			kinds = append(kinds, tk.Kind)
		}
	}
	want := []token.Kind{token.DocComment, token.Comment, token.Comment}
	if len(kinds) != len(want) {
		t.Fatalf("got %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("comment %d: got %s want %s", i, kinds[i], want[i])
		}
	}
}

func TestLexHeredocNowdocOpaque(t *testing.T) {
	cases := []string{
		"<?php $x = <<<EOT\nbody $a<$b if($c)\nEOT;\n",
		"<?php $x = <<<'EOT'\nraw $a<$b\nEOT;\n",
		"<?php $x = <<<EOT\n    indented close is allowed\n    EOT;\n",
	}
	for _, src := range cases {
		var heredoc string
		for _, tk := range Lex(src) {
			if tk.Kind == token.String && len(tk.Value) > 3 && tk.Value[:3] == "<<<" {
				heredoc = tk.Value
			}
		}
		if heredoc == "" {
			t.Errorf("heredoc not captured as a single String token in %q", src)
		}
		// lossless
		var got strings.Builder
		for _, tk := range Lex(src) {
			got.WriteString(tk.Value)
		}
		if got.String() != src {
			t.Errorf("not lossless\n src: %q\n got: %q", src, got.String())
		}
	}
}

func TestLexMultilineAttributeWithNowdoc(t *testing.T) {
	src := "<?php\n#[Deprecated(message: <<<'TXT'\nprose e.g. with. dots\nTXT)]\nclass A {}"
	var sawAttrStart bool
	var nowdoc string
	for _, tk := range Lex(src) {
		if tk.Kind == token.AttributeStart {
			sawAttrStart = true
		}
		if tk.Kind == token.String && strings.Contains(tk.Value, "prose e.g. with. dots") {
			nowdoc = tk.Value
		}
	}
	if !sawAttrStart {
		t.Fatal(`attribute opener "#[" not lexed as an AttributeStart token`)
	}
	// the nowdoc stays one opaque String token, so its body never leaks out as code
	if nowdoc == "" {
		t.Error("nowdoc body not captured as a single string token")
	}
	// lossless
	var got strings.Builder
	for _, tk := range Lex(src) {
		got.WriteString(tk.Value)
	}
	if got.String() != src {
		t.Errorf("not lossless:\n src: %q\n got: %q", src, got.String())
	}
}

func TestLexAttributeIsNative(t *testing.T) {
	toks := Lex("<?php #[Route('/x', methods: ['GET'])]\nclass C {}")
	start, end := -1, -1
	kinds := map[string]token.Kind{}
	for i, tk := range toks {
		switch tk.Kind {
		case token.AttributeStart:
			start = i
			if tk.Value != "#[" {
				t.Errorf("AttributeStart value = %q, want %q", tk.Value, "#[")
			}
		case token.AttributeEnd:
			end = i
			if tk.Value != "]" {
				t.Errorf("AttributeEnd value = %q, want %q", tk.Value, "]")
			}
		}
		kinds[tk.Value] = tk.Kind
	}
	if start < 0 || end < 0 || end < start {
		t.Fatalf("attribute delimiters not lexed: start=%d end=%d", start, end)
	}
	// the attribute's contents are ordinary tokens, visible to fixers
	if kinds["Route"] != token.Ident {
		t.Errorf("attribute name not lexed as Ident: %v", kinds["Route"])
	}
	if kinds["methods"] != token.Ident {
		t.Errorf("named argument not lexed natively: %v", kinds["methods"])
	}
	// the inner array "[" stays a plain Punct, only the outer "]" is AttributeEnd
	if kinds["["] != token.Punct {
		t.Errorf("inner array bracket kind = %v, want Punct", kinds["["])
	}
}

func TestLexKeywordsVsIdent(t *testing.T) {
	toks := Lex("<?php function foo() { return bar; }")
	kind := map[string]token.Kind{}
	for _, tk := range toks {
		if tk.Kind == token.Keyword || tk.Kind == token.Ident {
			kind[tk.Value] = tk.Kind
		}
	}
	if kind["function"] != token.Keyword || kind["return"] != token.Keyword {
		t.Errorf("keywords misclassified: %v", kind)
	}
	if kind["foo"] != token.Ident || kind["bar"] != token.Ident {
		t.Errorf("names misclassified: %v", kind)
	}
}

func TestKeywordLikeNamespaceSegmentIsIdent(t *testing.T) {
	// "Enum" here is a namespace segment, not the enum keyword
	for _, tk := range Lex(`<?php use App\Enum\Action;`) {
		if tk.Value == "Enum" && tk.Kind != token.Ident {
			t.Errorf("namespace segment Enum should be Ident, got %s", tk.Kind)
		}
	}
}

func TestKeywordAfterObjectOperatorIsIdent(t *testing.T) {
	// `list` is a keyword, but as a property name it is a plain identifier
	toks := Lex("<?php $this->list;")
	for _, tk := range toks {
		if tk.Value == "list" && tk.Kind != token.Ident {
			t.Errorf("property `list` after -> should be Ident, got %s", tk.Kind)
		}
	}
}

func TestLexMultiCharOperators(t *testing.T) {
	cases := map[string]string{
		"<?php $a === $b;": "===",
		"<?php $a=>$b;":    "=>",
		"<?php $a->b;":     "->",
		"<?php A::B;":      "::",
		"<?php $a ?? $b;":  "??",
		"<?php $a <=> $b;": "<=>",
		"<?php $a ??= $b;": "??=",
	}
	for src, op := range cases {
		found := false
		for _, tk := range Lex(src) {
			if tk.Kind == token.Punct && tk.Value == op {
				found = true
			}
		}
		if !found {
			t.Errorf("operator %q not tokenized as a single Punct in %q", op, src)
		}
	}
}
