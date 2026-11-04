package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDedicateAssertInternalTypeFixer.php
//
// PhpUnitDedicateAssertInternalType replaces assertInternalType("array", $x)
// with the dedicated assertIsArray($x) and assertNotInternalType with the
// negated form, inside test classes. The target option does not change the
// mapping. Risky: assumes PHPUnit's own methods.
type PhpUnitDedicateAssertInternalType struct{}

var phpUnitInternalTypeMap = map[string]string{
	"array":    "assertIsArray",
	"boolean":  "assertIsBool",
	"bool":     "assertIsBool",
	"double":   "assertIsFloat",
	"float":    "assertIsFloat",
	"integer":  "assertIsInt",
	"int":      "assertIsInt",
	"null":     "assertNull",
	"numeric":  "assertIsNumeric",
	"object":   "assertIsObject",
	"real":     "assertIsFloat",
	"resource": "assertIsResource",
	"string":   "assertIsString",
	"scalar":   "assertIsScalar",
	"callable": "assertIsCallable",
	"iterable": "assertIsIterable",
}

func (PhpUnitDedicateAssertInternalType) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDedicateAssertInternalTypeFixer`
}

func (PhpUnitDedicateAssertInternalType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDedicateAssertInternalTypeFixer.php"
}

// WithConfig accepts the target option, which does not affect behavior.
func (f PhpUnitDedicateAssertInternalType) WithConfig(map[string]any) fixer.Fixer {
	return f
}

func (PhpUnitDedicateAssertInternalType) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if dedicateInternalTypeClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func dedicateInternalTypeClass(s *tokens.Stream, startIndex, endIndex int) bool {
	anon := map[int]int{}
	for i := startIndex; i < endIndex; i++ {
		if !kwIs(s.At(i), "class") || !taIsAnonymousClass(s, i) {
			continue
		}
		open := nextPunctOfKind(s, i, "{")
		if open < 0 {
			continue
		}
		if end := s.MatchForward(open); end > open {
			anon[end] = open
		}
	}

	changed := false
	for index := endIndex - 1; index > startIndex; index-- {
		if open, ok := anon[index]; ok {
			index = open
			continue
		}
		t := s.At(index)
		if t.Kind != token.Ident {
			continue
		}
		fn := strings.ToLower(t.Value)
		if fn != "assertinternaltype" && fn != "assertnotinternaltype" {
			continue
		}
		bracket := sigNext(s, index)
		if !isPunctVal(s, bracket, "(") {
			continue
		}
		typeIdx := sigNext(s, bracket)
		if typeIdx < 0 || !s.At(typeIdx).IsLiteral() {
			continue
		}
		expectedType := strings.Trim(s.At(typeIdx).Value, "'\"")
		newAssertion, ok := phpUnitInternalTypeMap[expectedType]
		if !ok {
			continue
		}
		comma := sigNext(s, typeIdx)
		if !isPunctVal(s, comma, ",") {
			continue
		}
		if fn == "assertnotinternaltype" {
			newAssertion = strings.ReplaceAll(newAssertion, "Is", "IsNot")
			newAssertion = strings.ReplaceAll(newAssertion, "Null", "NotNull")
		}
		next := sigNext(s, comma)
		if next < 0 {
			continue
		}
		s.ReplaceRange(index, next-1, []token.Token{
			{Kind: token.Ident, Value: newAssertion},
			{Kind: token.Punct, Value: "("},
		})
		changed = true
	}
	return changed
}
