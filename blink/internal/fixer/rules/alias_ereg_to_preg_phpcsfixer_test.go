package rules

import "testing"

// Cases ported from PHP-CS-Fixer's EregToPregFixerTest (tests/Fixer/Alias), then
// invented ones (named "extra"/"analyzer"/...); every expected output is what
// PHP-CS-Fixer 3.95 produces for the input.
func TestEregToPregPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, EregToPreg{}, []aliasUpstreamCase{
		{"Fix#0", nil, "<?php $x = 1;", "<?php $x = 1;"},
		{"Fix#1", nil, "<?php $x = \"ereg\";", "<?php $x = \"ereg\";"},
		{"Fix#2", nil, "<?php $x = ereg(\"[A-Z]\".\"foo\", $m);", "<?php $x = ereg(\"[A-Z]\".\"foo\", $m);"},
		{"Fix#3", nil, "<?php $x = ereg(\"^*broken\", $m);", "<?php $x = ereg(\"^*broken\", $m);"},
		{"Fix#4", nil, "<?php $x = Foo::split(\"[A-Z]\", $m);", "<?php $x = Foo::split(\"[A-Z]\", $m);"},
		{"Fix#5", nil, "<?php $x = $foo->split(\"[A-Z]\", $m);", "<?php $x = $foo->split(\"[A-Z]\", $m);"},
		{"Fix#6", nil, "<?php $x = Foo\\split(\"[A-Z]\", $m);", "<?php $x = Foo\\split(\"[A-Z]\", $m);"},
		{"Fix#7", nil, "<?php $x = ereg('[A-Z]');", "<?php $x = preg_match('/[A-Z]/D');"},
		{"Fix#8", nil, "<?php $x = ereg('[A-Z]', $m);", "<?php $x = preg_match('/[A-Z]/D', $m);"},
		{"Fix#9", nil, "<?php $x = ereg(\"[A-Z]\", $m);", "<?php $x = preg_match(\"/[A-Z]/D\", $m);"},
		{"Fix#10", nil, "<?php $x = eregi(\"[A-Z]\", $m);", "<?php $x = preg_match(\"/[A-Z]/Di\", $m);"},
		{"Fix#11", nil, "<?php $x = ereg(\"/[AZ]\", $m);", "<?php $x = preg_match(\"#/[AZ]#D\", $m);"},
		{"Fix#12", nil, "<?php $x = ereg(\"[AZ]/\", $m);", "<?php $x = preg_match(\"#[AZ]/#D\", $m);"},
		{"Fix#13", nil, "<?php $x = ereg(\"#[A]/\", $m);", "<?php $x = preg_match(\"!#[A]/!D\", $m);"},
		{"Fix#14", nil, "<?php $x = ereg(\"##[A!]//\", $m);", "<?php $x = preg_match(\"!##[A\\!]//!D\", $m);"},
		{"Fix#15", nil, "<?php $x = ereg(\"##[A!!]//\", $m);", "<?php $x = preg_match(\"/##[A!!]\\/\\//D\", $m);"},
		{"Fix#16", nil, "<?php $x = ereg(\"##[A!!]///\", $m);", "<?php $x = preg_match(\"#\\#\\#[A!!]///#D\", $m);"},
		{"Fix#17", nil, "<?php $x = ereg_replace(\"[A-Z]\", \"\", $m);", "<?php $x = preg_replace(\"/[A-Z]/D\", \"\", $m);"},
		{"Fix#18", nil, "<?php $x = eregi_replace(\"[A-Z]\", \"\", $m);", "<?php $x = preg_replace(\"/[A-Z]/Di\", \"\", $m);"},
		{"Fix#19", nil, "<?php $x = split(\"[A-Z]\", $m);", "<?php $x = preg_split(\"/[A-Z]/D\", $m);"},
		{"Fix#20", nil, "<?php $x = spliti(\"[A-Z]\", $m);", "<?php $x = preg_split(\"/[A-Z]/Di\", $m);"},
		{"binary lowercase", nil, "<?php $x = spliti(b\"[A-Z]\", $m);", "<?php $x = preg_split(b\"/[A-Z]/Di\", $m);"},
		{"binary uppercase", nil, "<?php $x = spliti(B\"[A-Z]\", $m);", "<?php $x = preg_split(B\"/[A-Z]/Di\", $m);"},
		{"Fix81#23", nil, "<?php $x = spliti(...);", "<?php $x = spliti(...);"},
		{"extra #1", nil, "<?php\nnamespace Acme;\n\nuse function Other\\split;\n\n$parts = split(',', $csv);", "<?php\nnamespace Acme;\n\nuse function Other\\split;\n\n$parts = split(',', $csv);"},
		{"extra #2", nil, "<?php\nnamespace Acme;\n\n$parts = split(',', $csv);\n$parts = \\split(',', $csv);\n$ok = ereg('^[a-z]+$', $name);\n$ok = ereg('(a|b', $name);\n$ok = ereg('[z-a]', $name);\n$ok = ereg('a**', $name);\n$ok = ereg('$*', $name);\n$ok = ereg('x{2,3}y{2}', $name);\n$ok = ereg('(?=a)b', $name);\n$ok = ereg('\\\\d+', $name);\n$ok = ereg('\\d+', $name);\n$ok = ereg(\"\\\\.\", $name);\n$ok = ereg('[[:alpha:]]', $name);\n$ok = ereg('[]a]', $name);\n$ok = ereg('', $name);\n$ok = ereg('a\\\\', $name);\n$ok = ereg($pattern, $name);\n$ok = ereg(\"a$b\", $name);", "<?php\nnamespace Acme;\n\n$parts = preg_split('/,/D', $csv);\n$parts = \\preg_split('/,/D', $csv);\n$ok = preg_match('/^[a-z]+$/D', $name);\n$ok = ereg('(a|b', $name);\n$ok = ereg('[z-a]', $name);\n$ok = ereg('a**', $name);\n$ok = ereg('$*', $name);\n$ok = preg_match('/x{2,3}y{2}/D', $name);\n$ok = preg_match('/(?=a)b/D', $name);\n$ok = preg_match('/\\\\d+/D', $name);\n$ok = preg_match('/\\d+/D', $name);\n$ok = preg_match(\"/\\\\./D\", $name);\n$ok = preg_match('/[[:alpha:]]/D', $name);\n$ok = preg_match('/[]a]/D', $name);\n$ok = preg_match('//D', $name);\n$ok = preg_match('/a\\\\/D', $name);\n$ok = ereg($pattern, $name);\n$ok = ereg(\"a$b\", $name);"},
		// blink has no PCRE: these are patterns PHP's preg_match refuses
		// (some only at match time, past the backtrack limit)
		{"pcre invalid", nil, `<?php
$a = ereg('\1', $s);
$a = ereg('(a)\2', $s);
$a = ereg('\8', $s);
$a = ereg('[\d-z]', $s);
$a = ereg('[\w-.]', $s);
$a = ereg('(?<!a+)b', $s);
$a = ereg('(?<=a*)b', $s);
$a = ereg('[[=a=]]', $s);
$a = ereg('[[.a.]]', $s);
$a = ereg('(?<1a>x)', $s);
$a = ereg('[:alpha:]', $s);
$a = ereg('a{65536}', $s);
$a = ereg('[a-\]]', $s);
$a = ereg('(?:(?:a?){1,20}){1,20}\b', $s);
$a = ereg('(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?!)', $s);`, `<?php
$a = ereg('\1', $s);
$a = ereg('(a)\2', $s);
$a = ereg('\8', $s);
$a = ereg('[\d-z]', $s);
$a = ereg('[\w-.]', $s);
$a = ereg('(?<!a+)b', $s);
$a = ereg('(?<=a*)b', $s);
$a = ereg('[[=a=]]', $s);
$a = ereg('[[.a.]]', $s);
$a = ereg('(?<1a>x)', $s);
$a = ereg('[:alpha:]', $s);
$a = ereg('a{65536}', $s);
$a = ereg('[a-\]]', $s);
$a = ereg('(?:(?:a?){1,20}){1,20}\b', $s);
$a = ereg('(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?:|)(?!)', $s);`},
		// patterns PHP's preg_match accepts and Go's regexp parser does not
		{"pcre valid", nil, `<?php
$a = ereg('a{1001}', $s);
$a = ereg('a{2000}', $s);
$a = ereg('[\b]', $s);
$a = ereg('(?<=a)b', $s);
$a = ereg('(?i)a', $s);
$a = ereg('(a)\1', $s);
$a = ereg('(?>a|b)+[[:^digit:]]\x41', $s);
$a = ereg('(?:|)(?:|)(?:|)(?:|)(?!)', $s);
$a = ereg('[[:alpha:]-]', $s);
$a = ereg('(?i:a)(?-i)b', $s);
$a = ereg('(?<!\d{2}|x)y', $s);
$a = ereg('^[^.[]+', $s);`, `<?php
$a = preg_match('/a{1001}/D', $s);
$a = preg_match('/a{2000}/D', $s);
$a = preg_match('/[\b]/D', $s);
$a = preg_match('/(?<=a)b/D', $s);
$a = preg_match('/(?i)a/D', $s);
$a = preg_match('/(a)\1/D', $s);
$a = preg_match('/(?>a|b)+[[:^digit:]]\x41/D', $s);
$a = preg_match('/(?:|)(?:|)(?:|)(?:|)(?!)/D', $s);
$a = preg_match('/[[:alpha:]-]/D', $s);
$a = preg_match('/(?i:a)(?-i)b/D', $s);
$a = preg_match('/(?<!\d{2}|x)y/D', $s);
$a = preg_match('/^[^.[]+/D', $s);`},
		{"latin-1 bytes", nil, "<?php $a = ereg('\xe9t\xe9', $s);", "<?php $a = preg_match('/\xe9t\xe9/D', $s);"},
	})
}
