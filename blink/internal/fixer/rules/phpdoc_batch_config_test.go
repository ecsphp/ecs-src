package rules

import "testing"

func TestPhpdocNoAliasTagConfig(t *testing.T) {
	t.Parallel()
	// default rewrites @type -> @var
	got, changed := apply(t, PhpdocNoAliasTag{}, "<?php\n/**\n * @type int $x\n */\n")
	if want := "<?php\n/**\n * @var int $x\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to rewrite @link -> @website; @type is left alone
	f := PhpdocNoAliasTag{}.WithConfig(map[string]any{"replacements": map[string]any{"link": "website"}})
	got, changed = apply(t, f.(fixerRule), "<?php\n/**\n * @link Foo\n */\n")
	if want := "<?php\n/**\n * @website Foo\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, f.(fixerRule), "<?php\n/**\n * @type int $x\n */\n"); changed {
		t.Fatal("type must not be rewritten when only link is configured")
	}
}

func TestPhpdocNoAliasTagQuoteSkipQuirk(t *testing.T) {
	t.Parallel()
	// php-cs-fixer's default map; the set configures it the same way
	full := PhpdocNoAliasTag{}.WithConfig(map[string]any{"replacements": map[string]any{
		"property-read": "property", "property-write": "property", "type": "var", "link": "see",
	}}).(fixerRule)

	// GeneralPhpdocTagRename's regex greedily skips from the first quote to the last
	// matching quote, so only the @property-read before the first quote is renamed
	src := "<?php\n/**\n * @property-read Foo<'a'> $a\n * @property-read Foo<'b'> $b\n * @property-read Foo<'c'> $c\n */\n"
	want := "<?php\n/**\n * @property Foo<'a'> $a\n * @property-read Foo<'b'> $b\n * @property-read Foo<'c'> $c\n */\n"
	got, changed := apply(t, full, src)
	if !changed || got != want {
		t.Fatalf("quote-skip: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, full, got)

	// no quotes: every alias tag is renamed
	src = "<?php\n/**\n * @property-read Foo $a\n * @property-write Foo $b\n */\n"
	want = "<?php\n/**\n * @property Foo $a\n * @property Foo $b\n */\n"
	if got, changed := apply(t, full, src); !changed || got != want {
		t.Fatalf("no-quote: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}

	// an unbalanced quote starts no skip span, so both tags are still renamed
	src = "<?php\n/**\n * @type a 'x\n * @type b\n */\n"
	want = "<?php\n/**\n * @var a 'x\n * @var b\n */\n"
	if got, changed := apply(t, full, src); !changed || got != want {
		t.Fatalf("unbalanced: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestPhpdocTypesConfig(t *testing.T) {
	t.Parallel()
	// default lowercases a meta type
	got, changed := apply(t, PhpdocTypes{}, "<?php\n/**\n * @return VOID\n */\n")
	if want := "<?php\n/**\n * @return void\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// groups=["simple"] no longer touches the meta type "void"
	f := PhpdocTypes{}.WithConfig(map[string]any{"groups": []any{"simple"}})
	if _, changed := apply(t, f.(fixerRule), "<?php\n/**\n * @return VOID\n */\n"); changed {
		t.Fatal("void must not be lowercased when only the simple group is enabled")
	}
	got, changed = apply(t, f.(fixerRule), "<?php\n/**\n * @return ARRAY\n */\n")
	if want := "<?php\n/**\n * @return array\n */\n"; !changed || got != want {
		t.Fatalf("simple: changed=%v got=%q want=%q", changed, got, want)
	}
}
