package rules

import (
	"reflect"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

const forbiddenFunctionsChecker = `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\ForbiddenFunctionsSniff`

func forbiddenFunctionsErrors(f fixer.Fixer, src string) []fixer.CodingStandardError {
	return f.(ForbiddenFunctions).Errors(tokens.New(lexer.Lex(src)))
}

// TestForbiddenFunctionsUpstream ports PHP_CodeSniffer's
// ForbiddenFunctionsUnitTest with the default list (sizeof, delete).
func TestForbiddenFunctionsUpstream(t *testing.T) {
	t.Parallel()
	src := `<?php
$size = sizeof($array);
$size = count($array);
delete($filepath);
unset($filepath);
$size = \sizeof($array);
$size = \my\ns\sizeof('abc');($array);

// No errors thrown for class methods.
$size = MyClass::sizeof($array);
$size = MyClass::count($array);
MyClass::delete($filepath);
MyClass::unset($filepath);

$size = $class->sizeof($array);
$size = $class->count($array);
$class->delete($filepath);
$class->unset($filepath);

function delete() {}
function unset() {}
function sizeof() {}
function count() {}

trait DelProvider {
  public function delete() {
    //irrelevant
  }
}

class LeftSideTest {
    use DelProvider {
        delete as protected unsetter;
    }
}

class RightSideTest {
    use DelProvider {
        delete as delete;
    }
}

class RightSideVisTest {
    use DelProvider {
        delete as protected delete;
        DelProvider::delete insteadof delete;
    }
}

namespace Something\sizeof;
$var = new   /*comment*/   Sizeof();
class SizeOf implements Something {}

function mymodule_form_callback(SizeOf $sizeof) {
}

$size = $class?->sizeof($array);

#[SizeOf(10)]
function doSomething() {}

$size = sizeof /*comment*/ ($array);
`
	sizeof := "The use of function sizeof() is forbidden; use count() instead"
	want := []fixer.CodingStandardError{
		{Line: 2, Message: sizeof, Checker: forbiddenFunctionsChecker + ".FoundWithAlternative"},
		{Line: 4, Message: "The use of function delete() is forbidden; use unset() instead", Checker: forbiddenFunctionsChecker + ".FoundWithAlternative"},
		{Line: 6, Message: sizeof, Checker: forbiddenFunctionsChecker + ".FoundWithAlternative"},
		{Line: 62, Message: sizeof, Checker: forbiddenFunctionsChecker + ".FoundWithAlternative"},
	}
	if got := forbiddenFunctionsErrors(ForbiddenFunctions{}, src); !reflect.DeepEqual(got, want) {
		t.Fatalf("errors:\n got %+v\nwant %+v", got, want)
	}
	s := tokens.New(lexer.Lex(src))
	if (ForbiddenFunctions{}).Fix(s) || s.Render() != src {
		t.Fatal("the sniff must never change the file")
	}
}

// TestForbiddenFunctionsConfigured runs the list from the issue: names are
// matched case insensitively, a null alternative reports "Found", and the
// keyword-like constructs PHP tokenizes on their own need no "(".
func TestForbiddenFunctionsConfigured(t *testing.T) {
	t.Parallel()
	f := ForbiddenFunctions{}.WithConfig(map[string]any{
		"forbiddenFunctions": map[string]any{"dump": nil, "DD": nil, "var_dump": nil, "print": nil, "exit": "return"},
	})
	src := "<?php\nDump($a);\n\\dd($b);\nFoo\\dump($c);\nnamespace\\dump($d);\n$this->dump();\n#[dump(1)]\nfunction f() {}\nprint 'x';\nexit;\nvar_dump(...);\n"
	want := []fixer.CodingStandardError{
		{Line: 2, Message: "The use of function dump() is forbidden", Checker: forbiddenFunctionsChecker + ".Found"},
		{Line: 3, Message: "The use of function dd() is forbidden", Checker: forbiddenFunctionsChecker + ".Found"},
		{Line: 9, Message: "The use of function print() is forbidden", Checker: forbiddenFunctionsChecker + ".Found"},
		{Line: 10, Message: "The use of function exit() is forbidden; use return() instead", Checker: forbiddenFunctionsChecker + ".FoundWithAlternative"},
		{Line: 11, Message: "The use of function var_dump() is forbidden", Checker: forbiddenFunctionsChecker + ".Found"},
	}
	if got := forbiddenFunctionsErrors(f, src); !reflect.DeepEqual(got, want) {
		t.Fatalf("errors:\n got %+v\nwant %+v", got, want)
	}

	// the configured list replaces the default one
	if got := forbiddenFunctionsErrors(f, "<?php\nsizeof($a);\n"); len(got) != 0 {
		t.Fatalf("sizeof is no longer forbidden, got %+v", got)
	}
}

// TestForbiddenFunctionsWarnings reports nothing with error false: the sniff
// then adds warnings, which ECS drops by default.
func TestForbiddenFunctionsWarnings(t *testing.T) {
	t.Parallel()
	f := ForbiddenFunctions{}.WithConfig(map[string]any{"error": false})
	if got := forbiddenFunctionsErrors(f, "<?php\nsizeof($a);\n"); len(got) != 0 {
		t.Fatalf("want no errors, got %+v", got)
	}
}
