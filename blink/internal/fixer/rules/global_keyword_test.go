package rules

import (
	"reflect"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

func globalKeywordErrors(src string) []fixer.CodingStandardError {
	return GlobalKeyword{}.Errors(tokens.New(lexer.Lex(src)))
}

// TestGlobalKeywordUpstream ports PHP_CodeSniffer's GlobalKeywordUnitTest: one
// error per "global" statement, on its line, naming the first variable.
func TestGlobalKeywordUpstream(t *testing.T) {
	t.Parallel()
	src := "<?php\n$var = 'hello';\n$var2 = 'hello';\n$GLOBALS['var'] = 'hello';\n\nfunction func1()\n{\n    global $var;\n    global $var, $var2;\n    echo $var;\n    echo $GLOBALS['var'];\n}\n?>\n"
	const checker = `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\GlobalKeywordSniff.NotAllowed`
	want := []fixer.CodingStandardError{
		{Line: 8, Message: `Use of the "global" keyword is forbidden; use "$GLOBALS['var']" instead`, Checker: checker},
		{Line: 9, Message: `Use of the "global" keyword is forbidden; use "$GLOBALS['var']" instead`, Checker: checker},
	}
	if got := globalKeywordErrors(src); !reflect.DeepEqual(got, want) {
		t.Fatalf("errors:\n got %+v\nwant %+v", got, want)
	}
	s := tokens.New(lexer.Lex(src))
	if (GlobalKeyword{}).Fix(s) || s.Render() != src {
		t.Fatal("the sniff must never change the file")
	}
}

// TestGlobalKeywordIgnoresNames keeps "global" used as a name out of the report.
func TestGlobalKeywordIgnoresNames(t *testing.T) {
	t.Parallel()
	src := "<?php\n$a->global;\nFoo::global();\nclass A { public function global() {} }\nfoo(global: 1);\n// global $x;\n$s = 'global $y';\n" +
		"enum Reach: string\n{\n    case Global = 'g';\n}\nfunction &global() {}\n"
	if got := globalKeywordErrors(src); len(got) != 0 {
		t.Fatalf("want no errors, got %+v", got)
	}
}

// TestGlobalKeywordVariableVariable names the variable after the "$" sigil, as
// PHP_CodeSniffer's findNext(T_VARIABLE) does.
func TestGlobalKeywordVariableVariable(t *testing.T) {
	t.Parallel()
	got := globalKeywordErrors("<?php\nfunction f() {\n    GLOBAL $$name;\n}\n")
	if len(got) != 1 || got[0].Line != 3 || got[0].Message != `Use of the "global" keyword is forbidden; use "$GLOBALS['name']" instead` {
		t.Fatalf("got %+v", got)
	}
}

// TestGlobalKeywordNoPlainVariable: a variable interpolated in a string or
// heredoc is no T_VARIABLE for PHP_CodeSniffer, and with none findNext returns
// false, which reads the file's first token.
func TestGlobalKeywordNoPlainVariable(t *testing.T) {
	t.Parallel()
	for src, want := range map[string]string{
		"<?php function g() { global ${'q'}; }\n":                                           "<?php",
		"<?php\nfunction g() {\n    global ${'q'};\n}\n":                                    "<?php",
		"<?= 1 ?><?php function g() { global ${'q'}; }\n":                                   "<?=",
		"a$b  c\nline2\n<?php function g() { global ${'q'}; }\n":                            "ab  c\n",
		"<?php function g() { global ${'q'}; echo \"$zz {$yy} ${x}\"; }\n":                  "<?php",
		"<?php function g() { global ${'q'}; echo \"{$a[\"k\"]} $b\"; $after; }\n":          "after",
		"<?php function g() { global ${'q'}; echo \"$a\\\"$b\"; $after; }\n":                "after",
		"<?php function g() { global ${'q'}; echo <<<EOT\n{$hd} $he\nEOT;\n$after; }\n":     "after",
		"<?php function g() { global ${'q'}; echo <<<\"EOT\"\n  {$hd}\n  EOT;\n$after; }\n": "after",
		"<?php function g() { global ${'q'}; echo `ls {$bt}`; }\n":                          "bt",
		"<?php function g() { global ${'q'}; echo \"x\" . \"{$s}\" . $after; }\n":           "after",
	} {
		got := globalKeywordErrors(src)
		if len(got) != 1 || got[0].Message != `Use of the "global" keyword is forbidden; use "$GLOBALS['`+want+`']" instead` {
			t.Errorf("%q: got %+v, want %q", src, got, want)
		}
	}
}
