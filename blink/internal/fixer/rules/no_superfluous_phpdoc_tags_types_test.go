package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// nspAsECS is NoSuperfluousPhpdocTags with PHP-CS-Fixer's defaults plus
// allow_hidden_params and remove_inheritdoc.
func nspAsECS(t *testing.T) fixerRule {
	t.Helper()
	f, ok := ByName(`PhpCsFixer\Fixer\Phpdoc\NoSuperfluousPhpdocTagsFixer`)
	if !ok {
		t.Fatal("NoSuperfluousPhpdocTagsFixer is not registered")
	}
	return f.(fixer.ConfigurableFixer).WithConfig(map[string]any{"allow_hidden_params": true, "remove_inheritdoc": true}).(fixerRule)
}

// nspInClass wraps class members in a namespaced class importing Other\Thing.
func nspInClass(members string) string {
	return "<?php\n\nnamespace App;\n\nuse Other\\Thing;\n\nfinal class Foo\n{\n" + members + "}\n"
}

type nspCase struct{ name, src, want string }

func assertNspCases(t *testing.T, cases []nspCase) {
	t.Helper()
	f := nspAsECS(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src, want := nspInClass(c.src), nspInClass(c.want)
			assertFix(t, f, src, want, src != want)
		})
	}
}

// The examples blink kept: an untyped @param, and a nullable native type the
// phpdoc type only lacks null for. PHP-CS-Fixer removes every tag.
func TestNoSuperfluousPhpdocTagsUntypedAndNullable(t *testing.T) {
	t.Parallel()
	f := nspAsECS(t)
	assertFix(t, f, `<?php

final class Report
{
    /**
     * @param $value
     */
    public function format(int $value): string
    {
        return (string) $value;
    }

    /**
     * @return string
     */
    public function name(): ?string
    {
        return null;
    }

    /**
     * @var int
     */
    private ?int $id = null;

    /**
     * @var int
     */
    #[Column]
    private ?int $other = null;
}
`, `<?php

final class Report
{
    /**
     */
    public function format(int $value): string
    {
        return (string) $value;
    }

    /**
     */
    public function name(): ?string
    {
        return null;
    }

    /**
     */
    private ?int $id = null;

    /**
     */
    #[Column]
    private ?int $other = null;
}
`, true)
}

