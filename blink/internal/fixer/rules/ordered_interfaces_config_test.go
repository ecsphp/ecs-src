package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestOrderedInterfacesConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		OrderedInterfaces{},
		OrderedInterfaces{}.WithConfig(map[string]any{"order": "length"}).(fixerRule),
		OrderedInterfaces{}.WithConfig(map[string]any{"direction": "descend"}).(fixerRule),
		OrderedInterfaces{}.WithConfig(map[string]any{"order": "length", "direction": "descend"}).(fixerRule),
		OrderedInterfaces{}.WithConfig(map[string]any{"case_sensitive": true}).(fixerRule),
		OrderedInterfaces{}.WithConfig(map[string]any{"case_sensitive": true, "direction": "descend"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [6]string
	}{
		{"<?php\nclass A implements Gamma, Alpha, Beta {}\n", [6]string{"<?php\nclass A implements Alpha, Beta, Gamma {}\n", "<?php\nclass A implements Beta, Gamma, Alpha {}\n", "<?php\nclass A implements Gamma, Beta, Alpha {}\n", "<?php\nclass A implements Gamma, Alpha, Beta {}\n", "<?php\nclass A implements Alpha, Beta, Gamma {}\n", "<?php\nclass A implements Gamma, Beta, Alpha {}\n"}},
		{"<?php\nclass A implements b, A, C, _d, a {}\n", [6]string{"<?php\nclass A implements _d, A, a, b, C {}\n", "<?php\nclass A implements b, A, C, a, _d {}\n", "<?php\nclass A implements C, b, A, a, _d {}\n", "<?php\nclass A implements _d, b, A, C, a {}\n", "<?php\nclass A implements A, C, _d, a, b {}\n", "<?php\nclass A implements b, a, _d, C, A {}\n"}},
		{"<?php\nclass A implements Zz, Yyy, Xxxx, W, Vv {}\n", [6]string{"<?php\nclass A implements Vv, W, Xxxx, Yyy, Zz {}\n", "<?php\nclass A implements W, Zz, Vv, Yyy, Xxxx {}\n", "<?php\nclass A implements Zz, Yyy, Xxxx, W, Vv {}\n", "<?php\nclass A implements Xxxx, Yyy, Zz, Vv, W {}\n", "<?php\nclass A implements Vv, W, Xxxx, Yyy, Zz {}\n", "<?php\nclass A implements Zz, Yyy, Xxxx, W, Vv {}\n"}},
		{"<?php\nclass A implements Aa, Bb, Cc, D {}\n", [6]string{"<?php\nclass A implements Aa, Bb, Cc, D {}\n", "<?php\nclass A implements D, Aa, Bb, Cc {}\n", "<?php\nclass A implements D, Cc, Bb, Aa {}\n", "<?php\nclass A implements Aa, Bb, Cc, D {}\n", "<?php\nclass A implements Aa, Bb, Cc, D {}\n", "<?php\nclass A implements D, Cc, Bb, Aa {}\n"}},
		{"<?php\nclass A implements \\Foo\\Bar, Foo_Baz, FooBar, \\Abc {}\n", [6]string{"<?php\nclass A implements \\Abc, \\Foo\\Bar, Foo_Baz, FooBar {}\n", "<?php\nclass A implements \\Abc, FooBar, Foo_Baz, \\Foo\\Bar {}\n", "<?php\nclass A implements FooBar, Foo_Baz, \\Foo\\Bar, \\Abc {}\n", "<?php\nclass A implements \\Foo\\Bar, Foo_Baz, FooBar, \\Abc {}\n", "<?php\nclass A implements \\Abc, \\Foo\\Bar, FooBar, Foo_Baz {}\n", "<?php\nclass A implements Foo_Baz, FooBar, \\Foo\\Bar, \\Abc {}\n"}},
		{"<?php\ninterface I extends Second, First, third {}\n", [6]string{"<?php\ninterface I extends First, Second, third {}\n", "<?php\ninterface I extends First, third, Second {}\n", "<?php\ninterface I extends third, Second, First {}\n", "<?php\ninterface I extends Second, First, third {}\n", "<?php\ninterface I extends First, Second, third {}\n", "<?php\ninterface I extends third, Second, First {}\n"}},
		{"<?php\n$x = new class implements B, A {};\nenum E: string implements Z, Y {}\n", [6]string{"<?php\n$x = new class implements A, B {};\nenum E: string implements Y, Z {}\n", "<?php\n$x = new class implements B, A {};\nenum E: string implements Z, Y {}\n", "<?php\n$x = new class implements B, A {};\nenum E: string implements Z, Y {}\n", "<?php\n$x = new class implements B, A {};\nenum E: string implements Z, Y {}\n", "<?php\n$x = new class implements A, B {};\nenum E: string implements Y, Z {}\n", "<?php\n$x = new class implements B, A {};\nenum E: string implements Z, Y {}\n"}},
		{"<?php\nclass A implements\n    Gamma, // g\n    Alpha /* a */,\n    beta\n{\n}\n", [6]string{"<?php\nclass A implements // g\n    Alpha /* a */,\n    beta,\n    Gamma\n{\n}\n", "<?php\nclass A implements\n    beta,\n    Gamma, // g\n    Alpha /* a */\n{\n}\n", "<?php\nclass A implements\n    Gamma,\n    beta, // g\n    Alpha /* a */\n{\n}\n", "<?php\nclass A implements\n    Gamma, // g\n    Alpha /* a */,\n    beta\n{\n}\n", "<?php\nclass A implements // g\n    Alpha /* a */,\n    Gamma,\n    beta\n{\n}\n", "<?php\nclass A implements\n    beta,\n    Gamma, // g\n    Alpha /* a */\n{\n}\n"}},
		{"<?php\nclass A implements Ébc, abc, Zed, éa {}\n", [6]string{"<?php\nclass A implements abc, Zed, Ébc, éa {}\n", "<?php\nclass A implements abc, Zed, éa, Ébc {}\n", "<?php\nclass A implements éa, Ébc, Zed, abc {}\n", "<?php\nclass A implements Ébc, abc, Zed, éa {}\n", "<?php\nclass A implements Zed, abc, Ébc, éa {}\n", "<?php\nclass A implements éa, Ébc, abc, Zed {}\n"}},
		{"<?php\nclass A extends B implements D, C {}\nclass B implements Single {}\n", [6]string{"<?php\nclass A extends B implements C, D {}\nclass B implements Single {}\n", "<?php\nclass A extends B implements D, C {}\nclass B implements Single {}\n", "<?php\nclass A extends B implements D, C {}\nclass B implements Single {}\n", "<?php\nclass A extends B implements D, C {}\nclass B implements Single {}\n", "<?php\nclass A extends B implements C, D {}\nclass B implements Single {}\n", "<?php\nclass A extends B implements D, C {}\nclass B implements Single {}\n"}},
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
