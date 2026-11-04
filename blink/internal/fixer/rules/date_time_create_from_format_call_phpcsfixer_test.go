package rules

import (
	"strings"
	"testing"
)

// Cases ported from PHP-CS-Fixer's DateTimeCreateFromFormatCallFixerTest
// (tests/Fixer/FunctionNotation); the upstream provider runs every case for both
// DateTime and DateTimeImmutable. Every expected output is what PHP-CS-Fixer
// produces for the input.
func TestDateTimeCreateFromFormatCallPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	var cases []aliasUpstreamCase
	for _, class := range []string{"DateTime", "DateTimeImmutable"} {
		lower := strings.ToLower(class)
		upper := strings.ToUpper(class)
		sub := func(s string) string { return strings.ReplaceAll(s, "{C}", class) }

		add := func(name, input, expected string) {
			cases = append(cases, aliasUpstreamCase{class + " " + name, nil, input, expected})
		}
		same := func(name, code string) { add(name, code, code) }

		add("leading separator", sub(`<?php \{C}::createFromFormat('Y-m-d', '2022-02-11');`), sub(`<?php \{C}::createFromFormat('!Y-m-d', '2022-02-11');`))
		add("import", sub(`<?php use {C}; {C}::createFromFormat('Y-m-d', '2022-02-11');`), sub(`<?php use {C}; {C}::createFromFormat('!Y-m-d', '2022-02-11');`))
		add("global", sub(`<?php {C}::createFromFormat('Y-m-d', '2022-02-11');`), sub(`<?php {C}::createFromFormat('!Y-m-d', '2022-02-11');`))
		same("import other class", sub(`<?php use \Example\{C}; {C}::createFromFormat('Y-m-d', '2022-02-11');`))
		same("import other class case", `<?php use \Example\`+lower+`; `+upper+`::createFromFormat('Y-m-d', '2022-02-11');`)
		add("double quote", sub(`<?php \{C}::createFromFormat("Y-m-d", '2022-02-11');`), sub(`<?php \{C}::createFromFormat("!Y-m-d", '2022-02-11');`))
		same("variable format", sub(`<?php \{C}::createFromFormat($foo, '2022-02-11');`))
		add("upper with space", `<?php \`+upper+`::createFromFormat( "Y-m-d", '2022-02-11');`, `<?php \`+upper+`::createFromFormat( "!Y-m-d", '2022-02-11');`)
		add("leading comment", sub(`<?php \{C}::createFromFormat(/* aaa */ 'Y-m-d', '2022-02-11');`), sub(`<?php \{C}::createFromFormat(/* aaa */ '!Y-m-d', '2022-02-11');`))
		add("comments everywhere", sub(`<?php /*1*//*2*/{C}/*3*/::/*4*/createFromFormat/*5*/(/*6*/"Y-m-d"/*7*/,/*8*/"2022-02-11"/*9*/)/*10*/ ?>`), sub(`<?php /*1*//*2*/{C}/*3*/::/*4*/createFromFormat/*5*/(/*6*/"!Y-m-d"/*7*/,/*8*/"2022-02-11"/*9*/)/*10*/ ?>`))
		same("single argument", sub(`<?php \{C}::createFromFormat('Y-m-d');`))
		same("two variables", sub(`<?php \{C}::createFromFormat($a, $b);`))
		same("three arguments", sub(`<?php \{C}::createFromFormat('Y-m-d', $b, $c);`))
		same("qualified name", sub(`<?php A\{C}::createFromFormat('Y-m-d', '2022-02-11');`))
		same("concat format", sub(`<?php A\{C}::createFromFormat('Y-m-d'."a", '2022-02-11');`))
		same("integer format", sub(`<?php \{C}::createFromFormat(123, '2022-02-11');`))
		add("namespaced shadowing",
			sub("<?php namespace {\n    \\{C}::createFromFormat('Y-m-d', '2022-02-11');\n}\n\nnamespace Bar {\n    class {C} extends Foo {}\n    {C}::createFromFormat('Y-m-d', '2022-02-11');\n}\n"),
			sub("<?php namespace {\n    \\{C}::createFromFormat('!Y-m-d', '2022-02-11');\n}\n\nnamespace Bar {\n    class {C} extends Foo {}\n    {C}::createFromFormat('Y-m-d', '2022-02-11');\n}\n"))
		add("binary string", sub(`<?php \{C}::createFromFormat(b'Y-m-d', '2022-02-11');`), sub(`<?php \{C}::createFromFormat(b'!Y-m-d', '2022-02-11');`))
		same("empty string", sub(`<?php \{C}::createFromFormat('', '2022-02-11');`))
	}
	aliasRunUpstreamCases(t, DateTimeCreateFromFormatCall{}, cases)
}
