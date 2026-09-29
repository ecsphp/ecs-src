package rules

import "testing"

func TestArraySyntax(t *testing.T) {
	got, changed := apply(t, ArraySyntax{}, "<?php $a = array(1, array(2, 3));")
	if want := "<?php $a = [1, [2, 3]];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// a method/const named array is not the construct
	if _, changed := apply(t, ArraySyntax{}, "<?php $obj->array(1);"); changed {
		t.Fatal("method call array() must not convert")
	}
	// a cast and a typehint are not the construct
	if _, changed := apply(t, ArraySyntax{}, "<?php $x = (array) $y; function f(array $a) {}"); changed {
		t.Fatal("cast/typehint array must not convert")
	}
}

func TestListSyntax(t *testing.T) {
	got, changed := apply(t, ListSyntax{}, "<?php list($a, $b) = $c;")
	if want := "<?php [$a, $b] = $c;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestArrayCommaSpacing(t *testing.T) {
	got, changed := apply(t, WhitespaceAfterCommaInArray{}, "<?php $a = [1,2,3];")
	if want := "<?php $a = [1, 2, 3];"; !changed || got != want {
		t.Fatalf("after: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, NoWhitespaceBeforeCommaInArray{}, "<?php $a = [1 , 2];")
	if want := "<?php $a = [1, 2];"; !changed || got != want {
		t.Fatalf("before: changed=%v got=%q want=%q", changed, got, want)
	}
	// function-call commas are not this fixer's job
	if _, changed := apply(t, WhitespaceAfterCommaInArray{}, "<?php foo(1,2);"); changed {
		t.Fatal("call args must be left to method_argument_space")
	}
}

func TestSingleQuote(t *testing.T) {
	got, changed := apply(t, SingleQuote{}, `<?php $a = "plain"; $b = "with $var"; $c = "tab\t";`)
	if want := `<?php $a = 'plain'; $b = "with $var"; $c = "tab\t";`; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestStandardizeNotEquals(t *testing.T) {
	got, changed := apply(t, StandardizeNotEquals{}, "<?php if ($a <> $b) {}")
	if want := "<?php if ($a != $b) {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoEmptyStatement(t *testing.T) {
	got, changed := apply(t, NoEmptyStatement{}, "<?php $a = 1;; $b = 2;;;")
	if want := "<?php $a = 1; $b = 2;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// for-header empty expressions stay
	if _, changed := apply(t, NoEmptyStatement{}, "<?php for (;;) {}"); changed {
		t.Fatal("for-header semicolons must not be removed")
	}
}

func TestLineEnding(t *testing.T) {
	got, changed := apply(t, LineEnding{}, "<?php\r\n$a = 1;\r\n")
	if want := "<?php\n$a = 1;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestMagicCasing(t *testing.T) {
	got, changed := apply(t, MagicConstantCasing{}, "<?php echo __line__, __dir__;")
	if want := "<?php echo __LINE__, __DIR__;"; !changed || got != want {
		t.Fatalf("const: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, MagicMethodCasing{}, "<?php class A { public function __CONSTRUCT() {} public function __tostring() {} }")
	if want := "<?php class A { public function __construct() {} public function __toString() {} }"; !changed || got != want {
		t.Fatalf("method: changed=%v got=%q want=%q", changed, got, want)
	}
}
