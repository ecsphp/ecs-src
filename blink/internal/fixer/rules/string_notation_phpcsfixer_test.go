package rules

import "testing"

// Cases ported from PHP-CS-Fixer's SingleQuoteFixerTest (tests/Fixer/StringNotation),
// then invented ones (named "extra ..."); every expected output is what
// PHP-CS-Fixer 3.95 produces for the input.
func TestSingleQuotePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	withQuotes := map[string]any{"strings_containing_single_quote_chars": true}
	aliasRunUpstreamCases(t, SingleQuote{}, []aliasUpstreamCase{
		{"case 0", nil, `<?php $a = "";`, `<?php $a = '';`},
		{"case 1", nil, `<?php $a = "foo bar";`, `<?php $a = 'foo bar';`},
		{"case 2", nil, `<?php $a = b"";`, `<?php $a = b'';`},
		{"case 3", nil, `<?php $a = B"";`, `<?php $a = B'';`},
		{"case 4", nil, `<?php $a = b"foo bar";`, `<?php $a = b'foo bar';`},
		{"case 5", nil, `<?php $a = B"foo bar";`, `<?php $a = B'foo bar';`},
		{"case 6", nil, `<?php $a = "foo
                    bar";`, `<?php $a = 'foo
                    bar';`},
		{"case 7", nil, `<?php $a = 'foo'."bar"."$baz";`, `<?php $a = 'foo'.'bar'."$baz";`},
		{"case 8", nil, `<?php $a = "foo \"bar\"";`, `<?php $a = 'foo "bar"';`},
		{"case 9", nil, `<?php $a = "\\foo\\bar\\\\";`, `<?php $a = '\\foo\\bar\\\\';`},
		{"case 10", nil, `<?php $a = "foo \$bar7";`, `<?php $a = 'foo $bar7';`},
		{"case 11", nil, `<?php $a = "foo \$(bar7)";`, `<?php $a = 'foo $(bar7)';`},
		{"case 12", nil, `<?php $a = "foo \\(\$bar8)";`, `<?php $a = 'foo \\($bar8)';`},
		{"case 13", nil, `<?php $a = "foo \" \$$bar";`, `<?php $a = "foo \" \$$bar";`},
		{"case 14", nil, `<?php $a = b"foo \" \$$bar";`, `<?php $a = b"foo \" \$$bar";`},
		{"case 15", nil, `<?php $a = B"foo \" \$$bar";`, `<?php $a = B"foo \" \$$bar";`},
		{"case 16", nil, `<?php $a = "foo 'bar'";`, `<?php $a = "foo 'bar'";`},
		{"case 17", nil, `<?php $a = b"foo 'bar'";`, `<?php $a = b"foo 'bar'";`},
		{"case 18", nil, `<?php $a = B"foo 'bar'";`, `<?php $a = B"foo 'bar'";`},
		{"case 19", nil, `<?php $a = "foo $bar";`, `<?php $a = "foo $bar";`},
		{"case 20", nil, `<?php $a = b"foo $bar";`, `<?php $a = b"foo $bar";`},
		{"case 21", nil, `<?php $a = B"foo $bar";`, `<?php $a = B"foo $bar";`},
		{"case 22", nil, `<?php $a = "foo ${bar}";`, `<?php $a = "foo ${bar}";`},
		{"case 23", nil, `<?php $a = b"foo ${bar}";`, `<?php $a = b"foo ${bar}";`},
		{"case 24", nil, `<?php $a = B"foo ${bar}";`, `<?php $a = B"foo ${bar}";`},
		{"case 25", nil, `<?php $a = "foo\n bar";`, `<?php $a = "foo\n bar";`},
		{"case 26", nil, `<?php $a = b"foo\n bar";`, `<?php $a = b"foo\n bar";`},
		{"case 27", nil, `<?php $a = B"foo\n bar";`, `<?php $a = B"foo\n bar";`},
		{"case 28", nil, `<?php $a = "\\\n";`, `<?php $a = "\\\n";`},
		{"case 29", withQuotes, `<?php $a = "foo 'bar'";`, `<?php $a = 'foo \'bar\'';`},
		{"case 30", withQuotes, `<?php
// none
$a = "start ' end";
// one escaped backslash
$b = "start \\' end";
// two escaped backslash
$c = "start \\\\' end";`, `<?php
// none
$a = 'start \' end';
// one escaped backslash
$b = 'start \\\' end';
// two escaped backslash
$c = 'start \\\\\' end';`},
		{"case 31", withQuotes, `<?php
// one unescaped backslash
$a = "start \' end";
// one escaped + one unescaped backslash
$b = "start \\\' end";`, `<?php
// one unescaped backslash
$a = "start \' end";
// one escaped + one unescaped backslash
$b = "start \\\' end";`},

		{"extra escaped backslashes", nil, `<?php
$a = "/^\\d{2,}$/";
$b = "Y-m-d\\TH:i:s";
$c = "\\";
$d = "a\\";
$e = "\\\\";
$f = "C:\\Users\\";
$g = "x\\\\\\y";
$h = "\\n\\x41\\u{41}\\8\\é";
$i = "line1
\\line2";
$j = b"\\d";`, `<?php
$a = '/^\\d{2,}$/';
$b = 'Y-m-d\\TH:i:s';
$c = '\\';
$d = 'a\\';
$e = '\\\\';
$f = 'C:\\Users\\';
$g = 'x\\\\\\y';
$h = '\\n\\x41\\u{41}\\8\\é';
$i = 'line1
\\line2';
$j = b'\\d';`},
		{"extra dollar that starts no variable", nil, `<?php
$a = "$";
$b = "$1";
$c = "$ {x}";
$d = "$$";
$e = "$}";`, `<?php
$a = '$';
$b = '$1';
$c = '$ {x}';
$d = '$$';
$e = '$}';`},
		{"extra escaped dollar and quote", nil, `<?php
$a = "\$";
$b = "\${x}";
$c = "a \${b} c";
$d = "{\$x} \$y";
$e = "\\$";
$f = "\\\$x";
$g = "\\\${x}";
$h = "\\\\\$";
$i = "\\\"";
$j = "\$\"\\";
$k = B"\$x\\y";`, `<?php
$a = '$';
$b = '${x}';
$c = 'a ${b} c';
$d = '{$x} $y';
$e = '\$';
$f = '\\$x';
$g = '\\${x}';
$h = '\\\\$';
$i = '\\"';
$j = '$"\\';
$k = B'$x\\y';`},
		{"extra other escape sequences", nil, `<?php
$a = "\n";
$b = "\t\r";
$c = "\u{1F600}";
$d = "\x41";
$e = "\101";
$f = "\e\v\f\0";
$g = "\8\9";
$h = "\{";
$i = "\é";
$j = "\\\n";
$k = "\'";`, `<?php
$a = "\n";
$b = "\t\r";
$c = "\u{1F600}";
$d = "\x41";
$e = "\101";
$f = "\e\v\f\0";
$g = "\8\9";
$h = "\{";
$i = "\é";
$j = "\\\n";
$k = "\'";`},
		{"extra interpolation", nil, `<?php
$a = "${name}";
$b = "a ${b} c";
$c = "{$name}";
$d = "$é";
$e = "\\{$x}";
$f = "\{$x}";
$g = "\\${x}";
$h = "$$x";
$i = "$a[k]";`, `<?php
$a = "${name}";
$b = "a ${b} c";
$c = "{$name}";
$d = "$é";
$e = "\\{$x}";
$f = "\{$x}";
$g = "\\${x}";
$h = "$$x";
$i = "$a[k]";`},
		{"extra single quote", nil, `<?php
$a = "it's";
$b = "a\\'b";
$c = "\\\\'";`, `<?php
$a = "it's";
$b = "a\\'b";
$c = "\\\\'";`},
		{"extra single quote and escaped backslash", withQuotes, `<?php
$a = "it's \\d";
$b = "\\'";
$c = "'\$'";
$d = "it's \d";`, `<?php
$a = 'it\'s \\d';
$b = '\\\'';
$c = '\'$\'';
$d = "it's \d";`},
		{"extra string in an interpolation", nil, `<?php $a = "{$a["k\\ey"]}";`, `<?php $a = "{$a['k\\ey']}";`},
		{"extra unterminated string, a parse error PHP-CS-Fixer leaves as is", nil, `<?php $a = "abc\"`, `<?php $a = "abc\"`},
	})
}
