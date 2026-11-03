package rules

import (
	"strings"
	"testing"
)

const (
	npZWSP  = "​" // ZERO WIDTH SPACE
	npFigSp = " " // FIGURE SPACE
	npNNBSP = " " // NARROW NO-BREAK SPACE
	npNBSP  = " " // NO-BREAK SPACE
)

// Cases ported from PHP-CS-Fixer's NonPrintableCharacterFixerTest.
func TestNonPrintableCharacterUpstream(t *testing.T) {
	t.Parallel()
	plain := map[string]any{"use_escape_sequences_in_strings": false}
	escape := map[string]any{"use_escape_sequences_in_strings": true}
	docblock := "<?php\n\n                /**\n                 * @param %sstring $p Param\n                 */\n                function f(string $p)\n                {\n                    echo $p;\n                }"
	doc := func(c string) string { return strings.Replace(docblock, "%s", c, 1) }
	runUpstreamFixCases(t, NonPrintableCharacter{}, []upstreamFixCase{
		{name: "#0", config: plain, expected: `<?php echo "Hello World !";`,
			input: `<?php echo "` + npZWSP + `Hello` + npFigSp + `World` + npNBSP + `!";`},
		{name: "#1", config: plain, expected: `<?php echo "Hello World !";`,
			input: `<?php echo "` + npZWSP + npZWSP + npZWSP + npZWSP + npZWSP + npZWSP + `Hello World !";`},
		{name: "#2", config: plain, expected: "<?php\n// echo\necho \"Hello World !\";",
			input: "<?php\n// ec" + npZWSP + "ho\necho \"Hello" + npNNBSP + "World" + npNBSP + "!\";"},
		{name: "#3", config: plain, expected: doc(""), input: doc(npZWSP)},
		{name: "#4", config: plain, expected: `<?php echo "$a[0] ${a}";`,
			input: `<?php echo "$a` + npZWSP + `[0]` + npZWSP + ` ${a` + npZWSP + `}";`},
		{name: "#5", config: plain, expected: `<?php echo '12345';?>abc<?php ?>`,
			input: `<?php echo '123` + npZWSP + `45';?>a` + npZWSP + `bc<?php ?>`},
		{name: "#6", config: plain, expected: `<?php echo "${foo` + npNBSP + `bar} is great!";`},
		{name: "#7", config: plain, expected: `<?php echo $foo` + npNBSP + `bar;`},
		{name: "#8", config: plain, expected: `<?php /* foo *` + npZWSP + `/ bar */`},
		{name: "#9", config: plain, expected: `<?php /** foo *` + npZWSP + `/ bar */`},
		{name: "#10", config: plain, expected: `<?php echo b"Hello World !";`,
			input: `<?php echo b"` + npZWSP + `Hello` + npFigSp + `World` + npNBSP + `!";`},
		{name: "#11", config: escape, expected: doc(""), input: doc(npZWSP)},
		{name: "#12", config: escape, expected: `<?php echo 'FooBar\\';`},
		{name: "#13", config: escape, expected: `<?php echo "Foo\u{200b}Bar";`,
			input: `<?php echo "Foo` + npZWSP + `Bar";`},
		{name: "#14", config: escape, expected: `<?php echo "Foo\u{200b}Bar";`,
			input: `<?php echo 'Foo` + npZWSP + `Bar';`},
		{name: "#15", config: escape, expected: `<?php echo "Foo\u{200b} Bar \\n \\ \$variableToEscape";`,
			input: `<?php echo 'Foo` + npZWSP + ` Bar \n \ $variableToEscape';`},
		{name: "#16", config: escape, expected: "<?php echo <<<'TXT'\nFooBar\\\nTXT;\n"},
		{name: "#17", config: escape, expected: "<?php echo <<<TXT\nFoo\\u{200b}Bar\nTXT;\n",
			input: "<?php echo <<<TXT\nFoo" + npZWSP + "Bar\nTXT;\n"},
		{name: "#18", config: escape, expected: "<?php echo <<<TXT\nFoo\\u{200b}Bar\nTXT;\n",
			input: "<?php echo <<<'TXT'\nFoo" + npZWSP + "Bar\nTXT;\n"},
		{name: "#19", config: escape, expected: "<?php echo <<<TXT\n" + `Foo\u{200b} Bar \\n \\ \$variableToEscape` + "\nTXT;\n",
			input: "<?php echo <<<'TXT'\nFoo" + npZWSP + ` Bar \n \ $variableToEscape` + "\nTXT;\n"},
		{name: "#20", config: escape, expected: `<?php echo '。';`},
		{name: "#21", config: escape, expected: `<?php echo "Double \" quote \u{200b} inside";`,
			input: `<?php echo 'Double " quote ` + npZWSP + ` inside';`},
		{name: "#22", config: escape, expected: `<?php echo "Single ' quote \u{200b} inside";`,
			input: `<?php echo 'Single \' quote ` + npZWSP + ` inside';`},
		{name: "#23", config: escape, expected: "<?php echo <<<STRING\n" + `    Quotes ' and " to be handled \u{200b} properly \\' and \\"` + "\nSTRING\n;",
			input: "<?php echo <<<'STRING'\n" + `    Quotes ' and " to be handled ` + npZWSP + ` properly \' and \"` + "\nSTRING\n;"},
		{name: "#24", config: escape, expected: `<?php echo "\\\u{200b}\"";`,
			input: `<?php echo '\\` + npZWSP + `"';`},
		{name: "#25", config: escape, expected: `<?php echo "\\\u{200b}'";`,
			input: `<?php echo '\\` + npZWSP + `\'';`},
		{name: "#26", config: escape, expected: `<?php echo "Backslash 1 \\ \u{200b}";`,
			input: `<?php echo 'Backslash 1 \ ` + npZWSP + `';`},
		{name: "#27", config: escape, expected: `<?php echo "Backslash 2 \\ \u{200b}";`,
			input: `<?php echo 'Backslash 2 \\ ` + npZWSP + `';`},
		{name: "#28", config: escape, expected: `<?php echo "Backslash 3 \\\\ \u{200b}";`,
			input: `<?php echo 'Backslash 3 \\\ ` + npZWSP + `';`},
		{name: "#29", config: escape, expected: `<?php echo "Backslash 4 \\\\ \u{200b}";`,
			input: `<?php echo 'Backslash 4 \\\\ ` + npZWSP + `';`},
		{name: "#30", config: escape,
			expected: "<?php \"String in single quotes, having non-breaking space: \\u{a0}, linebreak: \n, and single quote inside: ' is a dangerous mix.\";",
			input:    "<?php 'String in single quotes, having non-breaking space: " + npNBSP + ", linebreak: \n, and single quote inside: \\' is a dangerous mix.';"},
	})
}

func TestNonPrintableCharacter(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, NonPrintableCharacter{}, []upstreamFixCase{
		{name: "default escapes", expected: `<?php echo "a\u{a0}b";`, input: `<?php echo 'a` + npNBSP + `b';`},
		{name: "default plain elsewhere",
			expected: "a b<?php $ab = 1; // c d\n",
			input:    "a" + npNBSP + "b<?php $a" + npZWSP + "b = 1; // c" + npNBSP + "d\n"},
		{name: "interpolated",
			expected: `<?php echo "x\u{200b} $a {$b} y\u{2007}";`,
			input:    `<?php echo "x` + npZWSP + ` $a {$b} y` + npFigSp + `";`},
		// a literal run that starts with "'" is quoted over like a single-quoted string
		{name: "encapsed run starting with a quote",
			expected: `<?php echo "$a"\u{200b}"";`,
			input:    `<?php echo "$a'` + npZWSP + `'";`},
		{name: "keep in strings", config: map[string]any{"use_escape_sequences_in_strings": false},
			expected: "<?php echo <<<'TXT'\nab\nTXT;\n",
			input:    "<?php echo <<<'TXT'\na" + npZWSP + "b\nTXT;\n"},
	})
}
