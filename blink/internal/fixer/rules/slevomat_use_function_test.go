package rules

import (
	"strings"
	"testing"
)

// TestSlevomatUseFunctionImport: "function" in a function import is no
// declaration (T_STRING in PHPCS), see https://github.com/ecsphp/ecs-src/issues/422
func TestSlevomatUseFunctionImport(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"<?php\n\nuse function Foo\\bar;\n",
		"<?php\n\nuse Foo\\{function bar, function baz};\n",
	}
	for _, f := range All() {
		if !strings.HasPrefix(f.Name(), "SlevomatCodingStandard\\") {
			continue
		}
		for _, input := range inputs {
			t.Run(f.Name(), func(t *testing.T) {
				t.Parallel()
				apply(t, f, input)
			})
		}
	}
}
