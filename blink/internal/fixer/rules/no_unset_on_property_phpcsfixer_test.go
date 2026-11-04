package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoUnsetOnPropertyFixerTest
// (tests/Fixer/LanguageConstruct); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestNoUnsetOnPropertyPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, NoUnsetOnProperty{}, []portConstructsCase{
		{"It replaces an unset on a property with = null", nil, "<?php unset($foo->bar);", "<?php $foo->bar = null;"},
		{"It replaces an unset on a property with = null II", nil, "<?php unset($foo->bar );", "<?php $foo->bar = null ;"},
		{"It replaces an unset on a static property with = null", nil, "<?php unset(TestClass::$bar);", "<?php TestClass::$bar = null;"},
		{"It does not replace unset on a variable with = null", nil, "<?php $b->a; unset($foo);", "<?php $b->a; unset($foo);"},
		{"It replaces multiple unsets on variables with = null", nil, "<?php unset($foo->bar, $bar->foo, $bar->baz, $a->ba);", "<?php $foo->bar = null; $bar->foo = null; $bar->baz = null; $a->ba = null;"},
		{"It replaces multiple unsets, but not those that arent properties", nil, "<?php unset($foo->bar, $bar->foo, $bar);", "<?php $foo->bar = null; $bar->foo = null; unset($bar);"},
		{"It replaces multiple unsets, but not those that arent properties in multiple places", nil, "<?php unset($foo, $bar->foo, $bar);", "<?php unset($foo); $bar->foo = null; unset($bar);"},
		{"It replaces $this -> and self:: replacements", nil, "<?php unset($this->bar, self::$foo, $bar);", "<?php $this->bar = null; self::$foo = null; unset($bar);"},
		{"It does not replace unsets on arrays", nil, "<?php unset($bar->foo[0]);", "<?php unset($bar->foo[0]);"},
		{"It works in a more complex unset", nil, "<?php unset($bar->foo[0], self::$foo, \\Test\\Baz::$fooBar, $bar->foo[0], $this->foo, $a, $b);", "<?php unset($bar->foo[0]); self::$foo = null; \\Test\\Baz::$fooBar = null; unset($bar->foo[0]); $this->foo = null; unset($a); unset($b);"},
		{"It works with consecutive unsets", nil, "<?php unset($foo->bar, $foo, $bar, $baz, $this->ab);", "<?php $foo->bar = null; unset($foo); unset($bar); unset($baz); $this->ab = null;"},
		{"It works when around messy whitespace", nil, "<?php\n     unset($a, $this->b);\n     unset($this->a, $b);\n", "<?php\n     unset($a); $this->b = null;\n     $this->a = null; unset($b);\n"},
		{"It works with weirdly placed comments", nil, "<?php unset/*foo*/(/*bar*/$bar->foo[0], self::$foo/*baz*/, /*hello*/\\Test\\Baz::$fooBar/*comment*/, $bar->foo[0], $this->foo, $a, $b);\n                unset/*foo*/(/*bar*/$bar);", "<?php unset/*foo*/(/*bar*/$bar->foo[0]); self::$foo = null/*baz*/; /*hello*/\\Test\\Baz::$fooBar = null/*comment*/; unset($bar->foo[0]); $this->foo = null; unset($a); unset($b);\n                unset/*foo*/(/*bar*/$bar);"},
		{"It does not mess with consecutive unsets", nil, "<?php unset($a, $b, $c);\n                unset($this->a);", "<?php unset($a, $b, $c);\n                $this->a = null;"},
		{"It does not replace function call with class constant inside", nil, "<?php unset($foos[array_search(BadFoo::NAME, $foos)]);", "<?php unset($foos[array_search(BadFoo::NAME, $foos)]);"},
		{"It does not replace function call with class constant and property inside", nil, "<?php unset($this->property[array_search(\\Types::TYPE_RANDOM, $this->property)]);", "<?php unset($this->property[array_search(\\Types::TYPE_RANDOM, $this->property)]);"},
		{"It does not break complex expressions", nil, "<?php\n                unset(a()[b()[\"a\"]]);\n                unset(a()[b()]);\n                unset(a()[\"a\"]);\n                unset(c($a)->a);\n            ", "<?php\n                unset(a()[b()[\"a\"]]);\n                unset(a()[b()]);\n                unset(a()[\"a\"]);\n                unset(c($a)->a);\n            "},
		{"It replaces an unset on a property with = null 1", nil, "<?php unset($foo->bar,);", "<?php $foo->bar = null;"},
		{"It replaces multiple unsets, but not those that arent properties 1", nil, "<?php unset($foo->bar, $bar->foo, $bar,);", "<?php $foo->bar = null; $bar->foo = null; unset($bar,);"},
		{"It replaces an unset on a static property with = null 1", nil, "<?php unset(TestClass::$bar,);", "<?php TestClass::$bar = null;"},
		{"It does not replace unset on a variable with = null 1", nil, "<?php $b->a; unset($foo,);", "<?php $b->a; unset($foo,);"},
		{"It replaces multiple unsets on variables with = null 1", nil, "<?php unset($foo->bar, $bar->foo, $bar->baz, $a->ba,);", "<?php $foo->bar = null; $bar->foo = null; $bar->baz = null; $a->ba = null;"},
		{"It replaces multiple unsets, but not those that arent properties in multiple places 1", nil, "<?php unset($foo, $bar->foo, $bar,);", "<?php unset($foo); $bar->foo = null; unset($bar,);"},
		{"It replaces $this -> and self:: replacements 1", nil, "<?php unset($this->bar, self::$foo, $bar,);", "<?php $this->bar = null; self::$foo = null; unset($bar,);"},
		{"It does not replace unsets on arrays 1", nil, "<?php unset($bar->foo[0],);", "<?php unset($bar->foo[0],);"},
		{"It works in a more complex unset 1", nil, "<?php unset($bar->foo[0], self::$foo, \\Test\\Baz::$fooBar, $bar->foo[0], $this->foo, $a, $b,);", "<?php unset($bar->foo[0]); self::$foo = null; \\Test\\Baz::$fooBar = null; unset($bar->foo[0]); $this->foo = null; unset($a); unset($b,);"},
		{"It works with consecutive unsets 1", nil, "<?php unset($foo->bar, $foo, $bar, $baz, $this->ab,);", "<?php $foo->bar = null; unset($foo); unset($bar); unset($baz); $this->ab = null;"},
		{"It works when around messy whitespace 1", nil, "<?php\n     unset($a, $this->b,);\n     unset($this->a, $b,);\n", "<?php\n     unset($a); $this->b = null;\n     $this->a = null; unset($b,);\n"},
		{"It works with weirdly placed comments 11", nil, "<?php unset/*foo*/(/*bar*/$bar->foo[0], self::$foo/*baz*/, /*hello*/\\Test\\Baz::$fooBar/*comment*/, $bar->foo[0], $this->foo, $a, $b,);\n                unset/*foo*/(/*bar*/$bar,);", "<?php unset/*foo*/(/*bar*/$bar->foo[0]); self::$foo = null/*baz*/; /*hello*/\\Test\\Baz::$fooBar = null/*comment*/; unset($bar->foo[0]); $this->foo = null; unset($a); unset($b,);\n                unset/*foo*/(/*bar*/$bar,);"},
		{"It does not mess with consecutive unsets 1", nil, "<?php unset($a, $b, $c,);\n                unset($this->a,);", "<?php unset($a, $b, $c,);\n                $this->a = null;"},
		{"It does not replace function call with class constant inside 1", nil, "<?php unset($foos[array_search(BadFoo::NAME, $foos)],);", "<?php unset($foos[array_search(BadFoo::NAME, $foos)],);"},
		{"It does not replace function call with class constant and property inside 1", nil, "<?php unset($this->property[array_search(\\Types::TYPE_RANDOM, $this->property)],);", "<?php unset($this->property[array_search(\\Types::TYPE_RANDOM, $this->property)],);"},
		{"NoUnsetOnPropertyFixer #32", nil, "<?php unset($foo->bar, );", "<?php $foo->bar = null ;"},
		{"NoUnsetOnPropertyFixer #33", nil, "<?php unset($foo->bar ,);", "<?php $foo->bar = null ;"},
		{"NoUnsetOnPropertyFixer #34", nil, "<?php unset($foo->bar , );", "<?php $foo->bar = null  ;"},
	})
}
