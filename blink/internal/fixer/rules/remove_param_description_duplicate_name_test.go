package rules

import "testing"

// TestRemoveParamDescriptionDuplicateName drops a @param description that only
// repeats the parameter name, ignoring case and spacing.
func TestRemoveParamDescriptionDuplicateName(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param int|string $leadId Lead ID\n     * @param string $name The person name\n     */\n    function f($leadId, $name) {}\n}\n"
	want := "<?php\nclass A {\n    /**\n     * @param int|string $leadId\n     * @param string $name The person name\n     */\n    function f($leadId, $name) {}\n}\n"
	got, changed := apply(t, RemoveParamDescriptionDuplicateName{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, RemoveParamDescriptionDuplicateName{}, got)
}

// TestRemoveParamDescriptionDuplicateNameSingleLine handles a one-line docblock
// with trailing punctuation and differing case.
func TestRemoveParamDescriptionDuplicateNameSingleLine(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /** @param int $userId User Id. */\n    function f($userId) {}\n}\n"
	want := "<?php\nclass A {\n    /** @param int $userId */\n    function f($userId) {}\n}\n"
	got, changed := apply(t, RemoveParamDescriptionDuplicateName{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, RemoveParamDescriptionDuplicateName{}, got)
}

// TestRemoveParamDescriptionDuplicateNameAttributedMethod strips the duplicate
// even when the docblock sits ahead of a method attribute.
func TestRemoveParamDescriptionDuplicateNameAttributedMethod(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param int $leadId Lead ID\n     */\n    #[Route('/x/{leadId}')]\n    function f($leadId) {}\n}\n"
	want := "<?php\nclass A {\n    /**\n     * @param int $leadId\n     */\n    #[Route('/x/{leadId}')]\n    function f($leadId) {}\n}\n"
	got, changed := apply(t, RemoveParamDescriptionDuplicateName{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, RemoveParamDescriptionDuplicateName{}, got)
}

// TestRemoveParamDescriptionDuplicateNameKeepsRealDescription leaves a genuine
// description untouched.
func TestRemoveParamDescriptionDuplicateNameKeepsRealDescription(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param int $id Numeric identifier of the record\n     */\n    function f($id) {}\n}\n"
	if got, changed := apply(t, RemoveParamDescriptionDuplicateName{}, src); changed || got != src {
		t.Fatalf("real description must be left alone: changed=%v got=%q", changed, got)
	}
}
