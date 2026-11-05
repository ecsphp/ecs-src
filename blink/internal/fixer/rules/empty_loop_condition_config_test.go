package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestEmptyLoopConditionConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		EmptyLoopCondition{},
		EmptyLoopCondition{}.WithConfig(map[string]any{"style": "while"}).(fixerRule),
		EmptyLoopCondition{}.WithConfig(map[string]any{"style": "for"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [3]string
	}{
		{"<?php\nfor (;;) {\n    foo();\n}\nwhile (true) {\n    bar();\n}\n", [3]string{"<?php\nwhile (true)  {\n    foo();\n}\nwhile (true) {\n    bar();\n}\n", "<?php\nwhile (true)  {\n    foo();\n}\nwhile (true) {\n    bar();\n}\n", "<?php\nfor (;;) {\n    foo();\n}\n  for(;;){\n    bar();\n}\n"}},
		{"<?php\ndo {\n    foo();\n} while (true);\n", [3]string{"<?php\nwhile (true) {\n    foo();\n}  \n", "<?php\nwhile (true) {\n    foo();\n}  \n", "<?php\nfor(;;) {\n    foo();\n}  \n"}},
		{"<?php\ndo {\n    foo();\n} while (true) ;\n$a = 1;\n", [3]string{"<?php\nwhile (true) {\n    foo();\n}   \n$a = 1;\n", "<?php\nwhile (true) {\n    foo();\n}   \n$a = 1;\n", "<?php\nfor(;;) {\n    foo();\n}   \n$a = 1;\n"}},
		{"<?php\nfor(;;){}\nwhile(true){}\nfor ( ; ; ) {}\nwhile ( true ) {}\n", [3]string{"<?php\nwhile (true){}\nwhile(true){}\n     while (true){}\nwhile ( true ) {}\n", "<?php\nwhile (true){}\nwhile(true){}\n     while (true){}\nwhile ( true ) {}\n", "<?php\nfor(;;){}\nfor(;;){}\nfor ( ; ; ) {}\n    for(;;){}\n"}},
		{"<?php\nfor (/* a */;;) {}\nfor (; /* b */ ;) {}\nfor (;; /* c */) {}\nfor /* d */ (;;) {}\n", [3]string{"<?php\nwhile (true) /* a */ {}\n  while (true)/* b */ {}\n  while (true)/* c */ {}\n while (true)/* d */ {}\n", "<?php\nwhile (true) /* a */ {}\n  while (true)/* b */ {}\n  while (true)/* c */ {}\n while (true)/* d */ {}\n", "<?php\nfor (/* a */;;) {}\nfor (; /* b */ ;) {}\nfor (;; /* c */) {}\nfor /* d */ (;;) {}\n"}},
		{"<?php\nwhile (/* a */ true) {}\nwhile (true /* b */) {}\nwhile /* c */ (true) {}\n", [3]string{"<?php\nwhile (/* a */ true) {}\nwhile (true /* b */) {}\nwhile /* c */ (true) {}\n", "<?php\nwhile (/* a */ true) {}\nwhile (true /* b */) {}\nwhile /* c */ (true) {}\n", "<?php\nfor(;;) /* a */ {}\n  for(;;)/* b */ {}\n for(;;)/* c */ {}\n"}},
		{"<?php\ndo {\n} /* x */ while (true /* y */) /* z */;\n", [3]string{"<?php\nwhile (true) {\n} /* x */   /* y */ /* z */\n", "<?php\nwhile (true) {\n} /* x */   /* y */ /* z */\n", "<?php\nfor(;;) {\n} /* x */   /* y */ /* z */\n"}},
		{"<?php\nwhile (TRUE) {}\nwhile (True) {}\nwhile (\\true) {}\nwhile (true && $x) {}\nwhile (1) {}\nfor ($i = 0;;) {}\nfor (;;$i++) {}\n", [3]string{"<?php\nwhile (TRUE) {}\nwhile (True) {}\nwhile (\\true) {}\nwhile (true && $x) {}\nwhile (1) {}\nfor ($i = 0;;) {}\nfor (;;$i++) {}\n", "<?php\nwhile (TRUE) {}\nwhile (True) {}\nwhile (\\true) {}\nwhile (true && $x) {}\nwhile (1) {}\nfor ($i = 0;;) {}\nfor (;;$i++) {}\n", "<?php\nwhile (TRUE) {}\nwhile (True) {}\nwhile (\\true) {}\nwhile (true && $x) {}\nwhile (1) {}\nfor ($i = 0;;) {}\nfor (;;$i++) {}\n"}},
		{"<?php\nwhile (true):\n    foo();\nendwhile;\nfor (;;):\n    foo();\nendfor;\n", [3]string{"<?php\nwhile (true):\n    foo();\nendwhile;\n while (true):\n    foo();\nendfor;\n", "<?php\nwhile (true):\n    foo();\nendwhile;\n while (true):\n    foo();\nendfor;\n", "<?php\nfor(;;) :\n    foo();\nendwhile;\nfor (;;):\n    foo();\nendfor;\n"}},
		{"<?php\nwhile (true) {\n    for (;;) {\n        do {\n        } while (true);\n    }\n}\n", [3]string{"<?php\nwhile (true) {\n      while (true){\n        while (true) {\n        }  \n    }\n}\n", "<?php\nwhile (true) {\n      while (true){\n        while (true) {\n        }  \n    }\n}\n", "<?php\nfor(;;)  {\n    for (;;) {\n        for(;;) {\n        }  \n    }\n}\n"}},
		{"<?php\nif ($x) {\n} while (true) {}\n", [3]string{"<?php\nif ($x) {\n} while (true) {}\n", "<?php\nif ($x) {\n} while (true) {}\n", "<?php\nif ($x) {\n}   for(;;){}\n"}},
		{"<?php\nwhile (true) foo();\nfor (;;) foo();\n", [3]string{"<?php\nwhile (true) foo();\n  while (true)foo();\n", "<?php\nwhile (true) foo();\n  while (true)foo();\n", "<?php\nfor(;;)  foo();\nfor (;;) foo();\n"}},
		{"<?php\nfunction f() {\n    do {\n        yield 1;\n    }\n    while (true);\n}\n", [3]string{"<?php\nfunction f() {\n    while (true) {\n        yield 1;\n    }\n     \n}\n", "<?php\nfunction f() {\n    while (true) {\n        yield 1;\n    }\n     \n}\n", "<?php\nfunction f() {\n    for(;;) {\n        yield 1;\n    }\n     \n}\n"}},
		{"<?php for (;;) {}", [3]string{"<?php while (true)  {}", "<?php while (true)  {}", "<?php for (;;) {}"}},
		{"<?php for(;;){ foo(); }", [3]string{"<?php while (true){ foo(); }", "<?php while (true){ foo(); }", "<?php for(;;){ foo(); }"}},
		{"<?php for ( ; ; ) { foo(); }", [3]string{"<?php while (true)     { foo(); }", "<?php while (true)     { foo(); }", "<?php for ( ; ; ) { foo(); }"}},
		{"<?php for ($i=0;$i<3;$i++) {}\nfor (;$i<3;) {}\n", [3]string{"<?php for ($i=0;$i<3;$i++) {}\nfor (;$i<3;) {}\n", "<?php for ($i=0;$i<3;$i++) {}\nfor (;$i<3;) {}\n", "<?php for ($i=0;$i<3;$i++) {}\nfor (;$i<3;) {}\n"}},
		{"<?php $o->for(1);\n$o->while(true);\nFoo::while(true);\n", [3]string{"<?php $o->for(1);\n$o->while(true);\nFoo::while(true);\n", "<?php $o->for(1);\n$o->while(true);\nFoo::while(true);\n", "<?php $o->for(1);\n$o->while(true);\nFoo::while(true);\n"}},
		{"<?php\ndo {\n} while (true) ?>\n", [3]string{"<?php\nwhile (true) {\n}   ", "<?php\nwhile (true) {\n}   ", "<?php\nfor(;;) {\n}   "}},
		{"<?php\ndo $i++; while (true);\n", [3]string{"<?php\ndo $i++; while (true);\n", "<?php\ndo $i++; while (true);\n", "<?php\ndo $i++;  for(;;);\n"}},
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
