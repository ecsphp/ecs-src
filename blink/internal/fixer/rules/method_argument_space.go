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
	// pullCommaAfterHeredoc is set only by an explicit after_heredoc=true, which
	// pulls a comma on its own line up onto the heredoc-closing line.
	pullCommaAfterHeredoc bool
	// attributePlacement is "standalone" (each attribute and the parameter on its
	// own line) or "same_line"; "" resolves to the php-cs-fixer default "standalone".
	attributePlacement string
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
		f.pullCommaAfterHeredoc = v
	}
	if v, ok := config["attribute_placement"].(string); ok {
		f.attributePlacement = v
	}
	return f
}

// resolvedAttributePlacement returns the effective attribute_placement, defaulting
// to php-cs-fixer's "standalone".
func (f MethodArgumentSpace) resolvedAttributePlacement() string {
	if f.attributePlacement == "" {
		return "standalone"
	}
	return f.attributePlacement
}

func (f MethodArgumentSpace) Fix(s *tokens.Stream) bool {
	changed := false
	switch f.onMultiline {
	case "ignore":
		// leave multiline argument lists as they are
	case "ensure_single_line":
		if methodArgSpaceCollapse(s, f.keepMultipleSpacesAfterComma) {
			changed = true
		}
		if collapseAttributeArgs(s, false) {
			changed = true
		}
	case "ensure_single_line_for_single_argument":
		// a multiline call with a single argument collapses to one line (unless the
		// argument itself is multiline); with several arguments it goes fully multiline
		if reflowSingleArgOrMultiline(s) {
			changed = true
		}
		if collapseAttributeArgs(s, true) {
			changed = true
		}
		if applyAttributePlacement(s, f.resolvedAttributePlacement()) {
			changed = true
		}
	default:
		// "" and "ensure_fully_multiline": a call/declaration argument list that
		// already spans lines gets one argument per line.
		if reflowMultilineArgs(s) {
			changed = true
		}
		if applyAttributePlacement(s, f.resolvedAttributePlacement()) {
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
		case "(":
			// only a function/method call or declaration paren has its commas spaced;
			// "array(...)" and grouping parens are left alone (marked "a")
			if isCommaSpacedParen(s, i) {
				stack = append(stack, "(")
			} else {
				stack = append(stack, "a")
			}
			continue
		case "[", "{":
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
			// No space before the comma, unless the previous token is a comment or
			// comma, or after_heredoc is off and a heredoc precedes it.
			if i > 0 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
				p := prevSignificantIndex(s, i)
				prevComment := p >= 0 && (s.At(p).Kind == token.Comment || s.At(p).Kind == token.DocComment)
				prevComma := p >= 0 && s.At(p).Value == ","
				if !prevComment && !prevComma && (!f.keepSpaceAfterHeredoc || !methodArgSpacePrevIsHeredoc(s, i-1)) {
					s.RemoveAt(i - 1)
					i--
					changed = true
				}
			} else if f.pullCommaAfterHeredoc && i > 0 && s.At(i-1).Kind == token.Whitespace &&
				hasNewline(s.At(i-1).Value) && methodArgSpacePrevIsHeredoc(s, i-1) {
				// after_heredoc=true pulls the comma onto the heredoc-closing line
				s.RemoveAt(i - 1)
				i--
				changed = true
			}
			// After the comma: a newline keeps its multiline alignment; otherwise
			// exactly one space, including a trailing comma before ")" (php-cs-fixer
			// adds it), except before a comment that ends its line.
			if i+1 < s.Len() {
				next := s.At(i + 1)
				if next.Kind == token.Whitespace {
					if !hasNewline(next.Value) && next.Value != " " && !f.keepMultipleSpacesAfterComma && !masCommentLastLine(s, i+2) {
						s.SetValue(i+1, " ")
						changed = true
					}
				} else if !masCommentLastLine(s, i+1) {
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
func methodArgSpaceCollapse(s *tokens.Stream, keepMultiple bool) bool {
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
		if methodArgSpaceCollapseParen(s, open, closeIdx, keepMultiple) {
			changed = true
		}
	}
	return changed
}

// methodArgSpaceCollapseParen removes top-level newlines within the paren at open,
// dropping the whitespace next to "(" and ")" and collapsing the rest to a space.
func methodArgSpaceCollapseParen(s *tokens.Stream, open, closeIdx int, keepMultiple bool) bool {
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
		prev := prevSignificantIndex(s, j)
		if prev == open || nextSignificantIndex(s, j) == closeIdx {
			s.RemoveAt(j)
		} else if keepMultiple && prev >= 0 && s.At(prev).Value == "," {
			// keep_multiple_spaces_after_comma: drop the newline but keep the indent
			if nl := strings.LastIndexByte(t.Value, '\n'); nl >= 0 {
				s.SetValue(j, t.Value[nl+1:])
			}
		} else {
			s.SetValue(j, " ")
		}
		changed = true
	}
	return changed
}

// reflowSingleArgOrMultiline handles on_multiline=ensure_single_line_for_single_argument:
// a multiline call with one argument collapses to a single line, one with several
// goes fully multiline.
func reflowSingleArgOrMultiline(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !argListIsMultiline(s, open, closeIdx) || !isCallOrDeclParen(s, open) {
			continue
		}
		if sigNext(s, open) == closeIdx {
			continue // empty ()
		}
		if countTopLevelArgs(s, open, closeIdx) == 1 {
			if ensureSingleLineForParen(s, open, closeIdx) {
				changed = true
			}
		} else if reflowParen(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// countTopLevelArgs counts the arguments between open and close, ignoring a
// trailing comma and nested brackets.
func countTopLevelArgs(s *tokens.Stream, open, closeIdx int) int {
	if sigNext(s, open) == closeIdx {
		return 0
	}
	count := 1
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
			if depth == 0 && sigNext(s, j) != closeIdx {
				count++
			}
		}
	}
	return count
}

// argumentContentIsMultiline reports whether the single argument between open and
// close spans multiple lines at the top level (a comment, or a newline that is not
// the edge whitespace right after "(" or before ")"), skipping nested brackets.
func argumentContentIsMultiline(s *tokens.Stream, open, closeIdx int) bool {
	for i := open + 1; i < closeIdx; i++ {
		t := s.At(i)
		if (t.Kind == token.Comment || t.Kind == token.DocComment) && !isAttributeComment(t) {
			return true
		}
		if i == open+1 || i == closeIdx-1 {
			continue
		}
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
			if c := s.MatchForward(i); c > 0 {
				i = c
			}
			continue
		}
		if t.Kind == token.Whitespace && hasNewline(t.Value) {
			return true
		}
	}
	return false
}

// ensureSingleLineForParen collapses the outer newlines of a single-argument
// call to one line, leaving a multiline argument (or one carrying a line comment)
// untouched.
func ensureSingleLineForParen(s *tokens.Stream, open, closeIdx int) bool {
	if argumentContentIsMultiline(s, open, closeIdx) {
		return false
	}
	changed := false
	for i := closeIdx - 1; i > open; i-- {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == ")" || t.Value == "]" || t.Value == "}") {
			if c := s.MatchBackward(i); c >= 0 {
				i = c
			}
			continue
		}
		if t.Kind == token.Whitespace {
			if i > 0 {
				prev := s.At(i - 1)
				if prev.Kind == token.Comment && !strings.HasPrefix(prev.Value, "/*") {
					continue
				}
			}
			if nv := collapseNewlineHspace(t.Value); nv != t.Value {
				s.SetValue(i, nv)
				changed = true
			}
		}
	}
	return changed
}

