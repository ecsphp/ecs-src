package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// trailingCommaCategory classifies the block opened at open as one of
// "arrays", "array_destructuring", "arguments", "parameters", "match",
// "group_import", or "" when it is not a comma-list block.
func trailingCommaCategory(s *tokens.Stream, open int) string {
	switch s.At(open).Value {
	case "[":
		if isOffsetOpen(s, open) {
			return ""
		}
		if arrayNotationIsDestructuring(s, open) {
			return "array_destructuring"
		}
		return "arrays"
	case "{":
		p := sigPrev(s, open)
		if p < 0 {
			return ""
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == `\` {
			return "group_import"
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == ")" {
			if o := s.MatchBackward(p); o >= 0 {
				if q := sigPrev(s, o); q >= 0 && strings.EqualFold(s.At(q).Value, "match") {
					return "match"
				}
			}
		}
		return ""
	case "(":
		p := sigPrev(s, open)
		if p < 0 {
			return ""
		}
		prev := s.At(p)
		lower := strings.ToLower(prev.Value)
		switch {
		case (prev.Kind == token.Keyword || prev.Kind == token.Ident) && lower == "array":
			return "arrays"
		case (prev.Kind == token.Keyword || prev.Kind == token.Ident) && lower == "list":
			return "array_destructuring"
		case prev.Kind == token.Keyword && (lower == "function" || lower == "fn"):
			return "parameters"
		case prev.Kind == token.Ident:
			if pp := sigPrev(s, p); pp >= 0 && strings.EqualFold(s.At(pp).Value, "function") {
				return "parameters"
			}
			return "arguments"
		case prev.Kind == token.Variable:
			return "arguments"
		case prev.Kind == token.Keyword && (lower == "class" || lower == "static" || lower == "isset" || lower == "unset"):
			return "arguments"
		case prev.Kind == token.Punct && prev.Value == "]":
			return "arguments"
		}
	}
	return ""
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/TrailingCommaInMultilineFixer.php
//
// TrailingCommaInMultiline adds a trailing comma to the last element of a
// multi-line array literal (the fixer default: elements = ['arrays']). When
// Elements is set (elements option), the listed constructs are fixed instead.
// SkipAfterHeredoc (after_heredoc=false) leaves a heredoc end without a comma.
type TrailingCommaInMultiline struct {
	Elements         []string
	SkipAfterHeredoc bool
}

func (TrailingCommaInMultiline) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\TrailingCommaInMultilineFixer`
}

func (TrailingCommaInMultiline) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/TrailingCommaInMultilineFixer.php"
}

func (f TrailingCommaInMultiline) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["after_heredoc"].(bool); ok {
		f.SkipAfterHeredoc = !v
	}
	if v, ok := config["elements"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.Elements = append([]string{}, list...)
		}
	}
	return f
}

func (f TrailingCommaInMultiline) included(s *tokens.Stream, open, closeIdx int) bool {
	closer := s.At(closeIdx).Value
	if f.Elements == nil {
		return closer == "]" && !isOffsetOpen(s, open)
	}
	switch trailingCommaCategory(s, open) {
	case "arrays":
		return arrayNotationContains(f.Elements, "arrays")
	case "array_destructuring":
		return arrayNotationContains(f.Elements, "array_destructuring") ||
			(closer == ")" && arrayNotationContains(f.Elements, "arguments"))
	case "arguments":
		return arrayNotationContains(f.Elements, "arguments")
	case "parameters":
		return arrayNotationContains(f.Elements, "parameters")
	case "match":
		return arrayNotationContains(f.Elements, "match")
	}
	return false
}

