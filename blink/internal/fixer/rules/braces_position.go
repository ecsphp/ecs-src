package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php
//
// BracesPosition places opening braces per PSR-12: classes/interfaces/traits/
// enums and named functions/methods get their "{" on the next line, aligned
// with the declaration; control structures, closures and anonymous classes
// keep it on the same line after a single space. Each brace position is
// configurable; empty option fields keep this default behaviour. A body
// written on the line of its "{" is put on lines of its own, unless it is one
// PHP-CS-Fixer allows on one line (allow_single_line_anonymous_functions,
// allow_single_line_empty_anonymous_classes).
type BracesPosition struct {
	classesOpeningBrace            string
	functionsOpeningBrace          string
	anonymousClassesOpeningBrace   string
	anonymousFunctionsOpeningBrace string
	controlStructuresOpeningBrace  string
	// allowSingleLineAnonFnsSet records whether the option was given; the value
	// defaults to true (php-cs-fixer 3 default), which keeps a single-line
	// closure or property hook body on its line, while false puts every such
	// body on lines of its own.
	allowSingleLineAnonFns    bool
	allowSingleLineAnonFnsSet bool
	// noSingleLineEmptyAnonClasses is allow_single_line_empty_anonymous_classes
	// set to false: an empty anonymous class body is put on lines of its own too
	noSingleLineEmptyAnonClasses bool
}

const (
	bracesNextLine = "next_line_unless_newline_at_signature_end"
	bracesSameLine = "same_line"
)

func (f BracesPosition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["classes_opening_brace"].(string); ok {
		f.classesOpeningBrace = v
	}
	if v, ok := config["functions_opening_brace"].(string); ok {
		f.functionsOpeningBrace = v
	}
	if v, ok := config["anonymous_classes_opening_brace"].(string); ok {
		f.anonymousClassesOpeningBrace = v
	}
	if v, ok := config["anonymous_functions_opening_brace"].(string); ok {
		f.anonymousFunctionsOpeningBrace = v
	}
	if v, ok := config["control_structures_opening_brace"].(string); ok {
		f.controlStructuresOpeningBrace = v
	}
	if v, ok := config["allow_single_line_empty_anonymous_classes"].(bool); ok {
		f.noSingleLineEmptyAnonClasses = !v
	}
	if v, ok := config["allow_single_line_anonymous_functions"].(bool); ok {
		f.allowSingleLineAnonFns = v
		f.allowSingleLineAnonFnsSet = true
	}
	return f
}

func (BracesPosition) Name() string {
	return `PhpCsFixer\Fixer\Basic\BracesPositionFixer`
}

func (BracesPosition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php"
}

func (f BracesPosition) Fix(s *tokens.Stream) bool {
	changed := false
	// PHP-CS-Fixer's $allowSingleLineUntil: the "}" of the first body left on
	// one line, before which a body is broken into lines only when it already
	// spans lines (or, for an anonymous class allowed on one line, holds code)
	singleLineUntil := -1
	for i := 0; i < s.Len(); i++ {
		c, ok := f.bpConstruct(s, i)
		if !ok {
			continue
		}
		open := c.open
		closeIdx := s.MatchForward(open)
		lenBefore := s.Len()
		nested := singleLineUntil >= 0 && c.index < singleLineUntil

		// an unclosed "{" (PHP-CS-Fixer throws on it) still gets its position
		newlinesInside := closeIdx >= 0
		markSingleLine := false
		if newlinesInside && (c.allowSingleLine || c.allowSingleLineIfEmpty || nested) {
			newlinesInside = false
			for j := open + 1; j < closeIdx; j++ {
				t := s.At(j)
				if c.allowSingleLineIfEmpty && t.Kind != token.Whitespace && !isComment(t) || t.Kind == token.Whitespace && hasNewline(t.Value) {
					newlinesInside = true
					break
				}
			}
			markSingleLine = !newlinesInside && singleLineUntil < 0
		}
		if newlinesInside && bpOpenBraceNeedsNewline(s, open) {
			changed = editSlotAfter(s, open, "\n"+lineIndent(s, open)) || changed
		}

		nextLine := c.position == bracesNextLine && !funcSignatureMultiline(s, open)
		if target := bpBraceTarget(s, open, nextLine); target >= 0 {
			open = bpMoveBrace(s, open, target)
			changed = true
		}
		want := " "
		if nextLine {
			// PHP-CS-Fixer aligns the brace with the line holding the keyword
			want = "\n" + lineIndent(s, c.index)
		}
		switch {
		case !nextLine && bpFollowsLineComment(s, open):
		case open > 0 && s.At(open-1).Kind == token.Whitespace:
			if s.At(open-1).Value != want {
				s.SetValue(open-1, want)
				changed = true
			}
		default:
			s.InsertAt(open, token.Token{Kind: token.Whitespace, Value: want})
			open++
			changed = true
		}

		if newlinesInside && bpCloseBraceOnOwnLine(s, open) {
			changed = true
		}
		switch {
		case markSingleLine:
			singleLineUntil = s.MatchForward(open)
		case nested:
			singleLineUntil += s.Len() - lenBefore
		}
	}
	return changed
}