// A tag naming no type is superfluous unless a description follows it.
// "$this", "$self" and "$static" are phpdoc types, not variable names.
func TestNoSuperfluousPhpdocTagsUntypedTag(t *testing.T) {
	t.Parallel()
	assertNspCases(t, []nspCase{
		{"param on an untyped parameter",
			"    /**\n     * @param $value\n     */\n    public function f($value) {}\n",
			"    /**\n     */\n    public function f($value) {}\n"},
		{"param by reference and variadic",
			"    /**\n     * @param &$a\n     * @param ...$b\n     */\n    public function f(int &$a, int ...$b) {}\n",
			"    /**\n     */\n    public function f(int &$a, int ...$b) {}\n"},
		{"bare param",
			"    /**\n     * @param\n     */\n    public function f(int $a) {}\n",
			"    /**\n     */\n    public function f(int $a) {}\n"},
		{"param with a description",
			"    /**\n     * @param $value the value\n     */\n    public function f(int $value) {}\n",
			"    /**\n     * @param $value the value\n     */\n    public function f(int $value) {}\n"},
		{"param with a wrapped description",
			"    /**\n     * @param $value\n     *        the value\n     */\n    public function f(int $value) {}\n",
			"    /**\n     * @param $value\n     *        the value\n     */\n    public function f(int $value) {}\n"},
		{"param type without a variable",
			"    /**\n     * @param int\n     */\n    public function f(int $value) {}\n",
			"    /**\n     * @param int\n     */\n    public function f(int $value) {}\n"},
		{"param named like a self reference type",
			"    /**\n     * @param $self\n     * @param $STATIC\n     * @param $thisValue\n     */\n    public function f(int $self, int $STATIC, int $thisValue) {}\n",
			"    /**\n     * @param $self\n     * @param $STATIC\n     */\n    public function f(int $self, int $STATIC, int $thisValue) {}\n"},
		{"self reference type on an unused param",
			"    /**\n     * @param $self\n     */\n    public function f(int $other) {}\n",
			"    /**\n     */\n    public function f(int $other) {}\n"},
		{"two variables",
			"    /**\n     * @param $a $b\n     */\n    public function f(int $a, int $b) {}\n",
			"    /**\n     * @param $a $b\n     */\n    public function f(int $a, int $b) {}\n"},
		{"single-line param",
			"    /** @param $value */\n    public function f(int $value) {}\n",
			"    /**  */\n    public function f(int $value) {}\n"},
		{"closure param",
			"    public function f(): void\n    {\n        $g = /** @param $y */ function (int $y) {};\n    }\n",
			"    public function f(): void\n    {\n        $g = /**  */ function (int $y) {};\n    }\n"},
		{"bare return",
			"    /**\n     * @return\n     */\n    public function f(): ?string {}\n\n    /**\n     * @return\n     */\n    public function g() {}\n",
			"    /**\n     */\n    public function f(): ?string {}\n\n    /**\n     */\n    public function g() {}\n"},
		{"bare return with a wrapped description",
			"    /**\n     * @return\n     *         the name\n     */\n    public function f(): ?string {}\n",
			"    /**\n     * @return\n     *         the name\n     */\n    public function f(): ?string {}\n"},
		{"bare var and var with only a name",
			"    /**\n     * @var\n     */\n    private ?int $a = null;\n\n    /**\n     * @var $b\n     */\n    private $b;\n\n    /**\n     * @var $selfish\n     */\n    private ?int $c = null;\n",
			"    /**\n     */\n    private ?int $a = null;\n\n    /**\n     */\n    private $b;\n\n    /**\n     */\n    private ?int $c = null;\n"},
		{"var with a name and a description",
			"    /**\n     * @var $id the id\n     */\n    private ?int $id = null;\n\n    /**\n     * @var $a $b\n     */\n    private ?int $a = null;\n",
			"    /**\n     * @var $id the id\n     */\n    private ?int $id = null;\n\n    /**\n     * @var $a $b\n     */\n    private ?int $a = null;\n"},
		{"var with a wrapped description",
			"    /**\n     * @var\n     *      the id\n     */\n    private ?int $id = null;\n",
			"    /**\n     * @var\n     *      the id\n     */\n    private ?int $id = null;\n"},
		{"var self reference type",
			"    /**\n     * @var $this\n     */\n    private ?self $a = null;\n\n    /**\n     * @var $self\n     */\n    private ?int $b = null;\n",
			"    /**\n     * @var $this\n     */\n    private ?self $a = null;\n\n    /**\n     * @var $self\n     */\n    private ?int $b = null;\n"},
		{"var before a statement",
			"    public function f($x)\n    {\n        /** @var $x */\n        echo $x;\n        /**\n         * @var\n         */\n        global $db;\n        /** @var $x */\n        DEBUG or $x->log();\n        /** @var $x */\n        return $x;\n    }\n",
			"    public function f($x)\n    {\n        /** @var $x */\n        echo $x;\n        /**\n         * @var\n         */\n        global $db;\n        /** @var $x */\n        DEBUG or $x->log();\n        /** @var $x */\n        return $x;\n    }\n"},
		{"var on parameters",
			"    public function f(\n        /** @var */\n        array $a,\n        /** @var $b */\n        Enum $b,\n        /** @var array */\n        ?array $c,\n        /** @var callable */\n        callable $d,\n        /** @var callable */\n        ?callable $e,\n        /** @var */\n        \\Closure|callable $f,\n    ) {}\n",
			"    public function f(\n        /**  */\n        array $a,\n        /**  */\n        Enum $b,\n        /**  */\n        ?array $c,\n        /** @var callable */\n        callable $d,\n        /** @var callable */\n        ?callable $e,\n        /** @var */\n        \\Closure|callable $f,\n    ) {}\n"},
	})
}

