package rules

import "testing"

// Cases ported from PHP-CS-Fixer's GetClassToClassKeywordFixerTest (tests/Fixer/LanguageConstruct), then
// invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.95 produces for the input.
func TestGetClassToClassKeywordPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, GetClassToClassKeyword{}, []portConstructsCase{
		{"provideFixCases#0", nil, "\n<?php\n\n$before = get_class($before);\n$after = get_class($after);\n", "\n<?php\n\n$before = $before::class;\n$after = $after::class;\n"},
		{"provideFixCases#1", nil, "<?php get_class($abc);", "<?php $abc::class;"},
		{"provideFixCases#2", nil, "<?php get_class( $a );", "<?php $a::class  ;"},
		{"provideFixCases#3", nil, "<?php \\get_class($b);", "<?php $b::class;"},
		{"provideFixCases#4", nil, "<?php GET_class($c);", "<?php $c::class;"},
		{"provideFixCases#5", nil, "<?php get_class($d/* a */);", "<?php $d::class/* a */;"},
		{"provideFixCases#6", nil, "<?php get_class($e /** b */);", "<?php $e::class /** b */;"},
		{"provideFixCases#7", nil, "<?php get_class ( $f );", "<?php $f::class   ;"},
		{"provideFixCases#8", nil, "<?php \\get_class(/* x */ $g /* y */);", "<?php $g::class/* x */  /* y */;"},
		{"provideFixCases#9", nil, "<?php get_class(($h));", "<?php $h::class;"},
		{"provideFixCases#10", nil, "<?php\necho get_class(\n    $bar\n);\n", "<?php\necho $bar::class\n    \n;\n"},
		{"provideFixCases#11", nil, "<?php get_class;", "<?php get_class;"},
		{"provideFixCases#12", nil, "<?php get_class($this);", "<?php get_class($this);"},
		{"provideFixCases#13", nil, "<?php get_class();", "<?php get_class();"},
		{"provideFixCases#14", nil, "<?php get_class(/* $a */);", "<?php get_class(/* $a */);"},
		{"provideFixCases#15", nil, "<?php get_class(/** $date */);", "<?php get_class(/** $date */);"},
		{"provideFixCases#16", nil, "<?php $a = get_class(12);", "<?php $a = get_class(12);"},
		{"provideFixCases#17", nil, "<?php get_class($a.$b);", "<?php get_class($a.$b);"},
		{"provideFixCases#18", nil, "<?php get_class($a === $b);", "<?php get_class($a === $b);"},
		{"provideFixCases#19", nil, "<?php get_class($foo->bar);", "<?php get_class($foo->bar);"},
		{"provideFixCases#20", nil, "<?php get_class($$foo);", "<?php get_class($$foo);"},
		{"provideFixCases#21", nil, "<?php get_class($arr[$bar]);", "<?php get_class($arr[$bar]);"},
		{"provideFixCases#22", nil, "<?php \\a\\get_class($foo);", "<?php \\a\\get_class($foo);"},
		{"provideFixCases#23", nil, "<?php\nclass A\n{\n    public function get_class($foo) {}\n}\n", "<?php\nclass A\n{\n    public function get_class($foo) {}\n}\n"},
		{"provideFixCases#24", nil, "<?php get_class($a, $b);", "<?php get_class($a, $b);"},
		{"extra#0", nil, "<?php\n$a = get_class( $b );\n$c = \\get_class($d);\n$e = get_class(/* c */$f);\n", "<?php\n$a = $b::class  ;\n$c = $d::class;\n$e = $f::class/* c */;\n"},
		{"extra#1", nil, "<?php\n$a = get_class($b, );\n$c = get_class((($d)));\n$e = get_class($this);\n$f = get_class($$g);\n$h = get_class($i->j);\n", "<?php\n$a = get_class($b, );\n$c = $d::class;\n$e = get_class($this);\n$f = get_class($$g);\n$h = get_class($i->j);\n"},
		{"extra#2", nil, "<?php\nnamespace X;\n$a = get_class($b);\n", "<?php\nnamespace X;\n$a = $b::class;\n"},
		{"extra#3", nil, "<?php\nnamespace X;\nfunction get_class($x) {}\n$a = get_class($b);\n", "<?php\nnamespace X;\nfunction get_class($x) {}\n$a = get_class($b);\n"},
		{"extra#4", nil, "<?php\n$x = [get_class($a), get_class($b)];\n$y = GET_CLASS($c);\n$z = get_class(...);\n", "<?php\n$x = [$a::class, $b::class];\n$y = $c::class;\n$z = get_class(...);\n"},
		{"extra#5", nil, "<?php\n$x = get_class(/**/ $a /**/);\n$f = fn ($a) => get_class($a);\n", "<?php\n$x = $a::class/**/  /**/;\n$f = fn ($a) => $a::class;\n"},
		{"extra#6", nil, "<?php\nclass A { public function get_class($a) {} public function f($o) { return get_class($o) . $this->get_class($o); } }\n", "<?php\nclass A { public function get_class($a) {} public function f($o) { return $o::class . $this->get_class($o); } }\n"},
		{"extra#7", nil, "<?php\nnamespace X;\nuse function Y\\get_class;\n$a = get_class($b);\n", "<?php\nnamespace X;\nuse function Y\\get_class;\n$a = get_class($b);\n"},
		{"extra#8", nil, "<?php\nif (get_class($a) === get_class($b)) {}\necho get_class($c) ?>\n", "<?php\nif ($a::class === $b::class) {}\necho $c::class ?>\n"},
		{"extra#9", nil, "<?php\nfunction g($a) { yield from get_class($a); }\n", "<?php\nfunction g($a) { yield from $a::class; }\n"},
	})
}
