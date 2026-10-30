package rules

import (
	"testing"

	fixerpkg "blink/internal/fixer"
)

func TestMethodArgumentSpace(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		src     string
		want    string
		changed bool
	}{
		{"already correct", "<?php foo($a, $b);", "<?php foo($a, $b);", false},
		{"missing and stray spaces", "<?php foo($a ,$b,$c);", "<?php foo($a, $b, $c);", true},
		{"space before comma", "<?php bar($a , $b);", "<?php bar($a, $b);", true},
		{"inner paren spaces left alone", "<?php bar( $a , $b );", "<?php bar( $a, $b );", true},
		{"array comma left alone", "<?php $x = [1,2];", "<?php $x = [1,2];", false},
		{"array inside call", "<?php foo([1,2],$b);", "<?php foo([1,2], $b);", true},
		{"multiline arg list becomes fully multiline", "<?php foo($a,\n    $b);", "<?php foo(\n    $a,\n    $b\n);", true},
		{"newline only inside a nested arg is left alone", "<?php foo($a, [\n    1,\n]);", "<?php foo($a, [\n    1,\n]);", false},
		{"trailing comma before paren", "<?php foo($a,);", "<?php foo($a, );", true},
		{"blank line between call args is collapsed", "<?php foo(\n    $a,\n\n    $b\n);", "<?php foo(\n    $a,\n    $b\n);", true},
		{"blank line between attributed promoted params is collapsed", "<?php class C { public function __construct(\n    #[A]\n    int $a,\n\n    #[B]\n    int $b,\n) {} }", "<?php class C { public function __construct(\n    #[A]\n    int $a,\n    #[B]\n    int $b,\n) {} }", true},
		{"blank line before a commented call arg is kept", "<?php foo(\n    $a,\n\n    // next\n    $b\n);", "<?php foo(\n    $a,\n\n    // next\n    $b\n);", false},
		{"grouping paren after ! breaks after (", "<?php return !($a\n|| $b\n);", "<?php return !(\n    $a\n|| $b\n);", true},
		{"grouping paren after && breaks after (", "<?php $x = $a && ($b\n|| $c\n);", "<?php $x = $a && (\n    $b\n|| $c\n);", true},
		{"single-line grouping paren left alone", "<?php return !($a || $b);", "<?php return !($a || $b);", false},
		{"control-structure paren is not reflowed", "<?php if ($a\n|| $b\n) {}", "<?php if ($a\n|| $b\n) {}", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := apply(t, MethodArgumentSpace{}, tc.src)
			if got != tc.want || changed != tc.changed {
				t.Fatalf("changed=%v got=%q want=%q", changed, got, tc.want)
			}
			// Idempotent: running the fixer again must not change the result.
			again, changedAgain := apply(t, MethodArgumentSpace{}, got)
			if changedAgain || again != got {
				t.Fatalf("not idempotent: changed=%v got=%q", changedAgain, again)
			}
		})
	}
}

func TestMethodArgumentSpaceSourceURL(t *testing.T) {
	t.Parallel()
	r := MethodArgumentSpace{}
	if got := fixerpkg.SourceURLFor(r.Name()); got != r.SourceURL() {
		t.Fatalf("SourceURL mismatch: derived=%q declared=%q", got, r.SourceURL())
	}
}

// A single-argument call to the eval/exit/die language constructs is not a
// function call, so a heredoc argument must not be reflowed onto its own line.
func TestMethodArgumentSpaceEvalHeredocNotReflowed(t *testing.T) {
	t.Parallel()
	src := "<?php\neval(<<<EOPHP\nnamespace X;\nEOPHP\n);\n"
	if got, changed := apply(t, MethodArgumentSpace{}, src); changed || got != src {
		t.Fatalf("eval heredoc must be left inline; changed=%v got=%q", changed, got)
	}
	// a real method named eval is still a call: its heredoc arg reflows onto its own line
	call := "<?php\n$o->eval(<<<EOT\nbody\nEOT\n);\n"
	want := "<?php\n$o->eval(\n    <<<EOT\nbody\nEOT\n);\n"
	if got, changed := apply(t, MethodArgumentSpace{}, call); !changed || got != want {
		t.Fatalf("a real ->eval() method call must still be processed; changed=%v got=%q", changed, got)
	}
}
