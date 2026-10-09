package lexer

import (
	"strings"
	"testing"
)

// FuzzLexLossless generalizes TestLexLossless: for any input, concatenating the
// token values must reproduce the source byte for byte. Seed corpus runs on
// every `go test`; `go test -fuzz` explores further.
func FuzzLexLossless(f *testing.F) {
	seeds := []string{
		"",
		"<?php $x = 1;\n",
		"plain html only",
		"<html><?php echo $a ; ?><div>after</div>",
		"<?php\n// line comment\n$foo = 'bar';\n/** doc */\nfunction f() {}\n",
		"<?php $s = \"a ; b\"; $t = 'c ; d';\n",
		"text <?= $v ?> more",
		"<?php <<<EOT\nheredoc\nEOT;\n",
		"<?xml version=\"1.0\"?><?php $a;",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		var got strings.Builder
		for _, tk := range Lex(src) {
			got.WriteString(tk.Value)
		}
		if got.String() != src {
			t.Fatalf("lossless violated\n src: %q\n got: %q", src, got.String())
		}
	})
}
