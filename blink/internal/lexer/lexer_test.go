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
		"<?php $a === $b;":    "===",
		"<?php $a=>$b;":       "=>",
		"<?php $a->b;":        "->",
		"<?php A::B;":         "::",
		"<?php $a ?? $b;":     "??",
		"<?php $a <=> $b;":    "<=>",
		"<?php $a ??= $b;":    "??=",
		"<?php $a |> f(...);": "|>",
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

func TestLexAsymmetricVisibilityIsOneKeyword(t *testing.T) {
	cases := map[string]string{
		"<?php class A { public private(set) int $a; }":                       "private(set)",
		"<?php class A { public function __construct(protected(set) $a) {} }": "protected(set)",
		"<?php class A { PUBLIC(Set) int $a; }":                               "PUBLIC(Set)",
	}
	for src, want := range cases {
		found := false
		for _, tk := range Lex(src) {
			if tk.Value == want {
				found = tk.Kind == token.Keyword
			}
		}
		if !found {
			t.Errorf("%q not lexed as one Keyword in %q", want, src)
		}
	}
	// "private (set)" with a space, or a method call, is not the modifier
	for _, src := range []string{"<?php private (set);", "<?php $o->private(set);", "<?php private(settings);"} {
		for _, tk := range Lex(src) {
			if strings.Contains(tk.Value, "(") && tk.Kind != token.Punct {
				t.Errorf("%q lexed as %s in %q", tk.Value, tk.Kind, src)
			}
		}
	}
}

func TestReservedWordInNamePositionIsIdent(t *testing.T) {
	cases := []struct {
		src, word string
		want      token.Kind
	}{
		// enum case names
		{"<?php enum E: string { case Else = 'e'; }", "Else", token.Ident},
		{"<?php enum E { case Default; }", "Default", token.Ident},
		{"<?php enum E { case List /* c */ = 'l'; }", "List", token.Ident},
		{"<?php switch ($a) { case static::X: }", "static", token.Keyword},
		{"<?php switch ($a) { case new A(): }", "new", token.Keyword},
		// named arguments
		{"<?php f(else: 0);", "else", token.Ident},
		{"<?php f($a, array : 1);", "array", token.Ident},
		{"<?php #[A(class: B::class)] function f() {}", "class", token.Ident},
		{"<?php f(static::X);", "static", token.Keyword},
		{"<?php if ($a): else: endif;", "else", token.Keyword},
		// "enum" is the keyword only before a name, as PHP's scanner reads it
		{"<?php enum Suit: string {}", "enum", token.Keyword},
		{"<?php enum /* c */ Suit {}", "enum", token.Keyword},
		{"<?php $a = Enum::from(1);", "Enum", token.Ident},
		{"<?php $a = new Enum($b);", "Enum", token.Ident},
		{"<?php class enum extends B {}", "enum", token.Ident},
		// constant names: in a list, typed, and the type itself
		{"<?php class A { const A = 1, NEW = 2; }", "NEW", token.Ident},
		{"<?php class A { const string DEFAULT = 'x'; }", "DEFAULT", token.Ident},
		{"<?php class A { const int|string AS = 1; }", "AS", token.Ident},
		{"<?php class A { const array LIST = []; }", "array", token.Keyword},
		{"<?php class A { const array LIST = []; }", "LIST", token.Ident},
		// token_get_all(TOKEN_PARSE) keeps the keyword when the value spells it
		{"<?php class A { const PUBLIC = B::PUBLIC; }", "PUBLIC", token.Keyword},
		{"<?php class A { const PUBLIC = B::public; }", "PUBLIC", token.Ident},
		{"<?php class A { const NEW = 1, X = B::NEW; }", "X", token.Ident},
		{"<?php enum E { case Default = B::Default; }", "Default", token.Ident},
		// trait method aliases and the methods they rename
		{"<?php class C { use T { foo as protected list; } }", "list", token.Ident},
		{"<?php class C { use T { foo as final static; } }", "static", token.Ident},
		{"<?php class C { use T { foo as new; } }", "new", token.Ident},
		{"<?php class C { use T { foo as protected; } }", "protected", token.Keyword},
		{"<?php class C { use T { list as protected; } }", "list", token.Ident},
		{"<?php class C { use T { and as protected or; } }", "and", token.Ident},
		{"<?php foreach ($a as list($b)) {}", "list", token.Keyword},
		// a method returning by reference
		{"<?php class C { public function &and() {} }", "and", token.Ident},
		{"<?php $f = function &() use ($list) {};", "use", token.Keyword},
		// a function named readonly, and the modifier before a DNF type
		{"<?php $a = readonly('x');", "readonly", token.Ident},
		{"<?php class A { readonly (A&B)|C $x; }", "readonly", token.Keyword},
		{"<?php class A { public function __construct(readonly int $a) {} }", "readonly", token.Keyword},
	}
	for _, c := range cases {
		// the first token spelling the word is the one under test
		found := false
		for _, tk := range Lex(c.src) {
			if tk.Value == c.word && (tk.Kind == token.Keyword || tk.Kind == token.Ident) {
				found = true
				if tk.Kind != c.want {
					t.Errorf("%q in %q: got %s, want %s", c.word, c.src, tk.Kind, c.want)
				}
				break
			}
		}
		if !found {
			t.Errorf("%q not found in %q", c.word, c.src)
		}
	}
}

// PHP's scanner reads the full open tag in any case, followed by whitespace or
// the end of the file; otherwise "<?" is the short open tag.
func TestLexOpenTagAnyCase(t *testing.T) {
	cases := []struct{ src, tag, next string }{
		{"<?PHP\nfoo();", "<?PHP", "\n"},
		{"<?PhP echo 1;", "<?PhP", " "},
		{"<?PHP", "<?PHP", ""},
		{"<?php$x = 1;", "<?", "php"},
		{"<?PHPx ?>", "<?", "PHPx"},
	}
	for _, c := range cases {
		toks := Lex(c.src)
		if toks[0].Kind != token.OpenTag || toks[0].Value != c.tag {
			t.Errorf("%q: open tag %s %q, want %q", c.src, toks[0].Kind, toks[0].Value, c.tag)
			continue
		}
		next := ""
		if len(toks) > 1 {
			next = toks[1].Value
		}
		if next != c.next {
			t.Errorf("%q: after the tag %q, want %q", c.src, next, c.next)
		}
	}
	// no PHP code is lost to inline HTML
	if toks := Lex("<?PHP\nfoo();"); toks[len(toks)-1].Kind != token.Punct {
		t.Errorf("uppercase tag: code lexed as %s", toks[len(toks)-1].Kind)
	}
}

// PHP's scanner reads "/**" as a doc comment only when whitespace follows it.
func TestDocCommentNeedsWhitespaceAfterOpening(t *testing.T) {
	cases := map[string]token.Kind{
		"/** doc */":       token.DocComment,
		"/**\n * doc\n */": token.DocComment,
		"/**\tdoc */":      token.DocComment,
		"/**@var \\X*/":    token.Comment,
		"/***/":            token.Comment,
		"/***x */":         token.Comment,
		"/**/":             token.Comment,
	}
	for c, want := range cases {
		toks := Lex("<?php " + c)
		if got := toks[len(toks)-1]; got.Value != c || got.Kind != want {
			t.Errorf("%q: got %s %q, want %s", c, got.Kind, got.Value, want)
		}
	}
}
