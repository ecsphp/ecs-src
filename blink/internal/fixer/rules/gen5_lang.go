package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// This file implements three PHP-CS-Fixer ControlStructure rules on the flat
// token stream. Each is a deliberately narrow subset of the upstream fixer:
// only token cases that are provably semantics-preserving and idempotent are
// rewritten, everything else is left untouched.

// includeKeywords are the four language constructs handled by IncludeFixer.
var includeKeywords = map[string]bool{
	"include": true, "include_once": true, "require": true, "require_once": true,
}

// isMemberAccessPrev reports whether the significant token before i makes the
// name at i a member access ("$o->include", "Foo::for") rather than a language
// keyword.
func isMemberAccessPrev(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	return pt.Kind == token.Punct && (pt.Value == "->" || pt.Value == "?->" || pt.Value == "::")
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/IncludeFixer.php
//
// Include normalizes the whitespace between an include/require construct and its
// argument to exactly one space ("require  \"x\"" -> "require \"x\""). Only the
// spacing part of the upstream fixer is implemented; the paren-form
// ("include(\"x\")") is left untouched because removing those parentheses cannot
// be proven safe on a flat token stream.
type Include struct{}

func (Include) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\IncludeFixer`
}

func (Include) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/IncludeFixer.php"
}

func (Include) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !includeKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if isMemberAccessPrev(s, i) {
			continue
		}
		n := i + 1
		if n >= s.Len() {
			continue
		}
		nt := s.At(n)
		if nt.Kind == token.Whitespace {
			if hasNewline(nt.Value) {
				continue // keep multi-line spacing (alignment) untouched
			}
			j := sigNext(s, i)
			if j < 0 || (s.At(j).Kind == token.Punct && s.At(j).Value == "(") {
				continue // paren form: leave the parentheses alone
			}
			if nt.Value != " " {
				s.SetValue(n, " ")
				changed = true
			}
			continue
		}
		// Directly adjacent argument: insert a single space. Only the token kinds
		// that can legally abut the keyword are handled ("include$x", "include'x'",
		// "include\\Foo"); the paren form is skipped. PHP-CS-Fixer takes a
		// T_CONSTANT_ENCAPSED_STRING, not an interpolated string, heredoc or
		// backtick literal.
		switch {
		case nt.Kind == token.Variable:
		case nt.IsLiteral() && !nt.StartsHeredoc() && !nt.IsShellExec() && !isInterpolatedString(nt):
		case nt.Kind == token.Punct && nt.Value == `\`:
		default:
			continue
		}
		s.InsertAt(n, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}

// loopBodyKeywords are the loops EmptyLoopBody acts on (for/foreach/while).
var loopBodyKeywords = map[string]bool{
	"for": true, "foreach": true, "while": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopBodyFixer.php
//
// EmptyLoopBody rewrites an empty loop body in the style option: "semicolon"
// (the default) turns "while ($x) {}" into "while ($x);", "braces" does the
// reverse. A body holding a comment and the "while" of a do-while are kept.
type EmptyLoopBody struct {
	Style string
}

func (EmptyLoopBody) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\EmptyLoopBodyFixer`
}

func (EmptyLoopBody) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopBodyFixer.php"
}

func (f EmptyLoopBody) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["style"].(string); ok {
		f.Style = v
	}
	return f
}

