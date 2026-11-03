package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ModernSerializationMethodsFixerTest.
func TestModernSerializationMethodsUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, ModernSerializationMethods{}, []upstreamFixCase{
		{name: "rename when only old methods exists",
			expected: "<?php class Foo {\n    public function __serialize() {}\n    public function __unserialize(array $data) {}\n}",
			input:    "<?php class Foo {\n    public function __sleep() {}\n    public function __wakeup() {}\n}"},
		{name: "do not rename when new methods already exists",
			expected: "<?php class Foo {\n    public function __serialize() {}\n    public function __unserialize(array $data) {}\n    public function __sleep() {}\n    public function __wakeup() {}\n}"},
		{name: "rename only when only new methods do not exist",
			expected: "<?php\nclass Foo {\n    public function __serialize() {}\n    public function __sleep() {}\n    public function __unserialize(array $data) {}\n}\nclass Bar {\n    public function __serialize() {}\n    public function __unserialize(array $data) {}\n    public function __wakeup() {}\n}",
			input:    "<?php\nclass Foo {\n    public function __serialize() {}\n    public function __sleep() {}\n    public function __wakeup() {}\n}\nclass Bar {\n    public function __sleep() {}\n    public function __unserialize(array $data) {}\n    public function __wakeup() {}\n}"},
		{name: "do not touch constants",
			expected: "<?php class Foo {\n    public const __sleep = 'nice try';\n    public function __unserialize(array $data) {}\n}",
			input:    "<?php class Foo {\n    public const __sleep = 'nice try';\n    public function __wakeup() {}\n}"},
	})
}

func TestModernSerializationMethods(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, ModernSerializationMethods{}, []upstreamFixCase{
		// an anonymous class is its own class-like; the outer one keeps its methods
		{name: "nested anonymous class",
			expected: "<?php class A {\n    public function __unserialize(array $data) { return new class { function __serialize() {} function __unserialize(array $data) {} }; }\n    public function __serialize() {}\n}",
			input:    "<?php class A {\n    public function __wakeup() { return new class { function __sleep() {} function __wakeup() {} }; }\n    public function __sleep() {}\n}"},
		{name: "interface and trait",
			expected: "<?php interface I { function __unserialize(array $data); }\ntrait T { function __serialize() {} }",
			input:    "<?php interface I { function __wakeup(); }\ntrait T { function __sleep() {} }"},
		{name: "names are case-sensitive", expected: "<?php class A { function __Sleep() {} function __WAKEUP() {} }"},
		{name: "functions and closures are no methods",
			expected: "<?php function __sleep() {}\nclass A { function f() { $f = function () {}; } }"},
	})
}
