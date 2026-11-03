package rules

import "testing"

func TestPhpdocTypesOrderConfig(t *testing.T) {
	t.Parallel()
	doc := func(typ string) string { return "<?php\n/**\n * @param " + typ + " $x\n */\nfunction f($x) {}" }
	cfg := func(config map[string]any) fixerRule { return PhpdocTypesOrder{}.WithConfig(config).(fixerRule) }

	// defaults (sort_algorithm alpha, null_adjustment always_first), also for
	// an empty config: sorted at every level, null first
	for _, f := range []fixerRule{PhpdocTypesOrder{}, cfg(map[string]any{})} {
		assertFix(t, f, doc("string|null|Bar"), doc("null|Bar|string"), true)
		assertFix(t, f, doc("array<string|int, Foo|null>|false"), doc("array<int|string, null|Foo>|false"), true)
	}

	// a partial config keeps the other option's default
	assertFix(t, cfg(map[string]any{"null_adjustment": "always_last"}), doc("string|null|Bar"), doc("Bar|string|null"), true)
	assertFix(t, cfg(map[string]any{"sort_algorithm": "none"}), doc("string|null|Bar"), doc("null|string|Bar"), true)

	// no adjustment: null sorts like any other type
	assertFix(t, cfg(map[string]any{"null_adjustment": "none"}), doc("string|null|Bar"), doc("Bar|null|string"), true)

	// case_sensitive: uppercase sorts before lowercase
	assertFix(t, cfg(map[string]any{"case_sensitive": true}), doc("b|A|a"), doc("A|a|b"), true)

	// a leading backslash or nullable mark does not count
	assertFix(t, cfg(map[string]any{}), doc("\\Zed|Alpha|?Beta"), doc("Alpha|?Beta|\\Zed"), true)

	// spaces around the glue are dropped even when the order is kept, as in PHP
	assertFix(t, cfg(map[string]any{"null_adjustment": "always_last", "sort_algorithm": "none"}), doc("int | null"), doc("int|null"), true)

	// nested callables and array shapes are sorted too
	assertFix(t, cfg(map[string]any{}),
		doc("callable(string|int): void|array{a: string|int}"),
		doc("array{a: int|string}|callable(int|string): void"), true)

	// @method parameter types too
	assertFix(t, cfg(map[string]any{"null_adjustment": "always_last"}),
		"<?php\n/**\n * @method null|int find(null|string $id)\n */\nclass A {}",
		"<?php\n/**\n * @method int|null find(string|null $id)\n */\nclass A {}", true)

	// conditional types
	assertFix(t, cfg(map[string]any{}), doc("($y is int ? string|int : null)"), doc("($y is int ? int|string : null)"), true)

	// a type that does not parse is left alone
	broken := doc("string|null|int<")
	assertFix(t, cfg(map[string]any{}), broken, broken, false)
}

