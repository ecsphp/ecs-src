package rules

import "testing"

// Rules that know the visibility modifiers must know PHP 8.4's "private(set)" too.
func TestSetVisibilityInMemberRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"docblock before set-visibility is a phpdoc", PhpdocToComment{}, "<?php\nclass A {\n    /** @var int $a */\n    private(set) ?int $a = null;\n}\n", "<?php\nclass A {\n    /** @var int $a */\n    private(set) ?int $a = null;\n}\n"},
		{"protected set-visibility in a final class", ProtectedToPrivate{}, "<?php\nfinal class B {\n    protected(set) int $p = 1;\n    public function __construct(protected(set) int $r) {}\n}\n", "<?php\nfinal class B {\n    private(set) int $p = 1;\n    public function __construct(private(set) int $r) {}\n}\n"},
		{"one property per statement", SingleClassElementPerStatement{}, "<?php\nclass A {\n    private(set) int $c, $d;\n    protected private(set) int $e, $f;\n}\n", "<?php\nclass A {\n    private(set) int $c;\n    private(set) int $d;\n    protected private(set) int $e;\n    protected private(set) int $f;\n}\n"},
		{"properties before methods", OrderedClassElements{}, "<?php\nclass A { private(set) int $a = 1; public function x() {} protected(set) int $b = 2; }\n", "<?php\nclass A { private(set) int $a = 1; protected(set) int $b = 2; public function x() {} }\n"},
		{"property docblock spans lines", PhpdocLineSpan{}, "<?php\nclass A {\n    /** @var Foo */\n    protected(set) Foo $b;\n}\n", "<?php\nclass A {\n    /**\n     * @var Foo\n     */\n    protected(set) Foo $b;\n}\n"},
		{"property @var without name", PhpdocVarWithoutName{}, "<?php\nclass A {\n    /** @var int $a */\n    private(set) ?int $a = null;\n}\n", "<?php\nclass A {\n    /** @var int */\n    private(set) ?int $a = null;\n}\n"},
		{"superfluous property @var", NoSuperfluousPhpdocTags{}, "<?php\nfinal class A {\n    /**\n     * @var int\n     */\n    private(set) int $b;\n}\n", "<?php\nfinal class A {\n    /**\n     */\n    private(set) int $b;\n}\n"},
		{"superfluous @var of promoted properties", NoSuperfluousPhpdocTags{}, "<?php\nfinal class A {\n    public function __construct(\n        /**\n         * @var int\n         */\n        public private(set) int $t,\n        /**\n         * @var int\n         */\n        private(set) int $u,\n    ) {}\n}\n", "<?php\nfinal class A {\n    public function __construct(\n        /**\n         * @var int\n         */\n        public private(set) int $t,\n        /**\n         */\n        private(set) int $u,\n    ) {}\n}\n"},
		{"compact nullable type", CompactNullableTypeDeclaration{}, "<?php\nclass A {\n    private(set) ? int $h = null;\n}\n", "<?php\nclass A {\n    private(set) ?int $h = null;\n}\n"},
		{"type declaration spaces", TypeDeclarationSpaces{}, "<?php\nclass A {\n    public private(set)  int|string  $i;\n}\n", "<?php\nclass A {\n    public private(set)  int|string $i;\n}\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
