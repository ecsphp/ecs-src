package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestNoMixedEchoPrintConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		NoMixedEchoPrint{},
		NoMixedEchoPrint{}.WithConfig(map[string]any{"use": "echo"}).(fixerRule),
		NoMixedEchoPrint{}.WithConfig(map[string]any{"use": "print"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [3]string
	}{
		{"<?php\necho 'a';\nprint 'b';\necho 'c', 'd';\n", [3]string{"<?php\necho 'a';\necho 'b';\necho 'c', 'd';\n", "<?php\necho 'a';\necho 'b';\necho 'c', 'd';\n", "<?php\nprint 'a';\nprint 'b';\necho 'c', 'd';\n"}},
		{"<?php\necho foo(1, 2);\necho 'a' . implode(',', $b);\necho [1, 2][0];\necho $a ? [1, 2] : 3;\n", [3]string{"<?php\necho foo(1, 2);\necho 'a' . implode(',', $b);\necho [1, 2][0];\necho $a ? [1, 2] : 3;\n", "<?php\necho foo(1, 2);\necho 'a' . implode(',', $b);\necho [1, 2][0];\necho $a ? [1, 2] : 3;\n", "<?php\nprint foo(1, 2);\nprint 'a' . implode(',', $b);\nprint [1, 2][0];\nprint $a ? [1, 2] : 3;\n"}},
		{"<?php\necho $a[f(1)], 2;\necho $a[0], $b;\necho match ($x) { 1, 2 => 'a', default => 'b' };\n", [3]string{"<?php\necho $a[f(1)], 2;\necho $a[0], $b;\necho match ($x) { 1, 2 => 'a', default => 'b' };\n", "<?php\necho $a[f(1)], 2;\necho $a[0], $b;\necho match ($x) { 1, 2 => 'a', default => 'b' };\n", "<?php\necho $a[f(1)], 2;\necho $a[0], $b;\necho match ($x) { 1, 2 => 'a', default => 'b' };\n"}},
		{"<?php echo 'a' ?>\n<?= 'b' ?>\n<?php echo 'c', 'd' ?>\n", [3]string{"<?php echo 'a' ?>\n<?= 'b' ?>\n<?php echo 'c', 'd' ?>\n", "<?php echo 'a' ?>\n<?= 'b' ?>\n<?php echo 'c', 'd' ?>\n", "<?php print 'a' ?>\n<?= 'b' ?>\n<?php echo 'c', 'd' ?>\n"}},
		{"<?php\nECHO 'a';\nPrint 'b';\n$x = print 'c';\nif ($x) print 'd'; else print 'e';\nif ($x) echo 'f'; else echo 'g', 'h';\n", [3]string{"<?php\nECHO 'a';\necho 'b';\n$x = print 'c';\nif ($x) echo 'd'; else echo 'e';\nif ($x) echo 'f'; else echo 'g', 'h';\n", "<?php\nECHO 'a';\necho 'b';\n$x = print 'c';\nif ($x) echo 'd'; else echo 'e';\nif ($x) echo 'f'; else echo 'g', 'h';\n", "<?php\nprint 'a';\nPrint 'b';\n$x = print 'c';\nif ($x) print 'd'; else print 'e';\nif ($x) print 'f'; else echo 'g', 'h';\n"}},
		{"<?php\necho \"a{$b}c\", 'd';\necho \"a{$b}c\";\necho <<<EOT\nx, y\nEOT;\n", [3]string{"<?php\necho \"a{$b}c\", 'd';\necho \"a{$b}c\";\necho <<<EOT\nx, y\nEOT;\n", "<?php\necho \"a{$b}c\", 'd';\necho \"a{$b}c\";\necho <<<EOT\nx, y\nEOT;\n", "<?php\necho \"a{$b}c\", 'd';\nprint \"a{$b}c\";\nprint <<<EOT\nx, y\nEOT;\n"}},
		{"<?php\nfunction f() { echo 'a'; }\nwhile ($x) echo 'b';\nfoo() or print 'c';\n", [3]string{"<?php\nfunction f() { echo 'a'; }\nwhile ($x) echo 'b';\nfoo() or print 'c';\n", "<?php\nfunction f() { echo 'a'; }\nwhile ($x) echo 'b';\nfoo() or print 'c';\n", "<?php\nfunction f() { print 'a'; }\nwhile ($x) print 'b';\nfoo() or print 'c';\n"}},
		{"<?php\necho /* x */ 'a';\necho\n    'a',\n    'b';\n", [3]string{"<?php\necho /* x */ 'a';\necho\n    'a',\n    'b';\n", "<?php\necho /* x */ 'a';\necho\n    'a',\n    'b';\n", "<?php\nprint /* x */ 'a';\necho\n    'a',\n    'b';\n"}},
	}
	for _, c := range cases {
		for k, f := range fixers {
			got, changed := apply(t, f, c.src)
			if got != c.want[k] || changed != (c.want[k] != c.src) {
				t.Errorf("config %d src=%q\nchanged=%v got=%q\nwant=%q", k, c.src, changed, got, c.want[k])
			}
		}
	}
}
