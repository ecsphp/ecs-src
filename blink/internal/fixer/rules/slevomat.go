package rules

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// slevomat sniffs run on a PHP_CodeSniffer file, so their ports run on slvFile:
// the token list PHP_CodeSniffer builds (multi-line tokens split per line, a
// line comment owning its newline, interpolated strings kept whole) with the
// line, column, length and tab-expanded content ECS's tabWidth of 4 gives it,
// plus PHP_CodeSniffer's fixer. slvRun loops a sniff like ECS does, until a
// pass fixes nothing.

const slvTabWidth = 4

type slvTok struct {
	kind    token.Kind
	content string // PHP_CodeSniffer's content: tabs expanded to spaces
	orig    string // the source text
	line    int
	column  int
	length  int  // display length without the newline
	heredoc bool // a line of a heredoc or nowdoc
	closer  int  // the matching bracket of "(", "[", "{", "#[" and back, else -1
	doc     slvDocKind
}

type slvFile struct {
	toks        []slvTok
	fixed       []string
	touched     []bool
	changeset   []slvChange
	inChangeset bool
	inConflict  bool
	numFixes    int
	eol         string
	loop        int
	history     map[int]*slvHistory // Fixer::$oldTokenValues, kept across passes
}

type slvChange struct {
	i int
	v string
}

type slvHistory struct {
	curr, prev string
	loop       int
}

const slvMaxLoops = 50

// slvRun runs process over the stream's PHP_CodeSniffer file until a pass fixes
// nothing, relexing the stream after every pass that did.
func slvRun(s *tokens.Stream, process func(f *slvFile)) bool {
	changed := false
	src := s.Render()
	history := map[int]*slvHistory{}
	for loop := range slvMaxLoops {
		f := newSlvFile(lexer.Lex(src))
		f.loop, f.history = loop, history
		process(f)
		if f.numFixes == 0 {
			break
		}
		next := f.contents()
		if next == src {
			break
		}
		src = next
		changed = true
	}
	if changed {
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(src))
	}
	return changed
}

// slvEachToken calls process for every token match accepts, skipping the tokens
// before the pointer process returns, like ECS's File::process does.
func (f *slvFile) eachToken(match func(i int) bool, process func(i int) int) {
	until := 0
	for i := range f.toks {
		if i < until || !match(i) {
			continue
		}
		until = process(i)
	}
}

func newSlvFile(toks []token.Token) *slvFile {
	f := &slvFile{eol: "\n"}
	f.toks = slvSplit(toks)
	f.position()
	f.matchBrackets()
	f.fixed = make([]string, len(f.toks))
	f.touched = make([]bool, len(f.toks))
	for i, t := range f.toks {
		f.fixed[i] = t.orig
	}
	f.eol = slvRawEOL(toks)
	return f
}

// slvSplit turns blink tokens into PHP_CodeSniffer's: an interpolated string is
// one token, and every token holding a newline is split after each of them.
func slvSplit(toks []token.Token) []slvTok {
	var out []slvTok
	eol := slvRawEOL(toks)
	add := func(k token.Kind, v string, heredoc bool) {
		for v != "" {
			n := strings.IndexByte(v, '\n')
			if n < 0 || n == len(v)-1 {
				out = append(out, slvTok{kind: k, orig: v, heredoc: heredoc})
				return
			}
			out = append(out, slvTok{kind: k, orig: v[:n+1], heredoc: heredoc})
			v = v[n+1:]
		}
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if end := slvNameEnd(toks, i); end > i {
			var b strings.Builder
			for ; i <= end; i++ {
				b.WriteString(toks[i].Value)
			}
			i = end
			out = append(out, slvTok{kind: token.Ident, orig: b.String()})
			continue
		}
		if end := slvCastEnd(toks, i); end > i {
			var b strings.Builder
			for ; i <= end; i++ {
				b.WriteString(toks[i].Value)
			}
			i = end
			out = append(out, slvTok{kind: token.Punct, orig: b.String()})
			continue
		}
		switch {
		case t.Kind == token.String && t.Part == token.Head:
			var b strings.Builder
			for ; i < len(toks); i++ {
				b.WriteString(toks[i].Value)
				if toks[i].Kind == token.String && toks[i].Part == token.Tail {
					break
				}
			}
			v := b.String()
			add(token.String, v, slvIsHeredoc(v))
		case t.Kind == token.String:
			add(token.String, t.Value, slvIsHeredoc(t.Value))
		case (t.Kind == token.Comment && !strings.HasPrefix(t.Value, "/*") || t.Kind == token.CloseTag) &&
			i+1 < len(toks) && (toks[i+1].Kind == token.Whitespace || toks[i+1].Kind == token.InlineHTML):
			// PHP ends a line comment and a closing tag with the newline after it
			ws := toks[i+1].Value
			nl := 0
			if strings.HasPrefix(ws, "\r\n") {
				nl = 2
			} else if strings.HasPrefix(ws, "\n") {
				nl = 1
			}
			out = append(out, slvTok{kind: t.Kind, orig: t.Value + ws[:nl]})
			if nl > 0 {
				i++
				add(toks[i].Kind, ws[nl:], false)
			}
		case slvIsDocBlock(t):
			// PHP_CodeSniffer splits a docblock into its tags, stars and texts
			out = append(out, slvDocTokens(t.Value, eol)...)
		default:
			add(t.Kind, t.Value, false)
		}
	}
	return out
}

