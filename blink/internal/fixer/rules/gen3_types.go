package rules

import (
	"strings"

	"blink/internal/fixer"
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
// ("int$x" and "int  $x" -> "int $x"). This is the canonical rule replacing the
// deprecated function_typehint_space; it also covers native types that lex as
// keywords ("array", "callable"). Line breaks between type and variable are
// preserved.
//
// Elements selects where it applies ("function", "property", "constant"); nil
// means function and property.
type TypeDeclarationSpaces struct {
	Elements map[string]bool
}

func (TypeDeclarationSpaces) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\TypeDeclarationSpacesFixer`
}

func (TypeDeclarationSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/TypeDeclarationSpacesFixer.php"
}

func (f TypeDeclarationSpaces) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := singleSpaceStringList(config["elements"]); ok {
		f.Elements = map[string]bool{}
		for _, name := range list {
			f.Elements[name] = true
		}
	}
	return f
}

func (f TypeDeclarationSpaces) has(element string) bool {
	if f.Elements == nil {
		return element != "constant"
	}
	return f.Elements[element]
}

func (f TypeDeclarationSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if f.has("constant") && s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "const") {
			if typeDeclarationSpacesConst(s, i) {
				changed = true
			}
			continue
		}
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
		if inFunc := enclosingFuncParamOpen(s, i) >= 0; (inFunc && !f.has("function")) || (!inFunc && !f.has("property")) {
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

// typeDeclarationSpacesConst fixes the space between the type and the name of a
// typed constant ("const int  X = 1;") at the const keyword at i.
func typeDeclarationSpacesConst(s *tokens.Stream, i int) bool {
	eq := -1
	for j := i + 1; j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.Punct && (t.Value == "=" || t.Value == ";") {
			if t.Value == "=" {
				eq = j
			}
			break
		}
	}
	if eq < 0 {
		return false
	}
	name := prevSignificantIndex(s, eq)
	if name < 0 || s.At(name).Kind != token.Ident {
		return false
	}
	typeEnd := prevSignificantIndex(s, name)
	if typeEnd < 0 || typeEnd == i || !isTypeNameToken(s.At(typeEnd)) {
		return false
	}
	if typeEnd == name-1 {
		s.InsertAt(name, token.Token{Kind: token.Whitespace, Value: " "})
		return true
	}
	if typeEnd == name-2 && s.At(name-1).Kind == token.Whitespace {
		ws := s.At(name - 1).Value
		if ws != " " && !hasNewline(ws) {
			s.SetValue(name-1, " ")
			return true
		}
	}
	return false
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
//
// Options: UnionNull (use_nullable_type_declaration: true) also appends "|null" to
// plain unions. NoNullable (use_nullable_type_declaration: false) instead removes
// "?" and "|null" from parameters defaulting to null, skipping promoted
// constructor properties.
type NullableTypeDeclarationForDefaultNullValue struct {
	UnionNull  bool
	NoNullable bool
}

func (NullableTypeDeclarationForDefaultNullValue) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NullableTypeDeclarationForDefaultNullValueFixer`
}

func (NullableTypeDeclarationForDefaultNullValue) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NullableTypeDeclarationForDefaultNullValueFixer.php"
}

func (f NullableTypeDeclarationForDefaultNullValue) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["use_nullable_type_declaration"].(bool); ok {
		f.UnionNull = v
		f.NoNullable = !v
	}
	return f
}

// nullableTypeRunHas returns the index of the first token in the type run [start, end]
// whose value equals the given one, or -1.
func nullableTypeRunHas(s *tokens.Stream, start, end int, value string) int {
	for j := start; j <= end; j++ {
		if strings.EqualFold(s.At(j).Value, value) && s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

// nullableTypeRemove drops the nullable marker or "null" member from the type run
// [start, end] and reports whether it changed.
func nullableTypeRemove(s *tokens.Stream, start, end int, hasNullable bool) bool {
	if hasNullable {
		s.RemoveAt(start)
		if start < s.Len() && s.At(start).Kind == token.Whitespace {
			s.RemoveAt(start)
		}
		return true
	}
	if nullableTypeRunHas(s, start, end, "&") >= 0 {
		return false
	}
	k := nullableTypeRunHas(s, start, end, "null")
	if k < 0 {
		return false
	}
	lo, hi := k, k
	if n := nextSignificantIndex(s, k); n >= 0 && n <= end && s.At(n).Value == "|" {
		hi = n
		if hi+1 < s.Len() && s.At(hi+1).Kind == token.Whitespace {
			hi++
		}
	} else if p := prevSignificantIndex(s, k); p >= start && s.At(p).Value == "|" {
		lo = p
		if lo-1 >= start && s.At(lo-1).Kind == token.Whitespace {
			lo--
		}
	} else {
		return false
	}
	s.ReplaceRange(lo, hi, nil)
	return true
}

func (f NullableTypeDeclarationForDefaultNullValue) Fix(s *tokens.Stream) bool {
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
		// atomic single type: skip "mixed" and standalone "null"
		if start == p {
			switch strings.ToLower(s.At(p).Value) {
			case "mixed", "null":
				continue
			}
		}
		if f.NoNullable {
			if b := prevSignificantIndex(s, start); b >= 0 && s.At(b).Kind == token.Keyword && isVisibilityModifier(s.At(b).Value) {
				continue
			}
			if (hasNullable || hasUnion) && nullableTypeRemove(s, start, p, hasNullable) {
				changed = true
				i = start
			}
			continue
		}
		if hasUnion && !hasNullable && f.UnionNull {
			if nullableTypeRunHas(s, start, p, "&") >= 0 || nullableTypeRunHas(s, start, p, "null") >= 0 {
				continue
			}
			s.InsertAt(p+1, token.Token{Kind: token.Punct, Value: "|"})
			s.InsertAt(p+2, token.Token{Kind: token.Ident, Value: "null"})
			changed = true
			i = start
			continue
		}
		if hasNullable || hasUnion {
			continue
		}
		s.InsertAt(start, token.Token{Kind: token.Punct, Value: "?"})
		changed = true
		i = start
	}
	return changed
}
