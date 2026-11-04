package rules

import (
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// fixedSources collects every source a test fixes through apply, so TestMain
// can check every Candidate rule against all of them.
var fixedSources sync.Map

// TestMain runs the tests, then checks that no Candidate rule rules out a
// source it would change, over every source the tests fixed: thousands of
// snippets, most written to catch a rule's edge cases.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		code = checkCandidates()
	}
	os.Exit(code)
}

func checkCandidates() int {
	var srcs []string
	fixedSources.Range(func(k, _ any) bool {
		srcs = append(srcs, k.(string))
		return true
	})
	slices.Sort(srcs)
	failed := 0
	for _, f := range Builtin() {
		c, ok := f.(fixer.Candidate)
		if !ok {
			continue
		}
		for _, src := range srcs {
			if msg := candidateViolation(c, f, src); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
				failed++
			}
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: %d sources ruled out by IsCandidate that Fix changes\n", failed)
		return 1
	}
	return 0
}

// candidateViolation reports, when c rules src out, whether Fix would still
// change it or report a change: the runner would then skip a real fix.
func candidateViolation(c fixer.Candidate, f fixerRule, src string) string {
	if c.IsCandidate(tokens.New(lexer.Lex(src))) {
		return ""
	}
	s := tokens.New(lexer.Lex(src))
	if changed := f.Fix(s); changed || s.Render() != src {
		return fmt.Sprintf("%T: IsCandidate is false, but Fix changes (changed=%v) %q into %q", f, changed, src, s.Render())
	}
	return ""
}

// Each gated rule fixes its trigger in an unusual spelling (upper or mixed
// case), so IsCandidate must hold; and a file without the trigger is ruled out.
func TestCandidateTriggers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rule    fixerRule
		fixed   string // a source the rule changes, the trigger oddly spelled
		without string // a source without the trigger
	}{
		{NoUselessSprintf{}, "<?php echo SprintF('a');", "<?php echo sprint('a'); $sprintf = 'sprintf';"},
		{ImplodeCall{}, "<?php IMPLODE($a);", "<?php explode(',', $a); // implode($a)"},
		{NoAliasFunctions{}, "<?php SizeOf($a);", "<?php count($a);"},
		{EregToPreg{}, "<?php EREG('a', $b);", "<?php preg_match('/a/', $b);"},
		{FunctionToConstant{}, "<?php echo PHPVersion();", "<?php echo PHP_VERSION;"},
		{ModernizeTypesCasting{}, "<?php IntVal($a);", "<?php (int) $a;"},
		{MagicConstantCasing{}, "<?php echo __Line__;", "<?php echo __LINES__;"},
		{MagicMethodCasing{}, "<?php class A { function __TOSTRING() {} }", "<?php class A { function toString() {} }"},
		{NoAliasLanguageConstructCall{}, "<?php DIE;", "<?php exit;"},
		{CombineNestedDirname{}, "<?php DirName(DIRNAME($a));", "<?php basename(basename($a));"},
		{DirConstant{}, "<?php DIRNAME(__FILE__);", "<?php __DIR__;"},
		{SetTypeToCast{}, "<?php SETTYPE($a, 'int');", "<?php gettype($a);"},
		{ArrayPush{}, "<?php Array_Push($a, $b);", "<?php $a[] = $b;"},
		{PowToExponentiation{}, "<?php POW($a, 2);", "<?php $a ** 2;"},
		{IsNull{}, "<?php IS_NULL($a);", "<?php null === $a;"},
		{GetClassToClassKeyword{}, "<?php GET_CLASS($a);", "<?php $a::class;"},
		{FopenFlagOrder{}, "<?php FOpen($f, 'bw');", "<?php fclose($f);"},
		{ModernizeStrpos{}, "<?php if (StrPos($a, $b) !== false) {}", "<?php str_contains($a, $b);"},
		{StringLengthToEmpty{}, "<?php if (STRLEN($a) === 0) {}", "<?php mb_strlen($a);"},
		{ModernSerializationMethods{}, "<?php class A { function __sleep() {} }", "<?php class A { function __serialize() {} }"},
		{SwitchCaseSemicolonToColon{}, "<?php SWITCH ($a) { CASE 1; }", "<?php match ($a) { 1 => 2 };"},
		{SwitchCaseSpace{}, "<?php Switch ($a) { case 1 : }", "<?php if ($a) {}"},
		{SwitchContinueToBreak{}, "<?php SWITCH ($a) { case 1: CONTINUE; }", "<?php while ($a) { continue; }"},
		{NoBreakComment{}, "<?php SWITCH ($a) { case 1: f(); case 2: g(); }", "<?php if ($a) { f(); }"},
		{ListSyntax{}, "<?php LIST($a, $b) = $c;", "<?php [$a, $b] = $c;"},
		{NoUnsetCast{}, "<?php $a = (UNSET) $a;", "<?php $a = null;"},
		{NoMixedEchoPrint{}, "<?php PRINT 'a';", "<?php echo 'a';"},
		{LogicalOperators{}, "<?php $a AND $b;", "<?php $a && $b;"},
		{LowercaseStaticReference{}, "<?php class A { function f() { return SELF::X; } }", "<?php class A { function f() { return A::X; } }"},
		{NonPrintableCharacter{}, "<?php echo 'a\u200bb';", "<?php echo 'ab'; // a\\u200b escape is ASCII"},
		{NullableTypeDeclaration{}, "<?php function f(NULL|int $a) {}", "<?php function f(?int $a) {}"},
		{NullableTypeDeclarationForDefaultNullValue{}, "<?php function f(int $a = NULL) {}", "<?php function f(int $a = 1) {}"},
		{NoNullPropertyInitialization{}, "<?php CLASS A { public $a = NULL; }", "<?php interface A { const B = null; }"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%T", c.rule), func(t *testing.T) {
			cand, ok := c.rule.(fixer.Candidate)
			if !ok {
				t.Fatal("not a fixer.Candidate")
			}
			if !cand.IsCandidate(tokens.New(lexer.Lex(c.fixed))) {
				t.Errorf("IsCandidate false for %q", c.fixed)
			}
			if _, changed := apply(t, c.rule, c.fixed); !changed {
				t.Errorf("Fix leaves %q as is: the case does not exercise the trigger", c.fixed)
			}
			if cand.IsCandidate(tokens.New(lexer.Lex(c.without))) {
				t.Errorf("IsCandidate true for %q", c.without)
			}
		})
	}
}