// bpConstruct is one construct BracesPositionFixer handles: index is the
// token it starts from (past an array default value for a property), open
// its body's "{", position where that brace goes.
type bpConstruct struct {
	index, open                             int
	position                                string
	allowSingleLine, allowSingleLineIfEmpty bool
}

var bpControlKeywords = map[string]bool{
	"declare": true, "do": true, "else": true, "elseif": true, "finally": true, "for": true, "foreach": true,
	"if": true, "while": true, "try": true, "catch": true, "switch": true, "match": true,
}

// bpConstruct mirrors one step of PHP-CS-Fixer's loop over the tokens: the
// construct the token at i starts, if any. Its branches are those of
// BracesPositionFixer::applyFix: classy, function, control structure, property
// with a hook list, and property hook.
func (f BracesPosition) bpConstruct(s *tokens.Stream, i int) (bpConstruct, bool) {
	t := s.At(i)
	c := bpConstruct{index: i}
	switch {
	case taIsClassy(s, i):
		c.open = bpNextOf(s, i, false, false)
		if c.open < 0 {
			return c, false
		}
		if taIsAnonymousClass(s, i) {
			c.allowSingleLineIfEmpty = !f.noSingleLineEmptyAnonClasses
			c.position = bpPosition(f.anonymousClassesOpeningBrace, bracesSameLine)
		} else {
			c.position = bpPosition(f.classesOpeningBrace, bracesNextLine)
		}
	case kwIs(t, "function"):
		c.open = bpNextOf(s, i, true, false)
		if c.open < 0 || !bpIsBodyBrace(s, c.open) {
			return c, false
		}
		if taIsLambda(s, i) {
			c.allowSingleLine = !f.allowSingleLineAnonFnsSet || f.allowSingleLineAnonFns
			c.position = bpPosition(f.anonymousFunctionsOpeningBrace, bracesSameLine)
		} else {
			c.position = bpPosition(f.functionsOpeningBrace, bracesNextLine)
		}
	case t.Kind == token.Keyword && bpControlKeywords[strings.ToLower(t.Value)]:
		c.open = sigNext(s, bpParenthesisEnd(s, i))
		if c.open < 0 || !bpIsBodyBrace(s, c.open) {
			return c, false
		}
		c.position = bpPosition(f.controlStructuresOpeningBrace, bracesSameLine)
	case t.Kind == token.Variable && !isLoneDollar(t):
		next := sigNext(s, i)
		if !isPunctVal(s, next, "=") && !bpIsHookListBrace(s, next) {
			return c, false
		}
		// the default value: an array is skipped, so a "{" inside it is not the hook list
		if isPunctVal(s, next, "=") {
			next = sigNext(s, next)
			if isPunctVal(s, next, "[") {
				c.index = s.MatchForward(next)
			} else if next >= 0 && kwIs(s.At(next), "array") {
				if next = sigNext(s, next); isPunctVal(s, next, "(") {
					c.index = s.MatchForward(next)
				}
			}
		}
		if c.index < 0 {
			return c, false
		}
		c.open = bpNextOf(s, c.index, true, true)
		if c.open < 0 || !bpIsHookListBrace(s, c.open) {
			return c, false
		}
		multiline := false
		for j := c.open; j <= s.MatchForward(c.open) && !multiline; j++ {
			multiline = strings.Contains(s.At(j).Value, "\n")
		}
		if !multiline {
			return c, false
		}
		c.position = bpPosition(f.controlStructuresOpeningBrace, bracesSameLine)
	case t.Kind == token.Ident:
		p := sigPrev(s, i)
		if p < 0 || !isPunctVal(s, p, "}") && !isPunctVal(s, p, ";") && s.At(p).Kind != token.AttributeEnd && !bpIsHookListBrace(s, p) {
			return c, false
		}
		c.open = sigNext(s, bpParenthesisEnd(s, i))
		if c.open < 0 || !bpIsBodyBrace(s, c.open) {
			return c, false
		}
		c.allowSingleLine = !f.allowSingleLineAnonFnsSet || f.allowSingleLineAnonFns
		c.position = bpPosition(f.controlStructuresOpeningBrace, bracesSameLine)
	default:
		return c, false
	}
	return c, true
}

