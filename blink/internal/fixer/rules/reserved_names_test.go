package rules

import "testing"

// Reserved words used as names keep their case; the keywords around them are fixed.
func TestReservedWordNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"keywords lowercased, reserved-word names kept", LowercaseKeywords{}, "<?php\nCLASS A {\n    USE T { foo AS PROTECTED NEW; LIST AS PRIVATE; }\n    CONST A = 1, NEW = 2;\n    CONST PUBLIC = B::PUBLIC;\n    CONST STRING DEFAULT = 'x';\n    PUBLIC FUNCTION &LIST() { RETURN [NEW STATIC, STATIC::X]; }\n}\n", "<?php\nclass A {\n    use T { foo as protected NEW; LIST as private; }\n    const A = 1, NEW = 2;\n    const public = B::PUBLIC;\n    const STRING DEFAULT = 'x';\n    public function &LIST() { return [new static, static::X]; }\n}\n"},
		{"keywords lowercased in alternative syntax and imports", LowercaseKeywords{}, "<?php\nUSE FUNCTION Foo\\bar;\nSWITCH ($a) { DEFAULT: BREAK; }\nIF ($a): ECHO 1; ELSE: ECHO 2; ENDIF;\n", "<?php\nuse function Foo\\bar;\nswitch ($a) { default: break; }\nif ($a): echo 1; else: echo 2; endif;\n"},
		{"reserved-word constant name after a comma", SingleSpaceAroundConstruct{}, "<?php\nclass A {\n    const A = 1, NEW =2;\n}\n", "<?php\nclass A {\n    const A = 1, NEW =2;\n}\n"},
		{"a method named list is not a list()", ListSyntax{}, "<?php\nclass A {\n    PUBLIC FUNCTION &LIST(): array { return $this->list(1); }\n}\nlist($a) = $b;\n", "<?php\nclass A {\n    PUBLIC FUNCTION &LIST(): array { return $this->list(1); }\n}\n[$a] = $b;\n"},
		{"die as a named argument or enum case", NoAliasLanguageConstructCall{}, "<?php\nfoo(die: 1, exit: 2);\nenum E { case Die; }\ndie(1);\n", "<?php\nfoo(die: 1, exit: 2);\nenum E { case Die; }\nexit(1);\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