// PHP-CS-Fixer compares the phpdoc types with the native ones, then once more
// with null added to the phpdoc types, as lists that keep duplicates. A
// parameter defaulting to null counts as nullable, unless a "?" already says so.
func TestNoSuperfluousPhpdocTagsNullableType(t *testing.T) {
	t.Parallel()
	assertNspCases(t, []nspCase{
		{"return without null on a nullable type",
			"    /**\n     * @return string\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return int|string\n     */\n    public function b(): int|string|null {}\n\n    /**\n     * @return string\n     */\n    public function c(): string|null {}\n\n    /**\n     * @return static\n     */\n    public function d(): ?static {}\n\n    /**\n     * @return self\n     */\n    public function e(): ?self {}\n\n    /**\n     * @return Foo\n     */\n    public function f(): ?self {}\n\n    /**\n     * @return Thing\n     */\n    public function g(): ?Thing {}\n\n    /**\n     * @return string\n     */\n    abstract public function h(): ?string;\n",
			"    /**\n     */\n    public function a(): ?string {}\n\n    /**\n     */\n    public function b(): int|string|null {}\n\n    /**\n     */\n    public function c(): string|null {}\n\n    /**\n     */\n    public function d(): ?static {}\n\n    /**\n     */\n    public function e(): ?self {}\n\n    /**\n     */\n    public function f(): ?self {}\n\n    /**\n     */\n    public function g(): ?Thing {}\n\n    /**\n     */\n    abstract public function h(): ?string;\n"},
		{"return with null on a nullable type",
			"    /**\n     * @return string|null\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return null|string\n     */\n    public function b(): ?string {}\n\n    /**\n     * @return ?string\n     */\n    public function c(): null|string {}\n",
			"    /**\n     */\n    public function a(): ?string {}\n\n    /**\n     */\n    public function b(): ?string {}\n\n    /**\n     */\n    public function c(): null|string {}\n"},
		{"return kept on a nullable type",
			"    /**\n     * @return null\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return int\n     */\n    public function b(): int|string|null {}\n\n    /**\n     * @return int|null\n     */\n    public function c(): int|string|null {}\n\n    /**\n     * @return string the name\n     */\n    public function d(): ?string {}\n\n    /**\n     * @return mixed\n     */\n    public function e(): ?string {}\n\n    /**\n     * @return string[]\n     */\n    public function f(): ?array {}\n\n    /**\n     * @return string|string\n     */\n    public function g(): ?string {}\n",
			"    /**\n     * @return null\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return int\n     */\n    public function b(): int|string|null {}\n\n    /**\n     * @return int|null\n     */\n    public function c(): int|string|null {}\n\n    /**\n     * @return string the name\n     */\n    public function d(): ?string {}\n\n    /**\n     * @return mixed\n     */\n    public function e(): ?string {}\n\n    /**\n     * @return string[]\n     */\n    public function f(): ?array {}\n\n    /**\n     * @return string|string\n     */\n    public function g(): ?string {}\n"},
		{"return kept when only the phpdoc type has null",
			"    /**\n     * @return string|null\n     */\n    public function a(): string {}\n\n    /**\n     * @return ?string\n     */\n    public function b(): string {}\n\n    /**\n     * @return string|null\n     */\n    public function c(): null {}\n\n    /**\n     * @return string|string\n     */\n    public function d(): string {}\n",
			"    /**\n     * @return string|null\n     */\n    public function a(): string {}\n\n    /**\n     * @return ?string\n     */\n    public function b(): string {}\n\n    /**\n     * @return string|null\n     */\n    public function c(): null {}\n\n    /**\n     * @return string|string\n     */\n    public function d(): string {}\n"},
		{"var on a nullable property",
			"    /**\n     * @var int|string\n     */\n    public int|string|null $a = null;\n\n    /**\n     * @var int\n     */\n    private static ?int $b = null;\n\n    /**\n     * @var Thing\n     */\n    #[Column(nullable: true)]\n    #[Other]\n    private readonly ?Thing $c;\n",
			"    /**\n     */\n    public int|string|null $a = null;\n\n    /**\n     */\n    private static ?int $b = null;\n\n    /**\n     */\n    #[Column(nullable: true)]\n    #[Other]\n    private readonly ?Thing $c;\n"},
		{"var kept on a nullable property",
			"    /**\n     * @var null\n     */\n    private ?int $a = null;\n\n    /**\n     * @var int[]\n     */\n    private ?array $b = null;\n\n    /**\n     * @var int\n     */\n    public int|string|null $c = null;\n",
			"    /**\n     * @var null\n     */\n    private ?int $a = null;\n\n    /**\n     * @var int[]\n     */\n    private ?array $b = null;\n\n    /**\n     * @var int\n     */\n    public int|string|null $c = null;\n"},
		{"var on promoted parameters",
			"    public function __construct(\n        /**\n         * @var string\n         */\n        public ?string $a,\n        /**\n         * @var string\n         */\n        readonly ?string $b,\n    ) {}\n",
			"    public function __construct(\n        /**\n         * @var string\n         */\n        public ?string $a,\n        /**\n         */\n        readonly ?string $b,\n    ) {}\n"},
		{"param on a nullable parameter",
			"    /**\n     * @param string $a\n     * @param int ...$b\n     */\n    public function f(?string $a, ?int ...$b) {}\n\n    /**\n     * @param int &$a\n     * @param bool $b\n     */\n    public function g(?int &$a, #[\\SensitiveParameter] ?bool $b = false) {}\n",
			"    /**\n     */\n    public function f(?string $a, ?int ...$b) {}\n\n    /**\n     */\n    public function g(?int &$a, #[\\SensitiveParameter] ?bool $b = false) {}\n"},
		{"param defaulting to null",
			"    /**\n     * @param string|null $a\n     * @param int|null $b\n     * @param int|string|null $c\n     * @param string|null $d\n     * @param string $e\n     */\n    public function f(string $a = null, int &$b = null, int|string $c = null, string $d = NULL, ?string $e = null) {}\n\n    /**\n     * @param null|int $a\n     * @param int|null $b\n     */\n    public function g(null|int $a = null, int $b /* c */ = /* d */ null) {}\n",
			"    /**\n     */\n    public function f(string $a = null, int &$b = null, int|string $c = null, string $d = NULL, ?string $e = null) {}\n\n    /**\n     */\n    public function g(null|int $a = null, int $b /* c */ = /* d */ null) {}\n"},
		{"param kept when defaulting to null",
			"    /**\n     * @param string $a\n     * @param null $b\n     * @param int|null $c\n     * @param string|null $d\n     */\n    public function f(string|null $a = null, string $b = null, &$c = null, string $d = \\null) {}\n",
			"    /**\n     * @param string $a\n     * @param null $b\n     * @param int|null $c\n     * @param string|null $d\n     */\n    public function f(string|null $a = null, string $b = null, &$c = null, string $d = \\null) {}\n"},
	})
}

