package rules

import "testing"

// TestAddMissingParamName adds the positional variable name to a type-only @param.
func TestAddMissingParamName(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param string\n     */\n    function f($name) {}\n}\n"
	want := "<?php\nclass A {\n    /**\n     * @param string $name\n     */\n    function f($name) {}\n}\n"
	got, changed := apply(t, AddMissingParamName{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AddMissingParamName{}, got)
}

// TestAddMissingParamNameMultiLineArrayShape leaves a @param whose multi-line
// array-shape type already carries its name after the closing brace untouched.
func TestAddMissingParamNameMultiLineArrayShape(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param array{\n     *     timed: list<array{category: array{label: string}, lines: list<string>}>,\n     *     timeless: list<array{category: array{label: string}, lines: list<string>}>\n     * } $messages\n     */\n    function f(array $messages) {}\n}\n"
	if got, changed := apply(t, AddMissingParamName{}, src); changed || got != src {
		t.Fatalf("multi-line array-shape param with name must be left alone: changed=%v got=%q", changed, got)
	}
}
