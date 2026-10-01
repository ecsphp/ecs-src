package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ProtectedToPrivateFixer.php
//
// ProtectedToPrivate converts "protected" members to "private" in a final class
// (a final class cannot be extended, so protected is equivalent to private).
type ProtectedToPrivate struct{}

func (ProtectedToPrivate) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ProtectedToPrivateFixer`
}

func (ProtectedToPrivate) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ProtectedToPrivateFixer.php"
}

func (ProtectedToPrivate) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || !strings.EqualFold(s.At(i).Value, "class") {
			continue
		}
		prev, ok := prevSignificant(s, i)
		if !ok || prev.Kind != token.Keyword || !strings.EqualFold(prev.Value, "final") {
			continue
		}
		nameIdx := nextSignificantIndex(s, i)
		if nameIdx < 0 || s.At(nameIdx).Kind != token.Ident {
			continue
		}
		open := findBodyBrace(s, nameIdx)
		if open < 0 {
			continue
		}
		// a class extending a parent may inherit/override protected members;
		// ECS leaves those alone
		if classExtends(s, nameIdx, open) {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		// a class that uses a trait may expose protected members to it; ECS leaves
		// those alone
		if classUsesTrait(s, open, closeIdx) {
			continue
		}
		depth := 0
		for k := open; k < closeIdx; k++ {
			if s.At(k).Kind == token.Punct {
				switch s.At(k).Value {
				case "{":
					depth++
				case "}":
					depth--
				}
				continue
			}
			// only this class's own members (depth 1), not a nested class body,
			// and not a constant named "protected" (lexed as a keyword)
			if depth == 1 && s.At(k).Kind == token.Keyword && strings.EqualFold(s.At(k).Value, "protected") && !isConstNamePosition(s, k) {
				s.SetValue(k, "private")
				changed = true
			}
		}
	}
	return changed
}

func classExtends(s *tokens.Stream, nameIdx, open int) bool {
	for k := nameIdx + 1; k < open; k++ {
		if s.At(k).Kind == token.Keyword && strings.EqualFold(s.At(k).Value, "extends") {
			return true
		}
	}
	return false
}

// isConstNamePosition reports whether the token at k is the name (or type) of a
// "const" declaration, e.g. the "PROTECTED" in "const int PROTECTED", which the
// lexer classifies as a keyword but is not a visibility modifier.
func isConstNamePosition(s *tokens.Stream, k int) bool {
	j := prevSignificantIndex(s, k)
	for j >= 0 {
		t := s.At(j)
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "const") {
			return true
		}
		if t.Kind == token.Ident ||
			(t.Kind == token.Keyword && !strings.EqualFold(t.Value, "const")) ||
			(t.Kind == token.Punct && (t.Value == "?" || t.Value == `\`)) {
			// a type token before the name; keep walking back
			if t.Kind == token.Keyword && isModifierKeyword(t.Value) {
				return false
			}
			j = prevSignificantIndex(s, j)
			continue
		}
		return false
	}
	return false
}

func isModifierKeyword(v string) bool {
	switch strings.ToLower(v) {
	case "public", "private", "protected", "static", "final", "abstract", "readonly", "var":
		return true
	}
	return false
}

// classUsesTrait reports whether the class body (open..close) contains a
// top-level "use Trait;" import, which makes protected members reachable.
func classUsesTrait(s *tokens.Stream, open, close int) bool {
	depth := 0
	for k := open; k < close; k++ {
		t := s.At(k)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				depth++
			case "}":
				depth--
			}
			continue
		}
		if depth == 1 && t.Kind == token.Keyword && strings.EqualFold(t.Value, "use") {
			return true
		}
	}
	return false
}
