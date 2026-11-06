package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_doc_to_return_type).
func TestPhpdocToReturnTypePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpdocToReturnType{}, []aliasUpstreamCase{
		{
			"fq class return", nil,
			"<?php\n\n/** @return \\My\\Bar */\nfunction f1()\n{}\n",
			"<?php\n\n/** @return \\My\\Bar */\nfunction f1(): \\My\\Bar\n{}\n",
		},
		{
			"void return", nil,
			"<?php\n/** @return void */\nfunction f2()\n{}\n",
			"<?php\n/** @return void */\nfunction f2(): void\n{}\n",
		},
		{
			"object return", nil,
			"<?php\n/** @return object */\nfunction my_foo()\n{}\n",
			"<?php\n/** @return object */\nfunction my_foo(): object\n{}\n",
		},
		{
			"nullable return", nil,
			"<?php\n/** @return string|null */\nfunction f() {}\n",
			"<?php\n/** @return string|null */\nfunction f(): ?string {}\n",
		},
		{
			"union return", nil,
			"<?php\n/** @return int|string */\nfunction f() {}\n",
			"<?php\n/** @return int|string */\nfunction f(): int|string {}\n",
		},
		{
			"scalar_types false", map[string]any{"scalar_types": false},
			"<?php\n/** @return Foo */\nfunction foo() {}\n/** @return string */\nfunction bar() {}\n",
			"<?php\n/** @return Foo */\nfunction foo(): Foo {}\n/** @return string */\nfunction bar() {}\n",
		},
		{
			"already typed noop", nil,
			"<?php\n/** @return int */\nfunction f(): string {}\n",
			"<?php\n/** @return int */\nfunction f(): string {}\n",
		},
		{
			"two return annotations skipped", nil,
			"<?php\n/**\n * @return int\n * @return string\n */\nfunction f() {}\n",
			"<?php\n/**\n * @return int\n * @return string\n */\nfunction f() {}\n",
		},
		{
			"null return skipped", nil,
			"<?php\n/** @return null */\nfunction f() {}\n",
			"<?php\n/** @return null */\nfunction f() {}\n",
		},
		{
			"static return", nil,
			"<?php\nclass Foo {\n    /**\n     * @return static\n     */\n    public function create($prototype) {\n        return new static($prototype);\n    }\n}\n",
			"<?php\nclass Foo {\n    /**\n     * @return static\n     */\n    public function create($prototype): static {\n        return new static($prototype);\n    }\n}\n",
		},
		{
			"construct excluded", nil,
			"<?php\nclass Foo {\n    /** @return int */\n    public function __construct() {}\n}\n",
			"<?php\nclass Foo {\n    /** @return int */\n    public function __construct() {}\n}\n",
		},
		{
			"arrow function return", nil,
			"<?php\n/** @return int */\n$f = fn($x) => $x;\n",
			"<?php\n/** @return int */\n$f = fn($x) => $x;\n",
		},
		{
			"self return", nil,
			"<?php\nclass A {\n    /** @return self */\n    public function f() {}\n}\n",
			"<?php\nclass A {\n    /** @return self */\n    public function f(): self {}\n}\n",
		},
		{
			"no docblock noop", nil,
			"<?php\nfunction f() {}\n",
			"<?php\nfunction f() {}\n",
		},
		{
			"arrow function reachable docblock", nil,
			"<?php\n/** @return int */\nfn($x) => $x;\n",
			"<?php\n/** @return int */\nfn($x): int => $x;\n",
		},
		{
			"phpstan-type exclude return", nil,
			"<?php\n/** @phpstan-type Foo int */\n/** @return Foo */\nfunction f() {}\n",
			"<?php\n/** @phpstan-type Foo int */\n/** @return Foo */\nfunction f() {}\n",
		},
		{
			"namespaced self static union noop", nil,
			"<?php\n/** @return self|static */\nfunction f() {}\n",
			"<?php\n/** @return self|static */\nfunction f(): self {}\n",
		},
		{
			"array traversable iterable", nil,
			"<?php\n/** @return array|\\Traversable */\nfunction f() {}\n",
			"<?php\n/** @return array|\\Traversable */\nfunction f(): iterable {}\n",
		},
	})
}
