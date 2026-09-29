package rules

import (
	"strings"

	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/LowercaseCastFixer.php
//
// LowercaseCast lowercases the type inside a cast: "(INT)$x" -> "(int)$x".
type LowercaseCast struct{}

func (LowercaseCast) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\LowercaseCastFixer`
}

func (LowercaseCast) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/LowercaseCastFixer.php"
}

func (LowercaseCast) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		typeIdx, _, ok := castAt(s, i)
		if !ok {
			continue
		}
		v := s.At(typeIdx).Value
		if lower := strings.ToLower(v); lower != v {
			s.SetValue(typeIdx, lower)
			changed = true
		}
	}
	return changed
}

var shortScalarCast = map[string]string{
	"integer": "int",
	"boolean": "bool",
	"double":  "float",
	"real":    "float",
	"binary":  "string",
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/ShortScalarCastFixer.php
//
// ShortScalarCast rewrites long cast names to the short form: "(integer)" ->
// "(int)", "(boolean)" -> "(bool)", "(double)"/"(real)" -> "(float)".
type ShortScalarCast struct{}

func (ShortScalarCast) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\ShortScalarCastFixer`
}

func (ShortScalarCast) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/ShortScalarCastFixer.php"
}

func (ShortScalarCast) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		typeIdx, _, ok := castAt(s, i)
		if !ok {
			continue
		}
		if short, found := shortScalarCast[strings.ToLower(s.At(typeIdx).Value)]; found {
			s.SetValue(typeIdx, short)
			changed = true
		}
	}
	return changed
}
