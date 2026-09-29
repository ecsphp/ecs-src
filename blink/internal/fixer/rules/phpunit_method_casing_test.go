package rules

import "testing"

func TestPhpUnitMethodCasingDefaultsToCamelCase(t *testing.T) {
	src := "<?php\nclass FooTest extends TestCase\n{\n    public function test_it_works()\n    {\n    }\n}\n"
	want := "<?php\nclass FooTest extends TestCase\n{\n    public function testItWorks()\n    {\n    }\n}\n"
	if got, changed := apply(t, PhpUnitMethodCasing{}, src); !changed || got != want {
		t.Fatalf("default must camel-case test methods: changed=%v got=%q", changed, got)
	}
}

func TestPhpUnitMethodCasingSnakeConfig(t *testing.T) {
	src := "<?php\nclass FooTest extends TestCase\n{\n    public function testItWorks()\n    {\n    }\n}\n"
	want := "<?php\nclass FooTest extends TestCase\n{\n    public function test_it_works()\n    {\n    }\n}\n"

	snake := PhpUnitMethodCasing{}.WithConfig(map[string]any{"case": "snake_case"})
	if got, changed := apply(t, snake, src); !changed || got != want {
		t.Fatalf("snake_case config must snake-case test methods: changed=%v got=%q", changed, got)
	}
}

func TestPhpUnitMethodCasingCamelConfigLeavesCamel(t *testing.T) {
	src := "<?php\nclass FooTest extends TestCase\n{\n    public function testItWorks()\n    {\n    }\n}\n"

	camel := PhpUnitMethodCasing{}.WithConfig(map[string]any{"case": "camel_case"})
	if got, changed := apply(t, camel, src); changed || got != src {
		t.Fatalf("camel_case config must leave camelCase methods unchanged: changed=%v got=%q", changed, got)
	}
}
