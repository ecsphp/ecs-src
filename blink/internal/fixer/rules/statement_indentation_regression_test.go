package rules

import "testing"

// malformed input once drove the scope-pop loop past the root scope and panicked
// with index -1; Fix must now return without panicking on any input.
func TestStatementIndentationNoPanicOnMalformed(t *testing.T) {
	for _, src := range []string{"<?0(", "<?php }", "<?php {{{", "<?php case:"} {
		apply(t, StatementIndentation{}, src) // must not panic
	}
}
