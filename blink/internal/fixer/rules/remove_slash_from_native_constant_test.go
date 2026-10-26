package rules

import "testing"

func TestRemoveSlashFromNativeConstant(t *testing.T) {
	t.Parallel()
	f := RemoveSlashFromNativeConstant{}

	// leading backslash on true/false/null is stripped
	got, changed := apply(t, f, "<?php\nif (\\true) {\n    return \\false;\n}\nreturn \\null;\n")
	if want := "<?php\nif (true) {\n    return false;\n}\nreturn null;\n"; !changed || got != want {
		t.Fatalf("strip: changed=%v got=%q", changed, got)
	}

	// casing is preserved, only the slash goes
	got, changed = apply(t, f, "<?php\n$x = \\TRUE;\n")
	if want := "<?php\n$x = TRUE;\n"; !changed || got != want {
		t.Fatalf("casing: changed=%v got=%q", changed, got)
	}

	// without a backslash: no-op
	if _, changed := apply(t, f, "<?php\nreturn true;\n"); changed {
		t.Fatal("bare constant must be a no-op")
	}

	// namespace segment named true/null stays untouched
	if _, changed := apply(t, f, "<?php\nreturn Some\\True\\Value::NAME;\n"); changed {
		t.Fatal("namespace segment must be a no-op")
	}
	if _, changed := apply(t, f, "<?php\nreturn \\true\\Bar::NAME;\n"); changed {
		t.Fatal("leading segment before more namespace must be a no-op")
	}
}
