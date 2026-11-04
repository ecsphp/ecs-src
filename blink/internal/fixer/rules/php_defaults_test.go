package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// TestRegistryResolvesPHPCSFixerDefaults checks that a fixer looked up by name,
// as the ECS bridge does for a rule dumped with an empty config, behaves like
// PHP-CS-Fixer's default configuration.
func TestRegistryResolvesPHPCSFixerDefaults(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rule, in, want string
	}{
		{
			`PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`,
			"<?php\nif ($x === null) {}\nif ($a == 5) {}\nif ($b < 3) {}\n",
			"<?php\nif (null === $x) {}\nif (5 == $a) {}\nif ($b < 3) {}\n",
		},
		{
			// annotations defaults to [], which removes nothing
			`PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocAnnotationRemoveFixer`,
			"<?php\n/**\n * @author Foo\n * @package Bar\n */\nclass A {}\n",
			"<?php\n/**\n * @author Foo\n * @package Bar\n */\nclass A {}\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\AlignMultilineCommentFixer`,
			"<?php\nfunction f() {\n    /*\n     * block\n        */\n    /**\n       * doc\n     */\n    return 1;\n}\n",
			"<?php\nfunction f() {\n    /*\n     * block\n        */\n    /**\n     * doc\n     */\n    return 1;\n}\n",
		},
		{
			`PhpCsFixer\Fixer\Basic\BracesPositionFixer`,
			"<?php\n$f = function ()\n{\n    return 1;\n};\n",
			"<?php\n$f = function () {\n    return 1;\n};\n",
		},
		{
			`PhpCsFixer\Fixer\Operator\ConcatSpaceFixer`,
			"<?php\n$a = 'x' . 'y';\n",
			"<?php\n$a = 'x'.'y';\n",
		},
		{
			`PhpCsFixer\Fixer\FunctionNotation\FunctionDeclarationFixer`,
			"<?php\nfunction f($a, $b,) {}\n",
			"<?php\nfunction f($a, $b) {}\n",
		},
		{
			`PhpCsFixer\Fixer\LanguageConstruct\FunctionToConstantFixer`,
			"<?php\nclass A { function f() { return get_class(); } }\n",
			"<?php\nclass A { function f() { return self::class; } }\n",
		},
		{
			`PhpCsFixer\Fixer\FunctionNotation\MethodArgumentSpaceFixer`,
			"<?php\nfoo(<<<EOT\nx\nEOT , 2 , 3);\n",
			"<?php\nfoo(<<<EOT\nx\nEOT , 2, 3);\n",
		},
		{
			`PhpCsFixer\Fixer\Whitespace\NoSpacesAroundOffsetFixer`,
			"<?php\n$a = $b ['x'];\n$c = (object) ['k' => 1];\n$d = $e[ 1 ];\n",
			"<?php\n$a = $b['x'];\n$c = (object) ['k' => 1];\n$d = $e[1];\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\NoSuperfluousPhpdocTagsFixer`,
			"<?php\n/**\n * @param mixed $a\n * @param int $gone\n * @return mixed\n */\nfunction f($a) { return $a; }\n",
			"<?php\n/**\n */\nfunction f($a) { return $a; }\n",
		},
		{
			`PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`,
			"<?php\n$f = foo($a, static fn ($c): bool\n    => $c->ok(),);\n$g = bar(1, 2,);\n#[Attr(a: 1,)]\nclass C {}\n",
			"<?php\n$f = foo($a, static fn ($c): bool\n    => $c->ok(),);\n$g = bar(1, 2);\n#[Attr(a: 1,)]\nclass C {}\n",
		},
		{
			`PhpCsFixer\Fixer\ArrayNotation\NoWhitespaceBeforeCommaInArrayFixer`,
			"<?php\n$a = [<<<EOT\nx\nEOT , 2 , 3];\n",
			"<?php\n$a = [<<<EOT\nx\nEOT , 2, 3];\n",
		},
		{
			`PhpCsFixer\Fixer\FunctionNotation\NullableTypeDeclarationForDefaultNullValueFixer`,
			"<?php\nfunction f(int|string $a = null, int $b = null) {}\n",
			"<?php\nfunction f(int|string|null $a = null, ?int $b = null) {}\n",
		},
		{
			`PhpCsFixer\Fixer\Import\OrderedImportsFixer`,
			"<?php\nnamespace N;\nuse Bbb\\B;\nuse function Aaa\\f;\n",
			"<?php\nnamespace N;\nuse function Aaa\\f;\nuse Bbb\\B;\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\PhpdocInlineTagNormalizerFixer`,
			"<?php\n/**\n * See { @see Foo } and {@link http://x }.\n */\nfunction f() {}\n",
			"<?php\n/**\n * See { @see Foo } and {@link http://x}.\n */\nfunction f() {}\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\PhpdocLineSpanFixer`,
			"<?php\n/** Class doc. */\nclass A {\n    /** Prop. */\n    public $p;\n}\n/** Fn doc. */\nfunction f() {}\n",
			"<?php\n/**\n * Class doc.\n */\nclass A {\n    /**\n     * Prop.\n     */\n    public $p;\n}\n/**\n * Fn doc.\n */\nfunction f() {}\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\PhpdocNoAliasTagFixer`,
			"<?php\n/**\n * @property-read int $a\n * @property-write int $b\n * @type int\n */\nclass A {}\n",
			"<?php\n/**\n * @property int $a\n * @property int $b\n * @var int\n */\nclass A {}\n",
		},
		{
			`PhpCsFixer\Fixer\Phpdoc\PhpdocTypesFixer`,
			"<?php\n/**\n * @param Boolean $a\n * @param Resource $r\n * @return Never\n */\nfunction f($a, $r) {}\n",
			"<?php\n/**\n * @param boolean $a\n * @param resource $r\n * @return Never\n */\nfunction f($a, $r) {}\n",
		},
		{
			`PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`,
			"<?php\nfoo(a:1);\nfunction g() { yield  from [1]; }\nforeach ($a  as $b) {}\n$f = function () use($x) {};\n",
			"<?php\nfoo(a: 1);\nfunction g() { yield from [1]; }\nforeach ($a as $b) {}\n$f = function () use ($x) {};\n",
		},
		{
			`PhpCsFixer\Fixer\Whitespace\TypesSpacesFixer`,
			"<?php\ntry {} catch (A | B $e) {}\nfunction f(int | string $x) {}\n",
			"<?php\ntry {} catch (A|B $e) {}\nfunction f(int|string $x) {}\n",
		},
		{
			`PhpCsFixer\Fixer\Operator\UnaryOperatorSpacesFixer`,
			"<?php\n$a = - $b;\n$c = ! $d;\n$e ++;\n",
			"<?php\n$a = -$b;\n$c = !$d;\n$e++;\n",
		},
	}
	for _, c := range cases {
		f, ok := ByName(c.rule)
		if !ok {
			t.Fatalf("%s: not registered", c.rule)
		}
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("%s with no options:\n got %q\nwant %q", c.rule, got, c.want)
		}
	}
}

