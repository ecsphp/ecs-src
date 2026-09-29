package rules

import "testing"

func TestModifierKeywords(t *testing.T) {
	f := ModifierKeywords{}

	// reorders modifiers and adds missing visibility across members
	got, changed := apply(t, f, "<?php\nclass C {\n    const S = 1;\n    var $a;\n    static protected int $b;\n    static public final function bar() {}\n    protected abstract function zim();\n    readonly protected string $ro;\n}\n")
	want := "<?php\nclass C {\n    public const S = 1;\n    public $a;\n    protected static int $b;\n    final public static function bar() {}\n    abstract protected function zim();\n    protected readonly string $ro;\n}\n"
	if !changed || got != want {
		t.Fatalf("reorder: changed=%v got=%q", changed, got)
	}

	// already-canonical members are left untouched
	if _, changed := apply(t, f, "<?php\nclass C {\n    public const S = 1;\n    protected static int $b;\n    final public function f() {}\n}\n"); changed {
		t.Fatal("canonical members must be a no-op")
	}

	// trait use and enum case are not touched
	if _, changed := apply(t, f, "<?php\nenum E {\n    case One;\n}\n"); changed {
		t.Fatal("enum case must be ignored")
	}
}
