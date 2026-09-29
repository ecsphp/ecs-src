package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

var caseKeywords = map[string]bool{"case": true, "default": true}

// caseTerminator finds the ":" (or ";") that ends the case/default label at i,
// skipping ternary "?:" pairs and "::". It stops at a block brace for safety.
func caseTerminator(s *tokens.Stream, i int) (idx int, isSemicolon, ok bool) {
	ternary := 0
	for k := i + 1; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "?":
			ternary++
		case ":":
			if ternary > 0 {
				ternary--
			} else {
				return k, false, true
			}
		case ";":
			if ternary == 0 {
				return k, true, true
			}
		case "{", "}":
			return 0, false, false
		}
	}
	return 0, false, false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSemicolonToColonFixer.php
//
// SwitchCaseSemicolonToColon turns a "case 1;" terminator into "case 1:". Enum
// cases are left alone.
type SwitchCaseSemicolonToColon struct{}

func (SwitchCaseSemicolonToColon) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SwitchCaseSemicolonToColonFixer`
}

func (SwitchCaseSemicolonToColon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSemicolonToColonFixer.php"
}

func (SwitchCaseSemicolonToColon) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !caseKeywords[strings.ToLower(t.Value)] || inClassLikeBody(s, i) {
			continue
		}
		if idx, isSemicolon, ok := caseTerminator(s, i); ok && isSemicolon {
			s.SetValue(idx, ":")
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSpaceFixer.php
//
// SwitchCaseSpace removes whitespace between a case/default label and its colon
// ("case 1 :" -> "case 1:").
type SwitchCaseSpace struct{}

func (SwitchCaseSpace) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SwitchCaseSpaceFixer`
}

func (SwitchCaseSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSpaceFixer.php"
}

func (SwitchCaseSpace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !caseKeywords[strings.ToLower(t.Value)] || inClassLikeBody(s, i) {
			continue
		}
		idx, _, ok := caseTerminator(s, i)
		if !ok {
			continue
		}
		if idx > 0 && s.At(idx-1).Kind == token.Whitespace && !hasNewline(s.At(idx-1).Value) {
			s.RemoveAt(idx - 1)
			changed = true
		}
	}
	return changed
}
