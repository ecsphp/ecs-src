package rules

import "testing"

<<<<<<< HEAD
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
=======
// Cases ported from PHP-CS-Fixer's ClassKeywordRemoveFixerTest (tests/Fixer/LanguageConstruct),
// then invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.x produces for the input.
func TestClassKeywordRemovePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, ClassKeywordRemove{}, []portConstructsCase{
		{"provideFixCases#0", nil, "<?php\n                use Foo\\Bar\\Thing;\n\n                echo Thing::class;\n                ", "<?php\n                use Foo\\Bar\\Thing;\n\n                echo 'Foo\\Bar\\Thing';\n                "},
		{"provideFixCases#1", nil, "<?php\n                use Foo\\Bar;\n            \n                echo Bar\\Thing::class;\n                ", "<?php\n                use Foo\\Bar;\n            \n                echo 'Foo\\Bar\\Thing';\n                "},
		{"provideFixCases#2", nil, "<?php\n                namespace Foo;\n                use Foo\\Bar;\n                echo \\Foo\\Bar\\Baz::class;\n                ", "<?php\n                namespace Foo;\n                use Foo\\Bar;\n                echo 'Foo\\Bar\\Baz';\n                "},
		{"provideFixCases#3", nil, "<?php\n                use Foo\\Bar\\Thing as Alias;\n\n                echo Alias::class;\n                ", "<?php\n                use Foo\\Bar\\Thing as Alias;\n\n                echo 'Foo\\Bar\\Thing';\n                "},
		{"provideFixCases#4", nil, "<?php\n                use Foo\\Bar\\Dummy;\n                use Foo\\Bar\\Thing as Alias;\n\n                echo Dummy::class;\n                echo Alias::class;\n                ", "<?php\n                use Foo\\Bar\\Dummy;\n                use Foo\\Bar\\Thing as Alias;\n\n                echo 'Foo\\Bar\\Dummy';\n                echo 'Foo\\Bar\\Thing';\n                "},
		{"provideFixCases#5", nil, "<?php\n                echo \\DateTime::class;\n                ", "<?php\n                echo 'DateTime';\n                "},
		{"provideFixCases#6", nil, "<?php\n                echo Thing::class;\n                ", "<?php\n                echo 'Thing';\n                "},
		{"provideFixCases#7", nil, "<?php\n                class Foo {\n                    public function amazingFunction() {\n                        echo Thing::class;\n                    }\n                }\n                ", "<?php\n                class Foo {\n                    public function amazingFunction() {\n                        echo 'Thing';\n                    }\n                }\n                "},
		{"provideFixCases#8", nil, "<?php\n                namespace A\\B;\n\n                use Foo\\Bar;\n\n                echo Bar::class;\n                ", "<?php\n                namespace A\\B;\n\n                use Foo\\Bar;\n\n                echo 'Foo\\Bar';\n                "},
		{"provideFixCases#9", nil, "<?php\n\n                namespace A\\B {\n\n                    class D {\n\n                    }\n                }\n\n                namespace B\\B {\n                    class D {\n\n                    }\n                }\n\n                namespace C {\n                    use A\\B\\D;\n                    var_dump(D::class);\n                }\n\n                namespace C1 {\n                    use B\\B\\D;\n                    var_dump(D::class);\n                }\n                ", "<?php\n\n                namespace A\\B {\n\n                    class D {\n\n                    }\n                }\n\n                namespace B\\B {\n                    class D {\n\n                    }\n                }\n\n                namespace C {\n                    use A\\B\\D;\n                    var_dump('A\\B\\D');\n                }\n\n                namespace C1 {\n                    use B\\B\\D;\n                    var_dump('B\\B\\D');\n                }\n                "},
		{"provideFixCases#10", nil, "<?php\n                namespace Foo;\n                class Bar extends Baz {\n                    public function a() {\n                        return self::class;\n                    }\n                    public function b() {\n                        return static::class;\n                    }\n                    public function c() {\n                        return parent::class;\n                    }\n                }\n                ", "<?php\n                namespace Foo;\n                class Bar extends Baz {\n                    public function a() {\n                        return self::class;\n                    }\n                    public function b() {\n                        return static::class;\n                    }\n                    public function c() {\n                        return parent::class;\n                    }\n                }\n                "},
		{"provideFixCases#11", nil, "<?php\n                namespace Foo;\n                var_dump(Bar\\Baz::class);\n                ", "<?php\n                namespace Foo;\n                var_dump('Foo\\Bar\\Baz');\n                "},
		{"provideFixCases#12", nil, "<?php\n                namespace Foo\\Bar;\n                var_dump(Baz::class);\n                ", "<?php\n                namespace Foo\\Bar;\n                var_dump('Foo\\Bar\\Baz');\n                "},
		{"provideFixCases#13", nil, "<?php\n                use Foo\\Bar\\{ClassA, ClassB, ClassC as C};\n                use function Foo\\Bar\\{fn_a, fn_b, fn_c};\n                use const Foo\\Bar\\{ConstA, ConstB, ConstC};\n\n                echo ClassB::class;\n                echo C::class;\n                ", "<?php\n                use Foo\\Bar\\{ClassA, ClassB, ClassC as C};\n                use function Foo\\Bar\\{fn_a, fn_b, fn_c};\n                use const Foo\\Bar\\{ConstA, ConstB, ConstC};\n\n                echo 'Foo\\Bar\\ClassB';\n                echo 'Foo\\Bar\\ClassC';\n                "},
		{"provideFixCases#14", nil, "<?php\n                namespace {\n                    var_dump(Foo::class);\n                }\n                namespace A {\n                    use B\\C;\n                    var_dump(C::class);\n                }\n                namespace {\n                    var_dump(Bar\\Baz::class);\n                }\n                namespace B {\n                    use A\\C\\D;\n                    var_dump(D::class);\n                }\n                namespace {\n                    var_dump(Qux\\Quux::class);\n                }\n                ", "<?php\n                namespace {\n                    var_dump('Foo');\n                }\n                namespace A {\n                    use B\\C;\n                    var_dump('B\\C');\n                }\n                namespace {\n                    var_dump('Bar\\Baz');\n                }\n                namespace B {\n                    use A\\C\\D;\n                    var_dump('A\\C\\D');\n                }\n                namespace {\n                    var_dump('Qux\\Quux');\n                }\n                "},
		{"extra#0", nil, "<?php\nnamespace Foo;\necho \\Bar::class;\n", "<?php\nnamespace Foo;\necho 'Bar';\n"},
		{"extra#1", nil, "<?php\necho Thing::class;\necho $thing::class;\n", "<?php\necho 'Thing';\necho $thing::class;\n"},
		{"extra#2", nil, "<?php\nuse App\\Model\\User;\n$a = User::class;\n", "<?php\nuse App\\Model\\User;\n$a = 'App\\Model\\User';\n"},
		{"extra#3", nil, "<?php $x = SomeClass::class;", "<?php $x = 'SomeClass';"},
>>>>>>> 8f741397a6 (Blink: port ClassKeywordRemoveFixer from PHP-CS-Fixer)
	})
}