// A configuration that rewrites the other way round can rule nothing out:
// ListSyntax's long syntax rewrites any "[", and NullableTypeDeclaration's
// union syntax any "?".
func TestCandidateOtherWayRound(t *testing.T) {
	t.Parallel()
	if !(ListSyntax{Long: true}).IsCandidate(tokens.New(lexer.Lex("<?php [$a] = $b;"))) {
		t.Error("the long list syntax must stay a candidate without list()")
	}
	union := configuredFromJSON(t, NullableTypeDeclaration{}, `{"syntax":"union"}`)
	src := "<?php function f(?int $a) {}"
	if !union.(fixer.Candidate).IsCandidate(tokens.New(lexer.Lex(src))) {
		t.Error("the union syntax must stay a candidate without null")
	}
	if _, changed := apply(t, union, src); !changed {
		t.Errorf("the union syntax leaves %q as is", src)
	}
}

// The docblock rules are candidates only with a docblock (or, for the ones
// that also read plain comments, a comment).
func TestCandidateDocblockRules(t *testing.T) {
	t.Parallel()
	const (
		bare    = "<?php class A { public function f($a) { return $a; } }"
		comment = "<?php // a\nclass A { public function f($a) { return $a; } }"
		doc     = "<?php class A { /** @param int $a */ public function f($a) { return $a; } }"
	)
	for _, c := range []struct {
		rule        fixer.Candidate
		withComment bool // a plain comment is enough
	}{
		{PhpdocOrder{}, false},
		{PhpdocTypesOrder{}, false},
		{PhpdocLineSpan{}, false},
		{NoSuperfluousPhpdocTags{}, false},
		{TypeToVarTag{}, true},
		{RemoveUselessDefaultComment{}, true},
	} {
		t.Run(fmt.Sprintf("%T", c.rule), func(t *testing.T) {
			is := func(src string) bool { return c.rule.IsCandidate(tokens.New(lexer.Lex(src))) }
			if is(bare) {
				t.Error("candidate without any comment")
			}
			if is(comment) != c.withComment {
				t.Errorf("candidate with a plain comment: %v", is(comment))
			}
			if !is(doc) {
				t.Error("not a candidate with a docblock")
			}
		})
	}
}
