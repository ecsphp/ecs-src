package rules

import "testing"

// Cases ported from PHP-CS-Fixer's SemicolonAfterInstructionFixerTest (tests/Fixer/Semicolon), then
// invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.95 produces for the input.
func TestSemicolonAfterInstructionPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, SemicolonAfterInstruction{}, []portConstructsCase{
		{"provideFixCases#comment", nil, "<?php $a++//a ?>", "<?php $a++;//a ?>"},
		{"provideFixCases#comment II", nil, "<?php $b++ /**/ ?>", "<?php $b++; /**/ ?>"},
		{"provideFixCases#no space", nil, "<?php $b++?>", "<?php $b++;?>"},
		{"provideFixCases#0", nil, "<?php echo 123 ?>", "<?php echo 123; ?>"},
		{"provideFixCases#1", nil, "<?php echo 123\n\t?>", "<?php echo 123;\n\t?>"},
		{"provideFixCases#2", nil, "<?php ?>", "<?php ?>"},
		{"provideFixCases#3", nil, "<?php ; ?>", "<?php ; ?>"},
		{"provideFixCases#4", nil, "<?php if($a){}", "<?php if($a){}"},
		{"provideFixCases#5", nil, "<?php while($a > $b){}", "<?php while($a > $b){}"},
		{"provideFixCases#6", nil, "<?php if ($a == 5): ?>\nA is equal to 5\n<?php endif; ?>\n<?php switch ($foo): ?>\n<?php case 1: ?>\n...\n<?php endswitch ?>", "<?php if ($a == 5): ?>\nA is equal to 5\n<?php endif; ?>\n<?php switch ($foo): ?>\n<?php case 1: ?>\n...\n<?php endswitch; ?>"},
		{"provideFixCases#7", nil, "<?php if ($a == 5) { ?>\nA is equal to 5\n<?php } ?>", "<?php if ($a == 5) { ?>\nA is equal to 5\n<?php } ?>"},
		{"provideFixCases#open tag with echo", nil, "<?= '1_' ?> <?php ?><?= 1; ?>", "<?= '1_'; ?> <?php ?><?= 1; ?>"},
		{"extra#0", nil, "<?php echo 1 ?>\n<?php if ($a): ?>\n<?php endif ?>\n<?php ?>\n<?= $a ?>\n", "<?php echo 1; ?>\n<?php if ($a): ?>\n<?php endif; ?>\n<?php ?>\n<?= $a; ?>\n"},
		{"extra#1", nil, "<?php $a->{'b'} ?>\n<?php echo 1; ?>\n<?php echo 1 // c ?>\n<?php echo 1 /* c */ ?>\n<?php foo() ?>\n", "<?php $a->{'b'}; ?>\n<?php echo 1; ?>\n<?php echo 1; // c ?>\n<?php echo 1; /* c */ ?>\n<?php foo(); ?>\n"},
		{"extra#2", nil, "<?php if ($a) { ?>x<?php } else { ?>y<?php } ?>\n<?php ${'a'} ?>\n<?php $b = <<<EOT\nx\nEOT ?>\n", "<?php if ($a) { ?>x<?php } else { ?>y<?php } ?>\n<?php ${'a'}; ?>\n<?php $b = <<<EOT\nx\nEOT; ?>\n"},
	})
}
