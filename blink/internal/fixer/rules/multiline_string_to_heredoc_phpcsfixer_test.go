package rules

import "testing"

// Cases ported from PHP-CS-Fixer's MultilineStringToHeredocFixerTest
// (tests/Fixer/StringNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestMultilineStringToHeredocPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, MultilineStringToHeredoc{}, []aliasUpstreamCase{
		{"empty string", nil, "<?php $a = '';", "<?php $a = '';"},
		{"single line string", nil, "<?php $a = 'a b';", "<?php $a = 'a b';"},
		{"single line string with backslash-n", nil, `<?php $a = 'a\nb';`, `<?php $a = 'a\nb';`},
		{"simple single quoted", nil,
			"<?php\n$a = 'line1\nline2';",
			"<?php\n$a = <<<'EOD'\nline1\nline2\nEOD;"},
		{"simple double quoted", nil,
			"<?php\n$a = \"line1\nline2\";",
			"<?php\n$a = <<<EOD\nline1\nline2\nEOD;"},
		{"colliding closing marker - one", nil,
			"<?php\n$a = 'line1\nEOD\nline2';",
			"<?php\n$a = <<<'EOD_'\nline1\nEOD\nline2\nEOD_;"},
		{"colliding closing marker - two", nil,
			"<?php\n$a = 'line1\nEOD\nEOD_\nline2';",
			"<?php\n$a = <<<'EOD__'\nline1\nEOD\nEOD_\nline2\nEOD__;"},
		{"single quoted unescape", nil,
			`<?php
$a = 'line1
\\
\n
\'
\\\\\'
\"
\
';`,
			`<?php
$a = <<<'EOD'
line1
\
\n
'
\\'
\"
\

EOD;`},
		{"double quoted unescape", nil,
			`<?php
$a = "line1
\\
\n
\"
\\\\\"
\'
\
\"{$rawPath}\"
";`,
			`<?php
$a = <<<EOD
line1
\\
\n
"
\\\\"
\'
\
"{$rawPath}"

EOD;`},
		{"single quoted /w variable", nil,
			"<?php\n$a = 'line1$var\nline2';",
			"<?php\n$a = <<<'EOD'\nline1$var\nline2\nEOD;"},
		{"double quoted /w simple variable", nil,
			"<?php\n$a = \"line1$var\nline2\";",
			"<?php\n$a = <<<EOD\nline1$var\nline2\nEOD;"},
		{"double quoted /w simple curly variable", nil,
			"<?php\n$a = \"line1{$var}\nline2\";",
			"<?php\n$a = <<<EOD\nline1{$var}\nline2\nEOD;"},
		{"double quoted /w complex curly variable", nil,
			"<?php\n$a = \"{$arr['foo'][3]}\n{ $obj->values[3]->name }\n{${getName()}}\";",
			"<?php\n$a = <<<EOD\n{$arr['foo'][3]}\n{ $obj->values[3]->name }\n{${getName()}}\nEOD;"},
		{"simple strings prefixed with b/B", nil,
			"<?php\n$a = b'line1\nline2';\n$b = B\"line1\nline2\";",
			"<?php\n$a = <<<'EOD'\nline1\nline2\nEOD;\n$b = <<<EOD\nline1\nline2\nEOD;"},
		{"double quoted /w simple variable prefixed with b/B", nil,
			"<?php\n$a = b\"line1$var\nline2\";\n$b = B\"line1$var\nline2\";",
			"<?php\n$a = <<<EOD\nline1$var\nline2\nEOD;\n$b = <<<EOD\nline1$var\nline2\nEOD;"},
		{"test stateful fixing loop", nil,
			`<?php
"$a
{$b['x']}";
'c
d';

"$a
$b";
"$c
$d";

'a';
'b
c';

<<<'EOD'
EOD;
<<<EOD
$a $b
EOD;
<<<'EOD'
c d
EOD;
<<<EOD
$a $b
EOD;
<<<EOD
$a
$b
EOD;
<<<'EOD'
$c
$d
EOD;`,
			`<?php
<<<EOD
$a
{$b['x']}
EOD;
<<<'EOD'
c
d
EOD;

<<<EOD
$a
$b
EOD;
<<<EOD
$c
$d
EOD;

'a';
<<<'EOD'
b
c
EOD;

<<<'EOD'
EOD;
<<<EOD
$a $b
EOD;
<<<'EOD'
c d
EOD;
<<<EOD
$a $b
EOD;
<<<EOD
$a
$b
EOD;
<<<'EOD'
$c
$d
EOD;`},
	})
}
