package lexer

import (
	"fmt"
	"strings"
	"testing"

	"blink/internal/token"
)

// interpolationDump renders the tokens of src from its first String token on,
// up to the first ";", one per line as "Kind value", a String token with its
// part (W, H, M, T).
func interpolationDump(src string) string {
	var b strings.Builder
	started := false
	for _, tk := range Lex(src) {
		if tk.Kind == token.String {
			started = true
		}
		if !started {
			continue
		}
		if tk.Kind == token.Punct && tk.Value == ";" {
			break // the end of the statement under test
		}
		k := tk.Kind.String()
		if tk.Kind == token.String {
			k += string("WHMT"[tk.Part])
		}
		fmt.Fprintf(&b, "%s %q\n", k, tk.Value)
	}
	return b.String()
}

// The split points are token_get_all(TOKEN_PARSE)'s; each run of tokens that
// are not code (quotes, T_ENCAPSED_AND_WHITESPACE, T_CURLY_OPEN, ...) is one String.
func TestLexInterpolation(t *testing.T) {
	cases := []struct{ src, want string }{
		{`<?php $x = "a $b c";`, `StringH "\"a "
Variable "$b"
StringT " c\""
`},
		{`<?php $x = "$a[0]$a[-1]$a[$k]$a[Bar]";`, `StringH "\""
Variable "$a"
Punct "["
Number "0"
Punct "]"
Variable "$a"
Punct "["
Punct "-"
Number "1"
Punct "]"
Variable "$a"
Punct "["
Variable "$k"
Punct "]"
Variable "$a"
Punct "["
Ident "Bar"
Punct "]"
StringT "\""
`},
		// one level of property access only; "?->" too; "->" without a name is text
		{`<?php $x = "$o->p->q $o?->r $o-> x";`, `StringH "\""
Variable "$o"
Punct "->"
Ident "p"
StringM "->q "
Variable "$o"
Punct "?->"
Ident "r"
StringM " "
Variable "$o"
StringT "-> x\""
`},
		// "${name}" is all literal (T_STRING_VARNAME); "${name[" and "${expr}" lex code
		{`<?php $x = "${a} ${a[BAR]} ${f()}";`, `StringH "\"${a} ${a"
Punct "["
Ident "BAR"
Punct "]"
StringM "} ${"
Ident "f"
Punct "("
Punct ")"
StringT "}\""
`},
		{`<?php $x = "{$a['}']} \$b \{$c} {\$d}";`, `StringH "\"{"
Variable "$a"
Punct "["
StringW "'}'"
Punct "]"
StringM "} \\$b \\{"
Variable "$c"
StringT "} {\\$d}\""
`},
		{`<?php $x = b"x$y";`, `StringH "b\"x"
Variable "$y"
StringT "\""
`},
		{"<?php $x = <<<EOT\n  {$a[Bar::X]} $b\n  EOT;\n", `StringH "<<<EOT\n  {"
Variable "$a"
Punct "["
Ident "Bar"
Punct "::"
Ident "X"
Punct "]"
StringM "} "
Variable "$b"
StringT "\n  EOT"
`},
		// a label that only starts a line, or sits mid-line, does not close
		{"<?php $x = <<<\"EOT\"\nEOTX $a\nx EOT\nEOT;\n", `StringH "<<<\"EOT\"\nEOTX "
Variable "$a"
StringT "\nx EOT\nEOT"
`},
		// heredoc in an interpolated expression of a heredoc
		{"<?php $x = <<<A\n{$f(<<<B\n$c\nB)}\nA;\n", `StringH "<<<A\n{"
Variable "$f"
Punct "("
StringH "<<<B\n"
Variable "$c"
StringT "\nB"
Punct ")"
StringT "}\nA"
`},
	}
	for _, c := range cases {
		if got := interpolationDump(c.src); got != c.want {
			t.Errorf("%q:\n got:\n%s\nwant:\n%s", c.src, got, c.want)
		}
	}
}

// The names and calls inside a heredoc's "{$...}" are ordinary tokens.
func TestLexHeredocInterpolationExposesCode(t *testing.T) {
	src := "<?php\necho <<<EOT\n    {$x->m(new Foo(), strlen($y), PHP_EOL)} $a[Bar]\n    EOT;\n"
	toks := Lex(src)
	var code []string
	for _, tk := range toks {
		switch tk.Kind {
		case token.Ident, token.Keyword, token.Variable:
			code = append(code, tk.Kind.String()+":"+tk.Value)
		case token.Punct:
			code = append(code, tk.Value)
		}
	}
	want := "Keyword:echo Variable:$x -> Ident:m ( Keyword:new Ident:Foo ( ) , Ident:strlen ( Variable:$y ) , Ident:PHP_EOL ) Variable:$a [ Ident:Bar ] ;"
	if got := strings.Join(code, " "); got != want {
		t.Errorf("code tokens\n got %s\nwant %s", got, want)
	}
}

// Backtick literals split like double-quoted strings; a backtick in the code of
// a "{$...}" does not close the literal.
func TestLexBacktick(t *testing.T) {
	cases := []struct{ src, want string }{
		{"<?php $x = `ls -la`;", "StringW \"`ls -la`\"\n"},
		{"<?php $x = ``;", "StringW \"``\"\n"},
		{"<?php $x = `echo intval(1) # x // y /* z`;", "StringW \"`echo intval(1) # x // y /* z`\"\n"},
		{"<?php $x = `a \\` \\$b \"$c\" '${d}'`;", `StringH "` + "`a \\\\` \\\\$b \\\"" + `"
Variable "$c"
StringT "\" '${d}'` + "`" + `"
`},
		{"<?php $x = `echo {$a[\"`\"]} $b->c`;", `StringH "` + "`echo {" + `"
Variable "$a"
Punct "["
StringW "\"` + "`" + `\""
Punct "]"
StringM "} "
Variable "$b"
Punct "->"
Ident "c"
StringT "` + "`" + `"
`},
		{"<?php $x = `{$f(`pwd`)}`;", `StringH "` + "`{" + `"
Variable "$f"
Punct "("
StringW "` + "`pwd`" + `"
Punct ")"
StringT "}` + "`" + `"
`},
		{"<?php $x = \"a `$b` c\";", `StringH "\"a ` + "`" + `"
Variable "$b"
StringT "` + "`" + ` c\""
`},
	}
	for _, c := range cases {
		if got := interpolationDump(c.src); got != c.want {
			t.Errorf("%q:\n got:\n%s\nwant:\n%s", c.src, got, c.want)
		}
	}
}

func TestIsShellExec(t *testing.T) {
	src := "<?php $a = `ls $b x`; $c = \"`ls`\"; $d = 'x`'; $e = `pwd`;"
	var got []string
	for _, tk := range Lex(src) {
		if tk.Kind == token.String {
			got = append(got, fmt.Sprintf("%s:%v", string("WHMT"[tk.Part]), tk.IsShellExec()))
		}
	}
	want := "H:true T:true W:false W:false W:true"
	if g := strings.Join(got, " "); g != want {
		t.Errorf("got %s, want %s", g, want)
	}
}
