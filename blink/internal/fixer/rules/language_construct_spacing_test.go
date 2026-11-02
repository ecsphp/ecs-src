package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestLanguageConstructSpacing(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{"collapse multi space", `<?php echo   "x";`, `<?php echo "x";`},
		{"insert before string", `<?php echo"y";`, `<?php echo "y";`},
		{"print variable", `<?php print  $a;`, `<?php print $a;`},
		{"return collapses newline", "<?php return\n    $multi;", "<?php return $multi;"},
		{"throw glued", `<?php throw$e;`, `<?php throw $e;`},
		{"new collapse", `<?php $x = new    Bar();`, `<?php $x = new Bar();`},
		{"goto", `<?php goto  end;`, `<?php goto end;`},
		{"namespace decl", `<?php namespace   App;`, `<?php namespace App;`},
		{"return negative", `<?php return-1;`, `<?php return -1;`},

		// grouping parens / no content are left alone
		{"print paren", `<?php print($a);`, `<?php print($a);`},
		{"echo paren", `<?php echo($b);`, `<?php echo($b);`},
		{"require call", `<?php require('b.php');`, `<?php require('b.php');`},
		{"require space before paren kept", `<?php require   ("a.php");`, `<?php require ("a.php");`},

		// a qualified name or member access lexes the word non-construct -> untouched
		{"use qualified no space", `<?php use\Foo;`, `<?php use\Foo;`},
		{"new qualified no space", `<?php new\App\Foo();`, `<?php new\App\Foo();`},
		{"namespace operator", `<?php namespace\Foo\bar();`, `<?php namespace\Foo\bar();`},
		{"return qualified", `<?php return\Foo::BAR;`, `<?php return\Foo::BAR;`},
		{"member new", `<?php $o->new;`, `<?php $o->new;`},
		{"member return", `<?php $o->return(1);`, `<?php $o->return(1);`},

		// the space immediately after "use" only; inner double space is untouched
		{"use function inner kept", `<?php use function  foo;`, `<?php use function  foo;`},

		// yield from: internal gap and trailing content both get one space
		{"yield plain", `<?php yield   $v;`, `<?php yield $v;`},
		{"yield from internal", `<?php yield    from $h;`, `<?php yield from $h;`},
		{"yield from trailing", `<?php yield from   $g;`, `<?php yield from $g;`},
		{"yield from glued", `<?php yield from$i;`, `<?php yield from $i;`},
		{"yield from paren", `<?php yield from(gen());`, `<?php yield from(gen());`},
	}

	for _, tc := range cases {
		got, _ := apply(t, LanguageConstructSpacing{}, tc.src)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
		idempotent(t, LanguageConstructSpacing{}, got)
	}
}

func TestLanguageConstructSpacingSourceURL(t *testing.T) {
	t.Parallel()
	f := LanguageConstructSpacing{}
	const want = "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/LanguageConstructSpacingSniff.php"
	if got := f.SourceURL(); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
	// a PHP_CodeSniffer sniff is not a PHP-CS-Fixer rule, so SourceURLFor does
	// not apply; the URL is pinned literally above.
	_ = fixer.SourceURLFor
}
