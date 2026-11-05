package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestOrderedTypesCaseSensitive(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		OrderedTypes{},
		OrderedTypes{}.WithConfig(map[string]any{"case_sensitive": false}).(fixerRule),
		OrderedTypes{}.WithConfig(map[string]any{"case_sensitive": true}).(fixerRule),
		OrderedTypes{}.WithConfig(map[string]any{"case_sensitive": true, "null_adjustment": "always_last"}).(fixerRule),
		OrderedTypes{}.WithConfig(map[string]any{"case_sensitive": true, "null_adjustment": "none"}).(fixerRule),
		OrderedTypes{}.WithConfig(map[string]any{"case_sensitive": true, "sort_algorithm": "none"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [6]string
	}{
		{"<?php\nfunction f(b|A|C|a $x): void {}\n", [6]string{"<?php\nfunction f(A|a|b|C $x): void {}\n", "<?php\nfunction f(A|a|b|C $x): void {}\n", "<?php\nfunction f(A|C|a|b $x): void {}\n", "<?php\nfunction f(A|C|a|b $x): void {}\n", "<?php\nfunction f(A|C|a|b $x): void {}\n", "<?php\nfunction f(b|A|C|a $x): void {}\n"}},
		{"<?php\nfunction f(int|Foo|null|bool $x): string|Bar {}\n", [6]string{"<?php\nfunction f(null|bool|Foo|int $x): Bar|string {}\n", "<?php\nfunction f(null|bool|Foo|int $x): Bar|string {}\n", "<?php\nfunction f(null|Foo|bool|int $x): Bar|string {}\n", "<?php\nfunction f(Foo|bool|int|null $x): Bar|string {}\n", "<?php\nfunction f(Foo|bool|int|null $x): Bar|string {}\n", "<?php\nfunction f(null|int|Foo|bool $x): string|Bar {}\n"}},
		{"<?php\nfunction f(\\Foo|Bar|\\Abc|abc $x) {}\n", [6]string{"<?php\nfunction f(\\Abc|abc|Bar|\\Foo $x) {}\n", "<?php\nfunction f(\\Abc|abc|Bar|\\Foo $x) {}\n", "<?php\nfunction f(\\Abc|Bar|\\Foo|abc $x) {}\n", "<?php\nfunction f(\\Abc|Bar|\\Foo|abc $x) {}\n", "<?php\nfunction f(\\Abc|Bar|\\Foo|abc $x) {}\n", "<?php\nfunction f(\\Foo|Bar|\\Abc|abc $x) {}\n"}},
		{"<?php\nfunction f(Foo_Bar|FooBar|Foo\\Bar|_x $x) {}\n", [6]string{"<?php\nfunction f(_x|Foo\\Bar|Foo_Bar|FooBar $x) {}\n", "<?php\nfunction f(_x|Foo\\Bar|Foo_Bar|FooBar $x) {}\n", "<?php\nfunction f(FooBar|Foo\\Bar|Foo_Bar|_x $x) {}\n", "<?php\nfunction f(FooBar|Foo\\Bar|Foo_Bar|_x $x) {}\n", "<?php\nfunction f(FooBar|Foo\\Bar|Foo_Bar|_x $x) {}\n", "<?php\nfunction f(Foo_Bar|FooBar|Foo\\Bar|_x $x) {}\n"}},
		{"<?php\nclass C {\n    public B&a&C $x;\n    private Null|Zed|alpha $y;\n}\n", [6]string{"<?php\nclass C {\n    public a&B&C $x;\n    private Null|alpha|Zed $y;\n}\n", "<?php\nclass C {\n    public a&B&C $x;\n    private Null|alpha|Zed $y;\n}\n", "<?php\nclass C {\n    public B&C&a $x;\n    private Null|Zed|alpha $y;\n}\n", "<?php\nclass C {\n    public B&C&a $x;\n    private Zed|alpha|Null $y;\n}\n", "<?php\nclass C {\n    public B&C&a $x;\n    private Null|Zed|alpha $y;\n}\n", "<?php\nclass C {\n    public B&a&C $x;\n    private Null|Zed|alpha $y;\n}\n"}},
		{"<?php\nfunction f(Ébc|abc|Zed|éa $x) {}\n", [6]string{"<?php\nfunction f(abc|Zed|Ébc|éa $x) {}\n", "<?php\nfunction f(abc|Zed|Ébc|éa $x) {}\n", "<?php\nfunction f(Zed|abc|Ébc|éa $x) {}\n", "<?php\nfunction f(Zed|abc|Ébc|éa $x) {}\n", "<?php\nfunction f(Zed|abc|Ébc|éa $x) {}\n", "<?php\nfunction f(Ébc|abc|Zed|éa $x) {}\n"}},
	}
	for _, c := range cases {
		for k, f := range fixers {
			got, changed := apply(t, f, c.src)
			if got != c.want[k] || changed != (c.want[k] != c.src) {
				t.Errorf("config %d src=%q\nchanged=%v got=%q\nwant=%q", k, c.src, changed, got, c.want[k])
			}
		}
	}
}
