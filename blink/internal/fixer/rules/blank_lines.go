package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

var threeOrMoreNewlines = regexp.MustCompile(`\n{3,}`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoExtraBlankLinesFixer.php
//
// NoExtraBlankLines collapses two or more consecutive blank lines into one. It
// runs after no_whitespace_in_blank_line, so blank lines are already bare. The
// `tokens` option selects which locations to strip blank lines from; the zero
// value reproduces PHP-CS-Fixer's default of ["extra"].
type NoExtraBlankLines struct {
	// tokens is the configured set of token kinds to act on; nil means the
	// default ["extra"].
	tokens map[string]bool
}

func (NoExtraBlankLines) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoExtraBlankLinesFixer`
}

func (NoExtraBlankLines) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoExtraBlankLinesFixer.php"
}

func (f NoExtraBlankLines) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["tokens"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			set := make(map[string]bool, len(list))
			for _, item := range list {
				set[item] = true
			}
			f.tokens = set
		}
	}
	return f
}

func (f NoExtraBlankLines) enabled(name string) bool {
	if f.tokens == nil {
		return name == "extra"
	}
	return f.tokens[name]
}

func (f NoExtraBlankLines) Fix(s *tokens.Stream) bool {
	changed := false
	// keyword-anchored removals: strip blank lines after the statement.
	for _, kw := range []string{"throw", "return", "break", "continue", "switch"} {
		if f.enabled(kw) && noExtraBlankLinesKeyword(s, kw) {
			changed = true
		}
	}
	if f.enabled("use") && noExtraBlankLinesUse(s, false) {
		changed = true
	}
	if f.enabled("use_trait") && noExtraBlankLinesUse(s, true) {
		changed = true
	}
	if f.enabled("extra") && noExtraBlankLinesExtra(s) {
		changed = true
	}
	return changed
}

// noExtraBlankLinesExtra collapses runs of blank lines to a single blank line.
func noExtraBlankLinesExtra(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Whitespace {
			continue
		}
		// merge any adjacent whitespace tokens (e.g. left by import removal) so
		// a run of blank lines lives in one token the regex can collapse
		for i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.SetValue(i, s.At(i).Value+s.At(i+1).Value)
			s.RemoveAt(i + 1)
		}
		if v := threeOrMoreNewlines.ReplaceAllString(s.At(i).Value, "\n\n"); v != s.At(i).Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// noExtraBlankLinesKeyword strips blank lines after a statement introduced by kw
// (throw honours the extra statement-boundary guard PHP-CS-Fixer applies).
func noExtraBlankLinesKeyword(s *tokens.Stream, kw string) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !strings.EqualFold(t.Value, kw) {
			continue
		}
		if kw == "throw" && !noExtraBlankLinesThrowPrevOK(s, i) {
			continue
		}
		if noExtraBlankLinesFixAfter(s, i) {
			changed = true
		}
	}
	return changed
}

// noExtraBlankLinesThrowPrevOK reports whether the token before a throw marks a
// statement start, so an expression throw ("$x = throw ...") is left untouched.
func noExtraBlankLinesThrowPrevOK(s *tokens.Stream, index int) bool {
	p := prevSignificantIndex(s, index)
	if p < 0 {
		return true
	}
	t := s.At(p)
	if t.Kind == token.OpenTag {
		return true
	}
	if t.Kind == token.Punct {
		switch t.Value {
		case ";", "{", "}", ":":
			return true
		}
	}
	return false
}

// noExtraBlankLinesFixAfter removes blank lines following the line that holds the
// token at index, unless the token sits inside a closure, anonymous class or
// array literal that opened on the same line.
func noExtraBlankLinesFixAfter(s *tokens.Stream, index int) bool {
	for i := index - 1; i > 0; i-- {
		t := s.At(i)
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "function") && functionBraceKind(s, i) == braceClosure {
			return false
		}
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "class") {
			if prev, ok := prevSignificant(s, i); ok && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
				return false
			}
		}
		if t.Kind == token.Punct && t.Value == "[" && !isOffsetOpen(s, i) {
			return false
		}
		if t.Kind == token.Whitespace && hasNewline(t.Value) {
			break
		}
	}
	return noExtraBlankLinesRemoveAfter(s, index)
}

// noExtraBlankLinesRemoveAfter finds the whitespace that ends the line at index
// (respecting parentheses) and collapses its leading blank lines to one newline,
// keeping the trailing indentation.
func noExtraBlankLinesRemoveAfter(s *tokens.Stream, index int) bool {
	depth := 0
	end := -1
	for i := index; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(":
				depth++
				continue
			case ")":
				depth--
				if depth < 0 {
					return false
				}
				continue
			case "}":
				end = i
			}
		}
		if end < 0 && t.Kind == token.Whitespace && hasNewline(t.Value) {
			end = i
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return false
	}
	changed := false
	for i := end; i < s.Len() && s.At(i).Kind == token.Whitespace; i++ {
		v := s.At(i).Value
		if !hasNewline(v) {
			continue
		}
		nl := strings.LastIndexByte(v, '\n')
		nv := "\n" + v[nl+1:]
		if nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

// noExtraBlankLinesUse collapses blank lines between consecutive import (or, with
// wantTrait, trait) use statements. Closure use lists are never touched.
func noExtraBlankLinesUse(s *tokens.Stream, wantTrait bool) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !strings.EqualFold(t.Value, "use") {
			continue
		}
		if noExtraBlankLinesIsClosureUse(s, i) {
			continue
		}
		if noExtraBlankLinesInClassBody(s, i) != wantTrait {
			continue
		}
		semi := noExtraBlankLinesUseSemicolon(s, i)
		if semi < 0 {
			continue
		}
		n := nextSignificantIndex(s, semi)
		if n < 0 || s.At(n).Kind != token.Keyword || !strings.EqualFold(s.At(n).Value, "use") {
			continue
		}
		if noExtraBlankLinesIsClosureUse(s, n) || noExtraBlankLinesInClassBody(s, n) != wantTrait {
			continue
		}
		if !rangeHasNewline(s, i, n) {
			continue
		}
		if noExtraBlankLinesRemoveAfter(s, semi) {
			changed = true
		}
	}
	return changed
}

// noExtraBlankLinesIsClosureUse reports whether the use at index opens a closure
// "use (...)" binding list rather than an import statement.
func noExtraBlankLinesIsClosureUse(s *tokens.Stream, index int) bool {
	n := nextSignificantIndex(s, index)
	return n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "("
}

// noExtraBlankLinesUseSemicolon returns the index of the ";" ending the use
// statement at index, or -1 if a close tag comes first.
func noExtraBlankLinesUseSemicolon(s *tokens.Stream, index int) int {
	for j := index + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.CloseTag {
			return -1
		}
		if t.Kind == token.Punct && t.Value == ";" {
			return j
		}
	}
	return -1
}

// noExtraBlankLinesInClassBody reports whether index sits directly inside a
// class-like body (so a use there is a trait import, not a namespace import).
func noExtraBlankLinesInClassBody(s *tokens.Stream, index int) bool {
	var stack []int
	for j := range index {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "{":
			stack = append(stack, j)
		case "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) == 0 {
		return false
	}
	kind, _ := classifyBrace(s, stack[len(stack)-1])
	return kind == braceClassLike
}
