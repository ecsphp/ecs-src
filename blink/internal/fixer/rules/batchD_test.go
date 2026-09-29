package rules

import "testing"

func TestSimplifiedNullReturn(t *testing.T) {
	// no return type -> simplified
	got, changed := apply(t, SimplifiedNullReturn{}, "<?php function a() {\n    return null;\n}")
	if want := "<?php function a() {\n    return;\n}"; !changed || got != want {
		t.Fatalf("no type: changed=%v got=%q", changed, got)
	}
	// void return type -> simplified
	got, changed = apply(t, SimplifiedNullReturn{}, "<?php function c(): void {\n    return null;\n}")
	if want := "<?php function c(): void {\n    return;\n}"; !changed || got != want {
		t.Fatalf("void: changed=%v got=%q", changed, got)
	}
	// closure with no type -> simplified
	got, changed = apply(t, SimplifiedNullReturn{}, "<?php $f = function () {\n    return null;\n};")
	if want := "<?php $f = function () {\n    return;\n};"; !changed || got != want {
		t.Fatalf("closure: changed=%v got=%q", changed, got)
	}
	// nullable return type -> kept
	if _, changed := apply(t, SimplifiedNullReturn{}, "<?php function b(): ?int {\n    return null;\n}"); changed {
		t.Fatal("nullable ?int return type must keep explicit null")
	}
	// union with null -> kept
	if _, changed := apply(t, SimplifiedNullReturn{}, "<?php function d(): int|null {\n    return null;\n}"); changed {
		t.Fatal("int|null return type must keep explicit null")
	}
	// already simplified -> no-op
	if _, changed := apply(t, SimplifiedNullReturn{}, "<?php function a() {\n    return;\n}"); changed {
		t.Fatal("return; must not change")
	}
	// return null in a non-nullable typed function -> simplified
	got, changed = apply(t, SimplifiedNullReturn{}, "<?php function e(): int {\n    return null;\n}")
	if want := "<?php function e(): int {\n    return;\n}"; !changed || got != want {
		t.Fatalf("int: changed=%v got=%q", changed, got)
	}
}
