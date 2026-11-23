package rules

import "testing"

// Since v3.95.28 (PHP-CS-Fixer#9912), PHP-CS-Fixer fixes the case labels of a
// switch whatever its "default" label looks like. An enum declared in a case
// body keeps its own cases, and the return type ":" or "?" of a closure in a
// label is not the label's. Every want is PHP-CS-Fixer's output for src.
func TestSwitchCaseSpaceWithDefault(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, src, want string }{
		{
			"tidy default after a case",
			"<?php\n\nswitch ($x) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\n\nswitch ($x) {\n    case 1:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"several cases before a tidy default",
			"<?php\nswitch ($a) {\n    case \"prod\" :\n    case \"stg\"   :\n    case \"dev\":\n    case \"test\"    :\n    default:\n}\n",
			"<?php\nswitch ($a) {\n    case \"prod\":\n    case \"stg\":\n    case \"dev\":\n    case \"test\":\n    default:\n}\n",
		},
		{
			"tidy default before the cases",
			"<?php\nswitch ($a) {\n    default:\n        break;\n    case 1 :\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    default:\n        break;\n    case 1:\n        break;\n}\n",
		},
		{
			"semicolon terminators",
			"<?php\nswitch ($a) {\n    case 1 ;\n        break;\n    default;\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1;\n        break;\n    default;\n        break;\n}\n",
		},
		{
			"comment between default and its colon",
			"<?php\nswitch ($a) {\n    case 1 :\n        break;\n    default /* c */ :\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        break;\n    default /* c */ :\n        break;\n}\n",
		},
		{
			"nested switches",
			"<?php\nswitch ($a) {\n    case 1 :\n        switch ($b) {\n            case 2 :\n                break;\n            default :\n                break;\n        }\n        break;\n    default:\n        switch ($c) {\n            case 3 :\n                break;\n            default:\n                break;\n        }\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        switch ($b) {\n            case 2:\n                break;\n            default:\n                break;\n        }\n        break;\n    default:\n        switch ($c) {\n            case 3:\n                break;\n            default:\n                break;\n        }\n}\n",
		},
		{
			"alternative syntax",
			"<?php\nswitch ($a):\n    case 1 :\n        break;\n    default:\n        break;\nendswitch;\n",
			"<?php\nswitch ($a):\n    case 1:\n        break;\n    default:\n        break;\nendswitch;\n",
		},
		{
			"alternative syntax inside a braced switch",
			"<?php\nswitch ($a) {\n    case 1 :\n        switch ($b):\n            case 2 :\n                break;\n            default:\n                break;\n        endswitch;\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        switch ($b):\n            case 2:\n                break;\n            default:\n                break;\n        endswitch;\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"match default in a body and a label",
			"<?php\nswitch ($a) {\n    case 1 :\n        $b = match ($c) {\n            1 => 1,\n            default  => 2,\n        };\n        break;\n    case match ($d) { default => 2 } :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        $b = match ($c) {\n            1 => 1,\n            default  => 2,\n        };\n        break;\n    case match ($d) { default => 2 }:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"uppercase keywords",
			"<?php\nSWITCH ($a) {\n    CASE 1 :\n        break;\n    DEFAULT:\n        break;\n}\n",
			"<?php\nSWITCH ($a) {\n    CASE 1:\n        break;\n    DEFAULT:\n        break;\n}\n",
		},
		{
			"ternary coalesce and nullsafe labels",
			"<?php\nswitch ($a) {\n    case $b ? 1 : 2 :\n        break;\n    case $c ?: 3 :\n        break;\n    case $d ?? 4 :\n        break;\n    case $e?->f :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case $b ? 1 : 2:\n        break;\n    case $c ?: 3:\n        break;\n    case $d ?? 4:\n        break;\n    case $e?->f:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"comment or newline before a colon",
			"<?php\nswitch ($a) {\n    case 1 /* c */ :\n        break;\n    case 2\n    :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1 /* c */ :\n        break;\n    case 2:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"default used as a name",
			"<?php\nswitch ($a) {\n    case self::DEFAULT :\n        $b = $o->default;\n        foo(default: 1);\n        break;\n    default :\n        break;\n}\nswitch ($a) {\n    case 1 :\n        $b = $o->default;\n        $c = self::DEFAULT;\n        foo(default: 1);\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case self::DEFAULT:\n        $b = $o->default;\n        foo(default: 1);\n        break;\n    default:\n        break;\n}\nswitch ($a) {\n    case 1:\n        $b = $o->default;\n        $c = self::DEFAULT;\n        foo(default: 1);\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"later switch without default",
			"<?php\nswitch ($a) {\n    case 1 :\n        break;\n    default:\n        break;\n}\nswitch ($b) {\n    case 2 :\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        break;\n    default:\n        break;\n}\nswitch ($b) {\n    case 2:\n        break;\n}\n",
		},
		{
			"enum declared in a case body",
			"<?php\nswitch ($a) {\n    case 1 :\n        enum E\n        {\n            case A ;\n        }\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        enum E\n        {\n            case A ;\n        }\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"enum declared in a switch without default",
			"<?php\nswitch ($a) {\n    case 1 :\n        enum E\n        {\n            case A ;\n        }\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        enum E\n        {\n            case A ;\n        }\n        break;\n}\n",
		},
		{
			"enum declared in a tidy switch",
			"<?php\nswitch ($a) {\n    case 1:\n        enum E\n        {\n            case A ;\n        }\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        enum E\n        {\n            case A ;\n        }\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"return types in labels",
			"<?php\nswitch ($a) {\n    case fn() : int => 1 :\n        break;\n    case fn(): ?int => 2:\n        break ;\n    case function () use ($b): int { return 3; } :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case fn() : int => 1:\n        break;\n    case fn(): ?int => 2:\n        break ;\n    case function () use ($b): int { return 3; }:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"nullable return type without default",
			"<?php\nswitch ($a) {\n    case fn(): ?int => 1 :\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case fn(): ?int => 1:\n        break;\n}\n",
		},
		{
			"tidy labels",
			"<?php\nswitch ($a) {\n    case 1:\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        break;\n    default:\n        break;\n}\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertFix(t, SwitchCaseSpace{}, c.src, c.want, c.src != c.want)
		})
	}
}

