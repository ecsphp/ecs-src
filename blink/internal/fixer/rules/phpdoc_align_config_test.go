package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestPhpdocAlignConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		PhpdocAlign{},
		PhpdocAlign{}.WithConfig(map[string]any{"align": "left"}).(fixerRule),
		PhpdocAlign{}.WithConfig(map[string]any{"align": "left", "spacing": 2}).(fixerRule),
		PhpdocAlign{}.WithConfig(map[string]any{"align": "left", "spacing": map[string]any{"param": 2}}).(fixerRule),
		PhpdocAlign{}.WithConfig(map[string]any{"tags": []any{"param", "see", "phpstan-param", "return"}, "spacing": map[string]any{"_default": 3, "return": 1}}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [5]string
	}{
		{"/**\n * @param int $a\n */", [5]string{"/**\n * @param int $a\n */", "/**\n * @param int $a\n */", "/**\n * @param  int  $a\n */", "/**\n * @param  int  $a\n */", "/**\n * @param   int   $a\n */"}},
		{"/**\n * @param  int  $a\n */", [5]string{"/**\n * @param int $a\n */", "/**\n * @param int $a\n */", "/**\n * @param  int  $a\n */", "/**\n * @param  int  $a\n */", "/**\n * @param   int   $a\n */"}},
		{"/**\n * @param int $a\n * @param string $bb the description\n * @return bool\n */", [5]string{"/**\n * @param  int    $a\n * @param  string $bb the description\n * @return bool\n */", "/**\n * @param int $a\n * @param string $bb the description\n * @return bool\n */", "/**\n * @param  int  $a\n * @param  string  $bb  the description\n * @return  bool\n */", "/**\n * @param  int  $a\n * @param  string  $bb  the description\n * @return bool\n */", "/**\n * @param    int      $a\n * @param    string   $bb   the description\n * @return bool\n */"}},
		{"/**\n * @param int $a first\n * @param array<string, int> $long second\n * @throws Exception when it fails\n * @return bool yes\n */", [5]string{"/**\n * @param  int                $a    first\n * @param  array<string, int> $long second\n * @throws Exception          when it fails\n * @return bool               yes\n */", "/**\n * @param int $a first\n * @param array<string, int> $long second\n * @throws Exception when it fails\n * @return bool yes\n */", "/**\n * @param  int  $a  first\n * @param  array<string, int>  $long  second\n * @throws  Exception  when it fails\n * @return  bool  yes\n */", "/**\n * @param  int  $a  first\n * @param  array<string, int>  $long  second\n * @throws Exception when it fails\n * @return bool yes\n */", "/**\n * @param   int                  $a      first\n * @param   array<string, int>   $long   second\n * @throws Exception when it fails\n * @return bool yes\n */"}},
		{"/**\n * @return   array   the list\n */", [5]string{"/**\n * @return array the list\n */", "/**\n * @return array the list\n */", "/**\n * @return  array  the list\n */", "/**\n * @return array the list\n */", "/**\n * @return array the list\n */"}},
		{"/**\n * @var int $a\n */", [5]string{"/**\n * @var int $a\n */", "/**\n * @var int $a\n */", "/**\n * @var  int  $a\n */", "/**\n * @var int $a\n */", "/**\n * @var int $a\n */"}},
		{"/**\n * @var int\n */", [5]string{"/**\n * @var int\n */", "/**\n * @var int\n */", "/**\n * @var  int\n */", "/**\n * @var int\n */", "/**\n * @var int\n */"}},
		{"/** @var int $a */", [5]string{"/** @var int $a */", "/** @var int $a */", "/** @var int $a */", "/** @var int $a */", "/** @var int $a */"}},
		{"/** @param int $a */", [5]string{"/** @param int $a */", "/** @param int $a */", "/** @param int $a */", "/** @param int $a */", "/** @param int $a */"}},
		{"/**\n * @type string $a\n * @property int $b\n * @property-read string $cc\n */", [5]string{"/**\n * @type     string $a\n * @property int    $b\n * @property-read string $cc\n */", "/**\n * @type string $a\n * @property int $b\n * @property-read string $cc\n */", "/**\n * @type  string  $a\n * @property  int  $b\n * @property-read string $cc\n */", "/**\n * @type string $a\n * @property int $b\n * @property-read string $cc\n */", "/**\n * @type string $a\n * @property int $b\n * @property-read string $cc\n */"}},
		{"/**\n * @param int $a first line\n *   continued here\n * @param string $b\n */", [5]string{"/**\n * @param int    $a first line\n *                  continued here\n * @param string $b\n */", "/**\n * @param int $a first line\n *               continued here\n * @param string $b\n */", "/**\n * @param  int  $a  first line\n *                  continued here\n * @param  string  $b\n */", "/**\n * @param  int  $a  first line\n *                  continued here\n * @param  string  $b\n */", "/**\n * @param   int      $a   first line\n *                        continued here\n * @param   string   $b\n */"}},
		{"/**\n * @return int first line\n *   continued here\n */", [5]string{"/**\n * @return int first line\n *             continued here\n */", "/**\n * @return int first line\n *             continued here\n */", "/**\n * @return  int  first line\n *               continued here\n */", "/**\n * @return int first line\n *             continued here\n */", "/**\n * @return int first line\n *             continued here\n */"}},
		{"/**\n * @throws Exception first line\n * second line\n * third line\n */", [5]string{"/**\n * @throws Exception first line\n *                   second line\n *                   third line\n */", "/**\n * @throws Exception first line\n *                   second line\n *                   third line\n */", "/**\n * @throws  Exception  first line\n *                     second line\n *                     third line\n */", "/**\n * @throws Exception first line\n *                   second line\n *                   third line\n */", "/**\n * @throws Exception first line\n * second line\n * third line\n */"}},
		{"/**\n * @param int $a\n *\n * @return bool\n */", [5]string{"/**\n * @param int $a\n *\n * @return bool\n */", "/**\n * @param int $a\n *\n * @return bool\n */", "/**\n * @param  int  $a\n *\n * @return  bool\n */", "/**\n * @param  int  $a\n *\n * @return bool\n */", "/**\n * @param   int   $a\n *\n * @return bool\n */"}},
		{"/**\n * A summary.\n *\n * @param int $a\n * @param bool $b\n */", [5]string{"/**\n * A summary.\n *\n * @param int  $a\n * @param bool $b\n */", "/**\n * A summary.\n *\n * @param int $a\n * @param bool $b\n */", "/**\n * A summary.\n *\n * @param  int  $a\n * @param  bool  $b\n */", "/**\n * A summary.\n *\n * @param  int  $a\n * @param  bool  $b\n */", "/**\n * A summary.\n *\n * @param   int    $a\n * @param   bool   $b\n */"}},
		{"/**\n * @method   static \\Foo   bar(int $x)   Do the thing\n * @method $this baz()\n */", [5]string{"/**\n * @method static \\Foo  bar(int $x) Do the thing\n * @method        $this baz()\n */", "/**\n * @method static \\Foo bar(int $x) Do the thing\n * @method $this baz()\n */", "/**\n * @method  static  \\Foo  bar(int $x)  Do the thing\n * @method  $this  baz()\n */", "/**\n * @method static \\Foo bar(int $x) Do the thing\n * @method $this baz()\n */", "/**\n * @method   static \\Foo   bar(int $x)   Do the thing\n * @method $this baz()\n */"}},
		{"/**\n * @method static foo()\n * @method int bar()\n */", [5]string{"/**\n * @method static foo()\n * @method int    bar()\n */", "/**\n * @method static foo()\n * @method int bar()\n */", "/**\n * @method  static  foo()\n * @method  int  bar()\n */", "/**\n * @method static foo()\n * @method int bar()\n */", "/**\n * @method static foo()\n * @method int bar()\n */"}},
		{"/**\n * @method int bar()\n * @method string longer(int $a) with description\n */", [5]string{"/**\n * @method int    bar()\n * @method string longer(int $a) with description\n */", "/**\n * @method int bar()\n * @method string longer(int $a) with description\n */", "/**\n * @method  int  bar()\n * @method  string  longer(int $a)  with description\n */", "/**\n * @method int bar()\n * @method string longer(int $a) with description\n */", "/**\n * @method int bar()\n * @method string longer(int $a) with description\n */"}},
		{"/**\n * @method static int bar() description\n *   continued\n */", [5]string{"/**\n * @method static int bar() description\n *                          continued\n */", "/**\n * @method static int bar() description\n *                          continued\n */", "/**\n * @method  static  int  bar()  description\n *                              continued\n */", "/**\n * @method static int bar() description\n *                          continued\n */", "/**\n * @method static int bar() description\n *   continued\n */"}},
		{"/**\n * @param int\n */", [5]string{"/**\n * @param int\n */", "/**\n * @param int\n */", "/**\n * @param int\n */", "/**\n * @param int\n */", "/**\n * @param int\n */"}},
		{"/**\n * @param $a\n */", [5]string{"/**\n * @param $a\n */", "/**\n * @param $a\n */", "/**\n * @param  $a\n */", "/**\n * @param  $a\n */", "/**\n * @param     $a\n */"}},
		{"/**\n * @param int|string $a\n * @param (int|string)[] $b\n */", [5]string{"/**\n * @param int|string     $a\n * @param (int|string)[] $b\n */", "/**\n * @param int|string $a\n * @param (int|string)[] $b\n */", "/**\n * @param  int|string  $a\n * @param  (int|string)[]  $b\n */", "/**\n * @param  int|string  $a\n * @param  (int|string)[]  $b\n */", "/**\n * @param   int|string       $a\n * @param   (int|string)[]   $b\n */"}},
		{"/**\n * @param callable(int $x): bool $a\n * @param array{a: int, b: string} $b\n */", [5]string{"/**\n * @param callable(int $x): bool   $a\n * @param array{a: int, b: string} $b\n */", "/**\n * @param callable(int $x): bool $a\n * @param array{a: int, b: string} $b\n */", "/**\n * @param  callable(int $x): bool  $a\n * @param  array{a: int, b: string}  $b\n */", "/**\n * @param  callable(int $x): bool  $a\n * @param  array{a: int, b: string}  $b\n */", "/**\n * @param   callable(int $x): bool     $a\n * @param   array{a: int, b: string}   $b\n */"}},
		{"/**\n * @param étoile $a\n * @param int $bb\n */", [5]string{"/**\n * @param étoile  $a\n * @param int     $bb\n */", "/**\n * @param étoile $a\n * @param int $bb\n */", "/**\n * @param  étoile  $a\n * @param  int  $bb\n */", "/**\n * @param  étoile  $a\n * @param  int  $bb\n */", "/**\n * @param   étoile    $a\n * @param   int       $bb\n */"}},
		{"/**\n * @param int $é desc\n * @param int $bb desc\n */", [5]string{"/**\n * @param int $é desc\n * @param int $bb desc\n */", "/**\n * @param int $é desc\n * @param int $bb desc\n */", "/**\n * @param  int  $é  desc\n * @param  int  $bb  desc\n */", "/**\n * @param  int  $é  desc\n * @param  int  $bb  desc\n */", "/**\n * @param   int   $é   desc\n * @param   int   $bb   desc\n */"}},
		{"/**\n * @see Foo\n * @param int $a\n */", [5]string{"/**\n * @see Foo\n * @param int $a\n */", "/**\n * @see Foo\n * @param int $a\n */", "/**\n * @see Foo\n * @param  int  $a\n */", "/**\n * @see Foo\n * @param  int  $a\n */", "/**\n * @see     Foo\n * @param   int   $a\n */"}},
		{"/**\n * @deprecated use   other\n * @author  Someone\n */", [5]string{"/**\n * @deprecated use   other\n * @author  Someone\n */", "/**\n * @deprecated use   other\n * @author  Someone\n */", "/**\n * @deprecated use   other\n * @author  Someone\n */", "/**\n * @deprecated use   other\n * @author  Someone\n */", "/**\n * @deprecated use   other\n * @author  Someone\n */"}},
		{"/**\n * @phpstan-param int $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */", [5]string{"/**\n * @phpstan-param int $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */", "/**\n * @phpstan-param int $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */", "/**\n * @phpstan-param int $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */", "/**\n * @phpstan-param int $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */", "/**\n * @phpstan-param   int   $a\n * @psalm-return bool\n * @phpstan-method int foo()\n */"}},
		{"/**\n * @param int $a\n * @see Foo\n * @return bool\n */", [5]string{"/**\n * @param int $a\n * @see Foo\n * @return bool\n */", "/**\n * @param int $a\n * @see Foo\n * @return bool\n */", "/**\n * @param  int  $a\n * @see Foo\n * @return  bool\n */", "/**\n * @param  int  $a\n * @see Foo\n * @return bool\n */", "/**\n * @param    int    $a\n * @see      Foo\n * @return bool\n */"}},
		{"/**\n * @param int $a\n * @\n */", [5]string{"/**\n * @param int $a\n * @\n */", "/**\n * @param int $a\n * @\n */", "/**\n * @param  int  $a\n * @\n */", "/**\n * @param  int  $a\n * @\n */", "/**\n * @param   int   $a\n * @\n */"}},
		{"/**\n * @param int $a */", [5]string{"/**\n * @param int $a */", "/**\n * @param int $a */", "/**\n * @param int $a */", "/**\n * @param int $a */", "/**\n * @param int $a */"}},
		{"/**\n     * @param int $a\n     * @return bool\n     */", [5]string{"/**\n     * @param  int  $a\n     * @return bool\n     */", "/**\n     * @param int $a\n     * @return bool\n     */", "/**\n     * @param  int  $a\n     * @return  bool\n     */", "/**\n     * @param  int  $a\n     * @return bool\n     */", "/**\n     * @param    int    $a\n     * @return bool\n     */"}},
		{"class A\n{\n    /**\n     * @param int $a\n     * @param string $bb text\n     */\n    public function f($a, $bb) {}\n}", [5]string{"class A\n{\n    /**\n     * @param int    $a\n     * @param string $bb text\n     */\n    public function f($a, $bb) {}\n}", "class A\n{\n    /**\n     * @param int $a\n     * @param string $bb text\n     */\n    public function f($a, $bb) {}\n}", "class A\n{\n    /**\n     * @param  int  $a\n     * @param  string  $bb  text\n     */\n    public function f($a, $bb) {}\n}", "class A\n{\n    /**\n     * @param  int  $a\n     * @param  string  $bb  text\n     */\n    public function f($a, $bb) {}\n}", "class A\n{\n    /**\n     * @param   int      $a\n     * @param   string   $bb   text\n     */\n    public function f($a, $bb) {}\n}"}},
		{"class A\n{\n\t/**\n\t * @param int $a\n\t * @param string $bb text\n\t */\n\tpublic function f($a, $bb) {}\n}", [5]string{"class A\n{\n\t/**\n\t * @param int    $a\n\t * @param string $bb text\n\t */\n\tpublic function f($a, $bb) {}\n}", "class A\n{\n\t/**\n\t * @param int $a\n\t * @param string $bb text\n\t */\n\tpublic function f($a, $bb) {}\n}", "class A\n{\n\t/**\n\t * @param  int  $a\n\t * @param  string  $bb  text\n\t */\n\tpublic function f($a, $bb) {}\n}", "class A\n{\n\t/**\n\t * @param  int  $a\n\t * @param  string  $bb  text\n\t */\n\tpublic function f($a, $bb) {}\n}", "class A\n{\n\t/**\n\t * @param   int      $a\n\t * @param   string   $bb   text\n\t */\n\tpublic function f($a, $bb) {}\n}"}},
		{"/**\n * @param int $a @see other\n */", [5]string{"/**\n * @param int $a @see other\n */", "/**\n * @param int $a @see other\n */", "/**\n * @param  int  $a  @see other\n */", "/**\n * @param  int  $a  @see other\n */", "/**\n * @param   int   $a   @see other\n */"}},
		{"/**\n * @return $this\n * @param static $a\n */", [5]string{"/**\n * @return $this\n * @param  static $a\n */", "/**\n * @return $this\n * @param static $a\n */", "/**\n * @return  $this\n * @param  static  $a\n */", "/**\n * @return $this\n * @param  static  $a\n */", "/**\n * @return $this\n * @param    static   $a\n */"}},
		{"/**\n * @param int ...$a\n * @param int &$b\n */", [5]string{"/**\n * @param int ...$a\n * @param int &$b\n */", "/**\n * @param int ...$a\n * @param int &$b\n */", "/**\n * @param  int  ...$a\n * @param  int  &$b\n */", "/**\n * @param  int  ...$a\n * @param  int  &$b\n */", "/**\n * @param   int   ...$a\n * @param   int   &$b\n */"}},
		{"/**\n * @property-write int $a text\n * @property string $bb\n */", [5]string{"/**\n * @property-write int $a text\n * @property string $bb\n */", "/**\n * @property-write int $a text\n * @property string $bb\n */", "/**\n * @property-write int $a text\n * @property  string  $bb\n */", "/**\n * @property-write int $a text\n * @property string $bb\n */", "/**\n * @property-write int $a text\n * @property string $bb\n */"}},
		{"/**\n * @param int $a\n *   @see Foo\n * @return bool\n */", [5]string{"/**\n * @param  int  $a\n * @see Foo\n * @return bool\n */", "/**\n * @param int $a\n * @see Foo\n * @return bool\n */", "/**\n * @param  int  $a\n * @see Foo\n * @return  bool\n */", "/**\n * @param  int  $a\n * @see Foo\n * @return bool\n */", "/**\n * @param    int    $a\n * @see      Foo\n * @return bool\n */"}},
	}
	for _, c := range cases {
		for i, f := range fixers {
			got, changed := apply(t, f, "<?php\n"+c.src+"\n")
			if want := "<?php\n" + c.want[i] + "\n"; got != want || changed != (c.src != c.want[i]) {
				t.Errorf("config %d: %s\n got=%q changed=%v\nwant=%q", i, c.src, got, changed, want)
			}
		}
	}
}