// slvNameEnd returns the last token of a qualified name starting at i
// ("Foo\Bar", "\Foo", "namespace\Foo"), which PHP lexes as one token, or -1.
func slvNameEnd(toks []token.Token, i int) int {
	isSep := func(j int) bool { return j < len(toks) && toks[j].Kind == token.Punct && toks[j].Value == "\\" }
	isName := func(j int) bool {
		return j < len(toks) && (toks[j].Kind == token.Ident || toks[j].Kind == token.Keyword)
	}
	j := i
	if !isSep(j) {
		if !isName(j) || !isSep(j+1) {
			return -1
		}
		j++
	}
	end := -1
	for isSep(j) && isName(j+1) {
		end = j + 1
		j += 2
	}
	return end
}

// slvCastEnd returns the ")" of a cast starting at "(" at i, which PHP lexes as
// one token, or -1.
func slvCastEnd(toks []token.Token, i int) int {
	if toks[i].Kind != token.Punct || toks[i].Value != "(" {
		return -1
	}
	j := i + 1
	skip := func() {
		if j < len(toks) && toks[j].Kind == token.Whitespace && strings.Trim(toks[j].Value, " \t") == "" {
			j++
		}
	}
	skip()
	if j >= len(toks) || toks[j].Kind != token.Ident && toks[j].Kind != token.Keyword {
		return -1
	}
	switch strings.ToLower(toks[j].Value) {
	case "int", "integer", "bool", "boolean", "float", "double", "real",
		"string", "array", "object", "unset", "binary":
	default:
		return -1
	}
	j++
	skip()
	if j < len(toks) && toks[j].Kind == token.Punct && toks[j].Value == ")" {
		return j
	}
	return -1
}

