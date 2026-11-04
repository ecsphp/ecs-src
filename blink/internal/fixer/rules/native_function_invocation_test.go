package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

func nfiConfig(opts map[string]any) fixerRule {
	return NativeFunctionInvocation{}.WithConfig(opts).(fixerRule)
}

func TestNativeFunctionInvocationDefault(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{"optimized gets slash", "<?php count([1]);", "<?php \\count([1]);"},
		{"strlen optimized", "<?php strlen($x);", "<?php \\strlen($x);"},
		{"already qualified kept", "<?php \\count([1]);", "<?php \\count([1]);"},
		{"non-optimized untouched", "<?php strpos($a, $b);", "<?php strpos($a, $b);"},
		{"non-optimized slash stripped", "<?php \\strpos($a, $b);", "<?php strpos($a, $b);"},
		{"non-optimized no slash kept", "<?php json_encode($z);", "<?php json_encode($z);"},
		{"method call untouched", "<?php $o->count();", "<?php $o->count();"},
		{"static call untouched", "<?php Foo::count();", "<?php Foo::count();"},
		{"declaration untouched", "<?php function count() {}", "<?php function count() {}"},
		{"instantiation untouched", "<?php new Count();", "<?php new Count();"},
		{"qualified call untouched", "<?php Ns\\count();", "<?php Ns\\count();"},
		{"new qualified untouched", "<?php new \\Foo();", "<?php new \\Foo();"},
		{"first-class callable untouched", "<?php $f = count(...);", "<?php $f = count(...);"},
		{"spread call gets slash", "<?php count(...$a);", "<?php \\count(...$a);"},
	}
	for _, tc := range cases {
		got, _ := apply(t, NativeFunctionInvocation{}, tc.src)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
		idempotent(t, NativeFunctionInvocation{}, got)
	}
}

func TestNativeFunctionInvocationInclude(t *testing.T) {
	t.Parallel()
	// @all: every global call gets a slash, strict strips nothing
	all := nfiConfig(map[string]any{"include": []any{"@all"}})
	if got, _ := apply(t, all, "<?php json_encode($z); \\strpos($a, $b);"); got != "<?php \\json_encode($z); \\strpos($a, $b);" {
		t.Errorf("@all: got %q", got)
	}

	// explicit names only: names not listed are stripped, listed ones qualified
	names := nfiConfig(map[string]any{"include": []any{"strpos"}})
	if got, _ := apply(t, names, "<?php \\count([1]); strpos($a, $b);"); got != "<?php count([1]); \\strpos($a, $b);" {
		t.Errorf("names: got %q", got)
	}

	// exclude wins over the optimized set
	excl := nfiConfig(map[string]any{"exclude": []any{"count"}})
	if got, _ := apply(t, excl, "<?php \\count([1]); strlen($x);"); got != "<?php count([1]); \\strlen($x);" {
		t.Errorf("exclude: got %q", got)
	}

	// @internal covers a non-optimized native function
	internal := nfiConfig(map[string]any{"include": []any{"@internal"}})
	if got, _ := apply(t, internal, "<?php strpos($a, $b); myfunc();"); got != "<?php \\strpos($a, $b); myfunc();" {
		t.Errorf("@internal: got %q", got)
	}
}

func TestNativeFunctionInvocationStrictAndScope(t *testing.T) {
	t.Parallel()
	// strict=false: a non-included call keeps its existing slash
	loose := nfiConfig(map[string]any{"strict": false})
	if got, _ := apply(t, loose, "<?php \\strpos($a, $b);"); got != "<?php \\strpos($a, $b);" {
		t.Errorf("strict=false: got %q", got)
	}

	// scope=namespaced: inside a named namespace the optimized call is qualified
	nsp := nfiConfig(map[string]any{"scope": "namespaced"})
	if got, _ := apply(t, nsp, "<?php namespace App;\ncount([1]);"); got != "<?php namespace App;\n\\count([1]);" {
		t.Errorf("namespaced: got %q", got)
	}
	// a file with no namespace is the global namespace: slashes are stripped, none added
	if got, _ := apply(t, nsp, "<?php count([1]); \\strpos($a, $b);"); got != "<?php count([1]); strpos($a, $b);" {
		t.Errorf("namespaced global: got %q", got)
	}
}

func TestNativeFunctionInvocationSourceURL(t *testing.T) {
	t.Parallel()
	var f fixer.Fixer = NativeFunctionInvocation{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Errorf("SourceURL %q != SourceURLFor %q", got, want)
	}
}

// An attribute class named like a native function (Symfony's Count constraint)
// is not a call: NativeFunctionInvocation leaves it alone, so the
// GlobalNamespaceImport that follows does not import it as a function. Checked
// against the PHP engine with tipee's options for both rules.
func TestNativeFunctionInvocationThenGlobalNamespaceImportKeepsAttributes(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace App;\n\nuse Symfony\\Component\\Validator\\Constraints\\Count;\n\nfinal class Cmd\n{\n    #[NotNull, Count(min: 1), All([new Type('int')]), Valid]\n    public array $content;\n\n    public function f(array $a): int\n    {\n        return count($a);\n    }\n}\n"
	want := "<?php\n\nnamespace App;\n\nuse Symfony\\Component\\Validator\\Constraints\\Count;\nuse function count;\n\nfinal class Cmd\n{\n    #[NotNull, Count(min: 1), All([new Type('int')]), Valid]\n    public array $content;\n\n    public function f(array $a): int\n    {\n        return count($a);\n    }\n}\n"
	nfi := nfiConfig(map[string]any{"scope": "namespaced"})
	gni := GlobalNamespaceImport{}.WithConfig(map[string]any{"import_constants": true, "import_functions": true}).(fixerRule)
	got, _ := apply(t, nfi, src)
	if got, _ = apply(t, gni, got); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
