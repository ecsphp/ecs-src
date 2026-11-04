package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// convertLongArray rewrites a long "name(...)" construct to "[...]", where name
// is "array" or "list". Method/constant uses (after -> ?-> ::) and a method
// declared with that name (after "function") are skipped.
func convertLongArray(s *tokens.Stream, name string) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		// a method named "list" lexes as an identifier
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != name {
			continue
		}
		j := skipWhitespace(s, i+1)
		if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(j)
		if closeIdx < 0 {
			continue
		}
		s.SetValue(closeIdx, "]")
		s.SetValue(j, "[")
		s.ReplaceRange(i, j-1, nil)
		changed = true
	}
	return changed
}

// arrayNotationStringList reads a string-list option (a []string or []any of strings).
func arrayNotationStringList(v any) ([]string, bool) {
	switch list := v.(type) {
	case []string:
		return list, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, str)
		}
		return out, true
	}
	return nil, false
}

func arrayNotationContains(list []string, want string) bool {
	return slices.Contains(list, want)
}

// arrayNotationIsHeredoc reports whether t ends a heredoc or nowdoc (T_END_HEREDOC).
func arrayNotationIsHeredoc(t token.Token) bool {
	return t.EndsHeredoc()
}

// arrayNotationEnclosing returns the index of the innermost unclosed opener before i, or -1.
func arrayNotationEnclosing(s *tokens.Stream, i int) int {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			if depth == 0 {
				return j
			}
			depth--
		}
	}
	return -1
}

func arrayNotationIsAs(t token.Token) bool {
	return (t.Kind == token.Keyword || t.Kind == token.Ident) && len(t.Value) == 2 && strings.EqualFold(t.Value, "as")
}

// arrayNotationInForeachHead reports whether the token at i sits inside the
// "(... as ...)" head of a foreach, after the "as" keyword.
func arrayNotationInForeachHead(s *tokens.Stream, i int) bool {
	depth := 0
	seenAs := false
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case ")", "]", "}":
				depth++
			case "(", "[", "{":
				if depth > 0 {
					depth--
					continue
				}
				if t.Value != "(" || !seenAs {
					return false
				}
				p := sigPrev(s, j)
				return p >= 0 && strings.EqualFold(s.At(p).Value, "foreach")
			}
			continue
		}
		if depth == 0 && arrayNotationIsAs(t) {
			seenAs = true
		}
	}
	return false
}

// arrayNotationForeachHeadAt is the end of arrayNotationInForeachHead's scan:
// enc is the innermost unclosed opener before the token, and seenAs whether an
// "as" sits between them at that depth.
func arrayNotationForeachHeadAt(s *tokens.Stream, enc int, seenAs bool) bool {
	if enc < 0 || s.At(enc).Value != "(" || !seenAs {
		return false
	}
	p := sigPrev(s, enc)
	return p >= 0 && strings.EqualFold(s.At(p).Value, "foreach")
}

// arrayNotationTracker follows the bracket nesting of a forward walk over a
// stream, so the walk learns what arrayNotationEnclosing and
// arrayNotationInForeachHead would find by scanning backwards, without the
// scan. Those scans cost the distance to the enclosing bracket, so asking them
// for every "[" of a long array or method body is quadratic.
//
// Step every token in order. A query describes the position after the last
// stepped token, and matches the backward scan as long as the walk has not
// changed a stepped token's kind or bracket/"as" value, nor moved it.
type arrayNotationTracker struct {
	frames []arrayNotationFrame
}

type arrayNotationFrame struct {
	open   int  // index of the unclosed opener
	seenAs bool // an "as" was stepped directly inside it (not in a nested bracket)
}

// step accounts for t, the token at index i. Like the backward scans, it pairs
// any closer with the innermost opener, whatever their kinds.
func (tr *arrayNotationTracker) step(t token.Token, i int) {
	if t.Kind != token.Punct {
		if len(tr.frames) > 0 && arrayNotationIsAs(t) {
			tr.frames[len(tr.frames)-1].seenAs = true
		}
		return
	}
	switch t.Value {
	case "(", "[", "{":
		tr.frames = append(tr.frames, arrayNotationFrame{open: i})
	case ")", "]", "}":
		if len(tr.frames) > 0 {
			tr.frames = tr.frames[:len(tr.frames)-1]
		}
	}
}

