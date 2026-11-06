package rules

import "testing"

// Cases ported from PHP-CS-Fixer's StringImplicitBackslashesFixerTest
// (tests/Fixer/StringNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestStringImplicitBackslashesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, StringImplicitBackslashes{}, []aliasUpstreamCase{
		{"case 0", nil,
			`<?php $var = 'String (\\\'\r\n\x0\) for My\Prefix\\';`,
			`<?php $var = 'String (\\\'\r\n\x0\) for My\Prefix\\';`},
		{"case 1", map[string]any{"single_quoted": "escape"},
			`<?php $var = 'String (\\\'\r\n\x0\) for My\Prefix\\';`,
			`<?php $var = 'String (\\\'\\r\\n\\x0\\) for My\\Prefix\\';`},
		{"case 2", nil,
			`<?php
$var = "\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = "\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = "$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = <<<HEREDOC
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC;
$var = <<<HEREDOC
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC;
$var = <<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`,
			`<?php
$var = "\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = "\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = "$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = <<<HEREDOC
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC;
$var = <<<HEREDOC
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC;
$var = <<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`},
		{"case 3", nil,
			`<?php
$var = "\e\f\n\r\t\v \\ \$ \"";
$var = "$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = <<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = <<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`,
			`<?php
$var = "\e\f\n\r\t\v \\ \$ \"";
$var = "$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = <<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = <<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`},
		{"case 4", nil,
			`<?php
$var = "\0 \00 \000 \0000 \00000";
$var = "$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = <<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = <<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`,
			`<?php
$var = "\0 \00 \000 \0000 \00000";
$var = "$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = <<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = <<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`},
		{"case 5", nil,
			`<?php
$var = "\xA \x99 \u{0}";
$var = "$foo \xA \x99 \u{0} ${bar}";
$var = <<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = <<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`,
			`<?php
$var = "\xA \x99 \u{0}";
$var = "$foo \xA \x99 \u{0} ${bar}";
$var = <<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = <<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`},
		{"case 6", nil,
			`<?php
$var = 'backslash \ not escaped';
$var = 'code coverage';
$var = "backslash \\ already escaped";
$var = "code coverage";
$var = <<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = <<<HEREDOC
code coverage
HEREDOC;
$var = <<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = <<<'NOWDOC'
code coverage
NOWDOC;
`,
			`<?php
$var = 'backslash \ not escaped';
$var = 'code coverage';
$var = "backslash \\ already escaped";
$var = "code coverage";
$var = <<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = <<<HEREDOC
code coverage
HEREDOC;
$var = <<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = <<<'NOWDOC'
code coverage
NOWDOC;
`},
		{"case 7", map[string]any{"double_quoted": "unescape"},
			`<?php
$var = "\A\a \' \8\9 \xZ \u";
$var = "$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = "\A\a \' \8\9 \xZ \u";
$var = "$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 8", map[string]any{"heredoc": "unescape"},
			`<?php
$var = <<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = <<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`,
			`<?php
$var = <<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = <<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`},
		{"case 9", nil,
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`},
		{"case 10", map[string]any{"single_quoted": "escape"},
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = b'String (\\\'\\r\\n\\x0) for My\\Prefix\\';`},
		{"case 11", nil,
			`<?php
$var = b"\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = b"\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = b"$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = b<<<HEREDOC
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC;
$var = b<<<HEREDOC
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC;
$var = b<<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`,
			`<?php
$var = b"\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = b"\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = b"$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = b<<<HEREDOC
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC;
$var = b<<<HEREDOC
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC;
$var = b<<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`},
		{"case 12", nil,
			`<?php
$var = b"\e\f\n\r\t\v \\ \$ \"";
$var = b"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = b<<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = b<<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`,
			`<?php
$var = b"\e\f\n\r\t\v \\ \$ \"";
$var = b"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = b<<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = b<<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`},
		{"case 13", nil,
			`<?php
$var = b"\0 \00 \000 \0000 \00000";
$var = b"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = b<<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = b<<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`,
			`<?php
$var = b"\0 \00 \000 \0000 \00000";
$var = b"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = b<<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = b<<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`},
		{"case 14", nil,
			`<?php
$var = b"\xA \x99 \u{0}";
$var = b"$foo \xA \x99 \u{0} ${bar}";
$var = b<<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = b<<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`,
			`<?php
$var = b"\xA \x99 \u{0}";
$var = b"$foo \xA \x99 \u{0} ${bar}";
$var = b<<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = b<<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`},
		{"case 15", nil,
			`<?php
$var = b'backslash \ not escaped';
$var = b'code coverage';
$var = b"backslash \\ already escaped";
$var = b"code coverage";
$var = b<<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = b<<<HEREDOC
code coverage
HEREDOC;
$var = b<<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = b<<<'NOWDOC'
code coverage
NOWDOC;
`,
			`<?php
$var = b'backslash \ not escaped';
$var = b'code coverage';
$var = b"backslash \\ already escaped";
$var = b"code coverage";
$var = b<<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = b<<<HEREDOC
code coverage
HEREDOC;
$var = b<<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = b<<<'NOWDOC'
code coverage
NOWDOC;
`},
		{"case 16", map[string]any{"double_quoted": "unescape"},
			`<?php
$var = b"\A\a \' \8\9 \xZ \u";
$var = b"$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = b"\A\a \' \8\9 \xZ \u";
$var = b"$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 17", map[string]any{"heredoc": "unescape"},
			`<?php
$var = b<<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = b<<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`,
			`<?php
$var = b<<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = b<<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`},
		{"case 18", nil,
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`},
		{"case 19", map[string]any{"single_quoted": "escape"},
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = B'String (\\\'\\r\\n\\x0) for My\\Prefix\\';`},
		{"case 20", nil,
			`<?php
$var = B"\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = B"\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = B"$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = B<<<HEREDOC
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC;
$var = B<<<HEREDOC
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC;
$var = B<<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`,
			`<?php
$var = B"\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = B"\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = B"$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = B<<<HEREDOC
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC;
$var = B<<<HEREDOC
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC;
$var = B<<<'NOWDOC'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC;
`},
		{"case 21", nil,
			`<?php
$var = B"\e\f\n\r\t\v \\ \$ \"";
$var = B"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = B<<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = B<<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`,
			`<?php
$var = B"\e\f\n\r\t\v \\ \$ \"";
$var = B"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = B<<<HEREDOC
\e\f\n\r\t\v \\ \$
HEREDOC;
$var = B<<<HEREDOC
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC;
`},
		{"case 22", nil,
			`<?php
$var = B"\0 \00 \000 \0000 \00000";
$var = B"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = B<<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = B<<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`,
			`<?php
$var = B"\0 \00 \000 \0000 \00000";
$var = B"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = B<<<HEREDOC
\0 \00 \000 \0000 \00000
HEREDOC;
$var = B<<<HEREDOC
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC;
`},
		{"case 23", nil,
			`<?php
$var = B"\xA \x99 \u{0}";
$var = B"$foo \xA \x99 \u{0} ${bar}";
$var = B<<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = B<<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`,
			`<?php
$var = B"\xA \x99 \u{0}";
$var = B"$foo \xA \x99 \u{0} ${bar}";
$var = B<<<HEREDOC
\xA \x99 \u{0}
HEREDOC;
$var = B<<<HEREDOC
$foo \xA \x99 \u{0} ${bar}
HEREDOC;
`},
		{"case 24", nil,
			`<?php
$var = B'backslash \ not escaped';
$var = B'code coverage';
$var = B"backslash \\ already escaped";
$var = B"code coverage";
$var = B<<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = B<<<HEREDOC
code coverage
HEREDOC;
$var = B<<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = B<<<'NOWDOC'
code coverage
NOWDOC;
`,
			`<?php
$var = B'backslash \ not escaped';
$var = B'code coverage';
$var = B"backslash \\ already escaped";
$var = B"code coverage";
$var = B<<<HEREDOC
backslash \\ already escaped
HEREDOC;
$var = B<<<HEREDOC
code coverage
HEREDOC;
$var = B<<<'NOWDOC'
backslash \\ already escaped
NOWDOC;
$var = B<<<'NOWDOC'
code coverage
NOWDOC;
`},
		{"case 25", map[string]any{"double_quoted": "unescape"},
			`<?php
$var = B"\A\a \' \8\9 \xZ \u";
$var = B"$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = B"\A\a \' \8\9 \xZ \u";
$var = B"$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 26", map[string]any{"heredoc": "unescape"},
			`<?php
$var = B<<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = B<<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`,
			`<?php
$var = B<<<HEREDOC
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC;
$var = B<<<HEREDOC
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC;
`},
		{"case 27", nil,
			`<?php
$var = "\bar";
$var = "\\bar";
$var = "\\\bar";
$var = "\\\\bar";
$var = "\\\\\bar";
$var = "\\\\\\bar";`,
			`<?php
$var = "\\bar";
$var = "\\bar";
$var = "\\\\bar";
$var = "\\\\bar";
$var = "\\\\\\bar";
$var = "\\\\\\bar";`},
		{"case 28", map[string]any{"single_quoted": "escape"},
			`<?php
$var = '\bar';
$var = '\\bar';
$var = '\\\bar';
$var = '\\\\bar';
$var = '\\\\\bar';
$var = '\\\\\\bar';`,
			`<?php
$var = '\\bar';
$var = '\\bar';
$var = '\\\\bar';
$var = '\\\\bar';
$var = '\\\\\\bar';
$var = '\\\\\\bar';`},
		{"case 29", nil,
			`<?php
$var = <<<TXT
\bar
\\bar
\\\bar
\\\\bar
\\\\\bar
\\\\\\bar
TXT;
`,
			`<?php
$var = <<<TXT
\\bar
\\bar
\\\\bar
\\\\bar
\\\\\\bar
\\\\\\bar
TXT;
`},
		{"case 30", nil,
			`<?php
$var = <<<'TXT'
\bar
\\bar
\\\bar
\\\\bar
\\\\\bar
\\\\\\bar
TXT;
`,
			`<?php
$var = <<<'TXT'
\bar
\\bar
\\\bar
\\\\bar
\\\\\bar
\\\\\\bar
TXT;
`},
		{"unescaped backslashes in single quoted string - backslash", nil,
			`<?php
'\\';
'\\\\';
'\\\\\\';`,
			`<?php
'\\';
'\\\\';
'\\\\\\';`},
		{"unescaped backslashes in single quoted string - reserved double quote", nil,
			`<?php
'\"';
'\\"';
'\\\"';
'\\\\"';
'\\\\\"';
'\\\\\\"';
'\\\\\\\"';
'\\\\\\\\"';`,
			`<?php
'\"';
'\"';
'\\\"';
'\\\"';
'\\\\\"';
'\\\\\"';
'\\\\\\\"';
'\\\\\\\"';`},
		{"unescaped backslashes in single quoted string - reserved chars", nil,
			`<?php
'\b';
'\\b';
'\\\b';
'\\\\b';
'\\\\\b';
'\\\\\\b';
'\\\\\\\b';
'\\\\\\\\b';
'\$v';
'\\$v';
'\{$v}';
'\\{$v}';
'\n';
'\\n';`,
			`<?php
'\b';
'\b';
'\\\b';
'\\\b';
'\\\\\b';
'\\\\\b';
'\\\\\\\b';
'\\\\\\\b';
'\$v';
'\$v';
'\{$v}';
'\{$v}';
'\n';
'\n';`},
		{"unescaped backslashes in double quoted string - backslash", map[string]any{"double_quoted": "unescape"},
			`<?php
"\\";
"\\\\";
"\\\\\\";`,
			`<?php
"\\";
"\\\\";
"\\\\\\";`},
		{"unescaped backslashes in double quoted string - reserved chars", map[string]any{"double_quoted": "unescape"},
			`<?php
"\b";
"\\b";
"\\\b";
"\\\\b";
"\\\\\b";
"\\\\\\b";
"\\\\\\\b";
"\\\\\\\\b";
"\$v";
"\\$v";
"\{$v}";
"\\{$v}";
"\n";
"\\n";`,
			`<?php
"\b";
"\b";
"\\\b";
"\\\b";
"\\\\\b";
"\\\\\b";
"\\\\\\\b";
"\\\\\\\b";
"\$v";
"\\$v";
"\{$v}";
"\\{$v}";
"\n";
"\\n";`},
		{"unescaped backslashes in heredoc - backslash", map[string]any{"heredoc": "unescape"},
			`<?php
<<<EOD
\
\\
\\\
\\\\
\\\\\
\\\\\\
\\\\\\\
EOD;`,
			`<?php
<<<EOD
\
\
\\\
\\\
\\\\\
\\\\\
\\\\\\\
EOD;`},
		{"unescaped backslashes in heredoc - reserved single quote", map[string]any{"heredoc": "unescape"},
			`<?php
<<<EOD
\'
\\'
\\\'
\\\\'
\\\\\'
\\\\\\'
\\\\\\\'
EOD;`,
			`<?php
<<<EOD
\'
\'
\\\'
\\\'
\\\\\'
\\\\\'
\\\\\\\'
EOD;`},
		{"unescaped backslashes in heredoc - reserved double quote", map[string]any{"heredoc": "unescape"},
			`<?php
<<<EOD
\"
\\"
\\\"
\\\\"
\\\\\"
\\\\\\"
\\\\\\\"
EOD;`,
			`<?php
<<<EOD
\"
\"
\\\"
\\\"
\\\\\"
\\\\\"
\\\\\\\"
EOD;`},
		{"unescaped backslashes in heredoc - reserved chars", map[string]any{"heredoc": "unescape"},
			`<?php
<<<EOD
\$v
\\$v
\{$v}
\\{$v}
\n
\\n
\b
\\b
\\\b
\\\\b
\\\\\b
\\\\\\b
\\\\\\\b
EOD;`,
			`<?php
<<<EOD
\$v
\\$v
\{$v}
\\{$v}
\n
\\n
\b
\b
\\\b
\\\b
\\\\\b
\\\\\b
\\\\\\\b
EOD;`},
		{"ignored mixed implicit backslashes in single quoted string", map[string]any{"single_quoted": "ignore"},
			`<?php
$var = 'a\b\\c';`,
			`<?php
$var = 'a\b\\c';`},
		{"ignored mixed implicit backslashes in double quoted string", map[string]any{"double_quoted": "ignore"},
			`<?php
$var = "a\b\\c";`,
			`<?php
$var = "a\b\\c";`},
		{"ignored mixed implicit backslashes in heredoc", map[string]any{"heredoc": "ignore"},
			`<?php
$var = <<<HEREDOC
    a\b\\c
    HEREDOC;`,
			`<?php
$var = <<<HEREDOC
    a\b\\c
    HEREDOC;`},
		{"execution operator", nil,
			"<?php $var = `ls a\\b`;",
			"<?php $var = `ls a\\\\b`;"},
	})
}
