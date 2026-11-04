package rules

import (
	"regexp"
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php
//
// BracesPosition places opening braces per PSR-12: classes/interfaces/traits/
// enums and named functions/methods get their "{" on the next line, aligned
// with the declaration; control structures keep it on the same line after a
// single space. Closures and free blocks are left alone by default. Each brace
// position is configurable; empty option fields keep this default behaviour.
type BracesPosition struct {
	classesOpeningBrace            string
	functionsOpeningBrace          string
	anonymousClassesOpeningBrace   string
	anonymousFunctionsOpeningBrace string
	controlStructuresOpeningBrace  string
	// allowSingleLineAnonFnsSet records whether the option was given; the value
	// defaults to true (php-cs-fixer 3 default), which keeps a single-line
	// closure body on its line, while false puts every closure body on lines of
	// its own.
	allowSingleLineAnonFns    bool
	allowSingleLineAnonFnsSet bool
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
	// allow_single_line_empty_anonymous_classes is accepted but not applied: blink
	// already keeps single-line empty bodies (matching the default true).
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
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		// a property hook list is CT::T_PROPERTY_HOOK_BRACE_OPEN in
		// PHP-CS-Fixer, not a "{" this fixer moves
		if isPropertyHookBrace(s, i) {
			continue
		}
		kind, kw := classifyBrace(s, i)

		var category string
		switch kind {
		case braceClassLike:
			category = "classes"
			if taIsAnonymousClass(s, kw) {
				category = "anon_classes"
			}
		case braceFunctionDecl:
			category = "functions"
		case braceClosure:
			category = "anon_functions"
		case braceControl:
			category = "control"
		default:
			continue
		}

		// an empty class/function body already collapsed to "{}" stays on its
		// line - ECS keeps "class A {}" as-is (single_line_empty_body owns the
		// collapse); only reflow a body with content
		if (kind == braceClassLike || kind == braceFunctionDecl) && s.MatchForward(i) == i+1 {
			continue
		}

		// php-cs-fixer's $addNewlinesInsideBraces, for closures only so far
		newlinesInside := kind == braceClosure && f.closureBodyNeedsNewlines(s, i)
		if newlinesInside && bpOpenBraceNeedsNewline(s, i) {
			changed = editSlotAfter(s, i, "\n"+lineIndent(s, i)) || changed
		}

		if nextLine, handle := f.wantNextLine(category, s, i); handle {
			if target := bpBraceTarget(s, i, nextLine); target >= 0 {
				i = bpMoveBrace(s, i, target)
				changed = true
			}
			want := " "
			if nextLine {
				// php-cs-fixer aligns the brace with the line holding the keyword
				want = "\n" + lineIndent(s, kw)
			}
			switch {
			case !nextLine && bpFollowsLineComment(s, i):
			case i > 0 && s.At(i-1).Kind == token.Whitespace:
				if s.At(i-1).Value != want {
					s.SetValue(i-1, want)
					changed = true
				}
			default:
				s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: want})
				i++
				changed = true
			}
		}

		if newlinesInside && bpCloseBraceOnOwnLine(s, i) {
			changed = true
		}
	}
	return changed
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
		// PHP-CS-Fixer has already put a one-line body on its own line at this point;
		// blink has not, so stop at code or the brace would jump over it
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

// closureBodyNeedsNewlines reports whether the body of the anonymous function
// whose "{" is at open goes on lines of its own: always when
// allow_single_line_anonymous_functions is false, and otherwise only when
// some whitespace inside the braces already holds a line break.
func (f BracesPosition) closureBodyNeedsNewlines(s *tokens.Stream, open int) bool {
	if f.allowSingleLineAnonFnsSet && !f.allowSingleLineAnonFns {
		return true
	}
	closeIdx := s.MatchForward(open)
	for j := open + 1; j < closeIdx; j++ {
		if s.At(j).Kind == token.Whitespace && hasNewline(s.At(j).Value) {
			return true
		}
	}
	return false
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

// wantNextLine resolves the configured position for a brace category to whether
// the "{" goes on the next line. handle is false for closures left at their
// default (blink does not move anonymous-function braces unless configured).
func (f BracesPosition) wantNextLine(category string, s *tokens.Stream, brace int) (nextLine, handle bool) {
	var pos, def string
	switch category {
	case "classes":
		pos, def = f.classesOpeningBrace, bracesNextLine
	case "functions":
		pos, def = f.functionsOpeningBrace, bracesNextLine
	case "anon_classes":
		pos, def = f.anonymousClassesOpeningBrace, bracesSameLine
	case "control":
		pos, def = f.controlStructuresOpeningBrace, bracesSameLine
	case "anon_functions":
		if f.anonymousFunctionsOpeningBrace == "" {
			return false, false
		}
		pos, def = f.anonymousFunctionsOpeningBrace, bracesSameLine
	default:
		return false, false
	}
	if pos == "" {
		pos = def
	}
	if pos == bracesSameLine {
		return false, true
	}
	// next_line_unless_newline_at_signature_end
	return !funcSignatureMultiline(s, brace), true
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
