package rules

import (
	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// isOffsetOpen reports whether the "[" at open is an array offset access (as
// opposed to an array literal), based on the preceding significant token.
func isOffsetOpen(s *tokens.Stream, open int) bool {
	prev, ok := prevSignificant(s, open)
	if !ok {
		return false
	}
	if prev.Kind == token.Variable || prev.Kind == token.Ident || prev.Kind == token.String {
		return true
	}
	return prev.Value == ")" || prev.Value == "]"
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoSpacesAroundOffsetFixer.php
//
// NoSpacesAroundOffset removes single-line spaces just inside an array offset
// ("$a[ 0 ]" -> "$a[0]"). Array literals are left alone. Outside also removes
// whitespace before the "[" and NoInside stops fixing the inside; both are set
// from the positions option.
type NoSpacesAroundOffset struct {
	Outside  bool
	NoInside bool
}

func (NoSpacesAroundOffset) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoSpacesAroundOffsetFixer`
}

func (NoSpacesAroundOffset) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoSpacesAroundOffsetFixer.php"
}

func (f NoSpacesAroundOffset) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["positions"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.Outside = arrayNotationContains(list, "outside")
			f.NoInside = !arrayNotationContains(list, "inside")
		}
	}
	return f
}

func (f NoSpacesAroundOffset) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "[" || !isOffsetOpen(s, i) {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		if !f.NoInside {
			if closeIdx-1 > i && s.At(closeIdx-1).Kind == token.Whitespace && !hasNewline(s.At(closeIdx-1).Value) {
				s.RemoveAt(closeIdx - 1)
				changed = true
			}
			if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
				s.RemoveAt(i + 1)
				changed = true
			}
		}
		if f.Outside && i > 0 && s.At(i-1).Kind == token.Whitespace {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}
