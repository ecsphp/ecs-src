package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// ecs-src: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/LineLength/LineLengthFixer.php
//
// LineLength breaks array items, method parameters, call arguments, new and
// attribute arguments onto standalone lines when the opening line overflows the
// limit, and inlines a short multiline block back onto one line. It ports
// Symplify's LineLengthFixer: process every closer bottom-up so nested blocks
// resolve outer-first, decide by the real rendered line length, and skip keyed
// arrays, heredocs, commented and promoted-property blocks.
type LineLength struct {
	lineLength      int
	breakLongLines  bool
	inlineShortLine bool
	configured      bool
}

const (
	llDefaultLineLength = 120
	llIndent            = "    "
	llEnding            = "\n"
)

func (LineLength) Name() string {
	return `Symplify\CodingStandard\Fixer\LineLength\LineLengthFixer`
}

func (LineLength) SourceURL() string {
	return fixer.EcsSrcSourceBase + "LineLength/LineLengthFixer.php"
}

func (f LineLength) WithConfig(config map[string]any) fixer.Fixer {
	f.configured = true
	f.lineLength = configInt(config, "line_length", llDefaultLineLength)
	f.breakLongLines = true
	f.inlineShortLine = true
	if v, ok := config["break_long_lines"].(bool); ok {
		f.breakLongLines = v
	}
	if v, ok := config["inline_short_lines"].(bool); ok {
		f.inlineShortLine = v
	}
	return f
}

// resolved fills the zero-value defaults: a bare LineLength{} means all on.
func (f LineLength) resolved() LineLength {
	if f.configured {
		return f
	}
	return LineLength{
		lineLength:      llDefaultLineLength,
		breakLongLines:  true,
		inlineShortLine: true,
		configured:      true,
	}
}

func (f LineLength) Fix(s *tokens.Stream) bool {
	f = f.resolved()
	changed := false
	// bottom-up: an insert inside a block only shifts tokens to its right, so the
	// not-yet-visited lower indices stay valid; the outermost closer is processed
	// first, which is what makes nested calls resolve outer-to-inner
	n := s.Len()
	for pos := n - 1; pos >= 0; pos-- {
		if pos >= s.Len() {
			continue
		}
		t := s.At(pos)
		switch {
		case t.Kind == token.Punct && t.Value == ")":
			if f.processMethodCall(s, pos) {
				changed = true
			}
		case llIsAttribute(t):
			if f.processAttribute(s, pos) {
				changed = true
			}
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "new"):
			if f.processOpener(s, pos) {
				changed = true
			}
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "function"):
			if f.processOpener(s, pos) {
				changed = true
			}
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "use") && llIsClosureUse(s, pos):
			if f.processOpener(s, pos) {
				changed = true
			}
		case t.Kind == token.Punct && t.Value == "]" && llIsArrayClose(s, pos):
			if f.processArrayClose(s, pos) {
				changed = true
			}
		}
	}
	return changed
}

// processMethodCall handles a call/declaration closer ")" whose name is a plain
// lower-cased identifier, mirroring FunctionCallNameMatcher.
func (f LineLength) processMethodCall(s *tokens.Stream, closeIdx int) bool {
	open := s.MatchBackward(closeIdx)
	if open <= 0 {
		return false
	}
	name := open - 1
	if s.At(name).Kind != token.Ident {
		return false
	}
	if v := s.At(name).Value; v == "" || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	// empty argument list
	if open+1 < s.Len() && s.At(open+1).Kind == token.Punct && s.At(open+1).Value == ")" {
		return false
	}
	// a comment inside is unsafe to reflow
	for j := open; j <= closeIdx && j < s.Len(); j++ {
		if k := s.At(j).Kind; k == token.Comment || k == token.DocComment {
			return false
		}
	}
	return f.transform(s, open, closeIdx)
}

