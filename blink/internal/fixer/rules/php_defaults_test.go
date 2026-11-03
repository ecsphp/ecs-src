package rules

import (
	"testing"

	"blink/internal/fixer"
)

// TestRegistryResolvesPHPCSFixerDefaults checks that a fixer looked up by name,
// as the ECS bridge does for a rule dumped with an empty config, behaves like
// PHP-CS-Fixer's default configuration.
func TestRegistryResolvesPHPCSFixerDefaults(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rule, in, want string
	}{
		{
			`PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`,
			"<?php\nif ($x === null) {}\nif ($a == 5) {}\nif ($b < 3) {}\n",
			"<?php\nif (null === $x) {}\nif (5 == $a) {}\nif ($b < 3) {}\n",
		},
	}
	for _, c := range cases {
		f, ok := ByName(c.rule)
		if !ok {
			t.Fatalf("%s: not registered", c.rule)
		}
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("%s with no options:\n got %q\nwant %q", c.rule, got, c.want)
		}
	}
}

// TestPartialConfigKeepsPHPCSFixerDefaults checks that options a dumped config
// leaves out (they equal the default) keep PHP-CS-Fixer's default.
func TestPartialConfigKeepsPHPCSFixerDefaults(t *testing.T) {
	t.Parallel()
	f, _ := ByName(`PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`)
	f = f.(fixer.ConfigurableFixer).WithConfig(map[string]any{"less_and_greater": true})
	got, _ := apply(t, f, "<?php\nif ($x === null) {}\nif ($b < 3) {}\n")
	if want := "<?php\nif (null === $x) {}\nif (3 > $b) {}\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
