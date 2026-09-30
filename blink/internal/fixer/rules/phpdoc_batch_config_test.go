package rules

import "testing"

func TestPhpdocNoAliasTagConfig(t *testing.T) {
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

func TestPhpdocTypesConfig(t *testing.T) {
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