// PHP-CS-Fixer rejects a spacing of zero or less, so there is no expected output to match.
func TestPhpdocAlignInvalidSpacingKeepsDefault(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @param  int  $a\n */\n"
	want := "<?php\n/**\n * @param int $a\n */\n"
	for _, spacing := range []any{0, -1, map[string]any{"param": 0}, map[string]any{"param": -1}, map[string]any{"_default": -1}} {
		f := PhpdocAlign{}.WithConfig(map[string]any{"spacing": spacing}).(fixerRule)
		if got, _ := apply(t, f, src); got != want {
			t.Errorf("spacing %v\n got=%q\nwant=%q", spacing, got, want)
		}
	}
}

// The expected output comes from the PHP engine.
func TestPhpdocAlignCustomTagNames(t *testing.T) {
	t.Parallel()
	f := PhpdocAlign{}.WithConfig(map[string]any{"tags": []any{"foo:bar", "été", "param", "x.y"}, "spacing": 2}).(fixerRule)
	got, _ := apply(t, f, "<?php\n/**\n * @foo:bar x\n * @été y\n * @param int $a\n * @x.y z\n */\n")
	if want := "<?php\n/**\n * @foo:bar  x\n * @été    y\n * @param    int  $a\n * @x.y      z\n */\n"; got != want {
		t.Errorf("\n got=%q\nwant=%q", got, want)
	}
}

// The expected output comes from the PHP engine, with tags from array_diff: "param" is left out.
func TestPhpdocAlignTagsWithGaps(t *testing.T) {
	t.Parallel()
	f := PhpdocAlign{}.WithConfig(map[string]any{"tags": map[string]any{"0": "method", "2": "property"}}).(fixerRule)
	src := "<?php\n/**\n * @param     int     $a\n */\n"
	if got, changed := apply(t, f, src); changed || got != src {
		t.Fatalf("changed=%v got=%q", changed, got)
	}
}
