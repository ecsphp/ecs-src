package rules

import "testing"

// TestNullableTypeDeclarationForDefaultNullValueUnions covers unions,
// intersections and DNF types in both modes; the expected outputs come from
// PHP-CS-Fixer.
func TestNullableTypeDeclarationForDefaultNullValueUnions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		nullable bool
		in, want string
	}{
		{
			"use_nullable_type_declaration true",
			true,
			"<?php\nfunction f(A&B $d = null, ?int $e = null, int|string $u = null, A|(B&C) $g = null, (A&B)|C $h = null, int|null $i = null, mixed $j = null, null $k = null, Foo &$l = null) {}\nclass K { function __construct(private A&B $p = null, readonly int $q = null) {} }\n$c = fn (int $x = null) => 1;\n",
			"<?php\nfunction f((A&B)|null $d = null, ?int $e = null, int|string|null $u = null, A|(B&C)|null $g = null, (A&B)|C|null $h = null, int|null $i = null, mixed $j = null, null $k = null, ?Foo &$l = null) {}\nclass K { function __construct(private (A&B)|null $p = null, readonly ?int $q = null) {} }\n$c = fn (?int $x = null) => 1;\n",
		},
		{
			"asymmetric visibility is not a DNF group",
			true,
			"<?php\nclass A {\n    public function __construct(\n        public private(set) ?string $r = null,\n        public private(set) int $i = null,\n        protected private(set) (A&B)|null $d = null,\n    ) {}\n}\n",
			"<?php\nclass A {\n    public function __construct(\n        public private(set) ?string $r = null,\n        public private(set) ?int $i = null,\n        protected private(set) (A&B)|null $d = null,\n    ) {}\n}\n",
		},
		{
			"use_nullable_type_declaration false",
			false,
			"<?php\nfunction f((A&B)|null $d = null, ?int $e = null, int|string|null $u = null, null|int $n = null, A|(B&C)|null $g = null, null | (A&B) $h = null) {}\nclass K { function __construct(private ?int $p = null, readonly int|null $q = null) {} }\n",
			"<?php\nfunction f(A&B $d = null, int $e = null, int|string $u = null, int $n = null, A|(B&C) $g = null, A&B $h = null) {}\nclass K { function __construct(private ?int $p = null, readonly int|null $q = null) {} }\n",
		},
	}
	for _, c := range cases {
		f := NullableTypeDeclarationForDefaultNullValue{}.WithConfig(map[string]any{"use_nullable_type_declaration": c.nullable})
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
