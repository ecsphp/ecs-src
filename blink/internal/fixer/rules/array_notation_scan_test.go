package rules

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// arrayNotationScanAlphabet is what the random streams below are made of:
// brackets (balanced or not), the tokens the destructuring checks look for,
// and the filler they skip.
var arrayNotationScanAlphabet = []token.Token{
	{Kind: token.Punct, Value: "("}, {Kind: token.Punct, Value: ")"},
	{Kind: token.Punct, Value: "["}, {Kind: token.Punct, Value: "]"},
	{Kind: token.Punct, Value: "{"}, {Kind: token.Punct, Value: "}"},
	{Kind: token.Punct, Value: "["}, {Kind: token.Punct, Value: "]"},
	{Kind: token.Punct, Value: "="}, {Kind: token.Punct, Value: "=>"},
	{Kind: token.Punct, Value: ","}, {Kind: token.Punct, Value: ";"},
	{Kind: token.Punct, Value: "..."},
	{Kind: token.Keyword, Value: "as"}, {Kind: token.Ident, Value: "AS"},
	{Kind: token.Keyword, Value: "foreach"}, {Kind: token.Keyword, Value: "list"},
	{Kind: token.Keyword, Value: "array"}, {Kind: token.Keyword, Value: "function"},
	{Kind: token.Keyword, Value: "match"}, {Kind: token.Keyword, Value: "object"},
	{Kind: token.Ident, Value: "foo"}, {Kind: token.Variable, Value: "$a"},
	{Kind: token.String, Value: "'s'"}, {Kind: token.Number, Value: "1"},
	{Kind: token.Whitespace, Value: " "}, {Kind: token.Whitespace, Value: "\n"},
	{Kind: token.Whitespace, Value: "\n"}, {Kind: token.Comment, Value: "/* c */"},
	{Kind: token.AttributeStart, Value: "#["}, {Kind: token.AttributeEnd, Value: "]"},
}

func randomArrayNotationStream(r *rand.Rand) *tokens.Stream {
	n := 1 + r.IntN(60)
	toks := make([]token.Token, n)
	for i := range toks {
		toks[i] = arrayNotationScanAlphabet[r.IntN(len(arrayNotationScanAlphabet))]
	}
	return tokens.New(toks)
}

var arrayNotationScanSources = []string{
	"<?php foreach ($rows as ['id' => $id, 'tags' => [$a, $b]]) { $x = [$id => [$a]]; }",
	"<?php foreach ($rows as $k => [$a, [$b, $c]]) {}",
	"<?php foreach (f([1, 2]) as $k => $v) { [$a, $b] = $v; }",
	"<?php [[$a, $b], [$c]] = $x; list($a, [$b]) = $y; $z = [[1], 2];",
	"<?php $a[0][1] = [1]; $b = (object) ['k' => [1]]; f()[0] = [[2]];",
	"<?php $m = match ($x) { 1 => [1, 2], default => [[3]] }; fn ($a) => [$a];",
	"<?php class C { const MAP = ['a' => ['x', 'y'], 'b' => ['z']]; function f(array $a = []) { foreach ($a as [, $b]) {} } }",
	"<?php foreach ($a as $b) [$c] = $b; foreach ($a AS list($b, [$c])) {}",
}

// arrayNotationScanNear are the bounds the scans are tested with: always the
// forward pass, a mix of both, and the default (only backward scans here).
var arrayNotationScanNear = []int{0, 3, 256}

func newArrayNotationScanNear(s *tokens.Stream, near int) *arrayNotationScan {
	sc := newArrayNotationScan(s)
	sc.near = near
	return sc
}

