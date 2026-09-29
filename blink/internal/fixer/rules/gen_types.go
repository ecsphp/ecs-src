package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// isFunctionParamOpen reports whether the "(" at open opens a function, method
// or closure parameter list (keyword function/fn, or a named function ident
// preceded, optionally via "&", by the function keyword).
func isFunctionParamOpen(s *tokens.Stream, open int) bool {
	if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return false
	}
	p := prevSignificantIndex(s, open)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	if pt.Kind == token.Keyword {
		v := strings.ToLower(pt.Value)
		return v == "function" || v == "fn"
	}
	if pt.Kind == token.Ident {
		q := prevSignificantIndex(s, p)
		if q >= 0 && s.At(q).Kind == token.Punct && s.At(q).Value == "&" {
			q = prevSignificantIndex(s, q)
		}
		return q >= 0 && s.At(q).Kind == token.Keyword && strings.ToLower(s.At(q).Value) == "function"
	}
	return false
}

// enclosingFuncParamOpen returns the index of the "(" of the function parameter
// list that directly encloses i, or -1. It stops at a "{", "}" or ";" reached at
// paren depth zero so it never crosses out of the signature into a body.
func enclosingFuncParamOpen(s *tokens.Stream, i int) int {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			depth++
		case "(":
			if depth == 0 {
				if isFunctionParamOpen(s, j) {
					return j
				}
				return -1
			}
			depth--
		case "{", "}", ";":
			if depth == 0 {
				return -1
			}
		}
	}
	return -1
}

// isNullableTypePos reports whether the "?" at i is a nullable-type marker rather
// than a ternary. A nullable "?" follows a type-position token (":", "(", ",",
// "|" or a visibility/modifier keyword); a ternary "?" follows an expression.
func isNullableTypePos(s *tokens.Stream, i int) bool {
	p := prevSignificantIndex(s, i)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	if pt.Kind == token.Punct {
		switch pt.Value {
		case ":", "(", ",", "|":
			return true
		}
		return false
	}
	if pt.Kind == token.Keyword {
		switch strings.ToLower(pt.Value) {
		case "public", "private", "protected", "readonly", "static", "var":
			return true
		}
	}
	return false
}

// isTypeNameToken reports whether t can name a type (a class name / Ident, or a
// native type that happens to lex as a Keyword).
func isTypeNameToken(t token.Token) bool {
	if t.Kind == token.Ident {
		return true
	}
	if t.Kind == token.Keyword {
		return nativeTypeNames[strings.ToLower(t.Value)]
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/CompactNullableTypeDeclarationFixer.php
//
// CompactNullableTypeDeclaration removes the whitespace between a nullable "?"
// and its type ("? int" -> "?int"). It only touches a "?" in a type position, so
// ternary, elvis ("?:"), null-coalesce ("??") and nullsafe ("?->") are left alone.
type CompactNullableTypeDeclaration struct{}

func (CompactNullableTypeDeclaration) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\CompactNullableTypeDeclarationFixer`
}

func (CompactNullableTypeDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/CompactNullableTypeDeclarationFixer.php"
}

func (CompactNullableTypeDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "?" {
			continue
		}
		if !isNullableTypePos(s, i) {
			continue
		}
		n := nextSignificantIndex(s, i)
		if n < 0 || !isTypeNameToken(s.At(n)) {
			continue
		}
		// drop the single-line whitespace between "?" and the type
		for j := n - 1; j > i; j-- {
			if s.At(j).Kind == token.Whitespace && !hasNewline(s.At(j).Value) {
				s.RemoveAt(j)
				changed = true
			}
		}
	}
	return changed
}

// inReturnType reports whether the type token at i sits in a return-type
// position, i.e. scanning left across type parts ("?", "|", "&", "\", names)
// reaches a return-type colon.
func inReturnType(s *tokens.Stream, i int) bool {
	j := i
	for {
		p := prevSignificantIndex(s, j)
		if p < 0 {
			return false
		}
		pt := s.At(p)
		if pt.Kind == token.Punct && pt.Value == ":" {
			return isReturnTypeColon(s, p)
		}
		if pt.Kind == token.Ident ||
			(pt.Kind == token.Punct && (pt.Value == "?" || pt.Value == "|" || pt.Value == "&" || pt.Value == `\`)) {
			j = p
			continue
		}
		return false
	}
}

// isFunctionSignatureType reports whether the token at i is a type in a function
// signature: a parameter type inside the parameter list, or a return type.
func isFunctionSignatureType(s *tokens.Stream, i int) bool {
	if enclosingFuncParamOpen(s, i) >= 0 && typeContextPrev(s, i) && typeContextNext(s, i) {
		return true
	}
	return inReturnType(s, i)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeFunctionTypeDeclarationCasingFixer.php
//
// NativeFunctionTypeDeclarationCasing lowercases native types used in function
// parameter and return type declarations ("function f(INT $x): STRING" -> "int",
// "string"). It only rewrites reserved native type names, which cannot name a
// class, and only in a function-signature type position.
type NativeFunctionTypeDeclarationCasing struct{}

func (NativeFunctionTypeDeclarationCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\NativeFunctionTypeDeclarationCasingFixer`
}

func (NativeFunctionTypeDeclarationCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeFunctionTypeDeclarationCasingFixer.php"
}

func (NativeFunctionTypeDeclarationCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident && t.Kind != token.Keyword {
			continue
		}
		lower := strings.ToLower(t.Value)
		if !nativeTypeNames[lower] || t.Value == lower {
			continue
		}
		if !isFunctionSignatureType(s, i) {
			continue
		}
		s.SetValue(i, lower)
		changed = true
	}
	return changed
}