// Inputs and outputs checked against PHP-CS-Fixer 3.95.
func TestPhpdocTypesOrderPHPParity(t *testing.T) {
	t.Parallel()
	doc := func(typ string) string { return "<?php\n/**\n * @param " + typ + " $x\n */\nfunction f($x) {}" }
	method := func(sig string) string { return "<?php\n/**\n * @method " + sig + "\n */\nclass A {}" }
	def := configuredFromJSON(t, PhpdocTypesOrder{}, `{}`)
	symfony := configuredFromJSON(t, PhpdocTypesOrder{}, `{"null_adjustment": "always_last", "sort_algorithm": "none"}`)
	caseSensitive := configuredFromJSON(t, PhpdocTypesOrder{}, `{"case_sensitive": true}`)
	for _, c := range []struct {
		f       fixerRule
		in, out string
	}{
		// | and & mixed without parentheses: each "&"-joined run is one member
		// of the union, which keeps the spaces around its "&" unless re-sorted
		{def, doc("B&A|null"), doc("null|A&B")},
		{def, doc("C|B&A"), doc("A&B|C")},
		{def, doc("B&A|D&C|null"), doc("null|A&B|C&D")},
		{def, doc("B | A & C"), doc("A & C|B")},
		{symfony, doc("B | A&C"), doc("B|A&C")},
		{symfony, doc("B & A | C"), doc("B & A|C")},
		{symfony, doc("null | A&B"), doc("A&B|null")},

		// case_sensitive compares with PHP's <=>: numeric strings as numbers
		{caseSensitive, doc("10|9|null"), doc("null|9|10")},
		{caseSensitive, doc("010|9"), doc("9|010")},
		{caseSensitive, doc("-1|-2"), doc("-2|-1")},
		{caseSensitive, doc("b|10|A|9"), doc("9|10|A|b")},
		{caseSensitive, doc(`"10"|"9"`), doc(`"10"|"9"`)},

		// number literals of every format
		{def, doc("0x1F|null|5"), doc("null|0x1F|5")},
		{def, doc(".5|1.|null"), doc("null|.5|1.")},
		{def, doc("1_000|null"), doc("null|1_000")},
		{def, doc("1e3|100"), doc("100|1e3")},
		{def, doc("1.5|-.5|null"), doc("null|-.5|1.5")},
		{def, doc("1e+3|.5e-2|null"), doc("null|.5e-2|1e+3")},
		{def, doc("0XAB|0b11|0o7|null"), doc("null|0b11|0o7|0XAB")},
		{def, doc("array{1_000: B|A, 0x1: D|C}"), doc("array{1_000: A|B, 0x1: C|D}")},

		// covariant/contravariant prefixes; PHP registers no inner types after
		// one, and fails to parse a prefixed nullable or a prefix not in
		// lowercase (the annotation is then left alone)
		{def, doc("covariant B|A"), doc("A|covariant B")},
		{def, doc("contravariant B|A"), doc("A|contravariant B")},
		{def, doc("covariant B[]|A"), doc("A|covariant B[]")},
		{def, doc("covariant array<D|C>|A"), doc("A|covariant array<D|C>")},
		{def, doc("covariant|B|A"), doc("A|B|covariant")},
		{def, doc("covariant ?B|A"), doc("covariant ?B|A")},
		{def, doc("Covariant B|A"), doc("Covariant B|A")},

		// callable templates, with bounds and defaults
		{def, doc(`\Closure<T>(B|A): (D|C)`), doc(`\Closure<T>(A|B): (C|D)`)},
		{def, doc(`\Closure<T of B|A = D|C>(int): void`), doc(`\Closure<T of A|B = C|D>(int): void`)},
		{def, doc(`\Closure<T, U = D|C>(T): U`), doc(`\Closure<T, U = C|D>(T): U`)},
		{def, doc("Foo<T>(int)|B"), doc("B|Foo<T>(int)")},
		{def, method("B|A foo<T>(D|C $a)"), method("A|B foo<T>(C|D $a)")},
		{def, method("B|A foo<T of D|C>(D|C $a): void"), method("A|B foo<T of C|D>(C|D $a): void")},

		// \h and \s are Unicode (a no-break space here)
		{def, doc("array<int,\u00a0B|A>"), doc("array<int,\u00a0A|B>")},
		{def, doc("B[]\u00a0|\u00a0A"), doc("A|B[]")},
		{def, doc("(B|A)\u00a0|null"), doc("null|(A|B)")},
		{def, "<?php\n/**\n * @param\u00a0B|A $x\n */\nfunction f($x) {}", "<?php\n/**\n * @param\u00a0A|B $x\n */\nfunction f($x) {}"},
		// ... and identifier characters: when no type follows the spaces after a
		// glue, PCRE gives them back and the last one is a type of its own
		{def, "<?php\n/**\n * @return B&\u2000A&\u2000$x\n */\nfunction f() {}", "<?php\n/**\n * @return A&B&\u2000$x\n */\nfunction f() {}"},
	} {
		assertFix(t, c.f, c.in, c.out, c.in != c.out)
	}
}
