package rules

import "testing"

// Cases ported from PHP-CS-Fixer's DateTimeImmutableFixerTest
// (tests/Fixer/ClassUsage). Every expected output is what PHP-CS-Fixer produces
// for the input.
func TestDateTimeImmutablePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	cases := []aliasUpstreamCase{
		{"new bare", nil, "<?php new DateTime();", "<?php new DateTimeImmutable();"},
		{"new uppercase", nil, "<?php new DATETIME();", "<?php new DateTimeImmutable();"},
		{"new leading backslash", nil, `<?php new \DateTime();`, `<?php new \DateTimeImmutable();`},
		{"new qualified", nil, `<?php new Foo\DateTime();`, `<?php new Foo\DateTime();`},
		{"namespace bare not imported", nil, "<?php namespace Foo; new DateTime();", "<?php namespace Foo; new DateTime();"},
		{"namespace leading backslash", nil, `<?php namespace Foo; new \DateTime();`, `<?php namespace Foo; new \DateTimeImmutable();`},
		{"namespace imported", nil, `<?php namespace Foo; use DateTime; new DateTime();`, `<?php namespace Foo; use DateTime; new \DateTimeImmutable();`},
		{"namespace imported qualified use", nil, `<?php namespace Foo; use DateTime; new Bar\DateTime();`, `<?php namespace Foo; use DateTime; new Bar\DateTime();`},
		{"namespace sub-namespace import", nil, `<?php namespace Foo; use DateTime\Bar; new DateTime();`, `<?php namespace Foo; use DateTime\Bar; new DateTime();`},
		{"namespace aliasing import", nil, `<?php namespace Foo; use Bar\DateTime; new DateTime();`, `<?php namespace Foo; use Bar\DateTime; new DateTime();`},
		{"namespace mixed imports", nil, `<?php namespace Foo; use DateTime\Bar; use DateTime; use Baz\DateTime as BazDateTime; new DateTime();`, `<?php namespace Foo; use DateTime\Bar; use DateTime; use Baz\DateTime as BazDateTime; new \DateTimeImmutable();`},
		{"constant access", nil, `<?php $foo = DateTime::ISO8601;`, `<?php $foo = DateTime::ISO8601;`},
		{"constant access leading backslash", nil, `<?php $foo = \datetime::ISO8601 + 24;`, `<?php $foo = \datetime::ISO8601 + 24;`},
		{"static createFromFormat", nil, `<?php DateTime::createFromFormat('j-M-Y', '15-Feb-2009');`, `<?php DateTimeImmutable::createFromFormat('j-M-Y', '15-Feb-2009');`},
		{"static getLastErrors leading backslash", nil, `<?php \DateTime::getLastErrors();`, `<?php \DateTimeImmutable::getLastErrors();`},
		{"static qualified", nil, `<?php Foo\DateTime::createFromFormat();`, `<?php Foo\DateTime::createFromFormat();`},
		{"method call", nil, `<?php $foo->DateTime();`, `<?php $foo->DateTime();`},
		{"static call on other class", nil, `<?php Foo::DateTime();`, `<?php Foo::DateTime();`},
		{"qualified function-like", nil, `<?php Foo\DateTime();`, `<?php Foo\DateTime();`},
		{"date_create", nil, `<?php date_create("now");`, `<?php date_create_immutable("now");`},
		{"date_create mixed case", nil, `<?php Date_Create();`, `<?php date_create_immutable();`},
		{"date_create leading backslash", nil, `<?php \date_create();`, `<?php \date_create_immutable();`},
		{"date_create namespace", nil, `<?php namespace Foo; date_create();`, `<?php namespace Foo; date_create_immutable();`},
		{"date_create namespace leading backslash", nil, `<?php namespace Foo; \date_create();`, `<?php namespace Foo; \date_create_immutable();`},
		{"date_create_from_format", nil, `<?php date_create_from_format('j-M-Y', '15-Feb-2009');`, `<?php date_create_immutable_from_format('j-M-Y', '15-Feb-2009');`},
		{"date_create qualified", nil, `<?php Foo\date_create();`, `<?php Foo\date_create();`},
		{"date_create method", nil, `<?php $foo->date_create();`, `<?php $foo->date_create();`},
		{"date_create static", nil, `<?php Foo::date_create();`, `<?php Foo::date_create();`},
		{"date_create new", nil, `<?php new date_create();`, `<?php new date_create();`},
		{"date_create new leading backslash", nil, `<?php new \date_create();`, `<?php new \date_create();`},
		{"date_create new qualified", nil, `<?php new Foo\date_create();`, `<?php new Foo\date_create();`},
		{"datetime method declaration", nil, `<?php class Foo { public function datetime() {} }`, `<?php class Foo { public function datetime() {} }`},
		{"date_create method declaration", nil, `<?php class Foo { public function date_create() {} }`, `<?php class Foo { public function date_create() {} }`},
		{"datetime method declaration imported", nil, `<?php namespace Foo; use DateTime; class Bar { public function datetime() {} }`, `<?php namespace Foo; use DateTime; class Bar { public function datetime() {} }`},
		{
			"multiline namespaced imports",
			nil,
			"<?php\n                namespace Foo;\n                use DateTime\\Bar;\n                use DateTime;\n                use Baz\\DateTime as BazDateTime;\n                new DateTime();\n                new DateTime();\n                new DateTime();\n                new DateTime();\n                ",
			"<?php\n                namespace Foo;\n                use DateTime\\Bar;\n                use DateTime;\n                use Baz\\DateTime as BazDateTime;\n                new \\DateTimeImmutable();\n                new \\DateTimeImmutable();\n                new \\DateTimeImmutable();\n                new \\DateTimeImmutable();\n                ",
		},
	}
	aliasRunUpstreamCases(t, DateTimeImmutable{}, cases)
}
