package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededBracesFixer.php
//
// NoUnneededBraces removes superfluous braces that are not part of a control
// structure body: standalone "{ ... }" blocks and single-element group imports
// ("use A\{B};" -> "use A\B;"). The namespaces option is off by default, so
// bracketed namespaces are left untouched. The former name NoUnneededCurlyBracesFixer
// is kept as a deprecated alias (see deprecatedAliases in registry.go).
type NoUnneededBraces struct {
	Namespaces bool
}

func (f NoUnneededBraces) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["namespaces"].(bool); ok {
		f.Namespaces = v
	}
	return f
}

func (NoUnneededBraces) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUnneededBracesFixer`
}

func (NoUnneededBraces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededBracesFixer.php"
}

func (f NoUnneededBraces) Fix(s *tokens.Stream) bool {
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
	if f.Namespaces && nubFixNamespaceBlock(s) {
		changed = true
	}
	return changed
}

// nubFixNamespaceBlock turns a lone "namespace Foo { ... }" spanning the rest of
// the file into "namespace Foo; ...".
func nubFixNamespaceBlock(s *tokens.Stream) bool {
	nsIndex := -1
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if (t.Kind == token.Keyword || t.Kind == token.Ident) && strings.EqualFold(t.Value, "namespace") {
			if nsIndex != -1 {
				return false
			}
			nsIndex = i
		}
	}
	if nsIndex == -1 {
		return false
	}
	named := false
	idx := nubNextMeaningful(s, nsIndex)
	for idx != -1 && (s.At(idx).Kind == token.Ident || nubIsPunct(s.At(idx), `\`)) {
		named = true
		idx = nubNextMeaningful(s, idx)
	}
	if !named || idx == -1 || !nubIsPunct(s.At(idx), "{") {
		return false
	}
	closeIdx := s.MatchForward(idx)
	if closeIdx == -1 {
		return false
	}
	after := nubNextMeaningful(s, closeIdx)
	if after != -1 && (s.At(after).Kind != token.CloseTag || nubNextMeaningful(s, after) != -1) {
		return false
	}
	s.RemoveAt(closeIdx)
	if closeIdx < s.Len() && closeIdx > 0 && s.At(closeIdx).Kind == token.Whitespace && s.At(closeIdx-1).Kind == token.Whitespace {
		s.SetValue(closeIdx-1, s.At(closeIdx-1).Value+s.At(closeIdx).Value)
		s.RemoveAt(closeIdx)
	}
	s.Set(idx, token.Token{Kind: token.Punct, Value: ";"})
	if idx > 1 && s.At(idx-1).Kind == token.Whitespace && strings.Trim(s.At(idx-1).Value, " \t") == "" && s.At(idx-2).Kind != token.Comment && s.At(idx-2).Kind != token.DocComment {
		s.RemoveAt(idx - 1)
	}
	return true
}

func nubNextMeaningful(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		k := s.At(j).Kind
		if k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		return j
	}
	return -1
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