// A description goes on after one blank line and onto the closing line, as
// DocBlock::findAnnotationLength() reads it; a second blank line or a tag ends it.
func TestNoSuperfluousPhpdocTagsDescriptionBelowTag(t *testing.T) {
	t.Parallel()
	afterBlank := "    /**\n     * @return string\n     *\n     * the name\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return string\n     *\n     * the name\n     */\n    public function b(): string {}\n\n    /**\n     * @return\n     *\n     * the name\n     */\n    public function c(): string {}\n\n    /**\n     * @param $x\n     *\n     * the value\n     * @param int $y\n     *\n     * the other\n     */\n    public function d(int $x, int $y) {}\n\n    /**\n     * @var int\n     *\n     * the id\n     */\n    private ?int $e = null;\n\n    /**\n     * @var int\n     *\n     * the id\n     */\n    private int $f = 0;\n"
	onClosingLine := "    /**\n     * @return string\n     * the name */\n    public function a(): ?string {}\n\n    /**\n     * @return string\n     *\n     * the name */\n    public function b(): string {}\n"
	assertNspCases(t, []nspCase{
		{"description after a blank line", afterBlank, afterBlank},
		{"description on the closing line", onClosingLine, onClosingLine},
		{"no description below",
			"    /**\n     * @return string\n     *\n     * @see other\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return string\n     *\n     *\n     * the name\n     */\n    public function b(): string {}\n\n    /**\n     * @return string\n     *\n     */\n    public function c(): ?string {}\n\n    /**\n     * @return string\n     * ***\n     * the name\n     */\n    public function d(): string {}\n\n    /**\n     * @return string\n     * see *@other\n     */\n    public function e(): string {}\n",
			"    /**\n     *\n     * @see other\n     */\n    public function a(): ?string {}\n\n    /**\n     *\n     *\n     * the name\n     */\n    public function b(): string {}\n\n    /**\n     *\n     */\n    public function c(): ?string {}\n\n    /**\n     * @return string\n     * ***\n     * the name\n     */\n    public function d(): string {}\n\n    /**\n     * see *@other\n     */\n    public function e(): string {}\n"},
	})
}

