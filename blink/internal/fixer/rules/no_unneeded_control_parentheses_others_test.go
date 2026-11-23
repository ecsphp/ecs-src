package rules

import "testing"

func nucpAllStatements() fixerRule {
	return NoUnneededControlParentheses{}.WithConfig(map[string]any{"statements": []any{
		"break", "clone", "continue", "echo_print", "negative_instanceof", "others", "return", "switch_case", "yield", "yield_from",
	}}).(fixerRule)
}

// The expected output of every case comes from the PHP engine, with every statement configured.
func TestNoUnneededControlParenthesesAllStatements(t *testing.T) {
	t.Parallel()
	assertFix(t, nucpAllStatements(),
		"<?php\nfunction f(int $value, DateTimeInterface $date): float\n{\n    if ((($value) > 2) && true) {\n        $date = (DateTimeImmutable::createFromInterface($date))->modify('+1 day');\n    }\n\n    return ceil(($value / 12));\n}\n",
		"<?php\nfunction f(int $value, DateTimeInterface $date): float\n{\n    if (($value > 2) && true) {\n        $date = DateTimeImmutable::createFromInterface($date)->modify('+1 day');\n    }\n\n    return ceil($value / 12);\n}\n",
		true)
	assertFix(t, nucpAllStatements(), "?><?=($a)?>\n", "?><?= $a?>\n", true)

	cases := []struct{ src, want string }{
		// a variable between operators, in a sequence or before an access
		{"$a = ($b) + 1;", "$a = $b + 1;"},
		{"$a = 1 + ($b);", "$a = 1 + $b;"},
		{"$a = ($b) * ($c);", "$a = $b * $c;"},
		{"foo(($a) + 1);", "foo($a + 1);"},
		{"$x = [($a) => 1, 1 => ($b)];", "$x = [$a => 1, 1 => $b];"},
		{"if (($a) > 2) {}", "if ($a > 2) {}"},
		{"return ($a) + 1;", "return $a + 1;"},
		{"echo ($a) . 'x';", "echo $a . 'x';"},
		{"$a = ($b);", "$a = $b;"},
		{"$a = ($b)->c;", "$a = $b->c;"},
		{"$a = ($b)[0];", "$a = $b[0];"},
		{"$a = ($b)?->c;", "$a = $b?->c;"},
		{"$a = ($b)->c++;", "$a = $b->c++;"},
		{"$x = ($a)[0]--;", "$x = $a[0]--;"},
		{"($a)->b = 1;", "$a->b = 1;"},
		{"$x = ($a)->{'b'};", "$x = $a->{'b'};"},
		{"$x = ($a)?->b ?? $c;", "$x = $a?->b ?? $c;"},
		{"$x = ($a) & $b;", "$x = $a & $b;"},
		{"$x = ($a) and $b;", "$x = $a and $b;"},
		{"$a ?? ($b);", "$a ?? $b;"},
		{"$a .= ($b);", "$a .= $b;"},
		{"($a);", "$a;"},
		{"{ ($a); }", "{ $a; }"},
		{"if ($a) {} ($b);", "if ($a) {} $b;"},
		{"($a ? $b : $c);", "$a ? $b : $c;"},
		{"$x = match ($a) { ($b) => ($c), default => ($d) };", "$x = match ($a) { $b => $c, default => $d };"},
		{"$a->{($b)};", "$a->{$b};"},
		{"${($a)};", "${$a};"},
		{"[$a, $b] = ($c);", "[$a, $b] = $c;"},
		{"$x = $a[($b) + 1];", "$x = $a[$b + 1];"},
		{"$a = \"{$b[($c)]}\";", "$a = \"{$b[$c]}\";"},
		// the only argument of a call, and other sequence elements
		{"foo(($a), $b);", "foo($a, $b);"},
		{"foo($a, ($b));", "foo($a, $b);"},
		{"foo(($a + $b));", "foo($a + $b);"},
		{"foo((new Foo()));", "foo(new Foo());"},
		{"foo((clone $a));", "foo(clone $a);"},
		{"$o->m(($a));", "$o->m($a);"},
		{"new Foo(($a));", "new Foo($a);"},
		{"foo(...($a));", "foo(...$a);"},
		{"foo(\n    ($a),\n    ($b)\n);", "foo(\n    $a,\n    $b\n);"},
		{"$x = [(new Foo())];", "$x = [new Foo()];"},
		{"for (($i = 0); ($i < 10); ($i++)) {}", "for ($i = 0; $i < 10; $i++) {}"},
		{"$x = fn() => ($a + 1);", "$x = fn() => $a + 1;"},
		{"$f = fn &($a) => ($a);", "$f = fn &($a) => $a;"},
		// a call, a constant or a literal before an access
		{"(Foo::bar())->baz();", "Foo::bar()->baz();"},
		{"$x = (Foo::$bar)->baz;", "$x = Foo::$bar->baz;"},
		{"$x = (Foo::BAR)[0];", "$x = Foo::BAR[0];"},
		{"$x = (\\Foo\\Bar::baz())->qux();", "$x = \\Foo\\Bar::baz()->qux();"},
		{"$x = ($a->b())->c();", "$x = $a->b()->c();"},
		{"$x = ('abc')[0];", "$x = 'abc'[0];"},
		{"$x = ([1])[0];", "$x = [1][0];"},
		{"$x = (static::$a);", "$x = static::$a;"},
		{"$x = ($$a);", "$x = $$a;"},
		{"$x = (__DIR__) . '/x';", "$x = __DIR__ . '/x';"},
		// unary operators and casts
		{"$a = -($b);", "$a = -$b;"},
		{"$a = !($b) && $c;", "$a = !$b && $c;"},
		{"$a = @($b);", "$a = @$b;"},
		{"$x = - ($a);", "$x = - $a;"},
		{"$x = -($a)+1;", "$x = -$a +1;"},
		{"$x = !($a)->b && $c;", "$x = !$a->b && $c;"},
		{"$x = @(foo());", "$x = @foo();"},
		{"$a = (int) ($b) + 1;", "$a = (int) $b + 1;"},
		{"$a = (int) (($b));", "$a = (int) $b;"},
		{"$a = (int)($b)->c;", "$a = (int)$b->c;"},
		{"$x = ((int)($a));", "$x = ((int)$a);"},
		{"foo(((int) $b));", "foo((int) $b);"},
		// instanceof
		{"if (!($a instanceof B)) {}", "if (!$a instanceof B) {}"},
		{"$x = !($a instanceof B);", "$x = !$a instanceof B;"},
		{"$x = ($a instanceof B);", "$x = $a instanceof B;"},
		{"$x = ($a->b instanceof C);", "$x = $a->b instanceof C;"},
		{"$x = (($a instanceof B));", "$x = ($a instanceof B);"},
		{"$x = !(($a instanceof B));", "$x = !($a instanceof B);"},
		{"$x = fn () => ($a instanceof B);", "$x = fn () => $a instanceof B;"},
		{"for (; ($a instanceof B); ) {}", "for (; $a instanceof B; ) {}"},
		// control statements
		{"return($a);", "return $a;"},
		{"print($a);", "print $a;"},
		{"foo(print ($a));", "foo(print $a);"},
		{"function g() { foo(yield ($a)); }", "function g() { foo(yield $a); }"},
		{"function g() { yield from ($a); }", "function g() { yield from $a; }"},
		{"throw ($e);", "throw $e;"},
		{"throw($e);", "throw $e;"},
		{"while (1) { break (1); }", "while (1) { break 1; }"},
		{"switch ($a) { case ($b); }", "switch ($a) { case $b; }"},
		{"switch ($a) { case($b): break; }", "switch ($a) { case $b: break; }"},
		{"switch ($a): case ($b): break; endswitch;", "switch ($a): case $b: break; endswitch;"},
		{"return (new Foo());", "return new Foo();"},
		{"return (clone $x);", "return clone $x;"},
		{"exit();", "exit;"},
		{"exit ( );", "exit  ;"},
		{"$a or exit();", "$a or exit;"},
		// nested pairs, one pass
		{"$x = ((($a)));", "$x = $a;"},
		{"$x = ((($a) + 1));", "$x = ($a + 1);"},
		{"while ((($a))) {}", "while ($a) {}"},
		{"$x = $a ? (($b)) : 1;", "$x = $a ? ($b) : 1;"},
		{"$x = (($a) && ($b)) || ($c);", "$x = ($a && $b) || $c;"},
		{"$x = $a && (!($b));", "$x = $a && (!$b);"},
		{"$a = ((new Foo()));", "$a = (new Foo());"},
		// spacing
		{"$a=($b)+1;", "$a= $b +1;"},
		{"$a = ( $b );", "$a =  $b ;"},
		{"$a = (/* c */ $b);", "$a = /* c */ $b;"},
		{"$a = ($b)// c\n;", "$a = $b// c\n;"},
		// kept
		{"$a = ($b)::c();", "$a = ($b)::c();"},
		{"$x = ($a)::B;", "$x = ($a)::B;"},
		{"$x = ($a)::$b::$c;", "$x = ($a)::$b::$c;"},
		{"$a = ($b)();", "$a = ($b)();"},
		{"$x = (foo())();", "$x = (foo())();"},
		{"(function () {})();", "(function () {})();"},
		{"($a) ? $b : $c;", "($a) ? $b : $c;"},
		{"$a ? ($b) : $c;", "$a ? ($b) : $c;"},
		{"$a ?: ($b);", "$a ?: ($b);"},
		{"return ($a) ?: $b;", "return ($a) ?: $b;"},
		{"$x = $a ? ($b ? 1 : 2) : 3;", "$x = $a ? ($b ? 1 : 2) : 3;"},
		{"$x = ($a) instanceof B;", "$x = ($a) instanceof B;"},
		{"$x = ($a) |> f(...);", "$x = ($a) |> f(...);"},
		{"echo($a), ($b);", "echo($a), ($b);"},
		{"echo ($a)->b, ($c)->d;", "echo ($a)->b, ($c)->d;"},
		{"$a->{'b'}($c);", "$a->{'b'}($c);"},
		{"if ($a): ($b)(); endif;", "if ($a): ($b)(); endif;"},
		{"(new Foo())->bar();", "(new Foo())->bar();"},
		{"(new Foo)->bar();", "(new Foo)->bar();"},
		{"$a = (new Foo());", "$a = (new Foo());"},
		{"$x = (new Foo)['x'];", "$x = (new Foo)['x'];"},
		{"return (new Foo())->bar();", "return (new Foo())->bar();"},
		{"$a = new ($b);", "$a = new ($b);"},
		{"(clone $x)->y;", "(clone $x)->y;"},
		{"$a = (clone $x)->y;", "$a = (clone $x)->y;"},
		{"$a = (clone $x);", "$a = (clone $x);"},
		{"$x = ($a ?? $b)->c;", "$x = ($a ?? $b)->c;"},
		{"$x = ($a = $b);", "$x = ($a = $b);"},
		{"$x = (-$a) ** 2;", "$x = (-$a) ** 2;"},
		{"$x = (@$a);", "$x = (@$a);"},
		{"$x = (!$a);", "$x = (!$a);"},
		{"$a = ((int) $b) + 1;", "$a = ((int) $b) + 1;"},
		{"$a = (string) ($b . 'x');", "$a = (string) ($b . 'x');"},
		{"$x = ($a instanceof B) && $c;", "$x = ($a instanceof B) && $c;"},
		{"if (($a instanceof B) && $c) {}", "if (($a instanceof B) && $c) {}"},
		{"$x = (!$a instanceof B);", "$x = (!$a instanceof B);"},
		{"if (!($a instanceof B) || !($c instanceof D)) {}", "if (!($a instanceof B) || !($c instanceof D)) {}"},
		{"$x = ($a::class);", "$x = ($a::class);"},
		{"$x = (namespace\\FOO);", "$x = (namespace\\FOO);"},
		{"$x = (namespace\\bar())->baz;", "$x = (namespace\\bar())->baz;"},
		{"$x = (array(1))[0];", "$x = (array(1))[0];"},
		{"$a = (\"x$y\") . 'z';", "$a = (\"x$y\") . 'z';"},
		{"$a = (<<<EOT\nx\nEOT);", "$a = (<<<EOT\nx\nEOT);"},
		{"require ($a);", "require ($a);"},
		{"require_once($a);", "require_once($a);"},
		{"exit(1);", "exit(1);"},
	}
	for _, c := range cases {
		got, changed := apply(t, nucpAllStatements(), "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}

// PHP-CS-Fixer's default leaves out others, negative_instanceof and yield_from.
func TestNoUnneededControlParenthesesDefaultStatements(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"$a = ($b) + 1;", "$a = ($b) + 1;"},
		{"if (!($a instanceof B)) {}", "if (!($a instanceof B)) {}"},
		{"function g() { yield from ($a); }", "function g() { yield from ($a); }"},
		{"function g() { yield ($a); }", "function g() { yield $a; }"},
		{"return ($a) + 1;", "return $a + 1;"},
		{"throw ($e);", "throw ($e);"},
		{"echo ($a), ($b);", "echo ($a), ($b);"},
		{"switch ($a) { case ($b); }", "switch ($a) { case $b; }"},
		{"exit();", "exit();"},
	}
	for _, c := range cases {
		got, changed := apply(t, NoUnneededControlParentheses{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