// processOpener handles the new / function / closure-use keywords by locating
// their "(" block, then applying the array/function skip rules.
func (f LineLength) processOpener(s *tokens.Stream, pos int) bool {
	open := -1
	for j := pos + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.Punct && t.Value == ";" {
			return false
		}
		if t.Kind == token.Punct && t.Value == "(" {
			open = j
			break
		}
	}
	if open < 0 {
		return false
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	// no arguments
	if sigNext(s, open) == closeIdx {
		return false
	}
	// a __construct signature is left to StandaloneLineConstructorParamFixer, which
	// keeps each param on its own line; LineLengthFixer skips it (see upstream)
	if llIsConstructorDecl(s, pos, open) {
		return false
	}
	if f.shouldSkip(s, open, closeIdx) {
		return false
	}
	return f.transform(s, open, closeIdx)
}

// processArrayClose handles a standalone "[...]" literal.
func (f LineLength) processArrayClose(s *tokens.Stream, closeIdx int) bool {
	open := s.MatchBackward(closeIdx)
	if open < 0 {
		return false
	}
	if f.shouldSkip(s, open, closeIdx) {
		return false
	}
	return f.transform(s, open, closeIdx)
}

// shouldSkip mirrors LineLengthFixer::shouldSkip: empty block, heredoc, a keyed
// "=>" array, or any inner comment.
func (f LineLength) shouldSkip(s *tokens.Stream, open, closeIdx int) bool {
	if closeIdx-open <= 1 {
		return true
	}
	if nx := sigNext(s, open); nx >= 0 && nx < closeIdx && llIsHeredoc(s.At(nx)) {
		return true
	}
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Punct && t.Value == "=>" {
			return true
		}
		if k := t.Kind; k == token.Comment || k == token.DocComment {
			return true
		}
	}
	return false
}

// transform decides break vs inline the way LineLengthTransformer does.
func (f LineLength) transform(s *tokens.Stream, open, closeIdx int) bool {
	if llHasPromotedProperty(s, open, closeIdx) {
		return false
	}
	if f.firstLineLength(s, open) > f.lineLength {
		if f.breakLongLines {
			return f.breakItems(s, open, closeIdx)
		}
		return false
	}
	if f.fullLineLength(s, open, closeIdx) > f.lineLength {
		return false
	}
	if !f.inlineShortLine {
		return false
	}
	return f.inlineItems(s, open, closeIdx)
}

// firstLineLength renders the length of the physical line the opener sits on,
// from line start through the first newline after the opener (ports
// FirstLineLengthResolver + LineLengthAndPositionFactory).
func (f LineLength) firstLineLength(s *tokens.Stream, open int) int {
	length := 0
	cp := open
	for cp >= 0 {
		t := s.At(cp)
		if llIsNewlineOrOpenTag(t) {
			break
		}
		c := t.Value
		if nl := strings.LastIndexByte(c, '\n'); nl >= 0 {
			length += len(c) - nl - 1
			cp--
			break
		}
		length += len(c)
		cp--
	}
	if cp >= 0 && cp < s.Len() {
		c := s.At(cp).Value
		length += len(c) - strings.Count(c, "\n")
	}
	for cp2 := open + 1; cp2 < s.Len(); cp2++ {
		t := s.At(cp2)
		if llIsEndOfArgsLine(t) {
			break
		}
		c := t.Value
		if nl := strings.IndexByte(c, '\n'); nl >= 0 {
			length += nl
			break
		}
		length += len(c)
	}
	return length
}

