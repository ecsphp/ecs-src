package rules

import (
	"testing"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

func lexStream(src string) *tokens.Stream { return tokens.New(lexer.Lex(src)) }

func firstBrace(s *tokens.Stream) int {
	for i := range s.Len() {
		if s.At(i).Kind == token.Punct && s.At(i).Value == "{" {
			return i
		}
	}
	return -1
}

func TestBracesPositionClassAndFunction(t *testing.T) {
	got, changed := apply(t, BracesPosition{}, "<?php class A {\n    public function run() {\n        return 1;\n    }\n}")
	want := "<?php class A\n{\n    public function run()\n    {\n        return 1;\n    }\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	if _, changed := apply(t, BracesPosition{}, want); changed {
		t.Fatal("correctly placed braces should not change (idempotent)")
	}
}

func TestBracesPositionControl(t *testing.T) {
	got, changed := apply(t, BracesPosition{}, "<?php if ($a)\n{\n}\nwhile ($b){\n}")
	want := "<?php if ($a) {\n}\nwhile ($b) {\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestBracesPositionClosureLeftAlone(t *testing.T) {
	if _, changed := apply(t, BracesPosition{}, "<?php $f = function () {\n};"); changed {
		t.Fatal("closure brace must not move")
	}
}

func TestClassifyBrace(t *testing.T) {
	cases := []struct {
		src  string
		want braceKind
	}{
		{"<?php class A {}", braceClassLike},
		{"<?php interface I {}", braceClassLike},
		{"<?php function foo() {}", braceFunctionDecl},
		{"<?php $f = function () {};", braceClosure},
		{"<?php function &gen() {}", braceFunctionDecl},
		{"<?php if ($a) {}", braceControl},
		{"<?php foreach ($x as $y) {}", braceControl},
		{"<?php function foo(): int {}", braceFunctionDecl},
		{"<?php class A extends B implements C {}", braceClassLike},
	}
	for _, c := range cases {
		s := lexStream(c.src)
		brace := firstBrace(s)
		if brace < 0 {
			t.Fatalf("no brace in %q", c.src)
		}
		if kind, _ := classifyBrace(s, brace); kind != c.want {
			t.Errorf("%q: classify = %d, want %d", c.src, kind, c.want)
		}
	}
}

// a trait adaptation block "use T { ... }" is not a class/function body
func TestClassifyTraitBlockIsOther(t *testing.T) {
	s := lexStream("<?php class A { use T { m as n; } }")
	// the SECOND "{" is the trait adaptation block
	var braces []int
	for i := range s.Len() {
		if s.At(i).Kind == token.Punct && s.At(i).Value == "{" {
			braces = append(braces, i)
		}
	}
	if len(braces) < 2 {
		t.Fatal("expected two braces")
	}
	if kind, _ := classifyBrace(s, braces[1]); kind != braceOther {
		t.Fatalf("trait block classify = %d, want braceOther", kind)
	}
}
