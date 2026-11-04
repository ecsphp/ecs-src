package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestStandaloneLineRequiredParamAttribute(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"class A { #[Required] public function f(int $a) {} }", "class A { #[Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Required] public function f(int $a, int $b): void {} }", "class A { #[Required] public function f(\n    int $a,\n    int $b\n): void {} }"},
		{"class A { #[\\Required] public function f(int $a) {} }", "class A { #[\\Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo\\Required] public function f(int $a) {} }", "class A { #[Foo\\Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[\\Foo\\Required] public function f(int $a) {} }", "class A { #[\\Foo\\Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Required\\Foo] public function f(int $a) {} }", "class A { #[Required\\Foo] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo\\Required\\Bar] public function f(int $a) {} }", "class A { #[Foo\\Required\\Bar] public function f(\n    int $a\n) {} }"},
		{"class A { #[namespace\\Required] public function f(int $a) {} }", "class A { #[namespace\\Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Required()] public function f(int $a) {} }", "class A { #[Required()] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo, Required] public function f(int $a) {} }", "class A { #[Foo, Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo] #[Required] public function f(int $a) {} }", "class A { #[Foo] #[Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Required] #[Foo] public function f(int $a) {} }", "class A { #[Required] #[Foo] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(Required: true)] public function f(int $a) {} }", "class A { #[Foo(Required: true)] public function f(int $a) {} }"},
		{"class A { #[Foo(Required::X)] public function f(int $a) {} }", "class A { #[Foo(Required::X)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(x: Required::X)] public function f(int $a) {} }", "class A { #[Foo(x: Required::X)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(Required)] public function f(int $a) {} }", "class A { #[Foo(Required)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(new Required())] public function f(int $a) {} }", "class A { #[Foo(new Required())] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(Bar::Required)] public function f(int $a) {} }", "class A { #[Foo(Bar::Required)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(required: true)] public function f(int $a) {} }", "class A { #[Foo(required: true)] public function f(int $a) {} }"},
		{"class A { #[Foo(x: 1, required: true)] public function f(int $a) {} }", "class A { #[Foo(x: 1, required: true)] public function f(int $a) {} }"},
		{"class A { #[required] public function f(int $a) {} }", "class A { #[required] public function f(int $a) {} }"},
		{"class A { #[REQUIRED] public function f(int $a) {} }", "class A { #[REQUIRED] public function f(int $a) {} }"},
		{"class A { #[Foo\\required] public function f(int $a) {} }", "class A { #[Foo\\required] public function f(int $a) {} }"},
		{"class A { #[Foo('Required')] public function f(int $a) {} }", "class A { #[Foo('Required')] public function f(int $a) {} }"},
		{"class A { #[Foo(\"Required\")] public function f(int $a) {} }", "class A { #[Foo(\"Required\")] public function f(int $a) {} }"},
		{"class A { #[Foo('is Required here')] public function f(int $a) {} }", "class A { #[Foo('is Required here')] public function f(int $a) {} }"},
		{"class A { #[Foo(description: 'Required value')] public function f(int $a) {} }", "class A { #[Foo(description: 'Required value')] public function f(int $a) {} }"},
		{"class A { #[Foo /* Required */] public function f(int $a) {} }", "class A { #[Foo /* Required */] public function f(int $a) {} }"},
		{"class A { #[NotRequired] public function f(int $a) {} }", "class A { #[NotRequired] public function f(int $a) {} }"},
		{"class A { #[RequiredFoo] public function f(int $a) {} }", "class A { #[RequiredFoo] public function f(int $a) {} }"},
		{"class A { #[Foo_Required] public function f(int $a) {} }", "class A { #[Foo_Required] public function f(int $a) {} }"},
		{"class A { #[Required] protected function f(int $a) {} }", "class A { #[Required] protected function f(int $a) {} }"},
		{"class A { #[Required] private function f(int $a) {} }", "class A { #[Required] private function f(int $a) {} }"},
		{"class A { #[Required] function f(int $a) {} }", "class A { #[Required] function f(int $a) {} }"},
		{"class A { #[Required] public function f() {} }", "class A { #[Required] public function f() {} }"},
		{"class A { #[Required] public static function f(int $a) {} }", "class A { #[Required] public static function f(\n    int $a\n) {} }"},
		{"class A { #[Required] final public function f(int $a) {} }", "class A { #[Required] final public function f(\n    int $a\n) {} }"},
		{"abstract class A { #[Required] abstract public function f(int $a); }", "abstract class A { #[Required] abstract public function f(\n    int $a\n); }"},
		{"class A { /** @required */ public function f(int $a) {} }", "class A { /** @required */ public function f(\n    int $a\n) {} }"},
		{"class A { /** @Required */ public function f(int $a) {} }", "class A { /** @Required */ public function f(\n    int $a\n) {} }"},
		{"class A { /** @required-foo */ public function f(int $a) {} }", "class A { /** @required-foo */ public function f(\n    int $a\n) {} }"},
		{"class A { /** @requiredfoo */ public function f(int $a) {} }", "class A { /** @requiredfoo */ public function f(int $a) {} }"},
		{"class A { /** Required */ public function f(int $a) {} }", "class A { /** Required */ public function f(int $a) {} }"},
		{"class A { /* @required */ public function f(int $a) {} }", "class A { /* @required */ public function f(int $a) {} }"},
		{"class A { /** @required */ protected function f(int $a) {} }", "class A { /** @required */ protected function f(int $a) {} }"},
		{"class A { #[Foo(required: true)] /** @required */ public function f(int $a) {} }", "class A { #[Foo(required: true)] /** @required */ public function f(\n    int $a\n) {} }"},
		{"class A { /** @required */ #[Foo(required: true)] public function f(int $a) {} }", "class A { /** @required */ #[Foo(required: true)] public function f(\n    int $a\n) {} }"},
		{"#[Required] function f(int $a) {}", "#[Required] function f(int $a) {}"},
		{"class A { public function g() {} #[Required] public function f(int $a) {} }", "class A { public function g() {} #[Required] public function f(\n    int $a\n) {} }"},
		{"class A { #[Required] public function g() {} public function f(int $a) {} }", "class A { #[Required] public function g() {} public function f(int $a) {} }"},
		{"class A { #[Foo(/** @required */ x: 1)] public function f(int $a) {} }", "class A { #[Foo(/** @required */ x: 1)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(x: 1 /** @required */)] public function f(int $a) {} }", "class A { #[Foo(x: 1 /** @required */)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(/** Required */ x: 1)] public function f(int $a) {} }", "class A { #[Foo(/** Required */ x: 1)] public function f(int $a) {} }"},
		{"class A {\n    #[\n        Foo,\n        Required,\n    ]\n    public function f(int $a) {}\n}", "class A {\n    #[\n        Foo,\n        Required,\n    ]\n    public function f(\n        int $a\n    ) {}\n}"},
		{"class A { #[Foo(true ? Required : false)] public function f(int $a) {} }", "class A { #[Foo(true ? Required : false)] public function f(\n    int $a\n) {} }"},
		{"class A { #[Foo(x: true ? Required : false)] public function f(int $a) {} }", "class A { #[Foo(x: true ? Required : false)] public function f(\n    int $a\n) {} }"},
	}
	for _, c := range cases {
		got, changed := apply(t, StandaloneLineRequiredParam{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
