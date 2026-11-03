package rules

import "testing"

// TernaryOperatorSpaces spaces every "?" and ":" that PHP-CS-Fixer's tokenizer
// leaves as a plain ternary token: a return-type colon, a nullable "?", a named
// argument, a switch case, a goto label or an alternative-syntax colon is not one.
func TestTernaryOperatorSpacesTokenContexts(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, in, want string }{
		{"plain", "<?php $a?$b:$c;", "<?php $a ? $b : $c;"},
		{"extra spaces", "<?php $a  ?  $b  :  $c;", "<?php $a ? $b : $c;"},
		{"elvis", "<?php $a?:$b;", "<?php $a ?: $b;"},
		{"elvis spaced", "<?php $a ? : $b;", "<?php $a ?: $b;"},
		{"nested", "<?php $a?$b?1:2:($c?3:4);", "<?php $a ? $b ? 1 : 2 : ($c ? 3 : 4);"},
		{"multiline kept", "<?php $a\n    ? $b\n    : $c;", "<?php $a\n    ? $b\n    : $c;"},
		{"coalesce untouched", "<?php $a??$b; $a??=$b; $a?->b;", "<?php $a??$b; $a??=$b; $a?->b;"},
		{
			"static arrow fn return type in ternary",
			"<?php $f = null !== $p ? static fn (mixed $x): array => [$x] : null;",
			"<?php $f = null !== $p ? static fn (mixed $x): array => [$x] : null;",
		},
		{
			"arrow fn return type in unspaced ternary",
			"<?php $f = $p?fn ($x): int => 1:null;",
			"<?php $f = $p ? fn ($x): int => 1 : null;",
		},
		{
			"closure use return type in ternary",
			"<?php $f = $p ? function () use ($q): ?int { return $q; } : null;",
			"<?php $f = $p ? function () use ($q): ?int { return $q; } : null;",
		},
		{
			"by-ref arrow fn nullable return type",
			"<?php $f = $p ? fn &(): ?array => $x : null;",
			"<?php $f = $p ? fn &(): ?array => $x : null;",
		},
		{"enum backed type", "<?php enum Suit: string {}\nenum Mode:int {}", "<?php enum Suit: string {}\nenum Mode:int {}"},
		{"named args", "<?php f(a: 1, b: $x?1:2);", "<?php f(a: 1, b: $x ? 1 : 2);"},
		{"keyword named args", "<?php f(default: [], new: $b, array: $c);", "<?php f(default: [], new: $b, array: $c);"},
		{"keyword named arg in attribute", "<?php #[A(class: B::class)]\nclass C {}", "<?php #[A(class: B::class)]\nclass C {}"},
		{"match arm", "<?php match (true) { $a?1:2 => 3, default => 4 };", "<?php match (true) { $a ? 1 : 2 => 3, default => 4 };"},
		{
			"switch case",
			"<?php switch ($a) { case 1: case X: break; case $b?1:2: default: }",
			"<?php switch ($a) { case 1: case X: break; case $b ? 1 : 2: default: }",
		},
		{
			"brace switch nested in alternative switch",
			"<?php switch ($a):\ncase 1:\nswitch ($b) { case 2: break; }\nbreak;\ncase 3:\nbreak;\nendswitch;",
			"<?php switch ($a):\ncase 1:\nswitch ($b) { case 2: break; }\nbreak;\ncase 3:\nbreak;\nendswitch;",
		},
		{
			"enum and match cases inside a switch",
			"<?php switch ($a) { case 1: $x = match ($b) { default => 1 }; break; default: }\nenum E { case A; }",
			"<?php switch ($a) { case 1: $x = match ($b) { default => 1 }; break; default: }\nenum E { case A; }",
		},
		{
			"reserved-word function name before return type",
			"<?php class A { function &list(): ?int { return $x; } public function &print(): int {} }",
			"<?php class A { function &list(): ?int { return $x; } public function &print(): int {} }",
		},
		{"keyword function name", "<?php class A { function list(): ?int {} }", "<?php class A { function list(): ?int {} }"},
		{"print call in ternary", "<?php $a?print($b):$c;", "<?php $a ? print($b) : $c;"},
		{"goto label", "<?php a: goto a;", "<?php a: goto a;"},
		{
			"alternative syntax",
			"<?php if ($a): elseif ($b): else: endif; foreach ($c as $d): endforeach;",
			"<?php if ($a): elseif ($b): else: endif; foreach ($c as $d): endforeach;",
		},
		{"attribute before nullable param", "<?php function f(#[A] ?int $x) {}", "<?php function f(#[A] ?int $x) {}"},
		{"nullable property", "<?php class A { public ?int $a; private static ?A $b; }", "<?php class A { public ?int $a; private static ?A $b; }"},
		{"asymmetric visibility", "<?php class A { public private(set) ?int $a; }", "<?php class A { public private(set) ?int $a; }"},
		{"comment before colon kept", "<?php $a ? $b /* c */  : $c;", "<?php $a ? $b /* c */  : $c;"},
		{"static instanceof", "<?php $a instanceof static?1:2;", "<?php $a instanceof static ? 1 : 2;"},
	}
	for _, c := range cases {
		got, _ := apply(t, TernaryOperatorSpaces{}, c.in)
		if got != c.want {
			t.Errorf("%s\n  in: %q\n got: %q\nwant: %q", c.name, c.in, got, c.want)
		}
	}
}
