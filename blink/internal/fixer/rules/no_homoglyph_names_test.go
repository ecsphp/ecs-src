package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoHomoglyphNamesFixerTest.
func TestNoHomoglyphNamesUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, NoHomoglyphNames{}, []upstreamFixCase{
		{name: "#0", expected: `<?php $øøøøa = 1;`},
		{name: "#1", expected: `<?php $name = "This should not be changed";`},
		{name: "#2", expected: `<?php $name = "Это не меняется";`},
		{name: "#3", expected: `<?php $name = 'Это не меняется';`},
		{name: "#4", expected: `<?php // This should not be chаnged`},
		{name: "#5", expected: `<?php /* This should not be chаnged */`},
		{name: "#6", expected: `<?php $name = 'wrong';`, input: `<?php $nаmе = 'wrong';`},
		{name: "#7", expected: `<?php $a->name = 'wrong';`, input: `<?php $a->nаmе = 'wrong';`},
		{name: "#8", expected: `<?php class A { private $name; }`, input: `<?php class A { private $nаmе; }`},
		{name: "#9", expected: `<?php class Broken {}`, input: `<?php class Вroken {}`},
		{name: "#10", expected: `<?php interface Broken {}`, input: `<?php interface Вroken {}`},
		{name: "#11", expected: `<?php trait Broken {}`, input: `<?php trait Вroken {}`},
		{name: "#12", expected: `<?php $a = new Broken();`, input: `<?php $a = new Вroken();`},
		{name: "#13", expected: `<?php class A extends Broken {}`, input: `<?php class A extends Вroken {}`},
		{name: "#14", expected: `<?php class A implements Broken {}`, input: `<?php class A implements Вroken {}`},
		{name: "#15", expected: `<?php class A { use Broken; }`, input: `<?php class A { use Вroken; }`},
		{name: "#16", expected: `<?php echo Broken::class;`, input: `<?php echo Вroken::class;`},
		{name: "#17", expected: `<?php function name() {}`, input: `<?php function nаmе() {}`},
		{name: "#18", expected: `<?php name();`, input: `<?php nаmе();`},
		{name: "#19", expected: `<?php $first_name = "a";`, input: `<?php $first＿name = "a";`},
		{name: "#20", expected: `<?php class A { private string $name; }`, input: `<?php class A { private string $nаmе; }`},
		{name: "#21", expected: `<?php class A { private ? Foo\Bar $name; }`, input: `<?php class A { private ? Foo\Bar $nаmе; }`},
		{name: "#22", expected: `<?php class A { private array $name; }`, input: `<?php class A { private array $nаmе; }`},
	})
}

func TestNoHomoglyphNames(t *testing.T) {
	t.Parallel()
	const cyrA = "а" // CYRILLIC SMALL LETTER A
	runUpstreamFixCases(t, NoHomoglyphNames{}, []upstreamFixCase{
		// names interpolated in a string are T_VARIABLE / T_STRING too
		{name: "interpolated",
			expected: "<?php echo \"$name $a->name $b[name] {$c->name} ${d" + cyrA + "}\";",
			input:    "<?php echo \"$n" + cyrA + "me $a->n" + cyrA + "me $b[n" + cyrA + "me] {$c->n" + cyrA + "me} ${d" + cyrA + "}\";"},
		{name: "heredoc",
			expected: "<?php echo <<<EOT\n$name {$b->name()}\nEOT;\n",
			input:    "<?php echo <<<EOT\n$n" + cyrA + "me {$b->n" + cyrA + "me()}\nEOT;\n"},
		{name: "nowdoc", expected: "<?php echo <<<'EOT'\n$n" + cyrA + "me\nEOT;\n"},
		// a named argument's name is no T_STRING for PHP-CS-Fixer
		{name: "named argument",
			expected: "<?php foo(n" + cyrA + "me: $value, b: 1);",
			input:    "<?php foo(n" + cyrA + "me: $v" + cyrA + "lue, b: 1);"},
		{name: "invalid utf-8", expected: "<?php $n\xd0me = 1;"},
		// checked with PHP-CS-Fixer: the command text of a backtick literal is not a name
		{name: "backticks", expected: "<?php $w = `echo hеllo $var`;", input: "<?php $w = `echo hеllo $vаr`;"},
	})
}
