package rules

import "testing"

// TestOrderedClassElementsMatchesPHPCSFixer sorts one class under the default
// order and other configurations (custom orders, method:name, alpha and
// case-sensitive sorting); the expected outputs come from PHP-CS-Fixer.
func TestOrderedClassElementsMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	in := "<?php\nclass A\n{\n    private $a;\n    public $b;\n    // comment for m\n    public function m() {} // trailing\n    /** doc */\n    #[Attr]\n    protected static function s() {}\n    public function __construct() {}\n    const X = 1;\n    private const Y = 2;\n    use T;\n    public function __toString() { return ''; }\n    protected function setUp(): void {}\n    public static $st;\n    abstract protected function ab();\n    public function __destruct() {}\n    public readonly int $ro;\n    protected function tearDown(): void {}\n    public function &ref() {}\n}\nenum E: string\n{\n    const Z = 'z';\n    case B = 'b';\n    case A = 'a';\n    public function f() {}\n}\n$o = new class {\n    public function x() {}\n    public $y;\n};\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"def", map[string]any{}, "<?php\nclass A\n{\n    use T;\n    const X = 1;\n    private const Y = 2;\n    public $b;\n    public static $st;\n    public readonly int $ro;\n    private $a;\n    public function __construct() {}\n    public function __destruct() {}\n    public function __toString() { return ''; }\n    protected function setUp(): void {}\n    protected function tearDown(): void {}\n    // comment for m\n    public function m() {} // trailing\n    public function &ref() {}\n    /** doc */\n    #[Attr]\n    protected static function s() {}\n    abstract protected function ab();\n}\nenum E: string\n{\n    case B = 'b';\n    case A = 'a';\n    const Z = 'z';\n    public function f() {}\n}\n$o = new class {\n    public $y;\n    public function x() {}\n};\n"},
		{"usetrait", map[string]any{"order": []any{"use_trait"}}, "<?php\nclass A\n{\n    use T;\n    private $a;\n    public $b;\n    // comment for m\n    public function m() {} // trailing\n    /** doc */\n    #[Attr]\n    protected static function s() {}\n    public function __construct() {}\n    const X = 1;\n    private const Y = 2;\n    public function __toString() { return ''; }\n    protected function setUp(): void {}\n    public static $st;\n    abstract protected function ab();\n    public function __destruct() {}\n    public readonly int $ro;\n    protected function tearDown(): void {}\n    public function &ref() {}\n}\nenum E: string\n{\n    const Z = 'z';\n    case B = 'b';\n    case A = 'a';\n    public function f() {}\n}\n$o = new class {\n    public function x() {}\n    public $y;\n};\n"},
		{"alpha", map[string]any{"order": []any{"method_public", "property", "method:m"}, "sort_algorithm": "alpha"}, "<?php\nclass A\n{\n    public function &ref() {}\n    public function __construct() {}\n    public function __destruct() {}\n    public function __toString() { return ''; }\n    private $a;\n    public $b;\n    public readonly int $ro;\n    public static $st;\n    // comment for m\n    public function m() {} // trailing\n    abstract protected function ab();\n    /** doc */\n    #[Attr]\n    protected static function s() {}\n    protected function setUp(): void {}\n    use T;\n    protected function tearDown(): void {}\n    const X = 1;\n    private const Y = 2;\n}\nenum E: string\n{\n    public function f() {}\n    case A = 'a';\n    case B = 'b';\n    const Z = 'z';\n}\n$o = new class {\n    public function x() {}\n    public $y;\n};\n"},
		{"cs", map[string]any{"order": []any{"public", "protected", "private", "phpunit", "method_static", "property_readonly"}, "sort_algorithm": "alpha", "case_sensitive": true}, "<?php\nclass A\n{\n    public $b;\n    public static $st;\n    public function &ref() {}\n    const X = 1;\n    public function __construct() {}\n    public function __destruct() {}\n    public function __toString() { return ''; }\n    // comment for m\n    public function m() {} // trailing\n    abstract protected function ab();\n    private $a;\n    private const Y = 2;\n    protected function setUp(): void {}\n    protected function tearDown(): void {}\n    /** doc */\n    #[Attr]\n    protected static function s() {}\n    public readonly int $ro;\n    use T;\n}\nenum E: string\n{\n    case A = 'a';\n    case B = 'b';\n    const Z = 'z';\n    public function f() {}\n}\n$o = new class {\n    public $y;\n    public function x() {}\n};\n"},
	}
	for _, c := range cases {
		f := OrderedClassElements{}.WithConfig(c.config).(fixerRule)
		if got, _ := apply(t, f, in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