// ControlCaseStructuresAnalyzer counts the "{" and "}" PHP-CS-Fixer does not
// retag from the switch keyword on, so a braced switch whose subject holds one
// ends at its "}" and the body's labels belong to the enclosing switch, if any.
// Two defaults in one switch make the analyzer throw, which leaves the file as
// it is. Every want is PHP-CS-Fixer's output for src.
func TestSwitchCaseSpaceAnalyzerLevels(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, src, want string }{
		{
			"closure in the subject",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"closure in the subject without default",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n}\n",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n}\n",
		},
		{
			"closure in the subject with an untidy default",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n    default :\n        break;\n}\n",
			"<?php\nswitch ((function () { return 1; })()) {\n    case 1 :\n        break;\n    default :\n        break;\n}\n",
		},
		{
			"anonymous class in the subject",
			"<?php\nswitch (get_class(new class {})) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch (get_class(new class {})) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"braces the analyzer does not count in the subject",
			"<?php\nswitch (match ($x) { 1 => 'a', default => 'b' }) {\n    case 1 :\n        break;\n    default:\n        break;\n}\nswitch (\"{$x}\" . \"${x}\") {\n    case 2 :\n        break;\n    default:\n        break;\n}\nswitch ($o->{'a'} . $o?->{'b'} . ${'c'} . $ {'d'}) {\n    case 3 :\n        break;\n    default:\n        break;\n}\nswitch (Foo::{'A'}::B . (fn () => 1)()) {\n    case 4 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch (match ($x) { 1 => 'a', default => 'b' }) {\n    case 1:\n        break;\n    default:\n        break;\n}\nswitch (\"{$x}\" . \"${x}\") {\n    case 2:\n        break;\n    default:\n        break;\n}\nswitch ($o->{'a'} . $o?->{'b'} . ${'c'} . $ {'d'}) {\n    case 3:\n        break;\n    default:\n        break;\n}\nswitch (Foo::{'A'}::B . (fn () => 1)()) {\n    case 4:\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"class constant fetch in the subject",
			"<?php\nswitch (Foo::{'A'}) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch (Foo::{'A'}) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"spaced dynamic property in the subject",
			"<?php\nswitch ($o-> {'a'}) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($o-> {'a'}) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"closure in the subject of an alternative syntax switch",
			"<?php\nswitch ((function () { return 1; })()):\n    case 1 :\n        break;\n    default:\n        break;\nendswitch;\n",
			"<?php\nswitch ((function () { return 1; })()):\n    case 1:\n        break;\n    default:\n        break;\nendswitch;\n",
		},
		{
			"closure subject hands its labels to the outer switch",
			"<?php\nswitch ($a) {\n    case 1 :\n        switch ((function () { return 1; })()) {\n            case 2 :\n                break;\n        }\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        switch ((function () { return 1; })()) {\n            case 2:\n                break;\n        }\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"closure subject hands its default to the outer switch",
			"<?php\nswitch ($a) {\n    case 1 :\n        switch ((function () { return 1; })()) {\n            case 2 :\n                break;\n            default :\n                break;\n        }\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        switch ((function () { return 1; })()) {\n            case 2:\n                break;\n            default:\n                break;\n        }\n        break;\n}\n",
		},
		{
			"two defaults in one analysis leave the file alone",
			"<?php\nswitch ($b) {\n    case 3 :\n        break;\n}\nswitch ($a) {\n    case 1 :\n        switch ((function () { return 1; })()) {\n            case 2 :\n                break;\n            default:\n                break;\n        }\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($b) {\n    case 3 :\n        break;\n}\nswitch ($a) {\n    case 1 :\n        switch ((function () { return 1; })()) {\n            case 2 :\n                break;\n            default:\n                break;\n        }\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"match with a closure subject beside a default",
			"<?php\nswitch ($a) {\n    case 1 :\n        $b = match ((function () { return 1; })()) { 1 => 2, default => 3 };\n        break;\n    default:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1 :\n        $b = match ((function () { return 1; })()) { 1 => 2, default => 3 };\n        break;\n    default:\n        break;\n}\n",
		},
		{
			"match with a closure subject in a switch without default",
			"<?php\nswitch ($a) {\n    case 1 :\n        $b = match ((function () { return 1; })()) { 1 => 2, default => 3 };\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case 1:\n        $b = match ((function () { return 1; })()) { 1 => 2, default => 3 };\n        break;\n}\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertFix(t, SwitchCaseSpace{}, c.src, c.want, c.src != c.want)
		})
	}
}
