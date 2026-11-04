package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitStrictFixer.php
//
// PhpUnitStrict replaces loose PHPUnit assertions with their strict counterpart
// (assertEquals -> assertSame, ...) inside test classes, when the call has two
// or three arguments. The assertions option lists the methods to fix (all four
// by default). Risky: assumes the methods are PHPUnit's own.
type PhpUnitStrict struct {
	assertions []string
}

var phpUnitStrictMap = map[string]string{
	"assertAttributeEquals":    "assertAttributeSame",
	"assertAttributeNotEquals": "assertAttributeNotSame",
	"assertEquals":             "assertSame",
	"assertNotEquals":          "assertNotSame",
}

var phpUnitStrictDefault = []string{"assertAttributeEquals", "assertAttributeNotEquals", "assertEquals", "assertNotEquals"}

func (PhpUnitStrict) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitStrictFixer`
}

func (PhpUnitStrict) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitStrictFixer.php"
}

func (f PhpUnitStrict) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["assertions"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.assertions = append([]string{}, list...)
		}
	}
	return f
}

func (f PhpUnitStrict) Fix(s *tokens.Stream) bool {
	assertions := f.assertions
	if assertions == nil {
		assertions = phpUnitStrictDefault
	}
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		start, end := cls[0], cls[1]
		for _, methodBefore := range assertions {
			methodAfter, ok := phpUnitStrictMap[methodBefore]
			if !ok {
				continue
			}
			for index := start; index < end; index++ {
				methodIndex := phpUnitNextStringExact(s, index, methodBefore)
				if methodIndex < 0 {
					break
				}
				if puIsTheSameClassCall(s, methodIndex) {
					open := sigNext(s, methodIndex)
					if isPunctVal(s, open, "(") {
						if n := len(aliasArguments(s, open, s.MatchForward(open))); n == 2 || n == 3 {
							s.SetValue(methodIndex, methodAfter)
							changed = true
						}
					}
				}
				index = methodIndex
			}
		}
	}
	return changed
}

// phpUnitNextStringExact returns the index of the next Ident after `from` whose
// value equals name exactly (case-sensitive), mirroring getNextTokenOfKind with
// a [T_STRING, name] target. It is not bounded by the class end, like upstream.
func phpUnitNextStringExact(s *tokens.Stream, from int, name string) int {
	for i := from + 1; i < s.Len(); i++ {
		if t := s.At(i); t.Kind == token.Ident && t.Value == name {
			return i
		}
	}
	return -1
}
