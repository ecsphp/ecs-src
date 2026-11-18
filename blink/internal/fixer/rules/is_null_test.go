package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

func TestIsNull(t *testing.T) {
	t.Parallel()
	f := IsNull{}

	assertFix(t, f, "<?php\nif (is_null($x)) {}", "<?php\nif (null === $x) {}", true)
	assertFix(t, f, "<?php\nif (! is_null($x)) {}", "<?php\nif (null !== $x) {}", true)
	assertFix(t, f, "<?php\n$y = is_null($a->b());", "<?php\n$y = null === $a->b();", true)
	assertFix(t, f, "<?php\n$y = is_null($a['k']);", "<?php\n$y = null === $a['k'];", true)

	// argument with a top-level operator keeps its parentheses
	assertFix(t, f, "<?php\nif (! is_null($h = f())) {}", "<?php\nif (null !== ($h = f())) {}", true)

	// method/static call of the same name is left alone
	assertFix(t, f, "<?php\n$o->is_null($x);", "<?php\n$o->is_null($x);", false)
	assertFix(t, f, "<?php\nFoo::is_null($x);", "<?php\nFoo::is_null($x);", false)

	// no is_null present
	assertFix(t, f, "<?php\n$x = 1;", "<?php\n$x = 1;", false)
}

// Calls in a namespace and first-class callables, as FunctionsAnalyzer::isGlobalFunctionCall sees them.
func TestIsNullRegressions(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, IsNull{}, []upstreamCase{
		{"first-class callable as an argument", nil, "<?php\n$nulls = array_map(is_null(...), $values);\n", ""},
		{"function declared in the namespace", nil, "<?php\nnamespace A;\nfunction is_null($x): bool { return false; }\n$f = is_null($x);\n", ""},
		{"function imported from a namespace", nil, "<?php\nnamespace A;\nuse function B\\is_null;\n$f = is_null($x);\n", ""},
		{"global function in a namespace", nil, "<?php\nnamespace A;\n$f = null === $x;\n", "<?php\nnamespace A;\n$f = is_null($x);\n"},
	})
}

func TestIsNullSourceURL(t *testing.T) {
	t.Parallel()
	f := IsNull{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