func (f TrailingCommaInMultiline) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "]" && t.Value != ")" && t.Value != "}") {
			continue
		}
		if f.Elements == nil && t.Value != "]" {
			continue
		}
		open := s.MatchBackward(i)
		if open < 0 || !f.included(s, open, i) {
			continue
		}

		// last real element before the close (skip trailing whitespace/comments)
		p := i - 1
		for p > open && (s.At(p).Kind == token.Whitespace ||
			s.At(p).Kind == token.Comment || s.At(p).Kind == token.DocComment) {
			p--
		}
		if p <= open {
			continue // empty
		}
		if s.At(p).Value == "," || s.At(p).Value == "..." {
			continue // already trailing, or a spread (no trailing comma after it)
		}
		if f.SkipAfterHeredoc && arrayNotationIsHeredoc(s.At(p)) {
			continue
		}
		// multi-line only: a newline sits between the last element and the close
		multiline := false
		for k := p + 1; k < i; k++ {
			if s.At(k).Kind == token.Whitespace && hasNewline(s.At(k).Value) {
				multiline = true
				break
			}
		}
		if !multiline {
			continue
		}
		s.InsertAt(p+1, token.Token{Kind: token.Punct, Value: ","})
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoTrailingCommaInSinglelineFixer.php
//
// NoTrailingCommaInSingleline removes a trailing comma before a single-line
// closing ")", "]" or "}". When Elements is set (elements option), only the
// listed constructs are fixed.
type NoTrailingCommaInSingleline struct {
	Elements []string
}

func (NoTrailingCommaInSingleline) Name() string {
	return `PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`
}

func (NoTrailingCommaInSingleline) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoTrailingCommaInSinglelineFixer.php"
}

func (f NoTrailingCommaInSingleline) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["elements"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.Elements = append([]string{}, list...)
		}
	}
	return f
}

func (f NoTrailingCommaInSingleline) Fix(s *tokens.Stream) bool {
	if f.Elements != nil {
		return f.fixElements(s)
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "," {
			continue
		}
		j := i + 1
		for j < s.Len() && s.At(j).Kind == token.Whitespace {
			if hasNewline(s.At(j).Value) {
				j = -1
				break
			}
			j++
		}
		if j < 0 || j >= s.Len() {
			continue
		}
		if v := s.At(j).Value; v == ")" || v == "]" || v == "}" {
			s.RemoveAt(i)
			i--
			changed = true
		}
	}
	return changed
}

func (f NoTrailingCommaInSingleline) shouldClear(s *tokens.Stream, open int) bool {
	switch trailingCommaCategory(s, open) {
	case "arrays":
		return arrayNotationContains(f.Elements, "array")
	case "array_destructuring":
		return arrayNotationContains(f.Elements, "array_destructuring")
	case "group_import":
		return arrayNotationContains(f.Elements, "group_import")
	case "arguments", "parameters":
		if s.At(open).Value != "(" {
			return false
		}
		p := sigPrev(s, open)
		if p >= 0 && s.At(p).Kind == token.Keyword {
			if l := strings.ToLower(s.At(p).Value); l == "function" || l == "fn" {
				return false // closure parameter lists are not fixed
			}
		}
		return arrayNotationContains(f.Elements, "arguments")
	}
	// a call on a closing paren, e.g. "$f()(1,)"
	if s.At(open).Value == "(" {
		if p := sigPrev(s, open); p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == ")" {
			return arrayNotationContains(f.Elements, "arguments")
		}
	}
	return false
}

func (f NoTrailingCommaInSingleline) fixElements(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != ")" && t.Value != "]" && t.Value != "}") {
			continue
		}
		c := sigPrev(s, i)
		if c < 0 || s.At(c).Kind != token.Punct || s.At(c).Value != "," {
			continue
		}
		open := s.MatchBackward(i)
		if open < 0 {
			continue
		}
		multiline := false
		for k := open; k <= i; k++ {
			if strings.ContainsAny(s.At(k).Value, "\n\r") {
				multiline = true
				break
			}
		}
		if multiline || !f.shouldClear(s, open) {
			continue
		}
		for c >= 0 && s.At(c).Kind == token.Punct && s.At(c).Value == "," {
			s.RemoveAt(c)
			i--
			c = sigPrev(s, c)
		}
		for c >= 0 && c+1 < s.Len() && s.At(c+1).Kind == token.Whitespace {
			s.RemoveAt(c + 1)
			i--
		}
		changed = true
	}
	return changed
}
