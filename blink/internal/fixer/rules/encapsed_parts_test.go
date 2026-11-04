package rules

import (
	"fmt"
	"strings"
	"testing"

	"blink/internal/lexer"
	"blink/internal/tokens"
)

// The parts of every String token are the tokens token_get_all gives for the
// literals, leaving out the code (variables, names, punctuation) between chunks.
func TestEncapsedParts(t *testing.T) {
	t.Parallel()
	names := map[encPartKind]string{
		encOther: "Other", encConstString: "Const", encText: "Text", encStartHeredoc: "Start",
		encVarname: "Varname", encDollarOpen: "DollarOpen", encCurlyOpen: "CurlyOpen",
	}
	cases := []struct{ src, want string }{
		{`<?php $x = b'a' . "b" . B"c" . "${d}";`,
			`Const "b'a'" Const "\"b\"" Const "B\"c\"" Other "\"" DollarOpen "${" Varname "d" Other "}" Other "\""`},
		// the "}" after a simple variable is text, after "{$...}" or "${...}" it closes
		{`<?php $x = "a${b}c{$d}e${f[1]}g$h\{$i} {\$j}";`,
			`Other "\"" Text "a" DollarOpen "${" Varname "b" Other "}" Text "c" CurlyOpen "{" Other "}" Text "e" ` +
				`DollarOpen "${" Varname "f" Other "}" Text "g" Text "\\{" Text "} {\\$j}" Other "\""`},
		{"<?php $x = b<<<EOT\n  {$a} ${b}\r\n  EOT;",
			`Start "b<<<EOT\n" Text "  " CurlyOpen "{" Other "}" Text " " DollarOpen "${" Varname "b" Other "}" Text "\r\n" Other "  EOT"`},
		{"<?php $x = `ls $a`;", "Other \"`\" Text \"ls \" Other \"`\""},
		{"<?php $x = <<<'N'\nq$a\nN;", `Start "<<<'N'\n" Text "q$a\n" Other "N"`},
		{`<?php $x = "${f()}";`, `Other "\"" DollarOpen "${" Other "}" Other "\""`},
	}
	for _, c := range cases {
		s := tokens.New(lexer.Lex(c.src))
		var got []string
		eachStringChunk(s, func(i int, parts []encPart) {
			v := s.At(i).Value
			for _, p := range parts {
				got = append(got, fmt.Sprintf("%s %q", names[p.kind], v[p.start:p.end]))
			}
		})
		if g := strings.Join(got, " "); g != c.want {
			t.Errorf("%q\n got %s\nwant %s", c.src, g, c.want)
		}
	}
}
