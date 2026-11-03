package rules

import "testing"

// Cases ported from PHP-CS-Fixer's EchoTagSyntaxFixerTest (tests/Fixer/PhpTag), then
// invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.95 produces for the input.
func TestEchoTagSyntaxPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, EchoTagSyntax{}, []portConstructsCase{
		{"provideFixCases#0", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php echo 'Foo';", "<?= 'Foo';"},
		{"provideFixCases#1", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php print 'Foo';", "<?= 'Foo';"},
		{"provideFixCases#2", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php echo 'Foo'; ?> PLAIN TEXT", "<?= 'Foo'; ?> PLAIN TEXT"},
		{"provideFixCases#3", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php print 'Foo'; ?> PLAIN TEXT", "<?= 'Foo'; ?> PLAIN TEXT"},
		{"provideFixCases#4", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "PLAIN TEXT<?php echo 'Foo'; ?>", "PLAIN TEXT<?= 'Foo'; ?>"},
		{"provideFixCases#5", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "PLAIN TEXT<?php print 'Foo'; ?>", "PLAIN TEXT<?= 'Foo'; ?>"},
		{"provideFixCases#6", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php echo 'Foo'; ?> <?php echo 'Bar'; ?>", "<?= 'Foo'; ?> <?= 'Bar'; ?>"},
		{"provideFixCases#7", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php print 'Foo'; ?> <?php echo 'Bar'; ?>", "<?= 'Foo'; ?> <?= 'Bar'; ?>"},
		{"provideFixCases#8", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php echo 'Foo'; someThingElse();", "<?php echo 'Foo'; someThingElse();"},
		{"provideFixCases#9", map[string]any{"format": "short", "shorten_simple_statements_only": false}, "<?php echo 'Foo'; someThingElse();", "<?= 'Foo'; someThingElse();"},
		{"provideFixCases#10", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php /*this */ /** should be in the result*/ echo 'Foo';", "<?=/*this */ /** should be in the result*/ 'Foo';"},
		{"provideFixCases#11", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php /*comment*/ echo\n  1\n?>", "<?=/*comment*/\n  1\n?>"},
		{"provideFixCases#12", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php\n  /*comment*/ echo 1\n?>", "<?=/*comment*/ 1\n?>"},
		{"provideFixCases#13", map[string]any{"format": "short", "shorten_simple_statements_only": true}, "<?php\n  /*comment*/\n  echo\n  1\n?>", "<?=/*comment*/\n  1\n?>"},
		{"provideFixCases#14", map[string]any{"format": "long", "long_function": "echo"}, "<?= 1;", "<?php echo 1;"},
		{"provideFixCases#15", map[string]any{"format": "long", "long_function": "echo"}, "<?=1;", "<?php echo 1;"},
		{"provideFixCases#16", map[string]any{"format": "long", "long_function": "echo"}, "<?=/**/1;", "<?php echo /**/1;"},
		{"provideFixCases#17", map[string]any{"format": "long", "long_function": "echo"}, "<?=/**/ 1;", "<?php echo /**/ 1;"},
		{"provideFixCases#18", map[string]any{"format": "long", "long_function": "echo"}, "<?= 'Foo';", "<?php echo 'Foo';"},
		{"provideFixCases#19", map[string]any{"format": "long", "long_function": "echo"}, "<?= 'Foo'; ?> PLAIN TEXT", "<?php echo 'Foo'; ?> PLAIN TEXT"},
		{"provideFixCases#20", map[string]any{"format": "long", "long_function": "echo"}, "PLAIN TEXT<?= 'Foo'; ?>", "PLAIN TEXT<?php echo 'Foo'; ?>"},
		{"provideFixCases#21", map[string]any{"format": "long", "long_function": "echo"}, "<?= 'Foo'; ?> <?= 'Bar'; ?>", "<?php echo 'Foo'; ?> <?php echo 'Bar'; ?>"},
		{"provideFixCases#22", map[string]any{"format": "long", "long_function": "echo"}, "<?=foo();", "<?php echo foo();"},
		{"provideFixCases#23", map[string]any{"format": "long", "long_function": "print"}, "<?= 1;", "<?php print 1;"},
		{"provideFixCases#24", map[string]any{"format": "long", "long_function": "print"}, "<?=1;", "<?php print 1;"},
		{"provideFixCases#25", map[string]any{"format": "long", "long_function": "print"}, "<?=/**/1;", "<?php print /**/1;"},
		{"provideFixCases#26", map[string]any{"format": "long", "long_function": "print"}, "<?=/**/ 1;", "<?php print /**/ 1;"},
		{"provideFixCases#27", map[string]any{"format": "long", "long_function": "print"}, "<?= 'Foo';", "<?php print 'Foo';"},
		{"provideFixCases#28", map[string]any{"format": "long", "long_function": "print"}, "<?= 'Foo'; ?> PLAIN TEXT", "<?php print 'Foo'; ?> PLAIN TEXT"},
		{"provideFixCases#29", map[string]any{"format": "long", "long_function": "print"}, "PLAIN TEXT<?= 'Foo'; ?>", "PLAIN TEXT<?php print 'Foo'; ?>"},
		{"provideFixCases#30", map[string]any{"format": "long", "long_function": "print"}, "<?= 'Foo'; ?> <?= 'Bar'; ?>", "<?php print 'Foo'; ?> <?php print 'Bar'; ?>"},
		{"provideFixCases#31", map[string]any{"format": "long", "long_function": "print"}, "<?=foo();", "<?php print foo();"},
		{"provideFixCases#32", map[string]any{"format": "short"}, "<?php\n// comment\necho 'Foo';\n?>", "<?=// comment\n 'Foo';\n?>"},
		{"extra#0", nil, "<?= $a ?>\n<?=$a?>\n<?=\n$a?>\n<?= $a; ?>\n", "<?php echo $a ?>\n<?php echo $a?>\n<?php echo\n$a?>\n<?php echo $a; ?>\n"},
		{"extra#1", map[string]any{"long_function": "print"}, "<?= $a ?>\n<?=$a?>\n", "<?php print $a ?>\n<?php print $a?>\n"},
		{"extra#2", map[string]any{"format": "short"}, "<?php echo $a ?>\n<?php  echo $a; ?>\n<?php\necho $a ?>\n<?php /* c */ echo 1 ?>\n<?php // c\necho 1 ?>\n<?php print 1; $b = 2 ?>\n<?php print 1; ?>\n", "<?= $a ?>\n<?= $a; ?>\n<?= $a ?>\n<?=/* c */ 1 ?>\n<?=// c\n 1 ?>\n<?php print 1; $b = 2 ?>\n<?= 1; ?>\n"},
		{"extra#3", map[string]any{"format": "short", "shorten_simple_statements_only": false}, "<?php print 1; $b = 2 ?>\n<?php echo 1;\n$c = 3; ?>\n", "<?= 1; $b = 2 ?>\n<?= 1;\n$c = 3; ?>\n"},
		{"extra#4", map[string]any{"format": "short"}, "<? echo 1 ?>\n<?php\n\n  echo 2 ?>\n<?php /** d */ echo 3 ?>\n<?php $a = 1; echo $a ?>\n", "<?= 1 ?>\n<?= 2 ?>\n<?=/** d */ 3 ?>\n<?php $a = 1; echo $a ?>\n"},
		{"extra#5", nil, "<div><?= $a ?></div>\n<?php echo $b ?>\n", "<div><?php echo $a ?></div>\n<?php echo $b ?>\n"},
	})
}