// bpPosition is the configured brace position, or def when the option is unset.
func bpPosition(configured, def string) string {
	if configured == "" {
		return def
	}
	return configured
}

// bpParenthesisEnd is BracesPositionFixer::findParenthesisEnd: the ")" closing
// the "(" right after the token at i, or i when no "(" follows.
func bpParenthesisEnd(s *tokens.Stream, i int) int {
	n := sigNext(s, i)
	if !isPunctVal(s, n, "(") {
		return i
	}
	if closing := s.MatchForward(n); closing >= 0 {
		return closing
	}
	return i
}

// bpIsBodyBrace reports whether the "{" at i is a plain brace for PHP-CS-Fixer,
// not one of the braces its transformers retag: dynamic access ("$o->{$p}",
// "${$p}"), a group import ("use A\{B, C}") or a property hook list.
func bpIsBodyBrace(s *tokens.Stream, i int) bool {
	if !isPunctVal(s, i, "{") || isPropertyHookBrace(s, i) {
		return false
	}
	if p := sigPrev(s, i); p >= 0 {
		pt := s.At(p)
		if isLoneDollar(pt) || pt.Kind == token.Punct && (pt.Value == "->" || pt.Value == "?->" || pt.Value == "::" || pt.Value == `\`) {
			return false
		}
	}
	return true
}

// bpIsHookListBrace reports whether the token at i is the "{" of a property hook
// list (CT::T_PROPERTY_HOOK_BRACE_OPEN).
func bpIsHookListBrace(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "{") && isPropertyHookBrace(s, i)
}

// bpNextOf is the Tokens::getNextTokenOfKind call of each branch: the index of
// the first plain "{" after i or, when stopAtSemicolon, of the first ";" or
// property hook list "{"; property adds the tokens a property declaration
// stops at: ".", an interpolated string and "?>". It returns -1 when none.
func bpNextOf(s *tokens.Stream, i int, stopAtSemicolon, property bool) int {
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		switch {
		case t.Kind == token.Punct && t.Value == "{":
			if bpIsBodyBrace(s, j) || stopAtSemicolon && isPropertyHookBrace(s, j) {
				return j
			}
		case stopAtSemicolon && t.Kind == token.Punct && t.Value == ";":
			return j
		case property && (t.Kind == token.Punct && t.Value == "." || t.Kind == token.CloseTag || t.Kind == token.String && t.Part != token.Whole):
			return j
		}
	}
	return -1
}

// bpBraceTarget is PHP-CS-Fixer's $moveBraceToIndex: where the brace goes so that a
// comment next to it stays out of the way, or -1. On the same line the brace goes
// before the comments, as written after "// c" it would be part of the comment. On the
// next line it goes after a "//" or "#" comment that follows it.
func bpBraceTarget(s *tokens.Stream, open int, nextLine bool) int {
	if nextLine {
		return bpTargetAfterComment(s, open)
	}
	target := -1
	previous := prevMeaningfulIndex(s, open)
	for i := open - 1; i > previous; i-- {
		if !isComment(s.At(i)) {
			continue
		}
		i--
		if s.At(i).Kind == token.Whitespace {
			target = i
		} else if i == previous {
			target = previous + 1
		}
	}
	// PHP-CS-Fixer moves it here too and writes a parse error
	if target < 0 || isLineComment(s.At(target-1)) || s.MatchForward(open) < 0 {
		return -1
	}
	return target
}

func bpTargetAfterComment(s *tokens.Stream, open int) int {
	if before := s.At(open - 1); before.Kind == token.Whitespace && hasNewline(before.Value) {
		return -1
	}
	closing := s.MatchForward(open)
	if closing < 0 {
		return -1
	}
	target := -1
	for i := open + 1; i < closing; i++ {
		t := s.At(i)
		// PHP-CS-Fixer goes on past code when a comment kept the body on the brace
		// line ("{ /* c */ a(); // d"), and the brace jumps over that code
		if t.Kind == token.Whitespace && hasNewline(t.Value) || t.Kind != token.Whitespace && !isComment(t) {
			break
		}
		// blink's "// c ?>" comment keeps the space, so the close tag can follow directly
		if isLineComment(t) && s.At(i+1).Kind == token.Whitespace {
			target = i + 1
		}
	}
	return target
}

// bpFollowsLineComment reports a brace whose line starts after a "//" or "#" comment:
// on the same line as that comment it would be part of it.
func bpFollowsLineComment(s *tokens.Stream, open int) bool {
	before := open - 1
	if s.At(before).Kind == token.Whitespace {
		before--
	}
	return isLineComment(s.At(before))
}

var bpLeadingBreakRe = regexp.MustCompile(`^(?:[ \t]*?(?:\r\n|\n|\r))?[ \t]*`)

// bpMoveBrace moves the brace at open to target like PHP-CS-Fixer: the tokens between
// shift by one, and the whitespace left on either side is cleared or merged. It returns
// the new index of the brace.
func bpMoveBrace(s *tokens.Stream, open, target int) int {
	first, last := min(open-1, target), max(open+1, target)
	work := slices.Clone(s.Tokens()[first : last+1])
	at := func(i int) *token.Token { return &work[i-first] }
	clearAt := func(i int) { *at(i) = token.Token{Kind: token.Whitespace} }
	isSpace := func(i int) bool { return at(i).Kind == token.Whitespace }

	brace := *at(open)
	delta := 1
	if open > target {
		delta = -1
	}
	if isSpace(open + delta) {
		if delta == -1 && hasNewline(at(open-1).Value) {
			if isSpace(open + 1) {
				at(open + 1).Value = bpLeadingBreakRe.ReplaceAllString(at(open+1).Value, "")
			}
		} else if isSpace(open - 1) {
			clearAt(open - 1)
		}
	}
	for i := open; i != target; i += delta {
		*at(i) = *at(i + delta)
	}
	*at(target) = brace
	if isSpace(open) && isSpace(open+1) {
		at(open).Value += at(open + 1).Value
		clearAt(open + 1)
	}

	moved := make([]token.Token, 0, len(work))
	index := -1
	for i, t := range work {
		if i+first == target {
			index = first + len(moved)
		}
		if t.Value != "" {
			moved = append(moved, t)
		}
	}
	s.ReplaceRange(first, last, moved)
	return index
}

// bpOpenBraceNeedsNewline reports whether php-cs-fixer breaks the line after
// the "{" at open: not when a line break already follows it (comments
// directly after the brace aside), when a comment shares its line, or when
// the next meaningful token is "?>".
func bpOpenBraceNeedsNewline(s *tokens.Stream, open int) bool {
	for j := open + 1; j < s.Len()-1; j++ {
		t := s.At(j)
		if t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if t.Kind == token.Whitespace && hasNewline(t.Value) {
			return false
		}
		break
	}
	if j := open + 1; j < s.Len() {
		if s.At(j).Kind == token.Whitespace && !hasNewline(s.At(j).Value) {
			j++
		}
		if j < s.Len() && (s.At(j).Kind == token.Comment || s.At(j).Kind == token.DocComment) {
			return false
		}
	}
	if n := sigNext(s, open); n >= 0 && s.At(n).Kind == token.CloseTag {
		return false
	}
	return true
}

// bpCloseBraceOnOwnLine puts the "}" matching the "{" at open on a line of its
// own, at the indentation of the line holding the "{", unless it already
// starts a line or directly follows "<?php".
func bpCloseBraceOnOwnLine(s *tokens.Stream, open int) bool {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	if p := sigPrev(s, closeIdx); p >= 0 && s.At(p).Kind == token.OpenTag {
		return false
	}
	prev := closeIdx - 1
	for prev > 0 && s.At(prev).Value == "" {
		prev--
	}
	if s.At(prev).Kind == token.Whitespace && hasNewline(s.At(prev).Value) {
		return false
	}
	ws := "\n" + lineIndent(s, open)
	if s.At(prev).Kind == token.Whitespace {
		s.SetValue(prev, ws)
	} else {
		s.InsertAt(prev+1, token.Token{Kind: token.Whitespace, Value: ws})
	}
	return true
}

// funcSignatureMultiline is the next_line_unless_newline_at_signature_end
// exception of PHP-CS-Fixer's BracesPositionFixer: from the "{", step back over
// the tokens that can end a signature; the brace stays only when that lands on
// a ")" with a newline (or a newline and a comment) right before it.
func funcSignatureMultiline(s *tokens.Stream, brace int) bool {
	p := prevMeaningfulIndex(s, brace)
	for p >= 0 && bpIsSignatureEndToken(s, p) {
		p = prevMeaningfulIndex(s, p)
	}
	if p <= 0 || !isPunctVal(s, p, ")") {
		return false
	}
	p--
	if k := s.At(p).Kind; (k == token.Comment || k == token.DocComment) && p > 0 {
		p--
	}
	prev := s.At(p)
	return prev.Kind == token.Whitespace && hasNewline(prev.Value)
}

// bpIsSignatureEndToken reports whether the token at i is one of the kinds
// BracesPositionFixer steps back over: T_STRING, T_NS_SEPARATOR, T_STATIC,
// T_CALLABLE, CT::T_ARRAY_TYPEHINT, CT::T_TYPE_COLON, CT::T_NULLABLE_TYPE,
// CT::T_TYPE_ALTERNATION, CT::T_TYPE_INTERSECTION and DNF parentheses.
func bpIsSignatureEndToken(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Ident:
		return true
	case token.Keyword:
		return isKeywordValue(t, "static", "array", "callable")
	case token.Punct:
		switch t.Value {
		case ":", "?", `\`, "|", "&":
			return true
		case "(":
			return bpIsDNFParens(s, i, s.MatchForward(i))
		case ")":
			return bpIsDNFParens(s, s.MatchBackward(i), i)
		}
	}
	return false
}

// bpIsDNFParens reports whether the parentheses at open/closeIdx are a DNF
// type group: the "(" follows a "|" or the ")" precedes one.
func bpIsDNFParens(s *tokens.Stream, open, closeIdx int) bool {
	if open < 0 || closeIdx < 0 {
		return false
	}
	if p := prevMeaningfulIndex(s, open); p >= 0 && isPunctVal(s, p, "|") {
		return true
	}
	n := nextMeaningfulIndex(s, closeIdx)
	return n >= 0 && isPunctVal(s, n, "|")
}
