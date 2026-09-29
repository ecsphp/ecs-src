package rules

import (
	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareEqualNormalizeFixer.php
//
// DeclareEqualNormalize removes the spaces around "=" inside a declare header:
// "declare(strict_types = 1)" -> "declare(strict_types=1)". With Single set
// (space: single) exactly one space is forced around it instead.
type DeclareEqualNormalize struct {
	Single bool
}

func (DeclareEqualNormalize) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\DeclareEqualNormalizeFixer`
}

func (DeclareEqualNormalize) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareEqualNormalizeFixer.php"
}

func (f DeclareEqualNormalize) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["space"].(string); ok {
		f.Single = v == "single"
	}
	return f
}

func (f DeclareEqualNormalize) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "=" || !insideDeclareArgs(s, i) {
			continue
		}
		if f.Single {
			if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
				if s.At(i+1).Value != " " {
					s.SetValue(i+1, " ")
					changed = true
				}
			} else {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			}
			if i >= 1 && s.At(i-1).Kind == token.Whitespace {
				if s.At(i-1).Value != " " {
					s.SetValue(i-1, " ")
					changed = true
				}
			} else {
				s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
				i++
				changed = true
			}
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.RemoveAt(i + 1)
			changed = true
		}
		if i >= 1 && s.At(i-1).Kind == token.Whitespace {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}