// top returns the innermost unclosed opener and whether an "as" sits directly
// in it so far, or -1.
func (tr *arrayNotationTracker) top() (int, bool) {
	if len(tr.frames) == 0 {
		return -1, false
	}
	f := tr.frames[len(tr.frames)-1]
	return f.open, f.seenAs
}

// inForeachHead is arrayNotationInForeachHead for the next token to step.
func (tr *arrayNotationTracker) inForeachHead(s *tokens.Stream) bool {
	enc, seenAs := tr.top()
	return arrayNotationForeachHeadAt(s, enc, seenAs)
}

// arrayNotationScan answers arrayNotationEnclosing, arrayNotationInForeachHead
// and arrayNotationIsDestructuring for any index of a stream. Near answers come
// from a bounded backward scan; once one would look further back than near
// tokens, a forward pass (an arrayNotationTracker run over the whole stream)
// records the answer at every index, and destructuring answers are remembered.
//
// It describes the stream as it was when that pass ran. A rule that edits the
// stream can keep asking it only about indices below every edit made since,
// and only when those edits cannot change an answer: TrailingCommaInMultiline
// and arrayNotationToLong walk backwards, edit at or after the current token,
// and what they write does not change how earlier brackets classify.
type arrayNotationScan struct {
	s    *tokens.Stream
	near int      // longest backward scan before the forward pass
	info []uint32 // per index, once built: enclosing+1 | seenAs | memo bits
}

const (
	arrayNotationEnc    = 1<<29 - 1 // innermost unclosed opener before i, plus one (0: none)
	arrayNotationSeenAs = 1 << 29   // an "as" sits between that opener and i, at its depth
	arrayNotationKnown  = 1 << 30   // isDestructuring(i) is remembered...
	arrayNotationDestr  = 1 << 31   // ...and is true
)

// newArrayNotationScan returns a scan of s, or nil for a stream too long to
// index, which the callers take as "scan backwards".
func newArrayNotationScan(s *tokens.Stream) *arrayNotationScan {
	if s.Len() > arrayNotationEnc {
		return nil
	}
	// most brackets sit close to their enclosing one, and a short scan back is
	// cheaper than a pass over the whole file
	return &arrayNotationScan{s: s, near: 256}
}

// arrayNotationEnclosingNear scans back from i, over at most limit tokens, for
// what arrayNotationEnclosing and arrayNotationInForeachHead look for: the
// innermost unclosed opener (-1 for none) and whether an "as" sits between it
// and i at that depth. ok is false when it gave up.
func arrayNotationEnclosingNear(s *tokens.Stream, i, limit int) (enc int, seenAs, ok bool) {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if i-j > limit {
			return 0, false, false
		}
		t := s.At(j)
		if t.Kind != token.Punct {
			if depth == 0 && arrayNotationIsAs(t) {
				seenAs = true
			}
			continue
		}
		switch t.Value {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			if depth == 0 {
				return j, seenAs, true
			}
			depth--
		}
	}
	return -1, seenAs, true
}

func (sc *arrayNotationScan) build() {
	n := sc.s.Len()
	sc.info = make([]uint32, n)
	var tr arrayNotationTracker
	for i := range n {
		enc, seenAs := tr.top()
		v := uint32(enc + 1)
		if seenAs {
			v |= arrayNotationSeenAs
		}
		sc.info[i] = v
		// step ignores every other kind
		switch sc.s.Kind(i) {
		case token.Punct, token.Keyword, token.Ident:
			tr.step(sc.s.At(i), i)
		}
	}
}

// lookup returns the innermost unclosed opener before i (or -1) and whether an
// "as" sits between them at that depth.
func (sc *arrayNotationScan) lookup(i int) (int, bool) {
	if sc.info == nil {
		if enc, seenAs, ok := arrayNotationEnclosingNear(sc.s, i, sc.near); ok {
			return enc, seenAs
		}
		sc.build()
	}
	v := sc.info[i]
	return int(v&arrayNotationEnc) - 1, v&arrayNotationSeenAs != 0
}

// enclosing is arrayNotationEnclosing.
func (sc *arrayNotationScan) enclosing(i int) int {
	enc, _ := sc.lookup(i)
	return enc
}

// inForeachHead is arrayNotationInForeachHead.
func (sc *arrayNotationScan) inForeachHead(i int) bool {
	enc, seenAs := sc.lookup(i)
	return arrayNotationForeachHeadAt(sc.s, enc, seenAs)
}