// slvRawEOL is PHP_CodeSniffer's Common::detectLineEndings.
func slvRawEOL(toks []token.Token) string {
	for _, t := range toks {
		if n := strings.IndexAny(t.Value, "\r\n"); n >= 0 {
			if t.Value[n] == '\n' {
				return "\n"
			}
			if n+1 < len(t.Value) && t.Value[n+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		}
	}
	return "\n"
}

// slvDocKind is the PHP_CodeSniffer token type of a docblock piece.
type slvDocKind uint8

const (
	slvDocNone slvDocKind = iota
	slvDocOpen
	slvDocClose
	slvDocWhitespace
	slvDocStar
	slvDocTag
	slvDocString
)

// slvDocTokens splits a docblock like PHP_CodeSniffer's Comment tokenizer.
func slvDocTokens(comment, eol string) []slvTok {
	var out []slvTok
	add := func(v string, doc slvDocKind) {
		out = append(out, slvTok{kind: token.DocComment, orig: v, doc: doc})
	}
	n := len(comment)
	char := n - len(strings.TrimLeft(comment, "/*"))
	last := ""
	if n >= 2 {
		last = comment[n-2:]
	}
	var openTag string
	if char == n && last == "*/" {
		openTag, comment = comment[:n-2], last
	} else {
		openTag, comment = comment[:char], strings.TrimLeft(comment, "/*")
	}
	add(openTag, slvDocOpen)
	closeTag := comment[len(strings.TrimRight(comment, "/*")):]
	body := strings.TrimRight(comment, "/*")
	ws := func(s string, start int) string {
		e := start
		for e < len(s) && (s[e] == ' ' || s[e] == '\t') {
			e++
		}
		return s[start:e]
	}
	lines := strings.Split(body, eol)
	for ln, line := range lines {
		if ln != len(lines)-1 {
			line += eol
		}
		c := 0
		if sp := ws(line, 0); sp != "" {
			add(sp, slvDocWhitespace)
			c += len(sp)
			if c == len(line) {
				break
			}
		}
		if line == "" {
			continue
		}
		if ln > 0 && line[c] == '*' {
			c++
			add("*", slvDocStar)
		}
		// processLine
		if sp := ws(line, c); sp != "" {
			add(sp, slvDocWhitespace)
			c += len(sp)
		}
		if c >= len(line) {
			continue
		}
		if line[c] == '@' {
			e := c + 1
			for e < len(line) && !strings.ContainsRune(" \t\n\r\v\f", rune(line[e])) {
				e++
			}
			tag := line[c:e]
			if !strings.HasPrefix(strings.ToLower(tag), "@phpcs:") {
				add(tag, slvDocTag)
				c = e
				if sp := ws(line, c); sp != "" {
					add(sp, slvDocWhitespace)
					c += len(sp)
				}
			}
		}
		end := len(line)
		eolAt := strings.Index(line[c:], eol)
		if eolAt < 0 {
			eolAt = end
		} else {
			eolAt += c
		}
		if eolAt > c {
			add(line[c:eolAt], slvDocString)
		}
		if eolAt != end {
			add(line[eolAt:eolAt+len(eol)], slvDocWhitespace)
		}
	}
	add(closeTag, slvDocClose)
	return out
}

// slvIsDocBlock reports whether PHP_CodeSniffer tokenizes t as a docblock,
// which includes a "/**" comment PHP lexes as a plain one ("/**text */").
func slvIsDocBlock(t token.Token) bool {
	return t.Kind == token.DocComment || t.Kind == token.Comment && strings.HasPrefix(t.Value, "/**") && t.Value != "/**/"
}

func slvIsHeredoc(v string) bool {
	_, rest := token.SplitBinaryPrefix(v)
	return strings.HasPrefix(rest, "<<<")
}

func slvTabbed(k token.Kind) bool {
	return k == token.Whitespace || k == token.Comment || k == token.DocComment || k == token.String
}

// position sets line, column, length and content like PHP_CodeSniffer's
// Tokenizer::createPositionMap with a tab width of 4.
func (f *slvFile) position() {
	line, col := 1, 1
	for i := range f.toks {
		t := &f.toks[i]
		t.line, t.column, t.closer = line, col, -1
		t.content, t.length = t.orig, utf8.RuneCountInString(t.orig)
		if slvTabbed(t.kind) && strings.Contains(t.orig, "\t") {
			t.content, t.length = slvReplaceTabs(t.orig, col)
		}
		col += t.length
		if strings.Contains(t.orig, "\n") {
			line++
			col = 1
			t.length--
			if strings.HasSuffix(t.orig, "\r\n") {
				t.length--
			}
		}
	}
}

// slvReplaceTabs is PHP_CodeSniffer's Tokenizer::replaceTabsInToken.
func slvReplaceTabs(v string, col int) (string, int) {
	if strings.Trim(v, "\t") == "" {
		n := len(v)
		length := (slvTabWidth - (col-1)%slvTabWidth) + slvTabWidth*(n-1)
		return strings.Repeat(" ", length), length
	}
	parts := strings.Split(v, "\t")
	var b strings.Builder
	length := 0
	for n, p := range parts {
		if p != "" {
			b.WriteString(p)
			l := utf8.RuneCountInString(p)
			col += l
			length += l
		}
		if n == len(parts)-1 {
			break
		}
		pad := slvTabWidth - (col+slvTabWidth-1)%slvTabWidth
		col += pad
		length += pad
		b.WriteString(strings.Repeat(" ", pad))
	}
	return b.String(), length
}

func (f *slvFile) matchBrackets() {
	var stack []int
	for i, t := range f.toks {
		switch {
		case t.kind == token.AttributeStart || t.kind == token.Punct && (t.orig == "(" || t.orig == "[" || t.orig == "{"):
			stack = append(stack, i)
		case t.kind == token.AttributeEnd || t.kind == token.Punct && (t.orig == ")" || t.orig == "]" || t.orig == "}"):
			if len(stack) == 0 {
				continue
			}
			o := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			f.toks[o].closer, f.toks[i].closer = i, o
		}
	}
}

func (f *slvFile) contents() string {
	var b strings.Builder
	for _, v := range f.fixed {
		b.WriteString(v)
	}
	return b.String()
}

func (f *slvFile) is(i int, v string) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Punct && f.toks[i].orig == v
}

