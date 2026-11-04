package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php
//
// SpaceAfterSemicolon puts a single space after a semicolon followed by code:
// "$a=1;$b=2;" becomes "$a=1; $b=2;", and a run of spaces or tabs becomes one
// space unless a comment or ")" follows. A ";" before ")", a newline or inline
// HTML is left alone. With RemoveInEmptyFor set, a ";" directly followed by
// ";" gains no space, and the single-line whitespace inside an empty for
// expression ("for ($i = 0; ; ++$i)", "for (;; )") is removed.
type SpaceAfterSemicolon struct {
	RemoveInEmptyFor bool
}

func (SpaceAfterSemicolon) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`
}

func (SpaceAfterSemicolon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php"
}

func (f SpaceAfterSemicolon) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["remove_in_empty_for_expressions"].(bool); ok {
		f.RemoveInEmptyFor = v
	}
	return f
}

func (f SpaceAfterSemicolon) Fix(s *tokens.Stream) bool {
	changed := false
	// The PHP fixer clears tokens in place and remembers the for header's ")"
	// by index without adjusting it after an insertion; clearing to empty
	// placeholders (compacted at the end) keeps the same index arithmetic.
	insideForUntil := -1
	for i := 0; i < s.Len()-1; i++ {
		if f.RemoveInEmptyFor {
			if kwIs(s.At(i), "for") {
				if i = sigNext(s, i); i < 0 {
					break
				}
				insideForUntil = s.MatchForward(i)
				continue
			}
			if i == insideForUntil {
				insideForUntil = -1
				continue
			}
		}
		if !isPunctVal(s, i, ";") {
			continue
		}
		next := s.At(i + 1)
		if next.Kind != token.Whitespace {
			if !isPunctVal(s, i+1, ")") && next.Kind != token.InlineHTML &&
				(!f.RemoveInEmptyFor || !isPunctVal(s, i+1, ";")) {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			}
			continue
		}
		if insideForUntil >= 0 && (isPunctVal(s, i+2, ";") || i+2 == insideForUntil) && !hasNewline(next.Value) {
			clearTokenAt(s, i+1)
			changed = true
			continue
		}
		if i+2 < s.Len() && next.Value != " " && strings.Trim(next.Value, " \t") == "" {
			if k := s.At(i + 2).Kind; k != token.Comment && k != token.DocComment && !isPunctVal(s, i+2, ")") {
				s.SetValue(i+1, " ")
				changed = true
			}
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}