// PHP-CS-Fixer reads a type with an empty member ("string|") as a description,
// and maps only a lowercase "self" to the class.
func TestNoSuperfluousPhpdocTagsEmptyMemberAndSelfCase(t *testing.T) {
	t.Parallel()
	emptyMember := "    /**\n     * @return string|\n     */\n    public function a(): ?string {}\n\n    /**\n     * @return string|\n     */\n    public function b(): string {}\n\n    /**\n     * @return |string\n     */\n    public function c(): string {}\n\n    /**\n     * @return string||int\n     */\n    public function d(): int|string {}\n\n    /**\n     * @param int& $x\n     */\n    public function e(int &$x) {}\n\n    /**\n     * @var int|\n     */\n    private ?int $f = null;\n\n    /**\n     * @var int|\n     */\n    private int $g = 0;\n\n    /**\n     * @return ?\n     */\n    public function h(): null {}\n\n    /**\n     * @return &int\n     */\n    public function i(): int {}\n"
	assertNspCases(t, []nspCase{
		{"type with an empty member", emptyMember, emptyMember},
		{"self in another case",
			"    /**\n     * @return SELF\n     */\n    public function a(): ?self {}\n\n    /**\n     * @return SELF\n     */\n    public function b(): self {}\n\n    /**\n     * @return self\n     */\n    public function c(): ?SELF {}\n\n    /**\n     * @return Foo\n     */\n    public function d(): SELF {}\n\n    /**\n     * @return Self\n     */\n    public function e(): Self {}\n\n    /**\n     * @return self\n     */\n    public function f(): ?self {}\n",
			"    /**\n     * @return SELF\n     */\n    public function a(): ?self {}\n\n    /**\n     * @return SELF\n     */\n    public function b(): self {}\n\n    /**\n     * @return self\n     */\n    public function c(): ?SELF {}\n\n    /**\n     * @return Foo\n     */\n    public function d(): SELF {}\n\n    /**\n     */\n    public function e(): Self {}\n\n    /**\n     */\n    public function f(): ?self {}\n"},
	})
}

// PHP-CS-Fixer tracks classes, interfaces and enums, not traits: in a trait
// "self" stays "self" and the trait's name is a different type.
func TestNoSuperfluousPhpdocTagsSelfInTrait(t *testing.T) {
	t.Parallel()
	assertFix(t, nspAsECS(t), `<?php

namespace App;

trait Greets
{
    use Other;

    /**
     * @return Greets
     */
    public function a(): ?self {}

    /**
     * @return Greets
     */
    public function b(): self {}

    /**
     * @return self
     */
    public function c(): Greets {}

    /**
     * @return self
     */
    public function d(): ?self {}

    /**
     * @return Other
     */
    public function e(): ?Other {}
}

final class Host
{
    /**
     * @return Host
     */
    public function f(): ?self {}
}
`, `<?php

namespace App;

trait Greets
{
    use Other;

    /**
     * @return Greets
     */
    public function a(): ?self {}

    /**
     * @return Greets
     */
    public function b(): self {}

    /**
     * @return self
     */
    public function c(): Greets {}

    /**
     */
    public function d(): ?self {}

    /**
     */
    public function e(): ?Other {}
}

final class Host
{
    /**
     */
    public function f(): ?self {}
}
`, true)
}
