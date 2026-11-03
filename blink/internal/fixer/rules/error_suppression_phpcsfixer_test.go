package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ErrorSuppressionFixerTest (tests/Fixer/LanguageConstruct), then
// invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.95 produces for the input.
func TestErrorSuppressionPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, ErrorSuppression{}, []portConstructsCase{
		{"provideFixCases#0", nil, "<?php trigger_error(\"This is not a deprecation warning.\"); @f(); ?>", "<?php trigger_error(\"This is not a deprecation warning.\"); @f(); ?>"},
		{"provideFixCases#1", nil, "<?php trigger_error(\"This is not a deprecation warning.\", E_USER_WARNING); ?>", "<?php trigger_error(\"This is not a deprecation warning.\", E_USER_WARNING); ?>"},
		{"provideFixCases#2", nil, "<?php A\\B\\trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php A\\B\\trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#3", nil, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#4", map[string]any{"mute_deprecation_error": false}, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#5", nil, "<?php \\trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php @\\trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#6", nil, "<?php echo \"test\";trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php echo \"test\";@trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#7", nil, "<?php //\nTrigger_Error/**/(\"This is a deprecation warning.\", E_USER_DEPRECATED/***/); ?>", "<?php //\n@Trigger_Error/**/(\"This is a deprecation warning.\", E_USER_DEPRECATED/***/); ?>"},
		{"provideFixCases#8", nil, "<?php new trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php new trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#9", nil, "<?php new \\trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php new \\trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#10", nil, "<?php $foo->trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php $foo->trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#11", nil, "<?php Foo::trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>", "<?php Foo::trigger_error(\"This is not a deprecation warning.\", E_USER_DEPRECATED); ?>"},
		{"provideFixCases#12", map[string]any{"mute_deprecation_error": true, "noise_remaining_usages": true}, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @mkdir(\"dir\"); ?>", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); mkdir(\"dir\"); ?>"},
		{"provideFixCases#13", map[string]any{"noise_remaining_usages": true}, "<?php @$foo->isBar(); ?>", "<?php $foo->isBar(); ?>"},
		{"provideFixCases#14", map[string]any{"noise_remaining_usages": true}, "<?php @Foo::isBar(); ?>", "<?php Foo::isBar(); ?>"},
		{"provideFixCases#15", map[string]any{"mute_deprecation_error": true, "noise_remaining_usages": true, "noise_remaining_usages_exclude": []any{"mkdir"}}, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @mkdir(\"dir\"); ?>", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @mkdir(\"dir\"); ?>"},
		{"provideFixCases#16", map[string]any{"mute_deprecation_error": true, "noise_remaining_usages": true, "noise_remaining_usages_exclude": []any{"mkdir"}}, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @mkdir(\"dir\"); @unlink($path); ?>", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @mkdir(\"dir\"); unlink($path); ?>"},
		{"provideFixCases#17", map[string]any{"mute_deprecation_error": true, "noise_remaining_usages": true, "noise_remaining_usages_exclude": []any{"trigger_error"}}, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @trigger_error(\"This is not a deprecation warning.\", E_USER_WARNING); ?>", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED); @trigger_error(\"This is not a deprecation warning.\", E_USER_WARNING); ?>"},
		{"provideFixCases#18", nil, "<?php trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED, );", "<?php @trigger_error(\"This is a deprecation warning.\", E_USER_DEPRECATED, );"},
		{"provideFix81Cases#0", nil, "<?php $a = trigger_error(...);", "<?php $a = trigger_error(...);"},
		{"extra#0", nil, "<?php\nnamespace A;\nfunction trigger_error() {}\ntrigger_error('x', E_USER_DEPRECATED);\n", "<?php\nnamespace A;\nfunction trigger_error() {}\ntrigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#1", nil, "<?php\nnamespace A;\nuse function B\\trigger_error;\ntrigger_error('x', E_USER_DEPRECATED);\n", "<?php\nnamespace A;\nuse function B\\trigger_error;\ntrigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#2", nil, "<?php\nnamespace A;\nuse function trigger_error;\ntrigger_error('x', E_USER_DEPRECATED);\n", "<?php\nnamespace A;\nuse function trigger_error;\n@trigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#3", nil, "<?php\nnamespace A;\nuse function B\\trigger_error, B\\other;\ntrigger_error('x', E_USER_DEPRECATED);\n", "<?php\nnamespace A;\nuse function B\\trigger_error, B\\other;\n@trigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#4", nil, "<?php\nnamespace A;\ntrigger_error('x', \\E_USER_DEPRECATED);\ntrigger_error(sprintf('%s', $a), E_USER_DEPRECATED);\n", "<?php\nnamespace A;\n@trigger_error('x', \\E_USER_DEPRECATED);\n@trigger_error(sprintf('%s', $a), E_USER_DEPRECATED);\n"},
		{"extra#5", nil, "<?php\nnamespace A;\nclass B { public function trigger_error() {} public function f() { trigger_error('x', E_USER_DEPRECATED); } }\n", "<?php\nnamespace A;\nclass B { public function trigger_error() {} public function f() { @trigger_error('x', E_USER_DEPRECATED); } }\n"},
		{"extra#6", nil, "<?php\nnamespace A {\n    function trigger_error() {}\n    trigger_error('x', E_USER_DEPRECATED);\n}\nnamespace {\n    trigger_error('x', E_USER_DEPRECATED);\n}\n", "<?php\nnamespace A {\n    function trigger_error() {}\n    trigger_error('x', E_USER_DEPRECATED);\n}\nnamespace {\n    @trigger_error('x', E_USER_DEPRECATED);\n}\n"},
		{"extra#7", nil, "<?php\necho trigger_error('x', E_USER_DEPRECATED);\n$ok = trigger_error('x', E_USER_DEPRECATED) or die;\n", "<?php\necho @trigger_error('x', E_USER_DEPRECATED);\n$ok = @trigger_error('x', E_USER_DEPRECATED) or die;\n"},
		{"extra#8", nil, "<?php\n#[Attr, trigger_error('x', E_USER_DEPRECATED)]\nfunction f() { trigger_error('x', E_USER_DEPRECATED); }\n", "<?php\n#[Attr, trigger_error('x', E_USER_DEPRECATED)]\nfunction f() { @trigger_error('x', E_USER_DEPRECATED); }\n"},
		{"extra#9", map[string]any{"noise_remaining_usages": true, "noise_remaining_usages_exclude": []any{"unlink"}}, "<?php\n@$a['x'];\n@file_get_contents($f);\n@\\unlink($f);\n@ fopen($f);\n$x = @Unlink($y);\n", "<?php\n$a['x'];\nfile_get_contents($f);\n@\\unlink($f);\n fopen($f);\n$x = Unlink($y);\n"},
		{"extra#10", map[string]any{"noise_remaining_usages": true, "mute_deprecation_error": false}, "<?php\n@trigger_error('x', E_USER_DEPRECATED);\n@trigger_error('x');\n", "<?php\n@trigger_error('x', E_USER_DEPRECATED);\ntrigger_error('x');\n"},
		{"extra#11", nil, "<?php\nif ($a) trigger_error(\"x\", E_USER_DEPRECATED) ?>\n<?php trigger_error('y', E_USER_DEPRECATED);\n", "<?php\nif ($a) @trigger_error(\"x\", E_USER_DEPRECATED) ?>\n<?php @trigger_error('y', E_USER_DEPRECATED);\n"},
		{"extra#12", nil, "<?php\n$f = function () { trigger_error('x', E_USER_DEPRECATED); };\n$g = fn () => trigger_error('x', E_USER_DEPRECATED);\n", "<?php\n$f = function () { @trigger_error('x', E_USER_DEPRECATED); };\n$g = fn () => @trigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#13", nil, "<?php\ndeclare(strict_types=1);\n\nnamespace A;\n\ntrigger_error('x', E_USER_DEPRECATED);\n", "<?php\ndeclare(strict_types=1);\n\nnamespace A;\n\n@trigger_error('x', E_USER_DEPRECATED);\n"},
		{"extra#14", nil, "<?php\nfunction g() { yield from trigger_error('x', E_USER_DEPRECATED); }\n", "<?php\nfunction g() { yield from @trigger_error('x', E_USER_DEPRECATED); }\n"},
	})
}