// fullLineLength is the length the block would have inlined onto one line, with
// each whitespace run counted as one char and the multi-line prefix/suffix
// counted verbatim (ports LineLengthResolver::getLengthFromStartEnd).
func (f LineLength) fullLineLength(s *tokens.Stream, open, closeIdx int) int {
	length := 0
	p := open
	for p >= 0 {
		if llIsNewlineOrOpenTag(s.At(p)) {
			break
		}
		length += len(s.At(p).Value)
		p--
	}
	if p >= 0 {
		length += len(s.At(p).Value)
	}
	for j := open; j < closeIdx && j < s.Len(); j++ {
		if s.At(j).Kind == token.Whitespace {
			length++
		} else {
			length += len(s.At(j).Value)
		}
	}
	for e := closeIdx; e < s.Len(); e++ {
		c := s.At(e).Value
		if len(c) > 0 && c[0] == '\n' {
			break
		}
		length += len(c)
	}
	return length
}

// breakItems puts each item on its own line, the opener line unchanged and the
// closer on its own line (ports TokensNewliner::breakItems with LineKind::CALLS).
func (f LineLength) breakItems(s *tokens.Stream, open, closeIdx int) bool {
	level := llIndentLevel(s, open)
	itemWS := llEnding + strings.Repeat(llIndent, level+1)
	closeWS := llEnding + strings.Repeat(llIndent, level)
	// closer first
	changed := f.breakCloser(s, open, closeIdx, itemWS, closeWS)
	// commas, bottom-up
	for i := closeIdx - 1; i >= open+1; i-- {
		i = llSkipBlocksReversed(s, i)
		if i < 0 {
			break
		}
		if s.At(i).Kind != token.Punct || s.At(i).Value != "," {
			continue
		}
		if llIsLastItem(s, i) || llIsFollowedByComment(s, i) || llIsFollowedByEmptyLine(s, i) {
			continue
		}
		if llEnsureWS(s, i+1, 0, itemWS) {
			changed = true
		}
	}
	// opener last
	if f.breakOpener(s, open, itemWS) {
		changed = true
	}
	return changed
}

// breakCloser inserts the newline before the closer; for a non-method call that
// ends with "])" it breaks before the "]" instead (shouldAddNewlineEarlier).
func (f LineLength) breakCloser(s *tokens.Stream, open, closeIdx int, itemWS, closeWS string) bool {
	isMethodCall := llIsMethodCall(s, open)
	prev := closeIdx - 1
	// TokenFinder resolves "previousPreviousToken" to endIndex-3, not endIndex-2
	pp := closeIdx - 3
	// method([...]) breaks before the "]"; method([]) and method([[...]]) do not
	if !isMethodCall && prev > open && llIsArrayCloseTok(s, prev) && !llClosesEmptyArray(s, prev) {
		if pp > open && !llIsArrayEdgeTok(s, pp) {
			return llEnsureWS(s, prev, 0, itemWS)
		}
	}
	return llEnsureWS(s, closeIdx-1, 1, closeWS)
}

// breakOpener inserts the newline after the opener.
func (f LineLength) breakOpener(s *tokens.Stream, open int, itemWS string) bool {
	next := open + 1
	if next < s.Len() && s.At(next).Kind == token.Whitespace {
		return llEnsureWS(s, next, 0, itemWS)
	}
	if next < s.Len() && llIsArrayOpenTok(s, next) && !llIsMethodCall(s, open) {
		return llEnsureWS(s, open+1, 1, itemWS)
	}
	return llEnsureWS(s, open, 1, itemWS)
}

// inlineItems collapses a multiline block onto one line, preserving nested
// curly/array blocks and the whitespace around a heredoc (ports TokensInliner).
func (f LineLength) inlineItems(s *tokens.Stream, open, closeIdx int) bool {
	changed := false
	for i := open + 1; i < closeIdx; i++ {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == "{" {
			if m := s.MatchForward(i); m > i {
				i = m
			}
			continue
		}
		if llIsArrayOpenTok(s, i) {
			if m := s.MatchForward(i); m > i {
				i = m
			}
			continue
		}
		if t.Kind != token.Whitespace {
			continue
		}
		prev := s.At(i - 1)
		next := s.At(i + 1)
		if llIsHeredoc(prev) {
			continue
		}
		if (prev.Kind == token.Punct && (prev.Value == "(" || prev.Value == "[")) ||
			(next.Kind == token.Punct && (next.Value == ")" || next.Value == "]")) {
			if t.Value != "" {
				s.SetValue(i, "")
				changed = true
			}
		} else if t.Value != " " {
			s.SetValue(i, " ")
			changed = true
		}
	}
	return changed
}