// TestPartialConfigKeepsPHPCSFixerDefaults checks that options a dumped config
// leaves out (they equal the default) keep PHP-CS-Fixer's default.
func TestPartialConfigKeepsPHPCSFixerDefaults(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rule     string
		config   map[string]any
		in, want string
	}{
		{
			`PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`,
			map[string]any{"less_and_greater": true},
			"<?php\nif ($x === null) {}\nif ($b < 3) {}\n",
			"<?php\nif (null === $x) {}\nif (3 > $b) {}\n",
		},
		{
			// case_sensitive defaults to true
			`PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocAnnotationRemoveFixer`,
			map[string]any{"annotations": []any{"author"}},
			"<?php\n/**\n * @Author someone\n * @author other\n */\nfunction f() {}\n",
			"<?php\n/**\n * @Author someone\n */\nfunction f() {}\n",
		},
		{
			// allow_unused_params defaults to false
			`PhpCsFixer\Fixer\Phpdoc\NoSuperfluousPhpdocTagsFixer`,
			map[string]any{"allow_mixed": true},
			"<?php\n/**\n * @param mixed $a\n * @param int $gone\n */\nfunction f($a) {}\n",
			"<?php\n/**\n * @param mixed $a\n */\nfunction f($a) {}\n",
		},
		{
			// sort_algorithm and imports_order keep their defaults
			`PhpCsFixer\Fixer\Import\OrderedImportsFixer`,
			map[string]any{"case_sensitive": false},
			"<?php\nnamespace N;\nuse Bbb\\B;\nuse function Aaa\\f;\n",
			"<?php\nnamespace N;\nuse function Aaa\\f;\nuse Bbb\\B;\n",
		},
	}
	for _, c := range cases {
		f, _ := ByName(c.rule)
		f = f.(fixer.ConfigurableFixer).WithConfig(c.config)
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("%s with %v:\n got %q\nwant %q", c.rule, c.config, got, c.want)
		}
	}
}
