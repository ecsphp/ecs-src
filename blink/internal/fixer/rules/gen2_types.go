package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// isTypeUnionPart reports whether t can appear inside a type union/intersection
// expression around a "|"/"&" operator (a type name, "?", "\", or the operators
// themselves). Native types and class names both lex as Ident.
func isTypeUnionPart(t token.Token) bool {
	if isTypeNameToken(t) {
		return true
	}
	if t.Kind == token.Punct {
		switch t.Value {
		case "?", `\`, "|", "&":
			return true
		}
	}
	return false
}

// isVisibilityModifier reports whether v is a property visibility/modifier keyword
// that can precede a typed property or a promoted constructor parameter.
func isVisibilityModifier(v string) bool {
	switch strings.ToLower(v) {
	case "public", "private", "protected", "readonly", "static", "var":
		return true
	}
	return false
}

// typeRunBoundaryPrev returns the first significant token index to the left of a
// type union that is not itself a type part (the token the type starts after), or
// -1. It begins the walk from the operator at i.
func typeRunBoundaryPrev(s *tokens.Stream, i int) int {
	j := i
	for {
		p := prevSignificantIndex(s, j)
		if p < 0 {
			return -1
		}
		if isTypeUnionPart(s.At(p)) {
			j = p
			continue
		}
		return p
	}
}

// typeRunBoundaryNext returns the first significant token index to the right of a
// type union that is not itself a type part (the token the type ends before), or
// -1. It begins the walk from the operator at i.
func typeRunBoundaryNext(s *tokens.Stream, i int) int {
	j := i
	for {
		n := nextSignificantIndex(s, j)
		if n < 0 {
			return -1
		}
		if isTypeUnionPart(s.At(n)) {
			j = n
			continue
		}
		return n
	}
}

// isTypeUnionOperator reports whether the "|"/"&" at i separates type names inside
// a type declaration (union/intersection) rather than acting as a bitwise operator.
// It confirms one of three safe positions: a return type, a function parameter
// type, or a typed property; otherwise it returns false so bitwise operators and
// unconfirmed positions are never touched.
func isTypeUnionOperator(s *tokens.Stream, i int) bool {
	// left neighbour must end a type name; right neighbour must start one
	p := prevSignificantIndex(s, i)
	n := nextSignificantIndex(s, i)
	if p < 0 || n < 0 {
		return false
	}
	if !isTypeNameToken(s.At(p)) {
		return false
	}
	nt := s.At(n)
	if !isTypeNameToken(nt) && (nt.Kind != token.Punct || (nt.Value != "?" && nt.Value != `\`)) {
		return false
	}
	// return type: walking left across type parts reaches a return-type colon
	if inReturnType(s, i) {
		return true
	}
	// parameter or property type: the type run must end at the declared "$variable"
	end := typeRunBoundaryNext(s, i)
	if end < 0 || s.At(end).Kind != token.Variable {
		return false
	}
	start := typeRunBoundaryPrev(s, i)
	if start < 0 {
		return false
	}
	st := s.At(start)
	// parameter type: inside a function parameter list, starting after "(", "," or
	// a visibility modifier (promoted constructor property)
	if enclosingFuncParamOpen(s, i) >= 0 {
		if st.Kind == token.Punct && (st.Value == "(" || st.Value == ",") {
			return true
		}
		return st.Kind == token.Keyword && isVisibilityModifier(st.Value)
	}
	// typed property: the type run starts after a visibility/modifier keyword
	return st.Kind == token.Keyword && isVisibilityModifier(st.Value)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/TypesSpacesFixer.php
//
// TypesSpaces removes the single-line whitespace around a "|"/"&" that joins type
// names in a union/intersection ("int | string" -> "int|string", "A & B" ->
// "A&B"). It acts only in confirmed type positions (parameter, return, typed
// property) and never on a bitwise "|"/"&". Line breaks are preserved.
//
// Options: Single (space: single) forces one space around the operator instead.
// Once WithConfig is called, multi-catch types ("catch (A | B $e)") are handled
// too, using CatchSingle (space_multiple_catch) or, when unset, Single.
type TypesSpaces struct {
	Single      bool
	CatchActive bool
	CatchSingle bool
}

func (TypesSpaces) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\TypesSpacesFixer`
}

func (TypesSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/TypesSpacesFixer.php"
}

func (f TypesSpaces) WithConfig(config map[string]any) fixer.Fixer {
	f.CatchActive = true
	if v, ok := config["space"].(string); ok {
		f.Single = v == "single"
	}
	f.CatchSingle = f.Single
	if v, ok := config["space_multiple_catch"].(string); ok {
		f.CatchSingle = v == "single"
	}
	return f
}

func (f TypesSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if f.CatchActive && t.Kind == token.Keyword && strings.EqualFold(t.Value, "catch") {
			if ch, end := typesSpacesCatch(s, i, f.CatchSingle); end > i {
				changed = changed || ch
				i = end
				continue
			}
		}
		if t.Kind != token.Punct || (t.Value != "|" && t.Value != "&") {
			continue
		}
		if !isTypeUnionOperator(s, i) {
			continue
		}
		var ch bool
		i, ch = typesSpacesFix(s, i, f.Single)
		changed = changed || ch
	}
	return changed
}

// typesSpacesFix spaces the operator at i and returns its new index.
func typesSpacesFix(s *tokens.Stream, i int, single bool) (int, bool) {
	changed := false
	if i+1 < s.Len() {
		next := s.At(i + 1)
		switch {
		case next.Kind == token.Whitespace && !hasNewline(next.Value):
			if !single {
				s.RemoveAt(i + 1)
				changed = true
			} else if next.Value != " " {
				s.SetValue(i+1, " ")
				changed = true
			}
		case next.Kind != token.Whitespace && single:
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
	}
	if i >= 1 {
		prev := s.At(i - 1)
		switch {
		case prev.Kind == token.Whitespace && !hasNewline(prev.Value):
			if !single {
				s.RemoveAt(i - 1)
				i--
				changed = true
			} else if prev.Value != " " {
				s.SetValue(i-1, " ")
				changed = true
			}
		case prev.Kind != token.Whitespace && single:
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
			i++
			changed = true
		}
	}
	return i, changed
}

// typesSpacesCatch spaces every "|" directly inside the parentheses of the catch
// at i and returns whether anything changed and the index of the closing ")".
func typesSpacesCatch(s *tokens.Stream, i int, single bool) (bool, int) {
	open := nextSignificantIndex(s, i)
	if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return false, i
	}
	changed := false
	for j := open + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			return changed, j
		case "|":
			var ch bool
			j, ch = typesSpacesFix(s, j, single)
			changed = changed || ch
		}
	}
	return changed, i
}
