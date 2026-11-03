package rules

import "testing"

// Cases ported from PHP-CS-Fixer's BacktickToShellExecFixerTest (tests/Fixer/Alias), then
// invented ones (named "extra"/"analyzer"/...); every expected output is what
// PHP-CS-Fixer 3.95 produces for the input.
func TestBacktickToShellExecPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, BacktickToShellExec{}, []aliasUpstreamCase{
		{"plain", nil, "<?php `ls -lah`;", "<?php shell_exec(\"ls -lah\");"},
		{"with variables", nil, "<?php `$var1 ls ${var2} -lah {$var3} file1.txt {$var4[0]} file2.txt {$var5->call()}`;", "<?php shell_exec(\"$var1 ls ${var2} -lah {$var3} file1.txt {$var4[0]} file2.txt {$var5->call()}\");"},
		{"with single quote", nil, "<?php\n`echo a\\'b`;\n`echo 'ab'`;", "<?php\n`echo a\\'b`;\n`echo 'ab'`;"},
		{"with double quote", nil, "<?php\n`echo a\\\"b`;\n`echo 'a\"b'`;", "<?php\n`echo a\\\"b`;\n`echo 'a\"b'`;"},
		{"with backtick", nil, "<?php\n`echo 'a\\`b'`;\n`echo a\\\\\\`b`;", "<?php\n`echo 'a\\`b'`;\n`echo a\\\\\\`b`;"},
		{"extra #1", nil, "<?php $out = `$cmd`;", "<?php $out = shell_exec($cmd);"},
		{"extra #2", nil, "<?php $out = ``;", "<?php $out = shell_exec(\"\");"},
		{"extra #3", nil, "<?php $out = `ls $dir`;", "<?php $out = shell_exec(\"ls $dir\");"},
		{"extra #4", nil, "<?php $out = `ls $dir[0]` . `ls $obj->dir`;", "<?php $out = shell_exec(\"ls $dir[0]\") . shell_exec(\"ls $obj->dir\");"},
		{"extra #5", nil, "<?php $out = `ls {$dirs['main']}`;", "<?php $out = shell_exec(\"ls {$dirs['main']}\");"},
		{"extra #6", nil, "<?php $out = `curl http://example.com/a # x`;", "<?php $out = shell_exec(\"curl http://example.com/a # x\");"},
		{"extra #7", nil, "<?php $out = `echo \"x\"` . `echo 'y'`;", "<?php $out = `echo \"x\"` . `echo 'y'`;"},
		{"extra #8", nil, "<?php $out = `printf '\\n'`;", "<?php $out = `printf '\\n'`;"},
		{"extra #9", nil, "<?php $out = `ls \\$HOME`;", "<?php $out = shell_exec(\"ls \\$HOME\");"},
		{"extra #10", nil, "<?php $out = `ls\n-lah`;", "<?php $out = shell_exec(\"ls\n-lah\");"},
		{"extra #11", nil, "<?php echo \"`not a command`\", '`neither`';", "<?php echo \"`not a command`\", '`neither`';"},
		{"interpolation #1", nil, "<?php $a = `echo {$a} \"x\"`;", "<?php $a = `echo {$a} \"x\"`;"},
		{"interpolation #2", nil, "<?php $b = `grep {$p} 'it's' file`;", "<?php $b = `grep {$p} 'it's' file`;"},
		{"interpolation #3", nil, "<?php $c = `echo ${a} \"x\"`;", "<?php $c = `echo ${a} \"x\"`;"},
		{"interpolation #4", nil, "<?php $a = `echo {$a[\"`\"]}`;", "<?php $a = shell_exec(\"echo {$a[\"`\"]}\");"},
		{"interpolation #5", nil, "<?php $a = `$a[0]`;", "<?php $a = shell_exec(\"$a[0]\");"},
		{"interpolation #6", nil, "<?php $b = `$a[x]`;", "<?php $b = shell_exec(\"$a[x]\");"},
		{"interpolation #7", nil, "<?php $a = `$o->p`;", "<?php $a = shell_exec(\"$o->p\");"},
		{"interpolation #8", nil, "<?php $b = `$o?->p`;", "<?php $b = shell_exec(\"$o?->p\");"},
		{"interpolation #9", nil, "<?php $a = `{$o->p}`;", "<?php $a = shell_exec(\"{$o->p}\");"},
		{"interpolation #10", nil, "<?php $b = `${a}`;", "<?php $b = shell_exec(\"${a}\");"},
		{"interpolation #11", nil, "<?php $c = `echo {$a['x']}`;", "<?php $c = shell_exec(\"echo {$a['x']}\");"},
		{"interpolation #12", nil, "<?php $a = `${a}x`;", "<?php $a = shell_exec(\"${a}x\");"},
		{"interpolation #13", nil, "<?php $b = `x${a}`;", "<?php $b = shell_exec(\"x${a}\");"},
		{"interpolation #14", nil, "<?php $c = `${a}${b}`;", "<?php $c = shell_exec(\"${a}${b}\");"},
		{"interpolation #15", nil, "<?php $d = `\\${a}`;", "<?php $d = shell_exec(\"\\${a}\");"},
		{"interpolation #16", nil, "<?php $a = `echo {$a[\"it's $b\"]}`;", "<?php $a = `echo {$a[\"it's $b\"]}`;"},
		{"interpolation #17", nil, "<?php $b = `echo {$a[\"x $b\"]}`;", "<?php $b = shell_exec(\"echo {$a[\"x $b\"]}\");"},
		{"interpolation #18", nil, "<?php $c = `echo {$a[\"x\"]} y`;", "<?php $c = shell_exec(\"echo {$a[\"x\"]} y\");"},
		{"interpolation #19", nil, "<?php $a = `echo {$a} 'x'`;", "<?php $a = `echo {$a} 'x'`;"},
		{"interpolation #20", nil, "<?php $b = `\"$a\"`;", "<?php $b = `\"$a\"`;"},
		{"interpolation #21", nil, "<?php $c = `$a \"x\"`;", "<?php $c = `$a \"x\"`;"},
		{"interpolation #22", nil, "<?php $d = `$a`.`$b`;", "<?php $d = shell_exec($a).shell_exec($b);"},
		{"interpolation #23", nil, "<?php\n$a = \"{$a[`ls`]}\";\n$b = ` $a`;\n$c = `$a `;\n$d = `echo {$a /* it's */}`;\n$e = `echo {$a['it\\'s']}`;\n$f = <<<EOT\nx {$a[`ls -l`]} y\nEOT;\n$g = `echo \\\\`;\n$h = `{$a}{$b}`;\n$i = `$a$b[1]`;\n$j = `echo {$a[\"x{$b}y\"]}`;\n$k = `echo {$a[\"x{$b}'\"]}`;\n$l = `echo ${a['it\\'s']}`;\n$m = `echo {$a}` . \"`\" . `ls`;\n", "<?php\n$a = \"{$a[shell_exec(\"ls\")]}\";\n$b = shell_exec(\" $a\");\n$c = shell_exec(\"$a \");\n$d = shell_exec(\"echo {$a /* it's */}\");\n$e = shell_exec(\"echo {$a['it\\'s']}\");\n$f = <<<EOT\nx {$a[shell_exec(\"ls -l\")]} y\nEOT;\n$g = shell_exec(\"echo \\\\\");\n$h = shell_exec(\"{$a}{$b}\");\n$i = shell_exec(\"$a$b[1]\");\n$j = shell_exec(\"echo {$a[\"x{$b}y\"]}\");\n$k = `echo {$a[\"x{$b}'\"]}`;\n$l = shell_exec(\"echo ${a['it\\'s']}\");\n$m = shell_exec(\"echo {$a}\") . \"`\" . shell_exec(\"ls\");\n"},
		{"interpolation #24", nil, "<?php\n$a = `echo {$a[<<<EOT\nit's\nEOT]}`;\n$b = `echo {$a[<<<'EOT'\nit's\nEOT]}`;\n", "<?php\n$a = `echo {$a[<<<EOT\nit's\nEOT]}`;\n$b = `echo {$a[<<<'EOT'\nit's\nEOT]}`;\n"},
	})
}

// PHP-CS-Fixer pairs the backticks it meets one after the other, so a backtick
// literal nested in the "{$...}" of another one comes out as broken code
// (shell_exec("echo {$a[")lsshell_exec("]} x")). Blink leaves both alone.
func TestBacktickToShellExecNestedLiteral(t *testing.T) {
	t.Parallel()
	src := "<?php $a = `echo {$a[`ls`]} x`;"
	if got, changed := apply(t, BacktickToShellExec{}, src); changed || got != src {
		t.Fatalf("changed=%v got=%q", changed, got)
	}
}
