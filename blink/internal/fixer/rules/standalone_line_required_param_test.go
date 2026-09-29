package rules

import "testing"

func TestStandaloneLineRequiredParam(t *testing.T) {
	f := StandaloneLineRequiredParam{}

	// #[Required] public method: each parameter on its own line
	got, changed := apply(t, f, "<?php\nclass A\n{\n    #[Required]\n    public function autowire(FormModel $a, SubModel $b): void\n    {\n    }\n}\n")
	if want := "<?php\nclass A\n{\n    #[Required]\n    public function autowire(\n        FormModel $a,\n        SubModel $b\n    ): void\n    {\n    }\n}\n"; !changed || got != want {
		t.Fatalf("attribute: changed=%v got=%q", changed, got)
	}

	// @required doc annotation triggers the same break
	got, changed = apply(t, f, "<?php\nclass A\n{\n    /**\n     * @required\n     */\n    public function autowire(FormModel $a, SubModel $b): void\n    {\n    }\n}\n")
	if want := "<?php\nclass A\n{\n    /**\n     * @required\n     */\n    public function autowire(\n        FormModel $a,\n        SubModel $b\n    ): void\n    {\n    }\n}\n"; !changed || got != want {
		t.Fatalf("annotation: changed=%v got=%q", changed, got)
	}

	// a single parameter is still broken out
	got, changed = apply(t, f, "<?php\nclass A\n{\n    #[Required]\n    public function autowire(FormModel $a): void\n    {\n    }\n}\n")
	if want := "<?php\nclass A\n{\n    #[Required]\n    public function autowire(\n        FormModel $a\n    ): void\n    {\n    }\n}\n"; !changed || got != want {
		t.Fatalf("single: changed=%v got=%q", changed, got)
	}

	// non-public #[Required] method is left inline
	if _, changed := apply(t, f, "<?php\nclass A\n{\n    #[Required]\n    protected function autowire(FormModel $a, SubModel $b): void\n    {\n    }\n}\n"); changed {
		t.Fatal("only public methods are targeted")
	}

	// public method without #[Required]/@required is left inline
	if _, changed := apply(t, f, "<?php\nclass A\n{\n    public function autowire(FormModel $a, SubModel $b): void\n    {\n    }\n}\n"); changed {
		t.Fatal("method needs #[Required] or @required")
	}
}
