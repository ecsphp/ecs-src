package rules

import "testing"

// Every expected output is what PHP-CS-Fixer 3.95's ClassKeywordRemoveFixer
// produces for the input. Group and comma imports are out of scope for the port.
func TestClassKeywordRemovePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, ClassKeywordRemove{}, []aliasUpstreamCase{
		{
			"namespaced imports", nil,
			"<?php\nnamespace App;\nuse App\\Foo\\Bar;\nuse App\\Other\\Baz as Alias;\n$a = Bar::class;\n$b = Alias::class;\n$c = \\App\\Qux::class;\n$d = self::class;\n$e = static::class;\n$f = parent::class;\n$g = Direct::class;\n",
			"<?php\nnamespace App;\nuse App\\Foo\\Bar;\nuse App\\Other\\Baz as Alias;\n$a = 'App\\Foo\\Bar';\n$b = 'App\\Other\\Baz';\n$c = 'App\\Qux';\n$d = self::class;\n$e = static::class;\n$f = parent::class;\n$g = 'App\\Direct';\n",
		},
		{
			"global namespace imports", nil,
			"<?php\nuse App\\Foo\\Bar;\nuse App\\Sub;\n$a = Bar::class;\n$b = Sub\\Deep::class;\n$c = Unknown::class;\n$d = \\Global\\Thing::class;\n",
			"<?php\nuse App\\Foo\\Bar;\nuse App\\Sub;\n$a = 'App\\Foo\\Bar';\n$b = 'App\\Sub\\Deep';\n$c = 'Unknown';\n$d = 'Global\\Thing';\n",
		},
		{
			"single class ref", nil,
			"<?php\nnamespace App;\nuse App\\Foo\\Bar;\n$x = Bar::class;\n",
			"<?php\nnamespace App;\nuse App\\Foo\\Bar;\n$x = 'App\\Foo\\Bar';\n",
		},
		{
			"no imports plain", nil,
			"<?php\n$x = Foo::class;\n",
			"<?php\n$x = 'Foo';\n",
		},
		{
			"leading backslash drops namespace prefix", nil,
			"<?php\nnamespace App;\n$x = \\Other\\Thing::class;\n",
			"<?php\nnamespace App;\n$x = 'Other\\Thing';\n",
		},
		{
			"alias wins over suffix", nil,
			"<?php\nnamespace App;\nuse App\\Other\\Baz as Alias;\n$a = Alias::class;\n",
			"<?php\nnamespace App;\nuse App\\Other\\Baz as Alias;\n$a = 'App\\Other\\Baz';\n",
		},
		{
			"self inside class is left alone", nil,
			"<?php\nclass A { public function f() { return self::class; } }\n",
			"<?php\nclass A { public function f() { return self::class; } }\n",
		},
		{
			"instance class constant is left alone", nil,
			"<?php\n$x = $obj::class;\n",
			"<?php\n$x = $obj::class;\n",
		},
	})
}
