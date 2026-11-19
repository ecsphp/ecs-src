package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/ArrayIndentSniff.php
//
// ArrayIndent indents the keys of a multi-line array one Indent deeper than
// its opening line and its closing brace back to that line.
type ArrayIndent struct {
	Indent int
}

func (ArrayIndent) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Arrays\ArrayIndentSniff`
}

func (ArrayIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/ArrayIndentSniff.php"
}

func (f ArrayIndent) WithConfig(config map[string]any) fixer.Fixer {
	f.Indent = configInt(config, "indent", f.Indent)
	return f
}

func (f ArrayIndent) Fix(s *tokens.Stream) bool {
	return phpcsRunUntilStable(s, f.pass)
}

// phpcsLayout maps tokens to the PHPCS line pieces they split into.
type phpcsLayout struct {
	s       *tokens.Stream
	pieces  []phpcsIndentLine
	line    []int // line number of each piece
	first   []int // first piece of each token, -1 for an empty token
	last    []int // last piece of each token
	ternary map[int]bool
	repl    map[int]string
}

func newPhpcsLayout(s *tokens.Stream) *phpcsLayout {
	l := &phpcsLayout{
		s:       s,
		pieces:  phpcsSplitDocLines(s, phpcsIndentLines(s)),
		first:   make([]int, s.Len()),
		last:    make([]int, s.Len()),
		ternary: phpcsTernaryColons(s),
		repl:    map[int]string{},
	}
	for i := range l.first {
		l.first[i] = -1
	}
	line := 1
	for n, p := range l.pieces {
		l.line = append(l.line, line)
		if l.first[p.tok] < 0 {
			l.first[p.tok] = n
		}
		l.last[p.tok] = n
		if strings.HasSuffix(p.text, "\n") {
			line++
		}
	}
	next := len(l.pieces) - 1
	for i := len(l.first) - 1; i >= 0; i-- {
		if l.first[i] < 0 {
			l.first[i], l.last[i] = max(next, 0), max(next, 0)
		}
		next = l.first[i]
	}
	return l
}

func (l *phpcsLayout) col(i int) int       { return l.pieces[l.first[i]].col }
func (l *phpcsLayout) startLine(i int) int { return l.line[l.first[i]] }
func (l *phpcsLayout) endLine(i int) int   { return l.line[l.last[i]] }

// firstOnLine mirrors File::findFirstOnLine(T_WHITESPACE, i, true): the first
// piece on token i's line that is not whitespace.
func (l *phpcsLayout) firstOnLine(i int) int {
	n := l.first[i]
	found := n
	for m := n; m >= 0 && l.line[m] == l.line[n]; m-- {
		p := l.pieces[m]
		if l.s.At(p.tok).Kind != token.Whitespace && !p.keySpace {
			found = m
		}
	}
	return found
}

// replace mirrors Fixer::replaceToken, which changes a token once per loop.
func (l *phpcsLayout) replace(n int, v string) {
	if _, done := l.repl[n]; !done {
		l.repl[n] = v
	}
}

func (l *phpcsLayout) addBefore(n int, v string) {
	l.replace(n, v+l.pieces[n].text)
}

func (l *phpcsLayout) apply() bool {
	return phpcsReplaceLines(l.s, l.pieces, l.repl)
}

// phpcsTernaryColons marks the ":" tokens PHPCS tokenizes as T_INLINE_ELSE.
func phpcsTernaryColons(s *tokens.Stream) map[int]bool {
	colons := map[int]bool{}
	pending := []int{0}
	for i := range s.Len() {
		t := s.At(i)
		switch {
		case t.Kind == token.AttributeStart || (t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{")):
			pending = append(pending, 0)
		case t.Kind == token.AttributeEnd || (t.Kind == token.Punct && (t.Value == ")" || t.Value == "]" || t.Value == "}")):
			if len(pending) > 1 {
				pending = pending[:len(pending)-1]
			}
		case t.Kind == token.Punct && t.Value == "?" && !phpcsIsNullable(s, i):
			pending[len(pending)-1]++
		case t.Kind == token.Punct && t.Value == ":" && pending[len(pending)-1] > 0:
			pending[len(pending)-1]--
			colons[i] = true
		}
	}
	return colons
}

func (l *phpcsLayout) statementEnd(i int) bool {
	t := l.s.At(i)
	if t.Kind == token.CloseTag {
		return true
	}
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case ",", "=>", ";":
		return true
	case ":":
		return !l.ternary[i]
	}
	return false
}

func isPunct(t token.Token, v string) bool {
	return t.Kind == token.Punct && t.Value == v
}

// startOfStatement mirrors File::findStartOfStatement.
func (l *phpcsLayout) startOfStatement(start int) int {
	s := l.s
	lastNotEmpty := start
	for start > 0 && (l.statementEnd(start) || isPunct(s.At(start), "}")) {
		if isPunct(s.At(start), "}") {
			if open := s.MatchBackward(start); open > 0 {
				start = open
			}
		}
		start--
	}
	for i := start; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.OpenTag || l.statementEnd(i) || isPunct(t, "}") ||
			(t.Kind == token.Punct && (t.Value == "{" || t.Value == "[" || t.Value == "(")) {
			return lastNotEmpty
		}
		if isPunct(t, "]") || isPunct(t, ")") {
			if open := s.MatchBackward(i); open >= 0 {
				i = open
			}
		}
		if !isPhpcsEmpty(s.At(i)) {
			lastNotEmpty = i
		}
	}
	return 0
}

// fnEnd is the scope closer PHPCS gives an arrow function: the last non-empty
// token of its body.
func (l *phpcsLayout) fnEnd(fn int) int {
	s := l.s
	arrow := -1
	for i := fn + 1; i < s.Len(); i++ {
		if isPunct(s.At(i), "(") {
			if c := s.MatchForward(i); c > 0 {
				i = c
			}
			continue
		}
		if isPunct(s.At(i), "=>") {
			arrow = i
			break
		}
	}
	if arrow < 0 {
		return fn
	}
	end := s.Len()
	for i := arrow + 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
			if c := s.MatchForward(i); c > 0 {
				i = c
				continue
			}
		}
		if t.Kind == token.OpenTag || t.Kind == token.CloseTag ||
			(t.Kind == token.Punct && (t.Value == "," || t.Value == ";" || t.Value == ")" || t.Value == "]" || t.Value == "}" ||
				(t.Value == ":" && !l.ternary[i]))) {
			end = i
			break
		}
	}
	return phpcsPrevNonEmpty(s, end-1)
}

// next mirrors AbstractArraySniff::getNext: the next comma or double arrow at
// the array's own nesting level.
func (l *phpcsLayout) next(ptr, arrayEnd int) int {
	s := l.s
	for ptr < arrayEnd {
		t := s.At(ptr)
		switch {
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "fn"):
			ptr = l.fnEnd(ptr)
		case t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{"):
			if c := s.MatchForward(ptr); c > 0 {
				ptr = c
			}
		}
		if isPunct(s.At(ptr), ",") || isPunct(s.At(ptr), "=>") {
			return ptr
		}
		ptr++
	}
	return ptr
}

func (f ArrayIndent) pass(s *tokens.Stream) bool {
	l := newPhpcsLayout(s)
	eol := phpcsEolChar(s)
	for i := range s.Len() {
		t := s.At(i)
		var open int
		switch {
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "array"):
			open = phpcsNextNonEmpty(s, i+1)
			if open >= s.Len() || !isPunct(s.At(open), "(") {
				continue
			}
		case isPunct(t, "[") && phpcsShortArrayOpen(s, i):
			open = i
		default:
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || l.startLine(open) == l.startLine(closeIdx) {
			continue
		}
		f.multiLine(l, i, open, closeIdx, eol)
	}
	return l.apply()
}

func (f ArrayIndent) multiLine(l *phpcsLayout, stackPtr, arrayStart, arrayEnd int, eol string) {
	s := l.s
	prev := stackPtr - 1
	for prev >= 0 && (isPhpcsEmpty(s.At(prev)) || isPunct(s.At(prev), "=>")) {
		prev--
	}
	if prev < 0 {
		return
	}
	baseIndent := l.pieces[l.firstOnLine(l.startOfStatement(prev))].col - 1
	first := l.firstOnLine(stackPtr)
	startIndent := l.pieces[first].col - 1
	if startIndent < baseIndent {
		phpcsIndentTo(l, first, startIndent, baseIndent)
		return
	}

	expected := startIndent + f.Indent
	for _, start := range l.arrayItemStarts(arrayStart, arrayEnd) {
		p := phpcsPrevNonEmpty(s, start-1)
		if p >= 0 && l.endLine(p) == l.startLine(start) {
			continue
		}
		first := l.firstOnLine(start)
		if found := l.pieces[first].col - 1; found != expected {
			phpcsIndentTo(l, first, found, expected)
		}
	}

	p := arrayEnd - 1
	for p >= 0 && s.At(p).Kind == token.Whitespace {
		p--
	}
	if p >= 0 && l.endLine(p) == l.startLine(arrayEnd) {
		l.addBefore(l.first[arrayEnd], eol+strings.Repeat(" ", startIndent))
		return
	}
	if found := l.col(arrayEnd) - 1; found != startIndent {
		phpcsIndentTo(l, l.first[arrayEnd], found, startIndent)
	}
}

// phpcsIndentTo indents the line whose first content is piece n to want spaces.
func phpcsIndentTo(l *phpcsLayout, n, found, want int) {
	padding := strings.Repeat(" ", want)
	if found == 0 {
		l.addBefore(n, padding)
		return
	}
	l.replace(n-1, padding)
}

// arrayItemStarts returns the first token of each key, or of each value
// without a key, mirroring AbstractArraySniff::process.
func (l *phpcsLayout) arrayItemStarts(arrayStart, arrayEnd int) []int {
	s := l.s
	var starts []int
	current := arrayStart
	for {
		next := phpcsNextNonEmpty(s, current+1)
		if next < 0 || next >= arrayEnd {
			return starts
		}
		starts = append(starts, next)
		valueStart := next
		if end := l.next(next, arrayEnd); isPunct(s.At(end), "=>") {
			valueStart = phpcsNextNonEmpty(s, end+1)
			if valueStart >= s.Len() {
				return starts
			}
		}
		current = l.next(valueStart, arrayEnd)
	}
}