func (f EmptyLoopBody) Fix(s *tokens.Stream) bool {
	braces := f.Style == "braces"
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !loopBodyKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if isMemberAccessPrev(s, i) {
			continue
		}
		open := sigNext(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeParen := s.MatchForward(open)
		if closeParen < 0 {
			continue
		}
		if braces {
			if strings.EqualFold(t.Value, "while") && isWhilePartOfDoWhile(s, i) {
				continue
			}
			semicolon := sigNext(s, closeParen)
			if semicolon < 0 || s.At(semicolon).Kind != token.Punct || s.At(semicolon).Value != ";" {
				continue
			}
			s.SetValue(semicolon, "{")
			s.InsertAt(semicolon+1, token.Token{Kind: token.Punct, Value: "}"})
			changed = true
			continue
		}
		braceOpen := sigNext(s, closeParen)
		if braceOpen < 0 || s.At(braceOpen).Kind != token.Punct || s.At(braceOpen).Value != "{" {
			continue
		}
		// The body is empty only when the next non-whitespace token (comments are
		// NOT skipped, so a commented body is preserved) is the closing brace.
		braceClose := nextSignificantIndex(s, braceOpen)
		if braceClose < 0 || s.At(braceClose).Kind != token.Punct || s.At(braceClose).Value != "}" {
			continue
		}
		// Replace "{" with ";" and clear the "}" merging the whitespace around it
		// (clearTokenAndMergeSurroundingWhitespace): "{\n}\n" becomes ";\n\n".
		s.SetValue(braceOpen, ";")
		if next := braceClose + 1; braceClose > braceOpen+1 && next < s.Len() && s.At(next).Kind == token.Whitespace {
			s.SetValue(braceClose-1, s.At(braceClose-1).Value+s.At(next).Value)
			s.ReplaceRange(braceClose, next, nil)
		} else {
			s.RemoveAt(braceClose)
		}
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopConditionFixer.php
//
// EmptyLoopCondition writes an endless loop in the configured style. With the
// default `while` style, `for (;;)` and `do { } while (true);` become
// `while (true)`; with `for`, `while (true)` and `do { } while (true);` become
// `for(;;)`. Only the exact header `true` (lowercase, unqualified) or `;;`
// counts. As upstream does, the replaced tokens are cleared one by one, merging
// the whitespace around them, and the comments among them stay where they were;
// the new header goes where the `for`, `while` or `do` keyword was.
type EmptyLoopCondition struct {
	For bool // style: for (default while)
}

func (EmptyLoopCondition) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\EmptyLoopConditionFixer`
}

func (EmptyLoopCondition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopConditionFixer.php"
}

// WithConfig reads `style` (`while` or `for`).
func (f EmptyLoopCondition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["style"].(string); ok {
		f.For = v == "for"
	}
	return f
}

func (f EmptyLoopCondition) Fix(s *tokens.Stream) bool {
	replacement := []token.Token{
		{Kind: token.Keyword, Value: "while"},
		{Kind: token.Whitespace, Value: " "},
		{Kind: token.Punct, Value: "("},
		{Kind: token.Ident, Value: "true"},
		{Kind: token.Punct, Value: ")"},
	}
	if f.For {
		replacement = []token.Token{
			{Kind: token.Keyword, Value: "for"},
			{Kind: token.Punct, Value: "("},
			{Kind: token.Punct, Value: ";"},
			{Kind: token.Punct, Value: ";"},
			{Kind: token.Punct, Value: ")"},
		}
	}
	if !s.Index().HasAnyWord("for", "while") {
		return false
	}
	// look first, so a file without a loop to convert is left untouched (not
	// even its tags are folded, which would bump the stream version)
	found := false
	for i := s.Len() - 1; i > 0 && !found; i-- {
		_, _, found = f.loopAt(s, i)
	}
	if !found {
		return false
	}
	// the whitespace merging below depends on what the tags hold
	foldTagWhitespace(s)
	defer unfoldTagWhitespace(s)
	changed := false
	for i := s.Len() - 1; i > 0; i-- {
		clearTo, do, ok := f.loopAt(s, i)
		if !ok {
			continue
		}
		clearNotCommentsInRange(s, i, clearTo)
		at := i // where the replacement goes
		if do >= 0 {
			clearTokenAt(s, do)
			at = do
		}
		s.InsertSliceAt(at, slices.Clone(replacement))
		changed = true
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// loopAt reports whether the `for` or `while` keyword at i heads a loop to
// convert, the last token its rewrite clears (the `)` of the header, or the
// `;` of a do-while) and the `do` of a do-while (else -1). It does not change
// the stream.
func (f EmptyLoopCondition) loopAt(s *tokens.Stream, i int) (clearTo, do int, ok bool) {
	t := s.At(i)
	if t.Kind != token.Keyword {
		return 0, -1, false
	}
	kw := strings.ToLower(t.Value)
	if kw != "while" && (f.For || kw != "for") || isMemberAccessPrev(s, i) {
		return 0, -1, false
	}
	open := nextPunctOfKind(s, i, "(")
	if open < 0 {
		return 0, -1, false
	}
	closeParen := s.MatchForward(open)
	if closeParen < 0 {
		return 0, -1, false
	}
	switch {
	case kw == "for" && emptyForHeader(s, open, closeParen):
		return closeParen, -1, true
	case kw == "while" && whileTrueHeader(s, open, closeParen):
		if do := doOfWhile(s, i); do >= 0 {
			return sigNext(s, closeParen), do, true // the trailing `;` goes too
		}
		if f.For {
			return closeParen, -1, true
		}
	}
	return 0, -1, false
}

// clearNotCommentsInRange clears every token of [from, to] but the comments,
// merging the whitespace around each, like upstream's helper of that name.
func clearNotCommentsInRange(s *tokens.Stream, from, to int) {
	for j := from; j <= to; j++ {
		if k := s.At(j).Kind; k != token.Comment && k != token.DocComment {
			clearTokenAndMergeWS(s, j)
		}
	}
}

// emptyForHeader reports whether the parentheses open..closeParen hold `;;`.
func emptyForHeader(s *tokens.Stream, open, closeParen int) bool {
	a := sigNext(s, open)
	if a < 0 || !isPunctVal(s, a, ";") {
		return false
	}
	b := sigNext(s, a)
	return b >= 0 && isPunctVal(s, b, ";") && sigNext(s, b) == closeParen
}

// whileTrueHeader reports whether the parentheses open..closeParen hold `true`.
func whileTrueHeader(s *tokens.Stream, open, closeParen int) bool {
	a := sigNext(s, open)
	return a >= 0 && s.At(a).Kind == token.Ident && s.At(a).Value == "true" && sigNext(s, a) == closeParen
}

// doOfWhile returns the `do` keyword of the do-while whose `while` is at i, or
// -1 when the `while` does not follow a `do { }` body.
func doOfWhile(s *tokens.Stream, i int) int {
	end := sigPrev(s, i)
	if end < 0 || !isPunctVal(s, end, "}") {
		return -1
	}
	start := s.MatchBackward(end)
	if start < 0 {
		return -1
	}
	do := sigPrev(s, start)
	if do < 0 || !kwIs(s.At(do), "do") {
		return -1
	}
	return do
}

// foldTagWhitespace gives each open tag its first whitespace character and
// each close tag its line break, as PHP-CS-Fixer's T_OPEN_TAG and T_CLOSE_TAG
// hold them, so a port of code that clears tokens and merges whitespace sees
// the same neighbours. unfoldTagWhitespace splits them back.
func foldTagWhitespace(s *tokens.Stream) {
	for i := 0; i+1 < s.Len(); i++ {
		t, n := s.At(i), s.At(i+1)
		var head string
		switch {
		case t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=") && n.Kind == token.Whitespace:
			head = firstBlankOrNewline(n.Value)
		case t.Kind == token.CloseTag && n.Kind == token.InlineHTML:
			head = leadingNewline(n.Value)
		}
		if head == "" {
			continue
		}
		s.SetValue(i, t.Value+head)
		if rest := n.Value[len(head):]; rest == "" {
			s.RemoveAt(i + 1)
		} else {
			s.SetValue(i+1, rest)
		}
	}
}

// unfoldTagWhitespace undoes foldTagWhitespace, and joins the whitespace tokens
// left side by side.
func unfoldTagWhitespace(s *tokens.Stream) {
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		var tag string
		var kind token.Kind
		switch t.Kind {
		case token.OpenTag:
			tag, kind = strings.TrimRight(t.Value, " \t\r\n"), token.Whitespace
		case token.CloseTag:
			tag, kind = strings.TrimRight(t.Value, "\r\n"), token.InlineHTML
		case token.Whitespace:
			for i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
				s.SetValue(i, s.At(i).Value+s.At(i+1).Value)
				s.RemoveAt(i + 1)
			}
			continue
		default:
			continue
		}
		if tag == t.Value {
			continue
		}
		tail := t.Value[len(tag):]
		s.SetValue(i, tag)
		if i+1 < s.Len() && s.At(i+1).Kind == kind {
			s.SetValue(i+1, tail+s.At(i+1).Value)
		} else {
			s.InsertAt(i+1, token.Token{Kind: kind, Value: tail})
		}
	}
}
