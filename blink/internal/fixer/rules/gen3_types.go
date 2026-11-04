package rules

import (
	"slices"
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

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NullableTypeDeclarationForDefaultNullValueFixer.php
//
// NullableTypeDeclarationForDefaultNullValue prepends "?" to a non-nullable
// parameter type whose default value is null ("function f(int $x = null)" ->
// "function f(?int $x = null)"). It acts only inside function/method/closure
// parameter lists and skips a type that is already nullable, "mixed", standalone
// "null", or a union/intersection (which would need "|null", not a "?" prefix).
//
// Options: UnionNull (use_nullable_type_declaration: true) also appends "|null" to
// unions and DNF types, and wraps a lone intersection first ("A&B" ->
// "(A&B)|null"). NoNullable (use_nullable_type_declaration: false) instead removes
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

// ntdDNFGroupOpen returns the "(" of the disjunctive-normal-form group "(A&B)"
// closed by the ")" at closeIdx, or -1 when that ")" closes anything else.
func ntdDNFGroupOpen(s *tokens.Stream, closeIdx int) int {
	open := s.MatchBackward(closeIdx)
	if open < 0 {
		return -1
	}
	intersection := false
	for k := open + 1; k < closeIdx; k++ {
		t := s.At(k)
		if t.Kind == token.Punct && t.Value == "&" {
			intersection = true
			continue
		}
		if t.Kind == token.Whitespace || isTypeNameToken(t) || (t.Kind == token.Punct && t.Value == `\`) {
			continue
		}
		return -1
	}
	if !intersection {
		return -1
	}
	return open
}

// ntdTypeStart walks left from the last token of a parameter type (a name or
// the ")" of a DNF group) and returns the index of the type's first token, or
// -1 when end does not end a type.
func ntdTypeStart(s *tokens.Stream, end int) int {
	cur := end
	if t := s.At(end); t.Kind == token.Punct && t.Value == ")" {
		if cur = ntdDNFGroupOpen(s, end); cur < 0 {
			return -1
		}
	} else if !isTypeNameToken(t) {
		return -1
	}
	for {
		pp := prevSignificantIndex(s, cur)
		if pp < 0 {
			return cur
		}
		pt := s.At(pp)
		switch {
		case pt.Kind == token.Punct && (pt.Value == "|" || pt.Value == "&" || pt.Value == "?" || pt.Value == `\`):
			cur = pp
		case pt.Kind == token.Punct && pt.Value == ")":
			open := ntdDNFGroupOpen(s, pp)
			if open < 0 {
				return cur
			}
			cur = open
		case isTypeNameToken(pt):
			cur = pp
		default:
			return cur
		}
	}
}

// ntdTypeName is the type's text without whitespace, as TypeAnalysis names it.
func ntdTypeName(s *tokens.Stream, start, end int) string {
	var b strings.Builder
	for k := start; k <= end; k++ {
		if t := s.At(k); t.Kind != token.Whitespace && t.Kind != token.Comment && t.Kind != token.DocComment {
			b.WriteString(t.Value)
		}
	}
	return b.String()
}

// ntdIsNullable is TypeAnalysis::isNullable: "?T", or "null" among the members
// of a union.
func ntdIsNullable(name string) bool {
	if strings.HasPrefix(name, "?") {
		return true
	}
	return slices.Contains(strings.Split(strings.ToLower(name), "|"), "null")
}

// ntdRemoveNull is the use_nullable_type_declaration=false half of
// fixUnionTypeParameter: drop "null" and its "|" from the union [start, end],
// then the parentheses of a DNF group left alone with its intersection.
func ntdRemoveNull(s *tokens.Stream, start, end int, name string) {
	k := -1
	for j := start; j <= end; j++ {
		if t := s.At(j); t.Kind == token.Ident && strings.EqualFold(t.Value, "null") {
			k = j
			break
		}
	}
	if k < 0 {
		return
	}
	lo, hi := k, k
	if k == start {
		if hi+1 < s.Len() && s.At(hi+1).Kind == token.Whitespace {
			hi++
		}
		if n := nextSignificantIndex(s, k); n >= 0 && n <= end && s.At(n).Value == "|" {
			hi = n
			if hi+1 < s.Len() && s.At(hi+1).Kind == token.Whitespace {
				hi++
			}
		}
	} else {
		if lo-1 >= start && s.At(lo-1).Kind == token.Whitespace {
			lo--
		}
		if p := prevSignificantIndex(s, k); p >= start && s.At(p).Value == "|" {
			lo = p
			if lo-1 >= start && s.At(lo-1).Kind == token.Whitespace {
				lo--
			}
		}
	}
	s.ReplaceRange(lo, hi, nil)
	end -= hi - lo + 1
	if !strings.Contains(name, "&") || strings.Count(name, "|") != 1 {
		return
	}
	// the closing parenthesis first, so the opening one's index stays valid
	for j := end; j >= start; j-- {
		if t := s.At(j); t.Kind == token.Punct && t.Value == ")" {
			s.RemoveAt(j)
			if j-1 >= start && s.At(j-1).Kind == token.Whitespace {
				s.RemoveAt(j - 1)
			}
			break
		}
	}
	for j := start; j <= end && j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.Punct && t.Value == "(" {
			s.RemoveAt(j)
			if j < s.Len() && s.At(j).Kind == token.Whitespace {
				s.RemoveAt(j)
			}
			break
		}
	}
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
		start := ntdTypeStart(s, p)
		if start < 0 {
			continue
		}
		name := ntdTypeName(s, start, p)
		if n := strings.ToLower(strings.TrimPrefix(name, "?")); n == "mixed" || n == "null" {
			continue
		}
		nullable := ntdIsNullable(name)
		union := strings.ContainsAny(name, "|&")
		if f.NoNullable {
			if b := prevSignificantIndex(s, start); b >= 0 && s.At(b).Kind == token.Keyword && isVisibilityModifier(s.At(b).Value) {
				continue
			}
			if !nullable {
				continue
			}
			if union {
				ntdRemoveNull(s, start, p, name)
			} else {
				s.RemoveAt(start)
				if start < s.Len() && s.At(start).Kind == token.Whitespace {
					s.RemoveAt(start)
				}
			}
			changed = true
			i = start
			continue
		}
		if nullable || (union && !f.UnionNull) {
			continue
		}
		if union {
			// an intersection alone becomes a DNF group: "A&B" -> "(A&B)|null"
			if !strings.Contains(name, "|") {
				s.InsertAt(p+1, token.Token{Kind: token.Punct, Value: ")"})
				p++
			}
			s.InsertAt(p+1, token.Token{Kind: token.Punct, Value: "|"})
			s.InsertAt(p+2, token.Token{Kind: token.Ident, Value: "null"})
			if !strings.Contains(name, "|") {
				s.InsertAt(start, token.Token{Kind: token.Punct, Value: "("})
			}
		} else {
			s.InsertAt(start, token.Token{Kind: token.Punct, Value: "?"})
		}
		changed = true
		i = start
	}
	return changed
}
