package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitConstructFixer.php
//
// PhpUnitConstruct rewrites PHPUnit assertions against a true/false/null
// literal to the dedicated assertion, inside PHPUnit test classes:
// `$this->assertSame(true, $a)` becomes `$this->assertTrue($a)`. The assertions
// option lists the methods to fix (assertEquals, assertNotEquals, assertNotSame
// and assertSame by default). Risky upstream: it assumes the methods are
// PHPUnit's own.
type PhpUnitConstruct struct {
	// assertions is the assertions option; nil keeps the default (all four).
	assertions []string
}

// phpUnitConstructAssertions is the default (and allowed) assertions option.
var phpUnitConstructAssertions = []string{"assertEquals", "assertNotEquals", "assertNotSame", "assertSame"}

var (
	phpUnitConstructPositive = map[string]string{"false": "assertFalse", "null": "assertNull", "true": "assertTrue"}
	phpUnitConstructNegative = map[string]string{"false": "assertNotFalse", "null": "assertNotNull", "true": "assertNotTrue"}
)

func (PhpUnitConstruct) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitConstructFixer`
}

func (PhpUnitConstruct) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitConstructFixer.php"
}

func (f PhpUnitConstruct) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["assertions"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.assertions = append([]string{}, list...)
		}
	}
	return f
}

func (f PhpUnitConstruct) Fix(s *tokens.Stream) bool {
	assertions := f.assertions
	if assertions == nil {
		assertions = phpUnitConstructAssertions
	}
	if len(assertions) == 0 {
		return false
	}
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		for _, method := range assertions {
			mapping := phpUnitConstructNegative
			if method == "assertSame" || method == "assertEquals" {
				mapping = phpUnitConstructPositive
			}
			for index := cls[0]; index < cls[1]; index++ {
				next, fixed, ok := phpUnitConstructFixAssert(s, mapping, index, method)
				if !ok {
					break
				}
				changed = changed || fixed
				index = next
			}
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// phpUnitConstructFixAssert mirrors PhpUnitConstructFixer::fixAssert: it finds
// the next `method(` from index (to the end of the file, as Tokens::findSequence
// does) and rewrites it when its first argument is a lone native constant. It
// returns the index to continue from and whether it rewrote the call, or
// ok=false to stop for this method.
func phpUnitConstructFixAssert(s *tokens.Stream, mapping map[string]string, index int, method string) (next int, fixed, ok bool) {
	nameIdx, openIdx := -1, -1
	for i := index; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Ident || t.Value != method {
			continue
		}
		if n := sigNext(s, i); isPunctVal(s, n, "(") {
			nameIdx, openIdx = i, n
			break
		}
	}
	if nameIdx < 0 {
		return 0, false, false
	}
	if !puIsTheSameClassCall(s, nameIdx) {
		return 0, false, false
	}
	first := sigNext(s, openIdx)
	if first < 0 {
		return 0, false, false
	}
	if !puIsNativeConstant(s.At(first)) {
		return first, false, true
	}
	comma := sigNext(s, first)
	if comma < 0 {
		return 0, false, false
	}
	if !isPunctVal(s, comma, ",") {
		return comma, false, true
	}
	s.Set(nameIdx, token.Token{Kind: token.Ident, Value: mapping[strings.ToLower(s.At(first).Value)], Pos: s.At(nameIdx).Pos})
	stop := puNextNonWhitespace(s, comma)
	if stop < 0 {
		stop = s.Len()
	}
	for i := first; i < stop; i++ {
		clearTokenAt(s, i)
	}
	return comma, true, true
}
