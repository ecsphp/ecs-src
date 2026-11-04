package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// upstreamFixCase is one PHP-CS-Fixer doTest() case: input fixes to expected,
// and expected stays as it is. An empty input means expected must not change.
type upstreamFixCase struct {
	name     string
	expected string
	input    string
	config   map[string]any
}

func runUpstreamFixCases(t *testing.T, f fixer.Fixer, cases []upstreamFixCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := f
			if cf, ok := f.(fixer.ConfigurableFixer); ok && c.config != nil {
				r = cf.WithConfig(c.config)
			}
			if c.input != "" {
				if got, _ := apply(t, r, c.input); got != c.expected {
					t.Errorf("fix mismatch\ninput:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.expected)
				}
			}
			if got, changed := apply(t, r, c.expected); changed || got != c.expected {
				t.Errorf("expected output is not stable\ngot:\n%s\nwant:\n%s", got, c.expected)
			}
		})
	}
}

// Cases ported from PHP-CS-Fixer's SimpleToComplexStringVariableFixerTest.
func TestSimpleToComplexStringVariableUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, SimpleToComplexStringVariable{}, []upstreamFixCase{
		{name: "basic fix",
			expected: "<?php\n$name = \"World\";\necho \"Hello {$name}!\";",
			input:    "<?php\n$name = \"World\";\necho \"Hello ${name}!\";"},
		{name: "heredoc",
			expected: "<?php\n$name = 'World';\necho <<<TEST\nHello {$name}!\nTEST;\n",
			input:    "<?php\n$name = 'World';\necho <<<TEST\nHello ${name}!\nTEST;\n"},
		{name: "implicit", expected: "<?php\n$name = 'World';\necho \"Hello $name!\";"},
		{name: "implicit again", expected: "<?php\n$name = 'World';\necho \"Hello { $name }!\";"},
		{name: "escaped", expected: "<?php\n$name = 'World';\necho \"Hello \\${name}\";"},
		{name: "double dollar",
			expected: "<?php\n$name = 'World';\necho \"Hello \\${$name}\";",
			input:    "<?php\n$name = 'World';\necho \"Hello $${name}\";"},
		{name: "double dollar heredoc",
			expected: "<?php\n$name = 'World';\necho <<<TEST\nHello \\${$name}!\nTEST;\n",
			input:    "<?php\n$name = 'World';\necho <<<TEST\nHello $${name}!\nTEST;\n"},
		{name: "double dollar single quote", expected: "<?php\n$name = 'World';\necho 'Hello $${name}';"},
		{name: "array elements",
			expected: "<?php\n\"Hello {$array[0]}!\";\n\"Hello {$array['index']}!\";",
			input:    "<?php\n\"Hello ${array[0]}!\";\n\"Hello ${array['index']}!\";"},
	})
}

func TestSimpleToComplexStringVariable(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, SimpleToComplexStringVariable{}, []upstreamFixCase{
		// an expression, not a name, stays as it is
		{name: "expression", expected: `<?php echo "${$a} ${a . 'b'}";`},
		// a "${" in a string already split around a "{$...}"
		{name: "next to complex",
			expected: `<?php echo "{$a} {$b} x {$c->d}";`,
			input:    `<?php echo "{$a} ${b} x {$c->d}";`},
		{name: "escaped backslash before dollar is kept",
			expected: `<?php echo "\\${$a}";`,
			input:    `<?php echo "\\$${a}";`},
		{name: "nowdoc untouched", expected: "<?php echo <<<'TEST'\n${a}\nTEST;\n"},
		{name: "inside interpolated heredoc code",
			expected: "<?php echo <<<TEST\n{$a[\"{$b}\"]} {$c}\nTEST;\n",
			input:    "<?php echo <<<TEST\n{$a[\"${b}\"]} ${c}\nTEST;\n"},
		// checked with PHP-CS-Fixer
		{name: "backticks", expected: "<?php $x = `ls {$b}`;", input: "<?php $x = `ls ${b}`;"},
		{name: "chunks", expected: `<?php $y = "{$a[0]}x{$b} \${$c}"; $q = "{$a[${b}]} {$c}";`,
			input: `<?php $y = "${a[0]}x${b} $${c}"; $q = "{$a[${b}]} ${c}";`},
		{name: "heredoc offset", expected: "<?php $z = <<<EOT\n{$a[0]} {$b}\nEOT;\n", input: "<?php $z = <<<EOT\n${a[0]} ${b}\nEOT;\n"},
	})
}