// processAttribute expands a too-long "#[Attr(...)]" (one comment token) onto
// multiple lines by rewriting its text.
func (f LineLength) processAttribute(s *tokens.Stream, pos int) bool {
	if !f.breakLongLines {
		return false
	}
	v := s.At(pos).Value
	if !strings.Contains(v, "(") {
		return false
	}
	attrFirst := v
	if before, _, ok := strings.Cut(v, "\n"); ok {
		attrFirst = before
	}
	if f.attributeColumn(s, pos)+len(attrFirst) <= f.lineLength {
		return false
	}
	base := lineIndent(s, pos)
	nv := llExpandAttribute(v, base)
	if nv == "" || nv == v {
		return false
	}
	s.SetValue(pos, nv)
	return true
}

// attributeColumn is the rendered column at which the attribute token starts.
func (f LineLength) attributeColumn(s *tokens.Stream, idx int) int {
	length := 0
	cp := idx - 1
	for cp >= 0 {
		t := s.At(cp)
		if llIsNewlineOrOpenTag(t) {
			break
		}
		c := t.Value
		if nl := strings.LastIndexByte(c, '\n'); nl >= 0 {
			length += len(c) - nl - 1
			cp--
			break
		}
		length += len(c)
		cp--
	}
	if cp >= 0 && cp < s.Len() {
		c := s.At(cp).Value
		length += len(c) - strings.Count(c, "\n")
	}
	return length
}

// llExpandAttribute rewrites the first top-level "(...)" of an attribute so each
// argument sits on its own line, indented one level past base.
func llExpandAttribute(v, base string) string {
	open := strings.IndexByte(v, '(')
	if open < 0 {
		return ""
	}
	closeIdx := matchParen(v, open)
	if closeIdx < 0 {
		return ""
	}
	args, _ := splitTopLevelArgs(v[open+1 : closeIdx])
	if len(args) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(v[:open+1])
	for _, arg := range args {
		b.WriteString("\n")
		b.WriteString(base)
		b.WriteString(llIndent)
		b.WriteString(arg)
		b.WriteByte(',')
	}
	// drop the trailing comma of the last item (attributes keep none here)
	out := strings.TrimSuffix(b.String(), ",")
	out += "\n" + base + v[closeIdx:]
	return out
}

// llEnsureWS ports PhpCsFixer Tokens::ensureWhitespaceAtIndex: overwrite an
// existing whitespace slot in place (ignoring offset), otherwise insert a new
// whitespace token at index+offset.
func llEnsureWS(s *tokens.Stream, index, offset int, ws string) bool {
	if index >= 0 && index < s.Len() && s.At(index).Kind == token.Whitespace {
		if ws == "" {
			s.SetValue(index, "")
			return true
		}
		if s.At(index).Value != ws {
			s.SetValue(index, ws)
			return true
		}
		return false
	}
	at := index + offset
	if at < 0 || at > s.Len() {
		return false
	}
	s.InsertAt(at, token.Token{Kind: token.Whitespace, Value: ws})
	return true
}

// llIndentLevel counts the indent units on the opener's physical line (ports
// IndentDetector::detectOnPosition).
func llIndentLevel(s *tokens.Stream, idx int) int {
	for j := idx; j >= 1; j-- {
		c := s.At(j).Value
		if nl := strings.LastIndexByte(c, '\n'); nl >= 0 {
			return strings.Count(c[nl+1:], llIndent)
		}
	}
	return 0
}