// isDestructuring is arrayNotationIsDestructuring. Answers are remembered once
// the forward pass has run, which is when asking again gets expensive.
func (sc *arrayNotationScan) isDestructuring(open int) bool {
	if sc.info != nil && sc.info[open]&arrayNotationKnown != 0 {
		return sc.info[open]&arrayNotationDestr != 0
	}
	d := arrayNotationDestructuring(sc.s, open, sc)
	if sc.info != nil {
		sc.info[open] |= arrayNotationKnown
		if d {
			sc.info[open] |= arrayNotationDestr
		}
	}
	return d
}

// arrayNotationIsDestructuring reports whether the short "[" at open is an
// array destructuring target rather than an array literal.
func arrayNotationIsDestructuring(s *tokens.Stream, open int) bool {
	return arrayNotationDestructuring(s, open, nil)
}

// arrayNotationDestructuring is arrayNotationIsDestructuring, asking sc (when
// not nil) instead of scanning backwards.
func arrayNotationDestructuring(s *tokens.Stream, open int, sc *arrayNotationScan) bool {
	if closeIdx := s.MatchForward(open); closeIdx >= 0 {
		if n := sigNext(s, closeIdx); n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "=" {
			return true
		}
	}
	if p := sigPrev(s, open); p >= 0 {
		if arrayNotationIsAs(s.At(p)) {
			return true
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == "=>" {
			if sc != nil {
				if sc.inForeachHead(open) {
					return true
				}
			} else if arrayNotationInForeachHead(s, open) {
				return true
			}
		}
	}
	var enc int
	if sc != nil {
		enc = sc.enclosing(open)
	} else {
		enc = arrayNotationEnclosing(s, open)
	}
	if enc < 0 {
		return false
	}
	switch s.At(enc).Value {
	case "[":
		if isOffsetOpen(s, enc) {
			return false
		}
		if sc != nil {
			return sc.isDestructuring(enc)
		}
		return arrayNotationIsDestructuring(s, enc)
	case "(":
		q := sigPrev(s, enc)
		return q >= 0 && strings.EqualFold(s.At(q).Value, "list")
	}
	return false
}

// arrayNotationToLong rewrites short "[...]" to "array(...)" (literals) or
// "list(...)" (destructuring targets), depending on destructuring.
func arrayNotationToLong(s *tokens.Stream, destructuring bool) bool {
	// Each rewrite is at or after i and turns a matched "[...]" into "name(...)",
	// so the scan stays right for the brackets before i (see arrayNotationScan).
	return arrayNotationToLongWith(s, destructuring, newArrayNotationScan(s))
}

// arrayNotationToLongWith is arrayNotationToLong; a nil sc scans backwards
// instead, as a reference for tests.
func arrayNotationToLongWith(s *tokens.Stream, destructuring bool, sc *arrayNotationScan) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "[" || isOffsetOpen(s, i) {
			continue
		}
		if arrayNotationDestructuring(s, i, sc) != destructuring {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		name := "array"
		if destructuring {
			name = "list"
		}
		s.SetValue(closeIdx, ")")
		s.SetValue(i, "(")
		s.InsertAt(i, token.Token{Kind: token.Keyword, Value: name})
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php
//
// ArraySyntax rewrites long "array(...)" to the short "[...]" form. With
// syntax=long it does the opposite.
type ArraySyntax struct {
	Long bool
}

func (ArraySyntax) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer`
}

func (ArraySyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php"
}

func (f ArraySyntax) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.Long = v == "long"
	}
	return f
}

func (f ArraySyntax) Fix(s *tokens.Stream) bool {
	if f.Long {
		return arrayNotationToLong(s, false)
	}
	return convertLongArray(s, "array")
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php
//
// ListSyntax rewrites "list(...)" to the short "[...]" form. With syntax=long
// it does the opposite.
type ListSyntax struct {
	Long bool
}

func (ListSyntax) Name() string {
	return `PhpCsFixer\Fixer\ListNotation\ListSyntaxFixer`
}

func (ListSyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php"
}

func (f ListSyntax) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.Long = v == "long"
	}
	return f
}

func (f ListSyntax) Fix(s *tokens.Stream) bool {
	if f.Long {
		return arrayNotationToLong(s, true)
	}
	return convertLongArray(s, "list")
}

// enclosingIsArray reports whether the innermost open bracket at index i is "[".
func topBracketIsArray(stack []string) bool {
	return len(stack) > 0 && stack[len(stack)-1] == "["
}

// isDestructuringAssignOpen reports whether the "[" at open is a short-list
// destructuring target - its matching "]" is directly followed by a single "=",
// or it sits after "as" in a foreach head ("foreach ($x as ['k' => $v])").
func isDestructuringAssignOpen(s *tokens.Stream, open int) bool {
	return isDestructuringAssignOpenIn(s, open, nil)
}

// isDestructuringAssignOpenIn is isDestructuringAssignOpen, asking tr (when not
// nil, and stepped up to open) whether open sits in a foreach head.
func isDestructuringAssignOpenIn(s *tokens.Stream, open int, tr *arrayNotationTracker) bool {
	c := s.MatchForward(open)
	if c < 0 {
		return false
	}
	n := nextSignificantIndex(s, c)
	if n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "=" {
		return true
	}
	if tr != nil {
		return tr.inForeachHead(s)
	}
	return arrayNotationInForeachHead(s, open)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php
//
// NoWhitespaceBeforeCommaInArray removes single-line whitespace before a comma
// inside an array. SkipAfterHeredoc (after_heredoc=false) keeps the whitespace
// between a heredoc end and the comma.
type NoWhitespaceBeforeCommaInArray struct {
	SkipAfterHeredoc bool
}

func (NoWhitespaceBeforeCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NoWhitespaceBeforeCommaInArrayFixer`
}

func (NoWhitespaceBeforeCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php"
}

func (f NoWhitespaceBeforeCommaInArray) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["after_heredoc"].(bool); ok {
		f.SkipAfterHeredoc = !v
	}
	return f
}

func (f NoWhitespaceBeforeCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				stack = append(stack, t.Value)
			case ")", "]", "}":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			case ",":
				if topBracketIsArray(stack) && i > 0 &&
					s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
					if f.SkipAfterHeredoc && i > 1 && arrayNotationIsHeredoc(s.At(i-2)) {
						continue
					}
					// an empty element ("[, , $x]") keeps its space: the token before
					// the whitespace is itself a comma, matching php-cs-fixer
					if i > 1 && s.At(i-2).Kind == token.Punct && s.At(i-2).Value == "," {
						continue
					}
					s.RemoveAt(i - 1)
					i--
					changed = true
				}
			}
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php
//
// WhitespaceAfterCommaInArray ensures a single space after a comma inside an
// array (unless a newline follows, or the comma is a trailing one before "]").
// EnsureSingleSpace also collapses horizontal whitespace after the comma to one space.
type WhitespaceAfterCommaInArray struct {
	EnsureSingleSpace bool
}

func (WhitespaceAfterCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\WhitespaceAfterCommaInArrayFixer`
}

func (WhitespaceAfterCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php"
}

func (f WhitespaceAfterCommaInArray) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ensure_single_space"].(bool); ok {
		f.EnsureSingleSpace = v
	}
	return f
}

func arrayNotationIsHorizontal(t token.Token) bool {
	return t.Kind == token.Whitespace && t.Value != "" && strings.Trim(t.Value, " \t") == ""
}

func (f WhitespaceAfterCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	// the edits below only write whitespace after the current comma (the space
	// they insert is skipped, not stepped), and the tracker ignores whitespace
	var tr arrayNotationTracker
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		destructuring := t.Kind == token.Punct && t.Value == "[" &&
			((len(stack) > 0 && stack[len(stack)-1] == "d[") || isDestructuringAssignOpenIn(s, i, &tr))
		tr.step(t, i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "[":
			// a destructuring target ("[$a,$b] = ...", or nested in one) is not an
			// array literal, so php-cs-fixer does not space its commas
			if destructuring {
				stack = append(stack, "d[")
			} else {
				stack = append(stack, "[")
			}
		case "(", "{":
			stack = append(stack, t.Value)
		case ")", "]", "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ",":
			if !topBracketIsArray(stack) || i+1 >= s.Len() {
				continue
			}
			next := s.At(i + 1)
			if next.Kind == token.Whitespace {
				if f.EnsureSingleSpace && next.Value != " " && arrayNotationIsHorizontal(next) {
					afterComment := i+2 < s.Len() && (s.At(i+2).Kind == token.Comment || s.At(i+2).Kind == token.DocComment)
					if !afterComment || (i+3 < s.Len() && arrayNotationIsHorizontal(s.At(i+3))) {
						s.SetValue(i+1, " ")
						changed = true
					}
				}
				continue // already spaced or newline
			}
			if next.Kind == token.Punct && next.Value == "]" {
				continue // trailing comma
			}
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			i++
			changed = true
		}
	}
	return changed
}
