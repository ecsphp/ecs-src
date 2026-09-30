package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/MethodArgumentSpaceFixer.php
//
// MethodArgumentSpace normalizes single-line spacing around commas inside
// parentheses (calls and signatures): no space before a comma, exactly one
// space after it. Commas inside arrays "[...]", commas followed by a newline
// (multiline alignment) and a trailing comma right before ")" are left alone.
// The zero value reflows already-multiline lists fully (on_multiline default).
type MethodArgumentSpace struct {
	// onMultiline is "", "ensure_fully_multiline", "ensure_single_line",
	// "ensure_single_line_for_single_argument" or "ignore"; "" is the default.
	onMultiline                  string
	keepMultipleSpacesAfterComma bool
	// keepSpaceAfterHeredoc is set from after_heredoc=false; the zero value keeps
	// today's behaviour of always removing a space before a comma.
	keepSpaceAfterHeredoc bool
}

func (MethodArgumentSpace) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\MethodArgumentSpaceFixer`
}

func (MethodArgumentSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/MethodArgumentSpaceFixer.php"
}

func (f MethodArgumentSpace) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["on_multiline"].(string); ok {
		f.onMultiline = v
	}
	if v, ok := config["keep_multiple_spaces_after_comma"].(bool); ok {
		f.keepMultipleSpacesAfterComma = v
	}
	if v, ok := config["after_heredoc"].(bool); ok {
		f.keepSpaceAfterHeredoc = !v
	}
	// attribute_placement is accepted but not applied: blink folds attributes into
	// a single comment token, so it cannot reposition them.
	return f
}

func (f MethodArgumentSpace) Fix(s *tokens.Stream) bool {
	changed := false
	switch f.onMultiline {
	case "ignore":
		// leave multiline argument lists as they are
	case "ensure_single_line":
		if methodArgSpaceCollapse(s) {
			changed = true
		}
	default:
		// "", "ensure_fully_multiline" and "ensure_single_line_for_single_argument"
		// (approximated as fully multiline): a call/declaration argument list that
		// already spans lines gets one argument per line.
		if reflowMultilineArgs(s) {
			changed = true
		}
	}
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			stack = append(stack, t.Value)
			continue
		case ")", "]", "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		case ",":
			if len(stack) == 0 || stack[len(stack)-1] != "(" {
				continue
			}
			// Multiline arg list: leave alignment untouched.
			if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && hasNewline(s.At(i+1).Value) {
				continue
			}
			// No space before the comma (single-line only), unless after_heredoc is
			// off and a heredoc precedes the comma.
			if i > 0 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
				if !f.keepSpaceAfterHeredoc || !methodArgSpacePrevIsHeredoc(s, i-1) {
					s.RemoveAt(i - 1)
					i--
					changed = true
				}
			}
			// Exactly one space after the comma, except a trailing comma before ")".
			if i+1 < s.Len() {
				next := s.At(i + 1)
				if next.Kind == token.Whitespace {
					if next.Value != " " && !f.keepMultipleSpacesAfterComma {
						s.SetValue(i+1, " ")
						changed = true
					}
				} else if next.Kind != token.Punct || next.Value != ")" {
					s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
					i++
					changed = true
				}
			}
		}
	}
	return changed
}

// methodArgSpacePrevIsHeredoc reports whether the significant token before the
// whitespace at wsIdx is a heredoc/nowdoc literal.
func methodArgSpacePrevIsHeredoc(s *tokens.Stream, wsIdx int) bool {
	p := prevSignificantIndex(s, wsIdx)
	return p >= 0 && arrayNotationIsHeredoc(s.At(p))
}

// methodArgSpaceCollapse joins every already-multiline call/declaration argument
// list onto a single line. Only top-level newlines are removed; a newline kept
// inside a nested array or closure argument is left in place.
func methodArgSpaceCollapse(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !argListIsMultiline(s, open, closeIdx) {
			continue
		}
		if !isCallOrDeclParen(s, open) {
			continue
		}
		if methodArgSpaceCollapseParen(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// methodArgSpaceCollapseParen removes top-level newlines within the paren at open,
// dropping the whitespace next to "(" and ")" and collapsing the rest to a space.
func methodArgSpaceCollapseParen(s *tokens.Stream, open, closeIdx int) bool {
	changed := false
	depth := 0
	for j := closeIdx - 1; j > open; j-- {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case ")", "]", "}":
				depth++
			case "(", "[", "{":
				depth--
			}
			continue
		}
		if depth != 0 || t.Kind != token.Whitespace || !hasNewline(t.Value) {
			continue
		}
		if prevSignificantIndex(s, j) == open || nextSignificantIndex(s, j) == closeIdx {
			s.RemoveAt(j)
		} else {
			s.SetValue(j, " ")
		}
		changed = true
	}
	return changed
}

// reflowMultilineArgs makes every already-multiline call/declaration argument
// list fully multiline: "(" then each argument on its own line indented one
// level past the call, and ")" on its own line at the call's indentation.
func reflowMultilineArgs(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !argListIsMultiline(s, open, closeIdx) {
			continue
		}
		if !isCallOrDeclParen(s, open) {
			continue
		}
		if sigNext(s, open) == closeIdx {
			continue // empty ()
		}
		if reflowParen(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// reflowParen puts each top-level argument of the paren at open on its own line,
// with "(" and ")" on their own lines, indented one level past the call.
func reflowParen(s *tokens.Stream, open, closeIdx int) bool {
	changed := false
	base := lineIndentBefore(s, open)
	argNL := "\n" + base + "    "

	var commas []int
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			// skip a trailing comma (nothing but the closer follows it); decided
			// now, before edits shift indices
			if depth == 0 && sigNext(s, j) != closeIdx {
				commas = append(commas, j)
			}
		}
	}

	// apply right-to-left so indices stay valid: ")" first, then commas, then "("
	if editSlotBefore(s, closeIdx, "\n"+base) {
		changed = true
	}
	for _, c := range slices.Backward(commas) {
		if reflowAfterComma(s, c, base) {
			changed = true
		}
	}
	if editSlotAfter(s, open, argNL) {
		changed = true
	}
	return changed
}

// reflowAfterComma breaks a multiline argument list after a top-level comma. A
// trailing line comment ("arg, // note") stays on the argument's line and the
// break goes after the comment, matching php-cs-fixer; otherwise the break goes
// right after the comma.
func reflowAfterComma(s *tokens.Stream, comma int, base string) bool {
	n := comma + 1
	ws := -1
	if n < s.Len() && s.At(n).Kind == token.Whitespace && !hasNewline(s.At(n).Value) {
		ws = n
		n++
	}
	if n < s.Len() && isLineComment(s.At(n)) {
		changed := false
		if ws >= 0 {
			if s.At(ws).Value != " " {
				s.SetValue(ws, " ")
				changed = true
			}
		} else {
			s.InsertAt(comma+1, token.Token{Kind: token.Whitespace, Value: " "})
			n++
			changed = true
		}
		if editSlotAfter(s, n, "\n"+base+"    ") {
			changed = true
		}
		return changed
	}
	return editSlotAfter(s, comma, argNLAfterComma(s, comma, base))
}

// argListIsMultiline reports whether the argument list is split at the top level
// - a newline right after "(" or right after a top-level comma. Newlines that
// only occur inside a nested array/closure argument do not count, matching ECS's
// ensure_fully_multiline trigger.
func argListIsMultiline(s *tokens.Stream, open, closeIdx int) bool {
	if n := open + 1; n < closeIdx && s.At(n).Kind == token.Whitespace && hasNewline(s.At(n).Value) {
		return true
	}
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 && j+1 < closeIdx && s.At(j+1).Kind == token.Whitespace && hasNewline(s.At(j+1).Value) {
				return true
			}
		}
	}
	return false
}

// isCallOrDeclParen reports whether the "(" at open opens a function/method call
// or a function/closure declaration (not a control structure, array or grouping).
func isCallOrDeclParen(s *tokens.Stream, open int) bool {
	p := sigPrev(s, open)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Ident, token.Variable:
		return true
	case token.Punct:
		return t.Value == ")" || t.Value == "]"
	case token.Keyword:
		lv := strings.ToLower(t.Value)
		return lv == "function" || lv == "fn"
	}
	return false
}

// lineIndentBefore returns the indentation of the line containing token idx.
func lineIndentBefore(s *tokens.Stream, idx int) string {
	for i := idx - 1; i >= 0; i-- {
		if t := s.At(i); t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			if nl := strings.LastIndexByte(t.Value, '\n'); nl >= 0 {
				return t.Value[nl+1:]
			}
		}
	}
	return ""
}

// argNLAfterComma returns the newline+indent to place after a top-level comma
// in a multiline argument list. A blank line an author left between two
// arguments is preserved (ECS keeps blank lines in a multiline argument list and
// only normalizes the indentation), so a comma whose following whitespace holds a
// blank line keeps that blank rather than collapsing to a single newline.
func argNLAfterComma(s *tokens.Stream, comma int, base string) string {
	if comma+1 < s.Len() {
		ws := s.At(comma + 1)
		if ws.Kind == token.Whitespace {
			if newlines := strings.Count(ws.Value, "\n"); newlines >= 2 {
				return strings.Repeat("\n", newlines) + base + "    "
			}
		}
	}
	return "\n" + base + "    "
}

// editSlotAfter sets the whitespace immediately after token idx to val.
func editSlotAfter(s *tokens.Stream, idx int, val string) bool {
	if idx+1 < s.Len() && s.At(idx+1).Kind == token.Whitespace {
		if s.At(idx+1).Value != val {
			s.SetValue(idx+1, val)
			return true
		}
		return false
	}
	s.InsertAt(idx+1, token.Token{Kind: token.Whitespace, Value: val})
	return true
}

// editSlotBefore sets the whitespace immediately before token idx to val.
func editSlotBefore(s *tokens.Stream, idx int, val string) bool {
	if idx > 0 && s.At(idx-1).Kind == token.Whitespace {
		if s.At(idx-1).Value != val {
			s.SetValue(idx-1, val)
			return true
		}
		return false
	}
	s.InsertAt(idx, token.Token{Kind: token.Whitespace, Value: val})
	return true
}
