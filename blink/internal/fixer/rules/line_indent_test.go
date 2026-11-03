package rules

import "testing"

// TestLineIndentFromInlineHTMLAndTags pins the indentation the fixers built on
// IndentationTrait::getLineIndentation (BracesPosition,
// NoMultipleStatementsPerLine) and WhitespacesAnalyzer::detectIndent
// (ControlStructureContinuationPosition) give a new line when the code shares
// its line with inline HTML or a PHP open or close tag. Outputs are the PHP
// fixers' own; note the "<?php\n" cases, where PHP's open tag owns the line
// break and the indentation comes from the line before it.
func TestLineIndentFromInlineHTMLAndTags(t *testing.T) {
	t.Parallel()
	nextLineContinuation := ControlStructureContinuationPosition{}.WithConfig(map[string]any{"position": "next_line"}).(fixerRule)
	cases := []struct {
		name string
		rule fixerRule
		in   string
		want string
	}{
		{"BracesPosition/a1", BracesPosition{}, "<div>\n  <?php class A {\n  }\n", "<div>\n  <?php class A\n  {\n  }\n"},
		{"BracesPosition/a2", BracesPosition{}, "<?php\nif ($a):\n  ?>\n    <?php function z() {\n    }\n  ?>\n<?php\nendif;\n", "<?php\nif ($a):\n  ?>\n    <?php function z()\n    {\n    }\n  ?>\n<?php\nendif;\n"},
		{"BracesPosition/a3", BracesPosition{}, "<div>\n  <?php\nclass A {\n}\n", "<div>\n  <?php\nclass A\n  {\n}\n"},
		{"BracesPosition/a4", BracesPosition{}, "<?php\n\nclass B {\n    public function x() {\n        ?>\n      <p>\n        <?php if ($a) {\n        }\n        ?>\n    <?php\n    }\n}\n", "<?php\n\nclass B\n{\n    public function x()\n    {\n        ?>\n      <p>\n        <?php if ($a) {\n        }\n        ?>\n    <?php\n    }\n}\n"},
		{"BracesPosition/a5", BracesPosition{}, "<?php if ($a) { ?>\n    <b></b>\n<?php }\nfunction q() {\n}\n", "<?php if ($a) { ?>\n    <b></b>\n<?php }\nfunction q()\n{\n}\n"},
		{"BracesPosition/a6", BracesPosition{}, "<div>\r\n\t<?php class C {\r\n\t}\r\n", "<div>\r\n\t<?php class C\n\t{\r\n\t}\r\n"},
		{"NoMultipleStatementsPerLine/m1", NoMultipleStatementsPerLine{}, "<div>\n  <?php $a = 1; $b = 2; ?>\n</div>\n", "<div>\n  <?php $a = 1;\n  $b = 2; ?>\n</div>\n"},
		{"NoMultipleStatementsPerLine/m2", NoMultipleStatementsPerLine{}, "<?php\nif ($a):\n  ?>\n    <?php $a = 1; $b = 2; ?>\n<?php\nendif;\n", "<?php\nif ($a):\n  ?>\n    <?php $a = 1;\n    $b = 2; ?>\n<?php\nendif;\n"},
		{"NoMultipleStatementsPerLine/m3", NoMultipleStatementsPerLine{}, "<div>\n  <?php\n$a = 1; $b = 2;\n", "<div>\n  <?php\n$a = 1;\n  $b = 2;\n"},
		{"NoMultipleStatementsPerLine/m4", NoMultipleStatementsPerLine{}, "<?php\n\nfunction f() {\n    $a = 1; $b = 2;\n}\n", "<?php\n\nfunction f() {\n    $a = 1;\n    $b = 2;\n}\n"},
		{"ControlStructureContinuationPosition/c1", nextLineContinuation, "<?php if ($a) { ?>\n  <b></b>\n  <?php } else { ?>\n  <i></i>\n<?php }\n", "<?php if ($a) { ?>\n  <b></b>\n  <?php }\nelse { ?>\n  <i></i>\n<?php }\n"},
		{"ControlStructureContinuationPosition/c2", nextLineContinuation, "<div>\n  <?php if ($a) {\n  } else {\n  }\n", "<div>\n  <?php if ($a) {\n  }\n  else {\n  }\n"},
		{"ControlStructureContinuationPosition/c3", nextLineContinuation, "<?php if ($a) { ?>\n    x\n<?php\n} else { ?>\n    y\n<?php\n}\n", "<?php if ($a) { ?>\n    x\n<?php\n}\nelse { ?>\n    y\n<?php\n}\n"},
		{"ControlStructureContinuationPosition/c4", nextLineContinuation, "<?php\n\nfunction f() {\n    try {\n    } catch (E $e) {\n    } finally {\n    }\n}\n", "<?php\n\nfunction f() {\n    try {\n    }\n    catch (E $e) {\n    }\n    finally {\n    }\n}\n"},
		{"ControlStructureContinuationPosition/c5", nextLineContinuation, "<span>\n    <?php\n    if ($a) {\n    } else {\n    }\n", "<span>\n    <?php\n    if ($a) {\n    }\n    else {\n    }\n"},
		{"ControlStructureContinuationPosition/c6", nextLineContinuation, "<?php\n    if ($a) { ?>\n    x\n<?php\n} else { ?>\n    y\n<?php\n}\n", "<?php\n    if ($a) { ?>\n    x\n<?php\n}\n    else { ?>\n    y\n<?php\n}\n"},
	}
	for _, tc := range cases {
		got, _ := apply(t, tc.rule, tc.in)
		if got != tc.want {
			t.Errorf("%s:\n got: %q\nwant: %q", tc.name, got, tc.want)
		}
	}
}