// checkArrayNotationScan compares arrayNotationScan and arrayNotationTracker
// with the backward scans they replace, at every index of s.
func checkArrayNotationScan(t *testing.T, r *rand.Rand, s *tokens.Stream, label string) {
	t.Helper()
	for _, near := range arrayNotationScanNear {
		checkArrayNotationScanNear(t, r, s, newArrayNotationScanNear(s, near), label)
	}
	var tr arrayNotationTracker
	for i := range s.Len() {
		if got, _ := tr.top(); got != arrayNotationEnclosing(s, i) {
			t.Fatalf("%s: tracker enclosing at %d = %d, want %d", label, i, got, arrayNotationEnclosing(s, i))
		}
		if got, want := tr.inForeachHead(s), arrayNotationInForeachHead(s, i); got != want {
			t.Fatalf("%s: tracker inForeachHead at %d = %v, want %v", label, i, got, want)
		}
		if got, want := isDestructuringAssignOpenIn(s, i, &tr), isDestructuringAssignOpen(s, i); got != want {
			t.Fatalf("%s: tracker isDestructuringAssignOpen at %d = %v, want %v", label, i, got, want)
		}
		tr.step(s.At(i), i)
	}
}

func checkArrayNotationScanNear(t *testing.T, r *rand.Rand, s *tokens.Stream, sc *arrayNotationScan, label string) {
	t.Helper()
	for i := range s.Len() {
		if got, want := sc.enclosing(i), arrayNotationEnclosing(s, i); got != want {
			t.Fatalf("%s: enclosing(%d) = %d, want %d (near %d)", label, i, got, want, sc.near)
		}
		if got, want := sc.inForeachHead(i), arrayNotationInForeachHead(s, i); got != want {
			t.Fatalf("%s: inForeachHead(%d) = %v, want %v (near %d)", label, i, got, want, sc.near)
		}
	}
	// in random order, so that the memo is filled from any side
	for _, i := range r.Perm(s.Len()) {
		if got, want := sc.isDestructuring(i), arrayNotationIsDestructuring(s, i); got != want {
			t.Fatalf("%s: isDestructuring(%d) = %v, want %v (near %d)", label, i, got, want, sc.near)
		}
	}
}

// renderTokens shows token boundaries, which Render would hide.
func renderTokens(s *tokens.Stream) string {
	var b strings.Builder
	for i := range s.Len() {
		b.WriteString(s.At(i).Value)
		b.WriteByte('|')
	}
	return b.String()
}

// checkArrayNotationScanEdits runs the rules that keep one scan while they edit
// the stream, with and without the scan, and wants the same result.
func checkArrayNotationScanEdits(t *testing.T, toks []token.Token, label string) {
	t.Helper()
	tcms := []TrailingCommaInMultiline{
		{},
		{Elements: []string{"arrays", "arguments", "parameters"}},
		{Elements: []string{"arrays", "array_destructuring", "arguments", "parameters", "match"}},
		{Elements: []string{"array_destructuring"}, AfterHeredoc: true},
	}
	for _, near := range arrayNotationScanNear {
		for k, f := range tcms {
			a, b := tokens.New(append([]token.Token{}, toks...)), tokens.New(append([]token.Token{}, toks...))
			ca, cb := f.fix(a, newArrayNotationScanNear(a, near)), f.fix(b, nil)
			if ca != cb || renderTokens(a) != renderTokens(b) {
				t.Fatalf("%s: TrailingCommaInMultiline #%d differs with the scan (near %d):\n got %v %s\nwant %v %s", label, k, near, ca, renderTokens(a), cb, renderTokens(b))
			}
		}
		for _, destructuring := range []bool{false, true} {
			a, b := tokens.New(append([]token.Token{}, toks...)), tokens.New(append([]token.Token{}, toks...))
			ca, cb := arrayNotationToLongWith(a, destructuring, newArrayNotationScanNear(a, near)), arrayNotationToLongWith(b, destructuring, nil)
			if ca != cb || renderTokens(a) != renderTokens(b) {
				t.Fatalf("%s: arrayNotationToLong(%v) differs with the scan (near %d):\n got %v %s\nwant %v %s", label, destructuring, near, ca, renderTokens(a), cb, renderTokens(b))
			}
		}
	}
}

func TestArrayNotationScanMatchesBackwardScans(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(1, 2))
	for _, src := range arrayNotationScanSources {
		toks := lexer.Lex(src)
		checkArrayNotationScan(t, r, tokens.New(toks), src)
		checkArrayNotationScanEdits(t, toks, src)
	}
	for range 20000 {
		s := randomArrayNotationStream(r)
		label := "random " + renderTokens(s)
		checkArrayNotationScan(t, r, s, label)
		checkArrayNotationScanEdits(t, s.Tokens(), label)
	}
}
