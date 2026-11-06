package rules

import "testing"

// Cases ported from PHP-CS-Fixer's EscapeImplicitBackslashesFixerTest
// (tests/Fixer/StringNotation); every expected output is what PHP-CS-Fixer
// produces for the input. The deprecated fixer runs StringImplicitBackslashes
// with the options it translates.
func TestEscapeImplicitBackslashesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	const name = `PhpCsFixer\Fixer\StringNotation\EscapeImplicitBackslashesFixer`
	for _, c := range []aliasUpstreamCase{
		{"case 0", nil,
			`<?php $var = 'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = 'String (\\\'\r\n\x0) for My\Prefix\\';`},
		{"case 1", map[string]any{"single_quoted": true},
			`<?php $var = 'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = 'String (\\\'\\r\\n\\x0) for My\\Prefix\\';`},
		{"case 2", nil,
			`<?php
$var = "\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = "\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = "$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = <<<HEREDOC_SYNTAX
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`,
			`<?php
$var = "\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = "\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = "$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = <<<HEREDOC_SYNTAX
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`},
		{"case 3", nil,
			`<?php
$var = "\e\f\n\r\t\v \\ \$ \"";
$var = "$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = <<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = "\e\f\n\r\t\v \\ \$ \"";
$var = "$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = <<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`},
		{"case 4", nil,
			`<?php
$var = "\0 \00 \000 \0000 \00000";
$var = "$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = <<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = "\0 \00 \000 \0000 \00000";
$var = "$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = <<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`},
		{"case 5", nil,
			`<?php
$var = "\xA \x99 \u{0}";
$var = "$foo \xA \x99 \u{0} ${bar}";
$var = <<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = "\xA \x99 \u{0}";
$var = "$foo \xA \x99 \u{0} ${bar}";
$var = <<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`},
		{"case 6", nil,
			`<?php
$var = 'backslash \\ already escaped';
$var = 'code coverage';
$var = "backslash \\ already escaped";
$var = "code coverage";
$var = <<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`,
			`<?php
$var = 'backslash \\ already escaped';
$var = 'code coverage';
$var = "backslash \\ already escaped";
$var = "code coverage";
$var = <<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = <<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`},
		{"case 7", map[string]any{"double_quoted": false},
			`<?php
$var = "\A\a \' \8\9 \xZ \u";
$var = "$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = "\A\a \' \8\9 \xZ \u";
$var = "$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 8", map[string]any{"heredoc_syntax": false},
			`<?php
$var = <<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = <<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = <<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
`},
		{"case 9", nil,
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`},
		{"case 10", map[string]any{"single_quoted": true},
			`<?php $var = b'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = b'String (\\\'\\r\\n\\x0) for My\\Prefix\\';`},
		{"case 11", nil,
			`<?php
$var = b"\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = b"\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = b"$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = b<<<HEREDOC_SYNTAX
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`,
			`<?php
$var = b"\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = b"\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = b"$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = b<<<HEREDOC_SYNTAX
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`},
		{"case 12", nil,
			`<?php
$var = b"\e\f\n\r\t\v \\ \$ \"";
$var = b"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = b<<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = b"\e\f\n\r\t\v \\ \$ \"";
$var = b"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = b<<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`},
		{"case 13", nil,
			`<?php
$var = b"\0 \00 \000 \0000 \00000";
$var = b"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = b<<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = b"\0 \00 \000 \0000 \00000";
$var = b"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = b<<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`},
		{"case 14", nil,
			`<?php
$var = b"\xA \x99 \u{0}";
$var = b"$foo \xA \x99 \u{0} ${bar}";
$var = b<<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = b"\xA \x99 \u{0}";
$var = b"$foo \xA \x99 \u{0} ${bar}";
$var = b<<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`},
		{"case 15", nil,
			`<?php
$var = b'backslash \\ already escaped';
$var = b'code coverage';
$var = b"backslash \\ already escaped";
$var = b"code coverage";
$var = b<<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`,
			`<?php
$var = b'backslash \\ already escaped';
$var = b'code coverage';
$var = b"backslash \\ already escaped";
$var = b"code coverage";
$var = b<<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = b<<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`},
		{"case 16", map[string]any{"double_quoted": false},
			`<?php
$var = b"\A\a \' \8\9 \xZ \u";
$var = b"$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = b"\A\a \' \8\9 \xZ \u";
$var = b"$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 17", map[string]any{"heredoc_syntax": false},
			`<?php
$var = b<<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = b<<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = b<<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
`},
		{"case 18", nil,
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`},
		{"case 19", map[string]any{"single_quoted": true},
			`<?php $var = B'String (\\\'\r\n\x0) for My\Prefix\\';`,
			`<?php $var = B'String (\\\'\\r\\n\\x0) for My\\Prefix\\';`},
		{"case 20", nil,
			`<?php
$var = B"\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z";
$var = B"\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z \' \8\9 \xZ \u";
$var = B"$foo \A \a \' \8\9 \xZ \u ${bar}";
$var = B<<<HEREDOC_SYNTAX
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\"
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \A \a \" \' \8\9 \xZ \u ${bar}
HEREDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`,
			`<?php
$var = B"\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z";
$var = B"\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z \\' \\8\\9 \\xZ \\u";
$var = B"$foo \\A \\a \\' \\8\\9 \\xZ \\u ${bar}";
$var = B<<<HEREDOC_SYNTAX
\\A\\B\\C\\D\\E\\F\\G\\H\\I\\J\\K\\L\\M\\N\\O\\P\\Q\\R\\S\\T\\U\\V\\W\\X\\Y\\Z
\\a\\b\\c\\d\\g\\h\\i\\j\\k\\l\\m\\o\\p\\q\\s\\w\\y\\z
\\"
\\'
\\8\\9
\\xZ
\\u
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \\A \\a \\" \\' \\8\\9 \\xZ \\u ${bar}
HEREDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
\A\B\C\D\E\F\G\H\I\J\K\L\M\N\O\P\Q\R\S\T\U\V\W\X\Y\Z
\a\b\c\d\g\h\i\j\k\l\m\o\p\q\s\w\y\z
\'
\8\9
\xZ
\u
NOWDOC_SYNTAX;
`},
		{"case 21", nil,
			`<?php
$var = B"\e\f\n\r\t\v \\ \$ \"";
$var = B"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = B<<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = B"\e\f\n\r\t\v \\ \$ \"";
$var = B"$foo \e\f\n\r\t\v \\ \$ \" ${bar}";
$var = B<<<HEREDOC_SYNTAX
\e\f\n\r\t\v \\ \$
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \e\f\n\r\t\v \\ \$ ${bar}
HEREDOC_SYNTAX;
`},
		{"case 22", nil,
			`<?php
$var = B"\0 \00 \000 \0000 \00000";
$var = B"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = B<<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = B"\0 \00 \000 \0000 \00000";
$var = B"$foo \0 \00 \000 \0000 \00000 ${bar}";
$var = B<<<HEREDOC_SYNTAX
\0 \00 \000 \0000 \00000
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \0 \00 \000 \0000 \00000 ${bar}
HEREDOC_SYNTAX;
`},
		{"case 23", nil,
			`<?php
$var = B"\xA \x99 \u{0}";
$var = B"$foo \xA \x99 \u{0} ${bar}";
$var = B<<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = B"\xA \x99 \u{0}";
$var = B"$foo \xA \x99 \u{0} ${bar}";
$var = B<<<HEREDOC_SYNTAX
\xA \x99 \u{0}
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo \xA \x99 \u{0} ${bar}
HEREDOC_SYNTAX;
`},
		{"case 24", nil,
			`<?php
$var = B'backslash \\ already escaped';
$var = B'code coverage';
$var = B"backslash \\ already escaped";
$var = B"code coverage";
$var = B<<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`,
			`<?php
$var = B'backslash \\ already escaped';
$var = B'code coverage';
$var = B"backslash \\ already escaped";
$var = B"code coverage";
$var = B<<<HEREDOC_SYNTAX
backslash \\ already escaped
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
code coverage
HEREDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
backslash \\ already escaped
NOWDOC_SYNTAX;
$var = B<<<'NOWDOC_SYNTAX'
code coverage
NOWDOC_SYNTAX;
`},
		{"case 25", map[string]any{"double_quoted": false},
			`<?php
$var = B"\A\a \' \8\9 \xZ \u";
$var = B"$foo \A\a \' \8\9 \xZ \u ${bar}";`,
			`<?php
$var = B"\A\a \' \8\9 \xZ \u";
$var = B"$foo \A\a \' \8\9 \xZ \u ${bar}";`},
		{"case 26", map[string]any{"heredoc_syntax": false},
			`<?php
$var = B<<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
`,
			`<?php
$var = B<<<HEREDOC_SYNTAX
\A\Z
\a\z
\'
\8\9
\xZ
\u
HEREDOC_SYNTAX;
$var = B<<<HEREDOC_SYNTAX
$foo
\A\Z
\a\z
\'
\8\9
\xZ
\u
${bar}
HEREDOC_SYNTAX;
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
		{"case 28", map[string]any{"single_quoted": true},
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
		{"execution operator", nil,
			"<?php $var = `ls a\\b`;",
			"<?php $var = `ls a\\\\b`;"},
	} {
		f, ok := ByName(name)
		if c.cfg != nil {
			var options map[string]any
			options, ok = DeprecatedOptions(name, c.cfg)
			f = StringImplicitBackslashes{}.WithConfig(options)
		}
		if !ok {
			t.Fatalf("%s: %s does not resolve", c.name, name)
		}
		aliasRunUpstreamCases(t, f, []aliasUpstreamCase{{c.name, nil, c.input, c.expected}})
	}
}
