package rules

import "testing"

// Cases ported from PHP-CS-Fixer's FopenFlagsFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input and configuration.
func TestFopenFlagsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	bModeOff := map[string]any{"b_mode": false}

	doNotFix := []struct {
		name string
		code string
	}{
		{"not simple flags", `<?php $a = fopen($foo, "t".$a);`},
		{"wrong # of arguments", "<?php\n                    $b = fopen(\"br+\");\n                    $c = fopen($foo, \"w+\", 1, 2 , 3);\n                "},
		{"flags too long", `<?php $d = fopen($foo, "r+w+a+x+c+etXY");`},
		{"flags too short", `<?php $d = fopen($foo, "");`},
		{"static method call", `<?php $e = A::fopen($foo, "w+");`},
		{"method call", `<?php $f = $b->fopen($foo, "r+");`},
		{"comments, PHPDoc and literal", "<?php\n                    // fopen($foo, \"rw\");\n                    /* fopen($foo, \"rw\"); */\n                    echo(\"fopen($foo, \\\"rw\\\")\");\n                "},
		{"invalid flag values", "<?php\n                $a = fopen($foo, '');\n                $a = fopen($foo, 'k');\n                $a = fopen($foo, 'kz');\n                $a = fopen($foo, 'k+');\n                $a = fopen($foo, '+k');\n                $a = fopen($foo, 'xct++');\n                $a = fopen($foo, 'w+r+r+');\n                $a = fopen($foo, '+btrw+');\n                $a = fopen($foo, 'b+rw');\n                $a = fopen($foo, 'bbrw+');\n                $a = fopen($foo, 'brw++');\n                $a = fopen($foo, '++brw');\n                $a = fopen($foo, 'ybrw+');\n                $a = fopen($foo, 'rr');\n                $a = fopen($foo, 'ロ');\n                $a = fopen($foo, 'ロ+');\n                $a = fopen($foo, 'rロ');\n                $a = \\fopen($foo, 'w+ロ');\n                "},
		{"second argument not string", "<?php\n                    echo \"abc\"; // to pass the candidate check\n                    $a = fopen($foo, 1);\n                    $a = fopen($foo, $a);\n                    $a = fopen($foo, null);\n                "},
	}

	cases := []aliasUpstreamCase{
		{"missing b", nil,
			"<?php\n                    $a = fopen($foo, 'rw+');\n                ",
			"<?php\n                    $a = fopen($foo, 'rw+b');\n                "},
		{"has t and b", nil,
			"<?php\n                    $a = \\fopen($foo, \"rw+bt\");\n                ",
			"<?php\n                    $a = \\fopen($foo, \"rw+b\");\n                "},
		{"has t and no b and binary string mod", nil,
			"<?php\n                    $a = fopen($foo, b'trw+');\n                ",
			"<?php\n                    $a = fopen($foo, b'rw+b');\n                "},
		{"missing b but not configured", bModeOff,
			"<?php\n                    $a = fopen($foo, 'rw+t');\n                ",
			"<?php\n                    $a = fopen($foo, 'rw+');\n                "},
		{"t and superfluous b", bModeOff,
			"<?php\n                    $a = fopen($foo, 'r+bt');\n                    $a = fopen($foo, 'btw+r');\n                    $a = fopen($foo, 'r+tb');\n                    $a = fopen($foo, 'tbw+r');\n                ",
			"<?php\n                    $a = fopen($foo, 'r+');\n                    $a = fopen($foo, 'w+r');\n                    $a = fopen($foo, 'r+');\n                    $a = fopen($foo, 'w+r');\n                "},
		{"superfluous b", bModeOff,
			"<?php\n                    $a = fopen($foo, 'r+b');\n                    $a = fopen($foo, 'bw+r');\n                ",
			"<?php\n                    $a = fopen($foo, 'r+');\n                    $a = fopen($foo, 'w+r');\n                "},
	}

	for _, s := range doNotFix {
		cases = append(cases,
			aliasUpstreamCase{s.name + " with b_mode", nil, s.code, s.code},
			aliasUpstreamCase{s.name + " without b_mode", bModeOff, s.code, s.code},
		)
	}

	aliasRunUpstreamCases(t, FopenFlags{BMode: true}, cases)
}
