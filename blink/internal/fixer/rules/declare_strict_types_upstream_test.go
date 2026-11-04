package rules

import "testing"

// Cases ported from PHP-CS-Fixer's tests/Fixer/Strict/DeclareStrictTypesFixerTest.php (v3.95.27).
func TestDeclareStrictTypesUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, DeclareStrictTypes{}, declareStrictTypesUpstreamCases)
}

var declareStrictTypesUpstreamCases = []upstreamCase{
	{"provideFixCases/remove existing declaration", map[string]any{"strategy": "remove"}, "<?php ", "<?php declare(strict_types=1);"},
	{"provideFixCases/remove existing declaration with ticks", map[string]any{"strategy": "remove"}, "<?php declare(ticks=1);", "<?php declare(strict_types=1, ticks=1);"},
	{"provideFixCases/remove existing declaration with ticks before", map[string]any{"strategy": "remove"}, "<?php declare(ticks=1);", "<?php declare(ticks=1, strict_types=1);"},
	{"provideFixCases/remove existing declaration with strict_types=0", map[string]any{"strategy": "remove"}, "<?php ", "<?php declare(strict_types=0);"},
	{"provideFixCases/remove existing declaration with comments", map[string]any{"strategy": "remove"}, "<?php /**/ ", "<?php /**/ declare(strict_types=1);"},
	{"provideFixCases/update multi-declare statement", map[string]any{"strategy": "enforce"}, "<?php declare(strict_types=1, ticks=1);", "<?php declare(strict_types=0, ticks=1);"},
	{"provideFixCases#6", nil, "<?php\ndeclare(ticks=1);\n//\ndeclare(strict_types=1);\n\nnamespace A\\B\\C;\nclass A {\n}", ""},
	{"provideFixCases#7", nil, "<?php declare/* A b C*/(strict_types=1);", ""},
	{"provideFixCases#8", nil, "<?php declare(strict_types=1, ticks=1);", ""},
	{"provideFixCases#9", nil, "<?php declare(ticks=1, strict_types=1);", ""},
	{"provideFixCases#10", nil, "<?php declare(ticks=1, strict_types=1);", "<?php declare(ticks=1, strict_types=0);"},
	{"provideFixCases/monolithic file with closing tag", nil, "<?php /**/ /**/ deClarE  (strict_types=1)    ?>", "<?php /**/ /**/ deClarE  (STRICT_TYPES=1)    ?>"},
	{"provideFixCases/monolithic file with closing tag and extra new line", nil, "<?php /**/ /**/ deClarE  (strict_types=1)    ?>\n", "<?php /**/ /**/ deClarE  (STRICT_TYPES=1)    ?>\n"},
	{"provideFixCases/monolithic file with closing tag and extra content", nil, "<?php /**/ /**/ deClarE  (STRICT_TYPES=1)    ?>Test", ""},
	{"provideFixCases#14", nil, "<?php            DECLARE  (    strict_types=1   )   ;", ""},
	{"provideFixCases#15", nil, "<?php\n                /**/\n                declare(strict_types=1);", ""},
	{"provideFixCases#16", nil, "<?php declare(strict_types=1);\n\n                phpinfo();", "<?php\n\n                phpinfo();"},
	{"provideFixCases#17", nil, "<?php declare(strict_types=1);\n\n/**\n * Foo\n */\nphpinfo();", "<?php\n\n/**\n * Foo\n */\nphpinfo();"},
	{"provideFixCases#18", nil, "<?php declare(strict_types=1);\n\n// comment after empty line", "<?php\n\n// comment after empty line"},
	{"provideFixCases#19", nil, "<?php declare(strict_types=1);\n// comment without empty line before", "<?php\n// comment without empty line before"},
	{"provideFixCases#20", nil, "<?php declare(strict_types=1);\nphpinfo();", "<?php phpinfo();"},
	{"provideFixCases#21", nil, "<?php declare(strict_types=1);\n$a = 456;\n", "<?php\n$a = 456;\n"},
	{"provideFixCases#22", nil, "<?php declare(strict_types=1);\n/**/", "<?php /**/"},
	{"provideFixCases#23", nil, "<?php declare(strict_types=1);", "<?php declare(strict_types=0);"},
	{"provideFixCases#24", map[string]any{"strategy": "enforce"}, "<?php declare(strict_types=1);", "<?php declare(strict_types=0);"},
	{"provideFixCases#25", map[string]any{"strategy": "add_when_missing"}, "<?php declare(strict_types=0);", ""},
	{"provideFixCases#26", nil, "  <?php echo 123;", ""},
	{"provideFixCases#27", nil, "<?= 123;", ""},
	{"provideFixCases/empty file /wo open tag", nil, "", ""},
	{"provideFixCases/empty file /w open tag", nil, "<?php declare(strict_types=1);", "<?php"},
	{"provideFixCases/non-empty file /wo open tag", nil, "x", ""},
	{"provideFixCases/non-empty file /w open tag", nil, "x<?php", ""},
	{"provideFixCases/file with shebang /w open tag", nil, "#!x\n<?php declare(strict_types=1);", "#!x\n<?php"},
	{"provideFixCases/file with shebang /wo open tag", nil, "#!x\ny", ""},
	{"provideFixCases/file with shebang not followed by open tag", nil, "#!x\n#!not_a_shebang\n<?php", ""},
}
