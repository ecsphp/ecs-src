package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

var magicConstants = map[string]string{
	"__line__":                 "__LINE__",
	"__file__":                 "__FILE__",
	"__dir__":                  "__DIR__",
	"__function__":             "__FUNCTION__",
	"__class__":                "__CLASS__",
	"__trait__":                "__TRAIT__",
	"__method__":               "__METHOD__",
	"__namespace__":            "__NAMESPACE__",
	"__compiler_halt_offset__": "__COMPILER_HALT_OFFSET__",
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/MagicConstantCasingFixer.php
//
// MagicConstantCasing uppercases magic constants (__line__ -> __LINE__).
type MagicConstantCasing struct{}

func (MagicConstantCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\MagicConstantCasingFixer`
}

func (MagicConstantCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/MagicConstantCasingFixer.php"
}

func (MagicConstantCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		if canonical, ok := magicConstants[strings.ToLower(t.Value)]; ok && canonical != t.Value {
			s.SetValue(i, canonical)
			changed = true
		}
	}
	return changed
}

var magicMethods = map[string]string{
	"__construct":   "__construct",
	"__destruct":    "__destruct",
	"__call":        "__call",
	"__callstatic":  "__callStatic",
	"__get":         "__get",
	"__set":         "__set",
	"__isset":       "__isset",
	"__unset":       "__unset",
	"__sleep":       "__sleep",
	"__wakeup":      "__wakeup",
	"__serialize":   "__serialize",
	"__unserialize": "__unserialize",
	"__tostring":    "__toString",
	"__invoke":      "__invoke",
	"__set_state":   "__set_state",
	"__clone":       "__clone",
	"__debuginfo":   "__debugInfo",
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/MagicMethodCasingFixer.php
//
// MagicMethodCasing fixes the casing of magic methods (__CONSTRUCT ->
// __construct, __tostring -> __toString).
type MagicMethodCasing struct{}

func (MagicMethodCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\MagicMethodCasingFixer`
}

func (MagicMethodCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/MagicMethodCasingFixer.php"
}

func (MagicMethodCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		canonical, ok := magicMethods[strings.ToLower(t.Value)]
		if !ok || canonical == t.Value {
			continue
		}
		// a magic method is always a declaration or call ("__set("); a constant
		// named "__SET" (followed by "=") must not be recased
		if nextSignificantValue(s, i) != "(" {
			continue
		}
		s.SetValue(i, canonical)
		changed = true
	}
	return changed
}