func (f *slvFile) isKeyword(i int, v string) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Keyword && strings.EqualFold(f.toks[i].orig, v)
}

func (f *slvFile) isEmpty(i int) bool {
	k := f.toks[i].kind
	return k == token.Whitespace || k == token.Comment || k == token.DocComment
}

// nextEffective is TokenHelper::findNextEffective: the first non-whitespace,
// non-comment token from i (inclusive) before end, or -1.
func (f *slvFile) nextEffective(i, end int) int {
	end = min(end, len(f.toks))
	for ; i < end; i++ {
		if !f.isEmpty(i) {
			return i
		}
	}
	return -1
}

// prevEffective is TokenHelper::findPreviousEffective, searching back from i
// (inclusive).
func (f *slvFile) prevEffective(i int) int {
	for ; i >= 0; i-- {
		if !f.isEmpty(i) {
			return i
		}
	}
	return -1
}

// nextNonWhitespace is TokenHelper::findNextNonWhitespace.
func (f *slvFile) nextNonWhitespace(i, end int) int {
	end = min(end, len(f.toks))
	for ; i < end; i++ {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return -1
}

// prevNonWhitespace is TokenHelper::findPreviousNonWhitespace.
func (f *slvFile) prevNonWhitespace(i int) int {
	for ; i >= 0; i-- {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return -1
}

// firstOnLine is TokenHelper::findFirstTokenOnLine.
func (f *slvFile) firstOnLine(i int) int {
	if i == 0 {
		return 0
	}
	line := f.toks[i].line
	for i > 0 && f.toks[i-1].line == line {
		i--
	}
	return i
}

// lastOnLine is TokenHelper::findLastTokenOnLine.
func (f *slvFile) lastOnLine(i int) int {
	line := f.toks[i].line
	for i+1 < len(f.toks) && f.toks[i+1].line == line {
		i++
	}
	return i
}

// firstOnNextLine is TokenHelper::findFirstTokenOnNextLine, or -1.
func (f *slvFile) firstOnNextLine(i int) int {
	if i >= len(f.toks) {
		return -1
	}
	line := f.toks[i].line
	for i++; i < len(f.toks); i++ {
		if f.toks[i].line != line {
			return i
		}
	}
	return -1
}

// firstNonWhitespaceOnLine is TokenHelper::findFirstNonWhitespaceOnLine.
func (f *slvFile) firstNonWhitespaceOnLine(i int) int {
	first := f.firstOnLine(i)
	if n := f.nextNonWhitespace(first, len(f.toks)); n >= 0 && n <= i {
		return n
	}
	return i
}

// content is TokenHelper::getContent: the PHP_CodeSniffer content of the
// tokens start..end inclusive.
func (f *slvFile) content(start, end int) string {
	var b strings.Builder
	for i := start; i <= end && i < len(f.toks); i++ {
		b.WriteString(f.toks[i].content)
	}
	return b.String()
}

// indentation is IndentationHelper::getIndentation.
func (f *slvFile) indentation(i int) string {
	return f.content(f.firstOnLine(i), i-1)
}

// isShortArrayOpen tells a "[" PHP_CodeSniffer retokenizes as
// T_OPEN_SHORT_ARRAY from an array access "[".
func (f *slvFile) isShortArrayOpen(i int) bool {
	if !f.is(i, "[") || f.toks[i].closer < 0 {
		return false
	}
	p := f.prevEffective(i - 1)
	if p < 0 {
		return true
	}
	t := f.toks[p]
	switch t.kind {
	case token.Variable, token.Ident, token.String:
		return false
	case token.Punct:
		switch t.orig {
		case "]", "->", "?->":
			return false
		case "}":
			return f.closesScope(p)
		case ")":
			o := t.closer
			return o >= 0 && f.isControlParen(o)
		}
	}
	return true
}

// closesScope reports whether "}" at i closes a scope body rather than a
// dynamic name ($o->{...}, ${...}).
func (f *slvFile) closesScope(i int) bool {
	o := f.toks[i].closer
	if o < 0 {
		return false
	}
	p := f.prevEffective(o - 1)
	if p < 0 {
		return true
	}
	t := f.toks[p]
	return t.kind != token.Variable && (t.kind != token.Punct || t.orig != "->" && t.orig != "?->" && t.orig != "::" && t.orig != "$")
}

var slvControlKeywords = map[string]bool{
	"if": true, "elseif": true, "while": true, "for": true, "foreach": true,
	"switch": true, "catch": true, "declare": true, "match": true,
}

func (f *slvFile) isControlParen(open int) bool {
	p := f.prevEffective(open - 1)
	return p >= 0 && f.toks[p].kind == token.Keyword && slvControlKeywords[strings.ToLower(f.toks[p].orig)]
}

// arrayOpenClose is ArrayHelper::openClosePointers for a token in
// TokenHelper::ARRAY_TOKEN_CODES (T_ARRAY, T_OPEN_SHORT_ARRAY); ok is false for
// any other token.
func (f *slvFile) arrayOpenClose(i int) (open, closeIdx int, ok bool) {
	if f.isShortArrayOpen(i) {
		return i, f.toks[i].closer, true
	}
	if f.isKeyword(i, "array") {
		if o := f.nextEffective(i+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
			return o, f.toks[o].closer, true
		}
	}
	return 0, 0, false
}

func (f *slvFile) isArrayToken(i int) bool {
	_, _, ok := f.arrayOpenClose(i)
	return ok
}

// tokenContent is Fixer::getTokenContent.
func (f *slvFile) tokenContent(i int) string {
	if f.inChangeset {
		for _, c := range f.changeset {
			if c.i == i {
				return c.v
			}
		}
	}
	return f.fixed[i]
}

func (f *slvFile) beginChangeset() {
	f.inChangeset = true
	f.changeset = nil
}

// endChangeset is Fixer::endChangeset: it applies every change of the
// changeset or, when one is refused, rolls the applied ones back.
func (f *slvFile) endChangeset() {
	if f.inConflict {
		return
	}
	f.inChangeset = false
	var applied []slvChange
	for _, c := range f.changeset {
		prev := f.fixed[c.i]
		if !f.replaceToken(c.i, c.v) {
			for _, a := range applied {
				f.fixed[a.i] = a.v
				f.touched[a.i] = false
				f.numFixes--
			}
			break
		}
		applied = append(applied, slvChange{c.i, prev})
	}
	f.changeset = nil
}

// replaceToken is Fixer::replaceToken: outside a changeset a token takes one
// fix per pass, and a fix reverting the previous pass's fix of the token is a
// conflict that stops all fixing in this pass.
func (f *slvFile) replaceToken(i int, v string) bool {
	if f.inConflict {
		return false
	}
	if !f.inChangeset && f.touched[i] {
		return false
	}
	if f.inChangeset {
		for n := range f.changeset {
			if f.changeset[n].i == i {
				f.changeset[n].v = v
				return true
			}
		}
		f.changeset = append(f.changeset, slvChange{i, v})
		return true
	}
	if h, ok := f.history[i]; !ok {
		f.history[i] = &slvHistory{curr: v, prev: f.fixed[i], loop: f.loop}
	} else {
		if h.prev == v && h.loop == f.loop-1 {
			f.inConflict = true
			return false
		}
		h.prev, h.curr, h.loop = h.curr, v, f.loop
	}
	f.fixed[i] = v
	f.touched[i] = true
	f.numFixes++
	return true
}

var slvLeadingSpaces = regexp.MustCompile(`(?m)^ +`)

// slvSpacesToTabs is IndentationHelper::convertSpacesToTabs with a tab width of 4.
func slvSpacesToTabs(v string) string {
	if !strings.Contains(v, " ") {
		return v
	}
	return slvLeadingSpaces.ReplaceAllStringFunc(v, func(m string) string {
		return strings.Repeat("\t", len(m)/slvTabWidth) + m[:len(m)%slvTabWidth]
	})
}

// replace, add, addBefore and change are slevomat's FixerHelper.
func (f *slvFile) replace(i int, v string) { f.replaceToken(i, slvSpacesToTabs(v)) }

func (f *slvFile) add(i int, v string) {
	f.replaceToken(i, f.tokenContent(i)+slvSpacesToTabs(v))
}

func (f *slvFile) addBefore(i int, v string) {
	f.replaceToken(i, slvSpacesToTabs(v)+f.tokenContent(i))
}

func (f *slvFile) change(start, end int, v string) {
	for i := start; i <= end; i++ {
		f.replaceToken(i, "")
	}
	f.replace(start, v)
}

// enclosing returns the innermost "(", "[", "{" or "#[" whose pair holds i, or -1.
func (f *slvFile) enclosing(i int) int {
	for j := i - 1; j >= 0; j-- {
		t := f.toks[j]
		if t.closer > i && (t.kind == token.AttributeStart || t.kind == token.Punct && (t.orig == "(" || t.orig == "[" || t.orig == "{")) {
			return j
		}
	}
	return -1
}

// enclosingScope returns the innermost scope "{" holding i, or -1.
func (f *slvFile) enclosingScope(i int) int {
	for j := f.enclosing(i); j >= 0; j = f.enclosing(j) {
		if f.is(j, "{") && f.closesScope(f.toks[j].closer) {
			return j
		}
	}
	return -1
}

// braceOwner returns the keyword a scope "{" belongs to (function, class,
// match, if, ...), or -1.
func (f *slvFile) braceOwner(open int) int {
	for j := open - 1; j >= 0; j-- {
		t := f.toks[j]
		if t.kind == token.Variable {
			return -1
		}
		if t.kind == token.Punct {
			switch t.orig {
			case ")", "]":
				if t.closer >= 0 {
					j = t.closer
					continue
				}
			case ";", "{", "}", "(", "[", "->", "?->", "::", "=":
				return -1
			}
		}
		if t.kind == token.Keyword {
			switch strings.ToLower(t.orig) {
			case "function", "class", "interface", "trait", "enum", "match", "switch",
				"if", "elseif", "else", "for", "foreach", "while", "do", "try", "catch",
				"finally", "declare", "namespace":
				return j
			}
		}
	}
	return -1
}

// isTraitUseBlock tells the "{" of a trait use's block.
func (f *slvFile) isTraitUseBlock(open int) bool {
	for j := open - 1; j >= 0; j-- {
		switch {
		case f.isEmpty(j), f.isName(j), f.is(j, ","):
			continue
		case f.isKeyword(j, "use"):
			return true
		}
		return false
	}
	return false
}

// scopeCondition is PHP_CodeSniffer's scope_condition: it returns the owner of the brace at i: "use" for a
// trait use block, else the keyword braceOwner finds, or -1.
func (f *slvFile) scopeCondition(i int) int {
	open := i
	if f.is(i, "}") {
		open = f.toks[i].closer
	}
	if open < 0 {
		return -1
	}
	if f.isTraitUseBlock(open) {
		for j := open - 1; j >= 0; j-- {
			if f.isKeyword(j, "use") {
				return j
			}
		}
	}
	return f.braceOwner(open)
}

// isClosure tells a closure's "function" from a named function's.
func (f *slvFile) isClosure(fn int) bool {
	n := f.nextEffective(fn+1, len(f.toks))
	if f.is(n, "&") {
		n = f.nextEffective(n+1, len(f.toks))
	}
	return f.is(n, "(")
}

// inClosureBody reports whether a closure is the innermost scope holding i.
func (f *slvFile) inClosureBody(i int) bool {
	scope := f.enclosingScope(i)
	if scope < 0 {
		return false
	}
	owner := f.braceOwner(scope)
	return owner >= 0 && f.isKeyword(owner, "function") && f.isClosure(owner)
}

// isDoubleArrow tells T_DOUBLE_ARROW from the "=>" of an arrow function
// (T_FN_ARROW) or a match arm (T_MATCH_ARROW).
func (f *slvFile) isDoubleArrow(i int) bool {
	if f.fnOfArrow(i) >= 0 {
		return false
	}
	if e := f.enclosing(i); e >= 0 && f.is(e, "{") {
		if owner := f.braceOwner(e); owner >= 0 && f.isKeyword(owner, "match") {
			return false
		}
	}
	return true
}

// fnOfArrow returns the "fn" keyword whose "=>" is at arrow, or -1.
func (f *slvFile) fnOfArrow(arrow int) int {
	j := f.prevEffective(arrow - 1)
	if !f.is(j, ")") || f.isTypeParen(j) {
		// a return type sits between ")" and "=>"
		for j >= 0 && !f.is(j, ":") {
			t := f.toks[j]
			switch {
			case t.kind == token.Ident, t.kind == token.Keyword && !f.isKeyword(j, "fn"),
				t.kind == token.Punct && (t.orig == "?" || t.orig == "|" || t.orig == "&" || t.orig == "\\"):
			case f.is(j, ")") && t.closer >= 0:
				j = t.closer
			default:
				return -1
			}
			j = f.prevEffective(j - 1)
		}
		if j < 0 {
			return -1
		}
		j = f.prevEffective(j - 1)
	}
	if !f.is(j, ")") || f.toks[j].closer < 0 {
		return -1
	}
	fn := f.prevEffective(f.toks[j].closer - 1)
	if f.is(fn, "&") {
		fn = f.prevEffective(fn - 1)
	}
	if f.isKeyword(fn, "fn") {
		return fn
	}
	return -1
}

// isTypeParen tells the ")" of a DNF type group ("(A&B)|null") from the ")" of
// an arrow function's parameters.
func (f *slvFile) isTypeParen(closeIdx int) bool {
	o := f.toks[closeIdx].closer
	if o < 0 {
		return false
	}
	p := f.prevEffective(o - 1)
	return f.is(p, "|") || f.is(p, ":")
}

// scopeCloser returns the scope_closer PHP_CodeSniffer gives the token at i,
// or -1 when it has none.
func (f *slvFile) scopeCloser(i int) int {
	t := f.toks[i]
	if f.is(i, "{") {
		if t.closer >= 0 && f.closesScope(t.closer) {
			return t.closer
		}
		return -1
	}
	if t.kind != token.Keyword {
		return -1
	}
	switch strings.ToLower(t.orig) {
	case "fn":
		return f.arrowEnd(i)
	case "function", "class", "interface", "trait", "enum", "match", "switch",
		"if", "elseif", "else", "for", "foreach", "while", "do", "try", "catch",
		"finally", "declare":
		for j := i + 1; j < len(f.toks); j++ {
			switch {
			case f.is(j, "(") || f.is(j, "["):
				if f.toks[j].closer < 0 {
					return -1
				}
				j = f.toks[j].closer
			case f.is(j, "{"):
				return f.toks[j].closer
			case f.is(j, ";") || f.is(j, ":") && !f.isKeyword(i, "function"):
				return -1
			}
		}
	}
	return -1
}

// arrowEnd is the scope_closer PHP_CodeSniffer's tokenizer gives the arrow
// function at fn: the token ending its expression.
func (f *slvFile) arrowEnd(fn int) int {
	arrow := -1
	for j := fn + 1; j < len(f.toks); j++ {
		if f.is(j, "(") && f.toks[j].closer >= 0 {
			j = f.toks[j].closer
			continue
		}
		if f.is(j, "=>") {
			arrow = j
			break
		}
		if f.is(j, ";") || f.is(j, "{") {
			return -1
		}
	}
	if arrow < 0 {
		return -1
	}
	lastNonEmpty := func(j int) int {
		for k := j - 1; k > arrow; k-- {
			if !f.isEmpty(k) {
				return k
			}
		}
		return j
	}
	lastEnd := -1
	inTernary := false
	for j := arrow + 1; j < len(f.toks); j++ {
		t := f.toks[j]
		if f.is(j, "}") && t.closer >= 0 {
			if owner := f.braceOwner(t.closer); owner >= 0 && f.isKeyword(owner, "match") {
				if arrow < owner {
					continue
				}
				if lastEnd >= 0 {
					return lastEnd
				}
				return lastNonEmpty(j)
			}
		}
		if f.is(j, ":") && inTernary {
			inTernary = false
			continue
		}
		if t.kind == token.OpenTag || t.kind == token.CloseTag || t.kind == token.Punct && (t.orig == ":" || t.orig == "," || t.orig == ";" || t.orig == ")" || t.orig == "]" || t.orig == "}") {
			if lastEnd >= 0 && t.closer >= 0 && t.closer < arrow {
				return lastNonEmpty(j)
			}
			return j
		}
		if sc := f.scopeCloser(j); sc >= 0 && !f.isKeyword(j, "fn") {
			j = sc - 1
			continue
		}
		if (f.is(j, "(") || f.is(j, "[") || f.is(j, "{")) && t.closer >= 0 {
			j = t.closer
			lastEnd = j
			continue
		}
		if f.isKeyword(j, "array") || f.isKeyword(j, "list") {
			if o := f.nextEffective(j+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
				j = f.toks[o].closer
				lastEnd = j
				continue
			}
		}
		if f.is(j, "?") {
			inTernary = true
		}
	}
	return -1
}

// strnatcasecmp is PHP's strnatcasecmp (strnatcmp_ex with case folding).
func strnatcasecmp(a, b string) int { return strnatcmpEx(a, b, true) }

// strnatcmp is PHP's strnatcmp.
func strnatcmp(a, b string) int { return strnatcmpEx(a, b, false) }

func strnatcmpEx(a, b string, fold bool) int {
	if a == "" || b == "" {
		switch {
		case len(a) == len(b):
			return 0
		case len(a) > len(b):
			return 1
		}
		return -1
	}
	at := func(s string, i int) byte {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }
	isSpace := func(c byte) bool { return c == ' ' || c >= '\t' && c <= '\r' }
	upper := func(c byte) byte {
		if c >= 'a' && c <= 'z' {
			return c - 32
		}
		return c
	}
	ap, bp := 0, 0
	leading := true
	for {
		ca, cb := at(a, ap), at(b, bp)
		for leading && ca == '0' && ap+1 < len(a) && isDigit(a[ap+1]) {
			ap++
			ca = a[ap]
		}
		for leading && cb == '0' && bp+1 < len(b) && isDigit(b[bp+1]) {
			bp++
			cb = b[bp]
		}
		leading = false
		for isSpace(ca) {
			ap++
			ca = at(a, ap)
		}
		for isSpace(cb) {
			bp++
			cb = at(b, bp)
		}
		if isDigit(ca) && isDigit(cb) {
			var result int
			if ca == '0' || cb == '0' {
				result = natCompareLeft(a, &ap, b, &bp)
			} else {
				result = natCompareRight(a, &ap, b, &bp)
			}
			switch {
			case result != 0:
				return result
			case ap == len(a) && bp == len(b):
				return 0
			case ap == len(a):
				return -1
			case bp == len(b):
				return 1
			}
			ca, cb = a[ap], b[bp]
		}
		if fold {
			ca, cb = upper(ca), upper(cb)
		}
		if ca < cb {
			return -1
		}
		if ca > cb {
			return 1
		}
		ap++
		bp++
		switch {
		case ap >= len(a) && bp >= len(b):
			return 0
		case ap >= len(a):
			return -1
		case bp >= len(b):
			return 1
		}
	}
}

func natDigitAt(s string, i int) bool { return i < len(s) && s[i] >= '0' && s[i] <= '9' }

func natCompareRight(a string, ap *int, b string, bp *int) int {
	bias := 0
	for ; ; *ap, *bp = *ap+1, *bp+1 {
		da, db := natDigitAt(a, *ap), natDigitAt(b, *bp)
		switch {
		case !da && !db:
			return bias
		case !da:
			return -1
		case !db:
			return 1
		case a[*ap] < b[*bp]:
			if bias == 0 {
				bias = -1
			}
		case a[*ap] > b[*bp]:
			if bias == 0 {
				bias = 1
			}
		}
	}
}

func natCompareLeft(a string, ap *int, b string, bp *int) int {
	for ; ; *ap, *bp = *ap+1, *bp+1 {
		da, db := natDigitAt(a, *ap), natDigitAt(b, *bp)
		switch {
		case !da && !db:
			return 0
		case !da:
			return -1
		case !db:
			return 1
		case a[*ap] < b[*bp]:
			return -1
		case a[*ap] > b[*bp]:
			return 1
		}
	}
}
