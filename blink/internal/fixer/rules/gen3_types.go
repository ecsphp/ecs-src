package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// isTypeDeclarationVariable reports whether the Variable at v is preceded by a
// type declaration: either a function/method/closure parameter type, or a typed
// property (a type run that begins after a visibility/modifier keyword). p is the
// significant token before v and must already be a type-name token.
func isTypeDeclarationVariable(s *tokens.Stream, v, p int) bool {
	if enclosingFuncParamOpen(s, v) >= 0 {
		return true
	}
	boundary := typeRunBoundaryPrev(s, p)
	if boundary < 0 {
		return false
	}
	bt := s.At(boundary)
	return bt.Kind == token.Keyword && isVisibilityModifier(bt.Value)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/TypeDeclarationSpacesFixer.php
//
// TypeDeclarationSpaces forces exactly one space between a type declaration and
// the "$variable" it declares, in function parameters and typed properties
// ("int$x" and "int  $x" -> "int $x"), in function parameters and typed
// properties. This is the canonical rule replacing the deprecated
// function_typehint_space; it also covers native types that lex as keywords
// ("array", "callable"). Line breaks between type and variable are preserved.
type TypeDeclarationSpaces struct{}

func (TypeDeclarationSpaces) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\TypeDeclarationSpacesFixer`
}

func (TypeDeclarationSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/TypeDeclarationSpacesFixer.php"
}

func (TypeDeclarationSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Variable {
			continue
		}
		p := prevSignificantIndex(s, i)
		if p < 0 || !isTypeNameToken(s.At(p)) {
			continue
		}
		if !isTypeDeclarationVariable(s, i, p) {
			continue
		}
		// type token abuts the variable: insert a single space
		if p == i-1 {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
			continue
		}
		// type, single whitespace, variable: collapse to a single space
		if p == i-2 && s.At(i-1).Kind == token.Whitespace {
			ws := s.At(i - 1).Value
			if ws != " " && !hasNewline(ws) {
				s.SetValue(i-1, " ")
				changed = true
			}
		}
	}
	return changed
}

// hasNullDefault reports whether the parameter variable at v has a default value
// that is exactly "null" (the "=" then "null" then the parameter's "," or ")").
func hasNullDefault(s *tokens.Stream, v int) bool {
	eq := nextSignificantIndex(s, v)
	if eq < 0 || s.At(eq).Kind != token.Punct || s.At(eq).Value != "=" {
		return false
	}
	nv := nextSignificantIndex(s, eq)
	if nv < 0 || s.At(nv).Kind != token.Ident || strings.ToLower(s.At(nv).Value) != "null" {
		return false
	}
	after := nextSignificantIndex(s, nv)
	if after < 0 {
		return false
	}
	at := s.At(after)
	return at.Kind == token.Punct && (at.Value == "," || at.Value == ")")
}

// typeRunStart walks left from the type's last token at end across the type parts
// ("\", names, and the operators "?", "|", "&") and returns the index of the
// type's first token together with whether the run contains a union/intersection
// ("|" or "&") or a nullable marker ("?").
func typeRunStart(s *tokens.Stream, end int) (start int, hasUnion, hasNullable bool) {
	start = end
	for {
		pp := prevSignificantIndex(s, start)
		if pp < 0 {
			break
		}
		pt := s.At(pp)
		if pt.Kind == token.Punct {
			switch pt.Value {
			case "|", "&":
				hasUnion = true
				start = pp
				continue
			case "?":
				hasNullable = true
				start = pp
				continue
			case `\`:
				start = pp
				continue
			}
			break
		}
		if isTypeNameToken(pt) {
			start = pp
			continue
		}
		break
	}
	return start, hasUnion, hasNullable
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NullableTypeDeclarationForDefaultNullValueFixer.php
//
// NullableTypeDeclarationForDefaultNullValue prepends "?" to a non-nullable
// parameter type whose default value is null ("function f(int $x = null)" ->
// "function f(?int $x = null)"). It acts only inside function/method/closure
// parameter lists and skips a type that is already nullable, "mixed", standalone
// "null", or a union/intersection (which would need "|null", not a "?" prefix).
type NullableTypeDeclarationForDefaultNullValue struct{}

func (NullableTypeDeclarationForDefaultNullValue) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NullableTypeDeclarationForDefaultNullValueFixer`
}

func (NullableTypeDeclarationForDefaultNullValue) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NullableTypeDeclarationForDefaultNullValueFixer.php"
}

func (NullableTypeDeclarationForDefaultNullValue) Fix(s *tokens.Stream) bool {
	changed := false
	// Right-to-left so an inserted "?" (always left of the current index) never
	// disturbs positions still to be scanned.
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind != token.Variable {
			continue
		}
		if enclosingFuncParamOpen(s, i) < 0 {
			continue
		}
		if !hasNullDefault(s, i) {
			continue
		}
		p := prevSignificantIndex(s, i)
		if p < 0 {
			continue
		}
		// step over a by-reference "&" to reach the type's last token
		if s.At(p).Kind == token.Punct && s.At(p).Value == "&" {
			p = prevSignificantIndex(s, p)
			if p < 0 {
				continue
			}
		}
		if !isTypeNameToken(s.At(p)) {
			continue
		}
		start, hasUnion, hasNullable := typeRunStart(s, p)
		if hasNullable || hasUnion {
			continue
		}
		// atomic single type: skip "mixed" and standalone "null"
		if start == p {
			switch strings.ToLower(s.At(p).Value) {
			case "mixed", "null":
				continue
			}
		}
		s.InsertAt(start, token.Token{Kind: token.Punct, Value: "?"})
		changed = true
		i = start
	}
	return changed
}
