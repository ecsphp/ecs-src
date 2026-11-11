package rules

import "testing"

// Expectations are PHP-CS-Fixer's own output: correctLineBreaks keeps a gap
// that already has the wanted newlines, prepends missing ones and drops
// surplus ones from the start.
func TestClassAttributesSeparationLineBreaks(t *testing.T) {
	t.Parallel()
	f := ClassAttributesSeparation{}
	for _, c := range []struct{ name, src, want string }{
		{
			"blank line holding spaces is kept",
			"<?php\nclass K\n{\n    public $a;\n    \n    public $b;\n}\n",
			"<?php\nclass K\n{\n    public $a;\n    \n    public $b;\n}\n",
		},
		{
			"docblock gap holding spaces is kept",
			"<?php\nclass D\n{\n    public $a;\n    \n    /** doc */\n    public $b;\n}\n",
			"<?php\nclass D\n{\n    public $a;\n    \n    /** doc */\n    public $b;\n}\n",
		},
		{
			"missing newline is prepended",
			"<?php\nclass F\n{\n    public $a;\n    public $b;\n}\n",
			"<?php\nclass F\n{\n    public $a;\n\n    public $b;\n}\n",
		},
		{
			"surplus newline is dropped from the start",
			"<?php\nclass M\n{\n    public $a;\n    \n\n    public $b;\n}\n",
			"<?php\nclass M\n{\n    public $a;    \n\n    public $b;\n}\n",
		},
		{
			"enum cases default to no blank line",
			"<?php\nenum E: string\n{\n    case A = 'a';\n    \n    case B = 'b';\n}\n",
			"<?php\nenum E: string\n{\n    case A = 'a';    \n    case B = 'b';\n}\n",
		},
	} {
		got, _ := apply(t, f, c.src)
		if got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
