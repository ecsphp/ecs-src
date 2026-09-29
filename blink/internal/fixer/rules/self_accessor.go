package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfAccessorFixer.php
//
// SelfAccessor replaces a reference to the containing class's own name with
// "self" inside the class body: "new Foo()" -> "new self()",
// "Foo::CONST" -> "self::CONST", type declarations "Foo $x" -> "self $x".
type SelfAccessor struct{}

func (SelfAccessor) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\SelfAccessorFixer`
}

func (SelfAccessor) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfAccessorFixer.php"
}

func (SelfAccessor) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || !isClassLikeKeyword(strings.ToLower(s.At(i).Value)) {
			continue
		}
		// skip "new class" (anonymous) and "::class"
		if prev, ok := prevSignificant(s, i); ok {
			if prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
				continue
			}
			if prev.Kind == token.Punct && (prev.Value == "->" || prev.Value == "?->" || prev.Value == "::") {
				continue
			}
		}
		nameIdx := nextSignificantIndex(s, i)
		if nameIdx < 0 || s.At(nameIdx).Kind != token.Ident {
			continue
		}
		className := s.At(nameIdx).Value
		open := findBodyBrace(s, nameIdx)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		for k := open + 1; k < closeIdx; k++ {
			if s.At(k).Kind != token.Ident || s.At(k).Value != className {
				continue
			}
			if selfAccessorRef(s, k) {
				s.SetValue(k, "self")
				changed = true
			}
		}
	}
	return changed
}

func isClassLikeKeyword(lw string) bool {
	switch lw {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

// findBodyBrace returns the index of the "{" opening a class body declared at
// nameIdx, scanning past any "extends"/"implements" clause. Returns -1 if none.
func findBodyBrace(s *tokens.Stream, nameIdx int) int {
	for k := nameIdx + 1; k < s.Len(); k++ {
		if s.At(k).Kind == token.Punct {
			switch s.At(k).Value {
			case "{":
				return k
			case ";":
				return -1
			}
		}
	}
	return -1
}

// selfAccessorRef reports whether the class-name token at k is a reference to the
// class (a "new"/"instanceof" operand, a "::" access, or a type declaration)
// rather than a name in another position.
func selfAccessorRef(s *tokens.Stream, k int) bool {
	prev, hasPrev := prevSignificant(s, k)
	nextIdx := nextSignificantIndex(s, k)

	if hasPrev {
		if prev.Kind == token.Punct {
			switch prev.Value {
			case "->", "?->", "::", `\`:
				return false
			}
		}
		if prev.Kind == token.Keyword {
			switch strings.ToLower(prev.Value) {
			case "function", "const", "extends", "implements", "use", "namespace", "as":
				return false
			case "new", "instanceof":
				return true
			}
		}
	}
	if nextIdx >= 0 && s.At(nextIdx).Kind == token.Punct && s.At(nextIdx).Value == `\` {
		return false // heads a namespaced name
	}
	if nextIdx >= 0 && s.At(nextIdx).Kind == token.Punct && s.At(nextIdx).Value == "::" {
		return true
	}
	// a type declaration only counts inside a parameter list or a return type;
	// property types are left untouched by ECS
	if enclosingFuncParamOpen(s, k) >= 0 {
		return true
	}
	return selfAccessorInReturnType(s, k)
}

// selfAccessorInReturnType reports whether the type token at k sits in a return
// type, walking back over "?", "|", "&", "\" to a return-type colon.
func selfAccessorInReturnType(s *tokens.Stream, k int) bool {
	j := k
	for {
		p := prevSignificantIndex(s, j)
		if p < 0 {
			return false
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == ":" {
			return isReturnTypeColon(s, p)
		}
		if s.At(p).Kind == token.Ident ||
			(s.At(p).Kind == token.Punct && (s.At(p).Value == "?" || s.At(p).Value == "|" || s.At(p).Value == "&" || s.At(p).Value == `\`)) {
			j = p
			continue
		}
		return false
	}
}
