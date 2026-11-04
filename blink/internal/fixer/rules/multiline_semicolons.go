package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/MultilineWhitespaceBeforeSemicolonsFixer.php
//
// MultilineWhitespaceBeforeSemicolons removes multi-line whitespace before a
// semicolon (the default "no_multi_line" strategy): "$a = foo()\n    ;" becomes
// "$a = foo();". A "const" statement's own semicolon is left alone, and a
// comment sitting between the code and the ";" keeps the ";" attached to the
// code, moved ahead of the comment.
//
// With the "new_line_for_chained_calls" strategy, the ";" closing a multi-line
// call chain goes on its own line, indented like the chain's first line.
type MultilineWhitespaceBeforeSemicolons struct {
	newLineForChainedCalls bool
}

func (MultilineWhitespaceBeforeSemicolons) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\MultilineWhitespaceBeforeSemicolonsFixer`
}

func (MultilineWhitespaceBeforeSemicolons) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/MultilineWhitespaceBeforeSemicolonsFixer.php"
}

// WithConfig reads the "strategy" option ("no_multi_line" or
// "new_line_for_chained_calls").
func (f MultilineWhitespaceBeforeSemicolons) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["strategy"].(string); ok {
		f.newLineForChainedCalls = v == "new_line_for_chained_calls"
	}
	return f
}

// Fix follows applyFix index for index. PHP-CS-Fixer clears tokens instead of
// removing them and stops at the token count it started with; cleared counts
// the tokens blink removed, so that i+cleared is PHP-CS-Fixer's index.
func (f MultilineWhitespaceBeforeSemicolons) Fix(s *tokens.Stream) bool {
	changed := false
	count, cleared := s.Len(), 0
	for i := 0; i < s.Len() && i+cleared < count; i++ {
		t := s.At(i)
		if t.Kind == token.Keyword && strings.ToLower(t.Value) == "const" {
			// skip a const statement entirely, up to its terminating ";"
			for j := i + 1; j < s.Len(); j++ {
				if s.At(j).Kind == token.Punct && s.At(j).Value == ";" {
					i = j
					break
				}
			}
			continue
		}
		if t.Kind != token.Punct || t.Value != ";" || i == 0 {
			continue
		}
		if f.newLineForChainedCalls {
			if indent, ok := multilineSemicolonChainIndent(s, i); ok {
				if removed, ok := mswbsFixChainedCall(s, &i, indent); ok {
					cleared += removed
					changed = true
				}
				continue
			}
		}
		prevIdx := i - 1
		if s.At(prevIdx).Kind != token.Whitespace || !hasNewline(s.At(prevIdx).Value) {
			continue
		}
		// a comment between the code and the ";": keep the ";" with the code by
		// moving it ahead of the comment
		if i-2 >= 0 && isComment(s.At(i-2)) && startsWithNewline(s.At(prevIdx).Value) {
			sig := prevValueTokenIndex(s, i)
			s.RemoveAt(i)       // drop the ";"
			s.RemoveAt(prevIdx) // drop the whitespace
			s.InsertAt(sig+1, token.Token{Kind: token.Punct, Value: ";"})
			// the token after the old ";" is now at i; PHP-CS-Fixer's next index
			// is one past its two cleared tokens
			cleared += 2
			i--
			changed = true
			continue
		}
		// plain case: drop the multi-line whitespace so ";" follows the code
		s.RemoveAt(prevIdx)
		cleared++
		i--
		changed = true
	}
	return changed
}

func startsWithNewline(v string) bool {
	return strings.HasPrefix(v, "\n") || strings.HasPrefix(v, "\r")
}

// prevValueTokenIndex scans back from a ";" for the token the ";" belongs to,
// stopping at a number, name, variable, string or ")" (PHP-CS-Fixer's
// getPreviousSignificantTokenIndex).
func prevValueTokenIndex(s *tokens.Stream, i int) int {
	for j := i - 1; j > 0; j-- {
		t := s.At(j)
		if t.Kind == token.Number || t.Kind == token.Ident || t.Kind == token.Variable ||
			t.Kind == token.String || (t.Kind == token.Punct && t.Value == ")") {
			return j
		}
	}
	return i
}

// mswbsFixChainedCall moves the ";" at *i onto a new line indented by indent,
// after the end of the line it is on; *i is left on the moved ";". It returns
// how many tokens it removed.
func mswbsFixChainedCall(s *tokens.Stream, i *int, indent string) (int, bool) {
	idx := *i
	if p := s.At(idx - 1); p.Kind == token.Whitespace && p.Value == "\n"+indent {
		return 0, false
	}
	removed := 1
	if s.At(idx-1).Kind == token.Whitespace {
		s.RemoveAt(idx - 1)
		idx--
		removed++
	}
	s.RemoveAt(idx)
	// the line ending after the ";" (getNewLineIndex): the first whitespace or
	// comment holding a newline, scanning only whitespace and comments
	at := idx
	for at < s.Len() {
		t := s.At(at)
		if t.Kind != token.Whitespace && !isComment(t) {
			break
		}
		if strings.Contains(t.Value, "\n") {
			break
		}
		at++
	}
	s.InsertSliceAt(at, []token.Token{{Kind: token.Whitespace, Value: "\n" + indent}, {Kind: token.Punct, Value: ";"}})
	*i = at + 1
	return removed, true
}

// multilineSemicolonChainIndent is findWhitespaceBeforeFirstCall: when the
// statement ending at the ";" at semi is a call chain spanning lines, it returns
// the indent of the chain's first line.
func multilineSemicolonChainIndent(s *tokens.Stream, semi int) (string, bool) {
	multiline := false
	index := semi
	prev := sigPrev(s, index)
	for prev >= 0 {
		t := s.At(prev)
		if t.Kind == token.OpenTag || (t.Kind == token.Punct && (t.Value == ";" || t.Value == ":" || t.Value == "{" || t.Value == "}")) ||
			(t.Kind == token.Keyword && strings.EqualFold(t.Value, "else")) {
			break
		}
		index = prev
		prev = sigPrev(s, index)
		if t.Kind == token.Punct && (t.Value == ")" || t.Value == "]") || t.Kind == token.AttributeEnd {
			if o := s.MatchBackward(index); o >= 0 {
				prev = o
			}
			continue
		}
		if t.Kind == token.Punct && (t.Value == "->" || t.Value == "?->" || t.Value == "::") {
			prev = sigPrev(s, index)
			if prev >= 0 {
				for k := prev; k <= index; k++ {
					if strings.Contains(s.At(k).Value, "\n") {
						multiline = true
						break
					}
				}
			}
		}
	}
	if !multiline {
		return "", false
	}
	return detectIndent(s, index), true
}
