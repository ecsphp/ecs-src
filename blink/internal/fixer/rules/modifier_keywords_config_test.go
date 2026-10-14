package rules

import "testing"

func TestModifierKeywordsConfig(t *testing.T) {
	t.Parallel()

	src := "<?php class C { const A = 1; static $b; static function c() {} }"

	// default adds visibility to const, property and method.
	wantAll := "<?php class C { public const A = 1; public static $b; public static function c() {} }"
	if got, changed := apply(t, ModifierKeywords{}, src); !changed || got != wantAll {
		t.Fatalf("default changed=%v got=%q", changed, got)
	}

	// elements:[method] fixes only the method, leaving const and property.
	f := ModifierKeywords{}.WithConfig(map[string]any{"elements": []any{"method"}})
	wantMethod := "<?php class C { const A = 1; static $b; public static function c() {} }"
	if got, changed := apply(t, f.(fixerRule), src); !changed || got != wantMethod {
		t.Fatalf("elements=[method] changed=%v got=%q", changed, got)
	}

	// elements:[const] fixes only the const.
	f = ModifierKeywords{}.WithConfig(map[string]any{"elements": []any{"const"}})
	wantConst := "<?php class C { public const A = 1; static $b; static function c() {} }"
	if got, changed := apply(t, f.(fixerRule), src); !changed || got != wantConst {
		t.Fatalf("elements=[const] changed=%v got=%q", changed, got)
	}
}
