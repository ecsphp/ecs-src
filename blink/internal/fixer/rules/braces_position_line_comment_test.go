package rules

import "testing"

// PHP-CS-Fixer moves the brace in front of the comment. Blink keeps the input, because
// writing the brace on the line of the comment would make it part of the comment.
func TestBracesPositionKeepsBraceOutOfLineComment(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		"if ($b) // c\n{\n    echo 1;\n}",
		"if ($b) # c\n{\n    echo 1;\n}",
		"$a = new class // c\n{\n    public $x;\n};",
		"$a = new readonly class // c\n{\n    public $x;\n};",
		"$a = new readonly class($b) implements C // c\n{\n    public $x;\n};",
	} {
		src = "<?php\n" + src + "\n"
		if got, changed := apply(t, BracesPosition{}, src); changed || got != src {
			t.Errorf("%s changed=%v got=%q", src, changed, got)
		}
	}
}
