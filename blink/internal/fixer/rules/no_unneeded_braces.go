package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededCurlyBracesFixer.php
//
// NoUnneededBraces removes superfluous braces that are not part of a control
// structure body: standalone "{ ... }" blocks and single-element group imports
// ("use A\{B};" -> "use A\B;"). The namespaces option is off by default, so
// bracketed namespaces are left untouched. Named NoUnneededCurlyBracesFixer to
// match the set membership (the rule's former name).
type NoUnneededBraces struct{}

func (NoUnneededBraces) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`
}

func (NoUnneededBraces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededCurlyBracesFixer.php"
}

func (NoUnneededBraces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i > 0; i-- {
		if !nubIsPunct(s.At(i), "{") {
			continue
		}
		prev := nubPrevMeaningful(s, i)
		if prev == -1 {
			continue
		}
		if nubIsPunct(s.At(prev), `\`) {
			// group import: "use A\{B};" -> "use A\B;" when single element, one line
			if nubGroupImportOverComplete(s, i) {
				closeIndex := s.MatchForward(i)
				if closeIndex != -1 {
					s.RemoveAt(closeIndex)
					s.RemoveAt(i)
					changed = true
				}
			}
			continue
		}
		if nubRegularOverComplete(s, prev) {
			closeIndex := s.MatchForward(i)
			if closeIndex != -1 {
				s.RemoveAt(closeIndex)
				s.RemoveAt(i)
				changed = true
			}
		}
	}
	return changed
}

// nubRegularOverComplete reports whether a "{" whose previous meaningful token is
// at prev opens a superfluous block (prev is "{", "}", "<?php", ":" or ";").
func nubRegularOverComplete(s *tokens.Stream, prev int) bool {
	t := s.At(prev)
	if t.Kind == token.OpenTag {
		return true
	}
	return t.Kind == token.Punct && (t.Value == "{" || t.Value == "}" || t.Value == ":" || t.Value == ";")
}

// nubGroupImportOverComplete reports whether the group-import brace at openIndex
// wraps a single element on a single line.
func nubGroupImportOverComplete(s *tokens.Stream, openIndex int) bool {
	closeIndex := s.MatchForward(openIndex)
	if closeIndex == -1 {
		return false
	}
	for j := openIndex + 1; j < closeIndex; j++ {
		t := s.At(j)
		if t.Kind == token.Punct && t.Value == "," {
			return false
		}
		if t.Kind == token.Whitespace && strings.ContainsAny(t.Value, "\r\n") {
			return false
		}
	}
	return true
}

func nubIsPunct(t token.Token, v string) bool {
	return t.Kind == token.Punct && t.Value == v
}

func nubPrevMeaningful(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		k := s.At(j).Kind
		if k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		return j
	}
	return -1
}
