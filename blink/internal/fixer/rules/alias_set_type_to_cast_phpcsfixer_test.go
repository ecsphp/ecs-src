package rules

import "testing"

// Cases ported from PHP-CS-Fixer's SetTypeToCastFixerTest (tests/Fixer/Alias), then
// invented ones (named "extra"/"analyzer"/...); every expected output is what
// PHP-CS-Fixer 3.95 produces for the input.
func TestSetTypeToCastPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, SetTypeToCast{}, []aliasUpstreamCase{
		{"null cast", nil, "<?php settype($foo, \"null\");", "<?php $foo = null;"},
		{"null cast comments", nil, "<?php\n# 0\nsettype# 1\n# 2\n(# 3\n# 4\n$foo# 5\n# 6\n,# 7\n# 8\n\"null\"# 9\n)# 10\n;", "<?php\n# 0\n$foo = null# 1\n# 2\n# 3\n# 4\n# 5\n# 6\n# 7\n# 8\n# 9\n# 10\n;"},
		{"array + spacing", nil, "<?php settype  (  $foo  , 'array');", "<?php $foo = (array) $foo;"},
		{"bool + casing", nil, "<?php settype  (  $foo  , \"Bool\");", "<?php $foo = (bool) $foo;"},
		{"boolean with extra space", nil, "<?php settype  (  $foo  , \"boolean\");", "<?php $foo = (bool) $foo;"},
		{"double", nil, "<?php settype($foo, \"double\");", "<?php $foo = (float) $foo;"},
		{"float", nil, "<?php settype($foo, \"float\");", "<?php $foo = (float) $foo;"},
		{"float in loop", nil, "<?php while(a()){settype($foo, \"float\");}", "<?php while(a()){$foo = (float) $foo;}"},
		{"int full caps", nil, "<?php settype($foo, \"INT\");", "<?php $foo = (int) $foo;"},
		{"integer (simple)", nil, "<?php settype($foo, \"integer\");", "<?php $foo = (int) $foo;"},
		{"object", nil, "<?php echo 1; settype($foo, \"object\");", "<?php echo 1; $foo = (object) $foo;"},
		{"string", nil, "<?php settype($foo, \"string\");", "<?php $foo = (string) $foo;"},
		{"string in function body", nil, "<?php function A(){ settype($foo, \"string\"); return $foo; }", "<?php function A(){ $foo = (string) $foo; return $foo; }"},
		{"integer + no space", nil, "<?php settype($foo,\"integer\");", "<?php $foo = (int) $foo;"},
		{"no space comments", nil, "<?php /*0*//*1*/settype/*2*/(/*3*/$foo/*4*/,/*5*/\"integer\"/*6*/)/*7*/;/*8*/", "<?php /*0*//*1*/$foo = (int) $foo/*2*//*3*//*4*//*5*//*6*//*7*/;/*8*/"},
		{"comments with line breaks", nil, "<?php #0\n#1\nsettype#2\n#3\n(#4\n$foo#5\n,#6\n\"integer\"#7\n)#8\n;#9", "<?php #0\n#1\n$foo = (int) $foo#2\n#3\n#4\n#5\n#6\n#7\n#8\n;#9"},
		{"first argument is not a variable", nil, "<?php\n                    namespace A\\B;             // comment\n                    function settype($a, $b){} // \"\n\n                    settype(1, \"double\");\n                ", "<?php\n                    namespace A\\B;             // comment\n                    function settype($a, $b){} // \"\n\n                    settype(1, \"double\");\n                "},
		{"first argument is variable followed by operation", nil, "<?php\n                    namespace A\\B;                // comment\n                    function settype($a, $b){}    // \"\n\n                    settype($foo + 1, \"integer\"); // function must be overridden, so do not fix it\n                ", "<?php\n                    namespace A\\B;                // comment\n                    function settype($a, $b){}    // \"\n\n                    settype($foo + 1, \"integer\"); // function must be overridden, so do not fix it\n                "},
		{"wrong numbers of arguments", nil, "<?php settype($foo, \"integer\", $a);", "<?php settype($foo, \"integer\", $a);"},
		{"other namespace I", nil, "<?php a\\b\\settype($foo, \"integer\", $a);", "<?php a\\b\\settype($foo, \"integer\", $a);"},
		{"other namespace II", nil, "<?php \\b\\settype($foo, \"integer\", $a);", "<?php \\b\\settype($foo, \"integer\", $a);"},
		{"static call", nil, "<?php A::settype($foo, \"integer\");", "<?php A::settype($foo, \"integer\");"},
		{"member call", nil, "<?php $a->settype($foo, \"integer\");", "<?php $a->settype($foo, \"integer\");"},
		{"unknown type", nil, "<?php $a->settype($foo, \"foo\");", "<?php $a->settype($foo, \"foo\");"},
		{"return value used I", nil, "<?php $a = settype($foo, \"integer\");", "<?php $a = settype($foo, \"integer\");"},
		{"return value used II", nil, "<?php a(settype($foo, \"integer\"));", "<?php a(settype($foo, \"integer\"));"},
		{"return value used III", nil, "<?php $a = \"123\"; $b = [settype($a, \"integer\")];", "<?php $a = \"123\"; $b = [settype($a, \"integer\")];"},
		{"return value used IV", nil, "<?php $a = \"123\"; $b = [3 => settype($a, \"integer\")];", "<?php $a = \"123\"; $b = [3 => settype($a, \"integer\")];"},
		{"return value used V", nil, "<?= settype($foo, \"object\");", "<?= settype($foo, \"object\");"},
		{"wrapped statements, fixable after removing the useless parenthesis brace", nil, "<?php\n                    settype(/*1*//*2*/($a), 'int');\n                    settype($b, ('int'));\n                    settype(($c), ('int'));\n                    settype((($d)), (('int')));\n                ", "<?php\n                    settype(/*1*//*2*/($a), 'int');\n                    settype($b, ('int'));\n                    settype(($c), ('int'));\n                    settype((($d)), (('int')));\n                "},
		{"wrapped statements, not-fixable, even after removing the useless parenthesis brace", nil, "<?php\n                    namespace A\\B;                // comment\n                    function settype($a, $b){}    // \"\n\n                    settype($foo1, ((\"integer\").\"1\"));\n                    settype($foo2, (\"1\".(\"integer\")));\n                    settype($foo3, (((\"integer\").\"1\")));\n                    settype((($foo)+1), \"integer\");\n                ", "<?php\n                    namespace A\\B;                // comment\n                    function settype($a, $b){}    // \"\n\n                    settype($foo1, ((\"integer\").\"1\"));\n                    settype($foo2, (\"1\".(\"integer\")));\n                    settype($foo3, (((\"integer\").\"1\")));\n                    settype((($foo)+1), \"integer\");\n                "},
		{"nested is not an issue for this fixer, as these non may be fixed", nil, "<?php\n                    settype($foo, settype($foo, \"double\"));\n                    settype(settype($foo, \"double\"), \"double\");\n                ", "<?php\n                    settype($foo, settype($foo, \"double\"));\n                    settype(settype($foo, \"double\"), \"double\");\n                "},
		{"unknown type II", nil, "<?php settype($foo, \"stringX\");", "<?php settype($foo, \"stringX\");"},
		{"boolean", nil, "<?php settype($foo, \"boolean\", );", "<?php $foo = (bool) $foo;"},
		{"comments with line breaks II", nil, "<?php #0\n#1\nsettype#2\n#3\n(#4\n$foo#5\n,#6\n\"integer\"#7\n,#8\n)#9\n;#10", "<?php #0\n#1\n$foo = (int) $foo#2\n#3\n#4\n#5\n#6\n#7\n#8\n#9\n;#10"},
		{"first argument is complex statement", nil, "<?php settype($foo[1], \"bar\");", "<?php settype($foo[1], \"bar\");"},
		{"extra #1", nil, "<?php\nnamespace Acme;\n\nuse function Other\\settype;\n\nsettype($value, 'int');", "<?php\nnamespace Acme;\n\nuse function Other\\settype;\n\nsettype($value, 'int');"},
		{"extra #2", nil, "<?php\nnamespace Acme;\n\nsettype($value, 'int');\n\\settype($value, 'int');", "<?php\nnamespace Acme;\n\n$value = (int) $value;\n\\settype($value, 'int');"},
		{"extra #3", nil, "<?php\nsettype($value, \"$type\");\nsettype($value, <<<'T'\nint\nT);\nsettype($value, 'int' . '');\nsettype($value, b'int');\nsettype($value, 'NULL');\nsettype($value, '\"int\"');", "<?php\nsettype($value, \"$type\");\nsettype($value, <<<'T'\nint\nT);\nsettype($value, 'int' . '');\nsettype($value, b'int');\n$value = null;\n$value = (int) $value;"},
		{"extra #4", nil, "<?php\nif ($x) settype($value, 'int');\nforeach ($rows as $row) { settype($row, 'string'); }\nsettype($value, 'int') ?>", "<?php\nif ($x) settype($value, 'int');\nforeach ($rows as $row) { $row = (string) $row; }\n$value = (int) $value?>"},
		{"extra #5", nil, "<?php\nsettype($a, 'int'); // trailing\nsettype($b, 'bool')    ;", "<?php\n$a = (int) $a; // trailing\n$b = (bool) $b;"},
	})
}