func llSkipBlocksReversed(s *tokens.Stream, i int) int {
	if i < 0 || i >= s.Len() {
		return i
	}
	t := s.At(i)
	if t.Kind == token.Punct && (t.Value == ")" || t.Value == "]") {
		if open := s.MatchBackward(i); open >= 0 {
			return open
		}
	}
	return i
}

func llIsLastItem(s *tokens.Stream, i int) bool {
	return i+1 < s.Len() && strings.Contains(s.At(i+1).Value, "\n")
}

func llIsFollowedByComment(s *tokens.Stream, i int) bool {
	if i+2 >= s.Len() {
		return false
	}
	if isComment(s.At(i + 2)) {
		return true
	}
	return s.At(i+1).Kind == token.Whitespace && s.At(i+1).Value == " " && isComment(s.At(i+2))
}

func llIsFollowedByEmptyLine(s *tokens.Stream, i int) bool {
	if i+1 >= s.Len() {
		return false
	}
	t := s.At(i + 1)
	return t.Kind == token.Whitespace && strings.Count(t.Value, "\n") >= 2
}

// llIsMethodCall reports whether the block opener is preceded by "->"/"?->".
func llIsMethodCall(s *tokens.Stream, open int) bool {
	name := sigPrev(s, open)
	if name < 0 {
		return false
	}
	arrow := sigPrev(s, name)
	if arrow < 0 {
		return false
	}
	t := s.At(arrow)
	return t.Kind == token.Punct && (t.Value == "->" || t.Value == "?->")
}

// llIsConstructorDecl reports whether the "(" at open belongs to a
// "function __construct(" declaration opened by the keyword at pos.
func llIsConstructorDecl(s *tokens.Stream, pos, open int) bool {
	if s.At(pos).Kind != token.Keyword || !strings.EqualFold(s.At(pos).Value, "function") {
		return false
	}
	name := sigPrev(s, open)
	return name > pos && s.At(name).Kind == token.Ident && strings.EqualFold(s.At(name).Value, "__construct")
}

func llHasPromotedProperty(s *tokens.Stream, open, closeIdx int) bool {
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private", "readonly":
				return true
			}
		}
	}
	return false
}

func llIsAttribute(t token.Token) bool {
	return t.Kind == token.Comment && strings.HasPrefix(t.Value, "#[")
}

func llIsHeredoc(t token.Token) bool {
	return t.Kind == token.String && strings.HasPrefix(t.Value, "<<<")
}

func llIsClosureUse(s *tokens.Stream, pos int) bool {
	nx := sigNext(s, pos)
	return nx >= 0 && s.At(nx).Kind == token.Punct && s.At(nx).Value == "("
}

func llIsArrayClose(s *tokens.Stream, closeIdx int) bool {
	open := s.MatchBackward(closeIdx)
	return open >= 0 && isArrayLiteralOpen(s, open)
}

func llIsArrayOpenTok(s *tokens.Stream, i int) bool {
	t := s.At(i)
	return t.Kind == token.Punct && t.Value == "[" && isArrayLiteralOpen(s, i)
}

func llIsArrayCloseTok(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct || t.Value != "]" {
		return false
	}
	return llIsArrayClose(s, i)
}

func llIsArrayEdgeTok(s *tokens.Stream, i int) bool {
	t := s.At(i)
	return t.Kind == token.Punct && (t.Value == "[" || t.Value == "]")
}

// llClosesEmptyArray reports whether the "]" at i closes an empty "[]".
func llClosesEmptyArray(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	return open >= 0 && sigNext(s, open) == i
}

func llIsNewlineOrOpenTag(t token.Token) bool {
	return t.Kind == token.OpenTag || (len(t.Value) > 0 && t.Value[0] == '\n')
}

func llIsEndOfArgsLine(t token.Token) bool {
	if len(t.Value) > 0 && t.Value[0] == '\n' {
		return true
	}
	return t.Kind == token.Keyword && strings.EqualFold(t.Value, "use")
}
