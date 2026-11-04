package rules

import "testing"

func TestNativeConstantInvocationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPhpCsFixerCases(t, NativeConstantInvocation{}, nativeConstantInvocationUpstreamCases)
}

// Regression cases, checked against the PHP engine, with the options of a
// typical ECS config.
func TestNativeConstantInvocationRegressions(t *testing.T) {
	t.Parallel()
	config := map[string]any{"scope": "namespaced"}
	runPhpCsFixerCases(t, NativeConstantInvocation{}, []phpCsFixerCase{
		{"bitwise flags, defaults and types", "<?php\n\nnamespace App;\n\n$flags = \\JSON_THROW_ON_ERROR | \\JSON_PRETTY_PRINT;\necho \\PHP_EOL, __DIR__, (int) $x, \\M_PI;\nfunction f(int|string $a = \\PHP_INT_MAX, ?Foo $b = null): Bar|E_ALL {}\n", "<?php\n\nnamespace App;\n\n$flags = JSON_THROW_ON_ERROR | JSON_PRETTY_PRINT;\necho PHP_EOL, __DIR__, (int) $x, M_PI;\nfunction f(int|string $a = PHP_INT_MAX, ?Foo $b = null): Bar|E_ALL {}\n", config},
		{"imported constants and strict mode", "<?php\n\nnamespace App;\n\nuse const PHP_EOL;\n\necho PHP_EOL, true, MY_CONST, null, Foo\\BAR;\n", "<?php\n\nnamespace App;\n\nuse const PHP_EOL;\n\necho PHP_EOL, \\true, \\MY_CONST, \\null, Foo\\BAR;\n", config},
	})
}
