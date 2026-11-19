package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/OperatorSpacingSniff.php
//
// SquizOperatorSpacing requires exactly one space around arithmetic, bitwise,
// comparison, assignment and ternary operators: "$a  +$b" -> "$a + $b". The
// space before an assignment is left alone (IgnoreSpacingBeforeAssignments),
// and a comment on the neighbouring line keeps its operator where it is.
type SquizOperatorSpacing struct {
	IgnoreNewlines                 bool
	IgnoreSpacingBeforeAssignments bool
}

func (SquizOperatorSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\OperatorSpacingSniff`
}

func (SquizOperatorSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/OperatorSpacingSniff.php"
}

func (f SquizOperatorSpacing) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	if v, ok := config["ignoreSpacingBeforeAssignments"].(bool); ok {
		f.IgnoreSpacingBeforeAssignments = v
	}
	return f
}

func (f SquizOperatorSpacing) Fix(s *tokens.Stream) bool {
	// a newline is fixed one PHP_CodeSniffer whitespace token per pass, so
	// repeat like ECS does until a pass changes nothing
	changed := false
	history := map[int]*phpcsHistory{}
	for loop := range phpcsFixerLoops {
		if !f.pass(s, loop, history) {
			break
		}
		changed = true
	}
	return changed
}

func (f SquizOperatorSpacing) pass(s *tokens.Stream, loop int, history map[int]*phpcsHistory) bool {
	ops := newPhpcsOperatorContext(s)
	// edits are collected on the original indices ops was built on, then applied
	var e phpcsEdits
	for i := 0; i < s.Len(); i++ {
		if end, ok := ops.declare[i]; ok {
			if end < 0 {
				break
			}
			i = end
			continue
		}
		if !ops.isOperatorToken(s, i, false) || !ops.isOperator(s, i) {
			continue
		}
		f.fixBefore(s, i, &e)
		f.fixAfter(s, i, &e)
	}
	return e.apply(s, loop, history)
}

// fixBefore fixes the whitespace before the operator at i.
func (f SquizOperatorSpacing) fixBefore(s *tokens.Stream, i int, e *phpcsEdits) {
	if i == 0 {
		return
	}
	v := s.At(i).Value
	if s.At(i-1).Kind != token.Whitespace || !phpcsWhitespaceAt(s, i-1) {
		if v != ":" || s.At(i-1).Value != "?" {
			e.insert(i)
		}
		return
	}
	ws := i - 1
	commentNL := phpcsCommentNewline(s, ws)
	own := s.At(ws).Value[commentNL:]
	newline := commentNL > 0 || phpcsLineBreaks(own) > 0
	if v == "&" {
		if !f.spacingOK(s, ws, newline) {
			// only PHP_CodeSniffer's last whitespace token, the current line's, becomes " "
			e.replace(ws, s.At(ws).Value[:commentNL]+phpcsBeforeLastPiece(own)+" ", phpcsAfterLastPieceOf(own))
		}
		return
	}
	if (phpcsAssignment[v] || v == "=>") && f.IgnoreSpacingBeforeAssignments {
		return
	}
	if f.spacingOK(s, ws, newline) {
		return
	}
	if p := ws - 1; p >= 0 && isPhpcsCommentToken(s.At(p)) {
		return
	}
	e.replace(ws, " ", phpcsAfterLastPieceOf(own))
}

func (f SquizOperatorSpacing) fixAfter(s *tokens.Stream, i int, e *phpcsEdits) {
	next := i + 1
	if next >= s.Len() {
		return
	}
	if s.At(next).Kind != token.Whitespace {
		if s.At(i).Value != "?" || s.At(next).Value != ":" {
			e.insert(next)
		}
		return
	}
	if next+1 >= s.Len() {
		return
	}
	ws := s.At(next).Value
	newline := phpcsLineBreaks(ws) > 0
	if f.spacingOK(s, next, newline) {
		return
	}
	if newline && s.At(i).Value != "&" && isPhpcsCommentToken(s.At(next+1)) {
		return
	}
	// only PHP_CodeSniffer's first whitespace token, through the first line break, becomes " "
	e.replace(next, " "+ws[len(phpcsThroughFirstBreak(ws)):], phpcsThroughFirstBreak(ws))
}

// phpcsFixerLoops is PHP_CodeSniffer's cap on fixer passes per file.
const phpcsFixerLoops = 50

// phpcsEdits is one fixer pass's edits in the order the sniff made them, on
// the token indices it read, applied at once like PHP_CodeSniffer's fixer.
type phpcsEdits struct {
	list []phpcsEdit
}

type phpcsEdit struct {
	i      int
	insert bool   // a " " whitespace token goes in before i
	value  string // the new value of token i
	piece  string // the PHP_CodeSniffer whitespace token replaced, for conflicts
}

// phpcsHistory is PHP_CodeSniffer Fixer::$oldTokenValues for one token.
type phpcsHistory struct {
	curr, prev string
	loop       int
}

func (e *phpcsEdits) insert(i int) {
	e.list = append(e.list, phpcsEdit{i: i, insert: true})
}

func (e *phpcsEdits) replace(i int, value, piece string) {
	e.list = append(e.list, phpcsEdit{i: i, value: value, piece: piece})
}

// apply makes the pass's edits: a token is changed once per pass, and an edit
// that turns a token back into what the previous pass changed it from is a
// conflict that drops it and every later edit of the pass.
func (e *phpcsEdits) apply(s *tokens.Stream, loop int, history map[int]*phpcsHistory) bool {
	set := map[int]string{}
	var inserts []int
	for _, ed := range e.list {
		if ed.insert {
			inserts = append(inserts, ed.i)
			continue
		}
		// a no-op replacement still counts for PHP_CodeSniffer's conflict check
		if _, done := set[ed.i]; done {
			continue
		}
		h := history[ed.i]
		if h != nil && h.prev == " " && h.loop == loop-1 {
			break
		}
		if h == nil {
			history[ed.i] = &phpcsHistory{curr: " ", prev: ed.piece, loop: loop}
		} else {
			h.prev, h.curr, h.loop = h.curr, " ", loop
		}
		set[ed.i] = ed.value
	}
	changed := false
	for i, v := range set {
		if s.At(i).Value != v {
			s.SetValue(i, v)
			changed = true
		}
	}
	slices.Sort(inserts)
	for _, at := range slices.Backward(slices.Compact(inserts)) {
		s.InsertAt(at, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}

// spacingOK reports whether the whitespace at ws is the single space the sniff
// wants, or a line break it is told to ignore.
func (f SquizOperatorSpacing) spacingOK(s *tokens.Stream, ws int, newline bool) bool {
	if newline {
		return f.IgnoreNewlines
	}
	return phpcsTabWidth(s, ws) == 1
}

// phpcsBeforeLastPiece is v without PHP_CodeSniffer's last whitespace token:
// the text after the last line break, or else the last line including its break.
func phpcsBeforeLastPiece(v string) string {
	end := len(v) - len(phpcsAfterLastBreak(v))
	if end < len(v) {
		return v[:end]
	}
	rest := strings.TrimSuffix(v, "\n")
	if len(rest) == len(v) || strings.HasSuffix(rest, "\r") {
		rest = strings.TrimSuffix(rest, "\r")
	}
	return rest[:len(rest)-len(phpcsAfterLastBreak(rest))]
}

// isPhpcsCommentToken reports whether t is one of PHP_CodeSniffer's
// COMMENT_TOKENS: a comment, a docblock or a phpcs: annotation.
func isPhpcsCommentToken(t token.Token) bool {
	return (t.Kind == token.Comment && !strings.HasPrefix(t.Value, "#[")) || t.Kind == token.DocComment
}

// phpcsAfterLastPieceOf is PHP_CodeSniffer's last whitespace token of v.
func phpcsAfterLastPieceOf(v string) string {
	return v[len(phpcsBeforeLastPiece(v)):]
}
