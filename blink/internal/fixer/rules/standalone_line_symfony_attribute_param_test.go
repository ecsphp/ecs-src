package rules

import "testing"

func TestStandaloneLineSymfonyAttributeParam(t *testing.T) {
	f := StandaloneLineSymfonyAttributeParam{}

	// AsCommand always breaks, even with a single argument set
	got, changed := apply(t, f, "<?php\n#[AsCommand(name: 'app:some', description: 'Some description')]\nclass C\n{\n}\n")
	if want := "<?php\n#[AsCommand(\n    name: 'app:some',\n    description: 'Some description'\n)]\nclass C\n{\n}\n"; !changed || got != want {
		t.Fatalf("AsCommand: changed=%v got=%q", changed, got)
	}

	// Route with 2+ args breaks, indented to the attribute line
	got, changed = apply(t, f, "<?php\nclass C\n{\n    #[Route('/path', name: 'r')]\n    public function x() {}\n}\n")
	if want := "<?php\nclass C\n{\n    #[Route(\n        '/path',\n        name: 'r'\n    )]\n    public function x() {}\n}\n"; !changed || got != want {
		t.Fatalf("Route2: changed=%v got=%q", changed, got)
	}

	// Route with a single arg is left inline (only AsCommand breaks on <2 args)
	if _, changed := apply(t, f, "<?php\nclass C\n{\n    #[Route('/path')]\n    public function x() {}\n}\n"); changed {
		t.Fatal("single-arg Route must stay inline")
	}

	// attribute outside the allowlist is untouched
	if _, changed := apply(t, f, "<?php\n#[Foo(a: 1, b: 2)]\nclass C\n{\n}\n"); changed {
		t.Fatal("non-allowlisted attribute must stay inline")
	}
}