// collapseNewlineHspace removes every line break and the horizontal whitespace
// that follows it, matching php-cs-fixer's /\R\h*/ replacement.
func collapseNewlineHspace(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] == '\n' || v[i] == '\r' {
			for i < len(v) && (v[i] == '\n' || v[i] == '\r') {
				i++
			}
			for i < len(v) && (v[i] == ' ' || v[i] == '\t') {
				i++
			}
			i--
			continue
		}
		b.WriteByte(v[i])
	}
	return b.String()
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

// isAttributeComment reports whether t is a "#[...]" attribute (the lexer keeps
// it as one comment token), as opposed to a "//" or "#" line comment.
func isAttributeComment(t token.Token) bool {
	return t.Kind == token.Comment && strings.HasPrefix(t.Value, "#[")
}

// collapseAttributeArgs collapses a multiline argument list inside an attribute
// (e.g. "#[Attr(\n    'foo'\n)]" -> "#[Attr('foo')]") on the attribute's token
// text. With singleArgOnly it only collapses a call that has a single argument,
// matching on_multiline=ensure_single_line_for_single_argument.
func collapseAttributeArgs(s *tokens.Stream, singleArgOnly bool) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if !isAttributeComment(t) || !strings.ContainsRune(t.Value, '\n') {
			continue
		}
		if nv, ok := collapseAttributeText(t.Value, singleArgOnly); ok && nv != t.Value {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

// collapseAttributeText rewrites the first top-level "(...)" in an attribute onto
// one line. It returns the rewritten text and whether a collapse applied.
func collapseAttributeText(attr string, singleArgOnly bool) (string, bool) {
	open := strings.IndexByte(attr, '(')
	if open < 0 {
		return attr, false
	}
	closeIdx := matchParen(attr, open)
	if closeIdx < 0 {
		return attr, false
	}
	args, trailingComma := splitTopLevelArgs(attr[open+1 : closeIdx])
	if len(args) == 0 {
		return attr, false
	}
	if singleArgOnly && len(args) != 1 {
		return attr, false
	}
	inner := strings.Join(args, ", ")
	if trailingComma {
		inner += ","
	}
	return attr[:open+1] + inner + attr[closeIdx:], true
}

// applyAttributePlacement enforces attribute_placement inside every multiline
// call/declaration paren. "standalone" puts each attribute, and the parameter it
// decorates, on its own line; "same_line" keeps them on one line.
func applyAttributePlacement(s *tokens.Stream, placement string) bool {
	if placement == "ignore" {
		return false
	}
	changed := false
	for open := 0; open < s.Len(); open++ {
		if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !isCallOrDeclParen(s, open) || !argListIsMultiline(s, open, closeIdx) {
			continue
		}
		if placeAttributesInParen(s, open, closeIdx, placement) {
			changed = true
		}
	}
	return changed
}

// placeAttributesInParen rewrites the whitespace after each top-level attribute
// in the paren at open per placement. Applied right-to-left so indices stay valid.
func placeAttributesInParen(s *tokens.Stream, open, closeIdx int, placement string) bool {
	var attrs []int
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if isAttributeComment(t) {
			if depth == 0 {
				attrs = append(attrs, j)
			}
			continue
		}
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
	}
	changed := false
	for _, a := range slices.Backward(attrs) {
		var want string
		if placement == "same_line" {
			want = " "
		} else {
			want = "\n" + lineIndentBefore(s, a)
		}
		if editSlotAfter(s, a, want) {
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
	if reflowBreakAfterComments(s, open, closeIdx, base) {
		changed = true
	}
	if editSlotAfter(s, open, argNL) {
		changed = true
	}
	return changed
}

// reflowBreakAfterComments puts a top-level block comment that is followed by
// argument code on the same line onto its own line, matching php-cs-fixer. A
// comment trailing another token ("$e/* c */") or sitting before a comma/closer
// is left in place. Applied right-to-left so indices stay valid.
func reflowBreakAfterComments(s *tokens.Stream, open, closeIdx int, base string) bool {
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
		if depth != 0 || (t.Kind != token.Comment && t.Kind != token.DocComment) ||
			isLineComment(t) || isAttributeComment(t) {
			continue // attributes are placed by applyAttributePlacement, not here
		}
		// the comment must start its own line (preceded by a newline) to count as a
		// standalone leading comment rather than a trailing one on an argument's line
		if p := j - 1; p < 0 || s.At(p).Kind != token.Whitespace || !hasNewline(s.At(p).Value) {
			continue
		}
		nx := j + 1
		if nx < s.Len() && s.At(nx).Kind == token.Whitespace {
			if hasNewline(s.At(nx).Value) {
				continue
			}
			nx++
		}
		if nx >= closeIdx {
			continue
		}
		if c := s.At(nx); c.Kind == token.Comment || c.Kind == token.DocComment ||
			(c.Kind == token.Punct && (c.Value == "," || c.Value == ")" || c.Value == "]" || c.Value == "}")) {
			continue
		}
		if editSlotAfter(s, j, "\n"+base+"    ") {
			changed = true
		}
	}
	return changed
}

// reflowAfterComma breaks a multiline argument list after a top-level comma. A
// comment that sits on the comma's line ("arg, // note" or "arg, /* note */")
// stays there and the break goes after the comment, matching php-cs-fixer;
// otherwise the break goes right after the comma.
func reflowAfterComma(s *tokens.Stream, comma int, base string) bool {
	n := comma + 1
	ws := -1
	if n < s.Len() && s.At(n).Kind == token.Whitespace && !hasNewline(s.At(n).Value) {
		ws = n
		n++
	}
	if n < s.Len() && !isAttributeComment(s.At(n)) &&
		(isLineComment(s.At(n)) || s.At(n).Kind == token.Comment || s.At(n).Kind == token.DocComment) {
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
	// a newline directly after "(" or directly before ")" means the list is split
	// (php-cs-fixer's first/last-whitespace check)
	if n := open + 1; n < closeIdx && s.At(n).Kind == token.Whitespace && hasNewline(s.At(n).Value) {
		return true
	}
	if p := closeIdx - 1; p > open && s.At(p).Kind == token.Whitespace && hasNewline(s.At(p).Value) {
		return true
	}
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		// a top-level attribute that itself spans lines makes the list multiline
		if depth == 0 && isAttributeComment(t) && hasNewline(t.Value) {
			return true
		}
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
		// "class" matches an anonymous class constructor: `new class (...)`;
		// "use" matches a closure binding list
		return lv == "function" || lv == "fn" || lv == "class" || lv == "use"
	}
	return false
}

// isCommaSpacedParen reports whether the "(" at open has its commas spaced on a
// single line. This is every call/declaration paren plus list() destructuring;
// list() gets comma spacing but is not reflown to fully multiline (php-cs-fixer
// leaves a partially multiline list() alone).
func isCommaSpacedParen(s *tokens.Stream, open int) bool {
	if isCallOrDeclParen(s, open) {
		return true
	}
	p := sigPrev(s, open)
	return p >= 0 && s.At(p).Kind == token.Keyword && strings.ToLower(s.At(p).Value) == "list"
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

// masCommentLastLine reports whether the token at idx is a comment that ends its
// line (the following whitespace starts with a line break), matching php-cs-fixer's
// isCommentLastLineToken.
func masCommentLastLine(s *tokens.Stream, idx int) bool {
	if idx < 0 || idx >= s.Len() {
		return false
	}
	t := s.At(idx)
	if t.Kind != token.Comment && t.Kind != token.DocComment {
		return false
	}
	if idx+1 >= s.Len() {
		return false
	}
	nx := s.At(idx + 1)
	return nx.Kind == token.Whitespace && len(nx.Value) > 0 && (nx.Value[0] == '\n' || nx.Value[0] == '\r')
}
