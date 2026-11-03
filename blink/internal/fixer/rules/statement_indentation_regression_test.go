package rules

import "testing"

// malformed input once drove the scope-pop loop past the root scope and panicked
// with index -1; Fix must now return without panicking on any input.
func TestStatementIndentationNoPanicOnMalformed(t *testing.T) {
	t.Parallel()
	for _, src := range []string{"<?0(", "<?php }", "<?php {{{", "<?php case:"} {
		apply(t, StatementIndentation{}, src) // must not panic
	}
}

// A promoted constructor parameter with a leading attribute is still a param, not
// a class property, so the params after it must not gain an extra indent level.
func TestStatementIndentationAttributedPromotedParam(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nclass E\n{\n    public function __construct(\n        public $guard,\n        #[\\SensitiveParameter]\n        public $credentials,\n        public $remember,\n    ) {\n    }\n}\n"
	if got, changed := apply(t, StatementIndentation{}, src); changed || got != src {
		t.Fatalf("attributed promoted param must not be re-indented; changed=%v got=%q", changed, got)
	}
}

// A PHP 8.4 "final" promoted constructor property is still a param: "final"
// before the visibility must not make it a class property, or the params after
// it and the closing ") {" gain an extra indent level.
func TestStatementIndentationFinalPromotedParam(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nfinal class Flag\n{\n    public function __construct(\n        final public bool $on = false,\n        public int $level = 0,\n    ) {}\n}\n\nclass Box\n{\n    public function __construct(\n        public int $size = 0,\n        #[Marker]\n        final protected bool $open = false,\n    ) {\n    }\n}\n"
	if got, changed := apply(t, StatementIndentation{}, src); changed || got != src {
		t.Fatalf("final promoted param must not be re-indented; changed=%v got=%q", changed, got)
	}
}
