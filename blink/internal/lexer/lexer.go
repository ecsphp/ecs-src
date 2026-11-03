// Package lexer is a small, self-contained PHP tokenizer producing a flat,
// lossless token stream. It mirrors the shape of PHP's token_get_all rather
// than building an AST, so fixers can walk and mutate tokens by index.
//
// It classifies keywords vs names, emits multi-char operators (=> -> :: === ...)
// as single Punct tokens, and captures heredoc/nowdoc as one opaque String
// token. It is a pragmatic subset, not a full PHP grammar: string interpolation
// is not split and casts are not tokenized. Enough for coding-standard fixers.
package lexer

import (
	"slices"
	"strings"

	"blink/internal/token"
)

// Lex converts source into a flat token slice. Concatenating the Value of each
// returned token reproduces src exactly.
func Lex(src string) []token.Token {
	// most tokens are a few bytes; presize to avoid repeated slice regrowth
	l := &lexer{src: src, toks: make([]token.Token, 0, len(src)/3+8)}
	l.run()
	markAttributeCloses(l.toks)
	return l.toks
}

// markAttributeCloses retags the "]" that closes each "#[" attribute as an
// AttributeEnd token, so generic bracket-depth scans treat neither attribute
// delimiter as an array bracket (mirroring php-cs-fixer's T_ATTRIBUTE_CLOSE).
func markAttributeCloses(toks []token.Token) {
	for i := range toks {
		if toks[i].Kind != token.AttributeStart {
			continue
		}
		depth := 1
		for j := i + 1; j < len(toks); j++ {
			if toks[j].Kind != token.Punct {
				continue
			}
			switch toks[j].Value {
			case "[":
				depth++
			case "]":
				depth--
				if depth == 0 {
					toks[j].Kind = token.AttributeEnd
					j = len(toks)
				}
			}
		}
	}
}

type lexer struct {
	src   string
	pos   int
	inPHP bool
	toks  []token.Token
}

func (l *lexer) emit(k token.Kind, start int) {
	l.toks = append(l.toks, token.Token{Kind: k, Value: l.src[start:l.pos], Pos: start})
}

func (l *lexer) run() {
	for l.pos < len(l.src) {
		if l.inPHP {
			l.lexPHP()
		} else {
			l.lexHTML()
		}
	}
}

func (l *lexer) lexHTML() {
	start := l.pos
	for {
		idx := strings.Index(l.src[l.pos:], "<?")
		if idx < 0 {
			l.pos = len(l.src)
			break
		}
		l.pos += idx
		if l.isOpenTagHere() {
			break
		}
		l.pos += 2 // a "<?" that is not a PHP tag (e.g. "<?xml") stays inline HTML
	}
	if l.pos > start {
		l.toks = append(l.toks, token.Token{Kind: token.InlineHTML, Value: l.src[start:l.pos], Pos: start})
	}
	if l.pos < len(l.src) {
		l.lexOpenTag()
		l.inPHP = true
	}
}

// isOpenTagHere reports whether l.pos starts a real PHP open tag. "<?php" and
// "<?=" always are; a bare "<?" short open tag is too, except "<?xml" which is
// an XML declaration and stays inline HTML.
func (l *lexer) isOpenTagHere() bool {
	if l.atFullOpenTag() || l.hasPrefix("<?=") {
		return true
	}
	rest := l.src[l.pos+2:]
	if len(rest) >= 3 && strings.EqualFold(rest[:3], "xml") {
		return false
	}
	return true
}

// atFullOpenTag reports whether l.pos starts the full open tag: "<?php" in
// any case, followed by whitespace or the end of the file, as PHP's scanner
// reads it ("<?php$x" is the short tag "<?" then "php").
func (l *lexer) atFullOpenTag() bool {
	rest := l.src[l.pos:]
	if len(rest) < len("<?php") || !strings.EqualFold(rest[:len("<?php")], "<?php") {
		return false
	}
	return len(rest) == len("<?php") || isSpace(rest[len("<?php")])
}

func (l *lexer) lexOpenTag() {
	start := l.pos
	switch {
	case l.atFullOpenTag():
		l.pos += len("<?php")
	case l.hasPrefix("<?="):
		l.pos += len("<?=")
	default:
		l.pos += len("<?")
	}
	l.emit(token.OpenTag, start)
}

func (l *lexer) lexPHP() {
	start := l.pos
	c := l.src[l.pos]

	switch {
	case l.hasPrefix("?>"):
		l.pos += 2
		l.emit(token.CloseTag, start)
		l.inPHP = false
	case l.hasPrefix("<<<"):
		if end := l.scanHeredoc(); end > l.pos {
			l.pos = end
			l.emit(token.String, start)
		} else {
			if op := l.matchOperator(); op > 0 {
				l.pos += op
			} else {
				l.pos++
			}
			l.emit(token.Punct, start)
		}
	case isSpace(c):
		for l.pos < len(l.src) && isSpace(l.src[l.pos]) {
			l.pos++
		}
		l.emit(token.Whitespace, start)
	case c == '$':
		l.pos++
		for l.pos < len(l.src) && isIdent(l.src[l.pos]) {
			l.pos++
		}
		l.emit(token.Variable, start)
	case l.hasPrefix("#["):
		l.lexAttribute(start)
	case l.hasPrefix("//") || c == '#':
		l.lexLineComment(start)
	case l.hasPrefix("/*"):
		l.lexBlockComment(start)
	case c == '\'':
		l.lexString(start, '\'')
	case c == '"':
		l.lexString(start, '"')
	case isDigit(c) || (c == '.' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1])):
		// a leading-dot float (".1") is one DNUMBER in PHP's context-free scanner
		for l.pos < len(l.src) && isNumber(l.src[l.pos]) {
			l.pos++
		}
		l.emit(token.Number, start)
	case isIdentStart(c):
		for l.pos < len(l.src) && isIdent(l.src[l.pos]) {
			l.pos++
		}
		kind := l.identKind(l.src[start:l.pos])
		if kind == token.Keyword && l.atSetModifierSuffix(l.src[start:l.pos]) {
			l.pos += len("(set)")
		}
		l.emit(kind, start)
	default:
		if op := l.matchOperator(); op > 0 {
			l.pos += op
		} else {
			l.pos++
		}
		l.emit(token.Punct, start)
	}
}

// identKind classifies an identifier run as Keyword or Ident. A reserved word
// where PHP only accepts a name is an identifier, as token_get_all(TOKEN_PARSE)
// reports it (T_STRING).
func (l *lexer) identKind(word string) token.Kind {
	lw := strings.ToLower(word)
	if !keywords[lw] {
		return token.Ident
	}
	prev := l.lastSignificant()
	if prev == "->" || prev == "?->" || prev == "::" || prev == `\` {
		return token.Ident // property/method name or namespace segment
	}
	lp := strings.ToLower(prev)
	// a function or method name, also "function &list()"
	if lp == "function" || (lp == "&" && strings.EqualFold(l.significantBefore(1), "function")) {
		return token.Ident
	}
	// a namespace segment like "Enum\Action" is an identifier, not a keyword
	if l.pos < len(l.src) && l.src[l.pos] == '\\' {
		return token.Ident
	}
	next := l.skipTrivia(l.pos)
	// a constant or enum case name: no reserved word is followed by a single "="
	if l.byteAt(next) == '=' {
		if c := l.byteAt(next + 1); c != '=' && c != '>' {
			// TOKEN_PARSE retags the last token spelled like the constant name,
			// so "const PUBLIC = A::PUBLIC" keeps T_PUBLIC
			if lp != "case" && l.wordInValue(next+1, word) {
				return token.Keyword
			}
			return token.Ident
		}
	}
	// a constant name after "const", unless it is the type of "const array LIST"
	if lp == "const" && !isIdentStart(l.byteAt(next)) && l.byteAt(next) != '|' && l.byteAt(next) != '&' {
		return token.Ident
	}
	// a named argument "f(else: 1)"
	if (prev == "(" || prev == ",") && l.byteAt(next) == ':' && l.byteAt(next+1) != ':' {
		return token.Ident
	}
	// an enum case "case Default;"
	if lp == "case" && l.byteAt(next) == ';' {
		return token.Ident
	}
	// a trait method alias "foo as list;" or "foo as protected list;"
	if l.byteAt(next) == ';' {
		if lp == "as" && !memberModifiers[lw] {
			return token.Ident
		}
		if memberModifiers[lp] && strings.EqualFold(l.significantBefore(1), "as") {
			return token.Ident
		}
	}
	// the method of a trait alias "{ list as protected; }"
	if (prev == "{" || prev == ";") && l.wordAt(next, "as") {
		return token.Ident
	}
	// a call to a function named readonly, unless "(" opens a DNF type
	if lw == "readonly" && l.byteAt(next) == '(' && !l.dnfTypeAt(next) {
		return token.Ident
	}
	// "enum" is the keyword only before a name ("enum Suit", not "Enum::from()")
	if lw == "enum" && !l.enumKeywordAt(next) {
		return token.Ident
	}
	return token.Keyword
}

// memberModifiers are the modifiers PHP accepts in a trait method alias
// ("foo as protected bar;").
var memberModifiers = map[string]bool{
	"public": true, "protected": true, "private": true, "static": true,
	"abstract": true, "final": true, "readonly": true,
}

// significantBefore returns the value of the non-trivia token emitted n
// tokens before the last one (0 is lastSignificant), or "" when there is none.
func (l *lexer) significantBefore(n int) string {
	for _, tk := range slices.Backward(l.toks) {
		switch tk.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		if n == 0 {
			return tk.Value
		}
		n--
	}
	return ""
}

// wordAt reports whether the source at p spells word, in any case, as a whole name.
func (l *lexer) wordAt(p int, word string) bool {
	if p+len(word) > len(l.src) || !strings.EqualFold(l.src[p:p+len(word)], word) {
		return false
	}
	return !isIdent(l.byteAt(p + len(word)))
}

// wordInValue reports whether word (exact case) appears as a whole name in the
// constant value starting at p, which ends at the first "," or ";" outside
// brackets.
func (l *lexer) wordInValue(p int, word string) bool {
	depth := 0
	for p < len(l.src) {
		switch c := l.src[p]; {
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			if depth == 0 {
				return false
			}
			depth--
		case (c == ',' || c == ';') && depth == 0:
			return false
		case c == '\'' || c == '"':
			for p++; p < len(l.src) && l.src[p] != c; p++ {
				if l.src[p] == '\\' {
					p++
				}
			}
		case c == '/' && (l.byteAt(p+1) == '*' || l.byteAt(p+1) == '/'), c == '#':
			p = l.skipTrivia(p)
			continue
		case isIdentStart(c):
			start := p
			for p < len(l.src) && isIdent(l.src[p]) {
				p++
			}
			if l.src[start:p] == word {
				return true
			}
			continue
		}
		p++
	}
	return false
}

// dnfTypeAt reports whether the "(" at p opens the conjunction of a DNF type,
// "(A&B)|C": names, "\" and "&" up to ")", then "|".
func (l *lexer) dnfTypeAt(p int) bool {
	for p++; p < len(l.src) && l.src[p] != ')'; p++ {
		if c := l.src[p]; !isIdent(c) && !isSpace(c) && c != '\\' && c != '&' {
			return false
		}
	}
	return l.byteAt(l.skipTrivia(p+1)) == '|'
}

// enumKeywordAt reports whether "enum", ending at l.pos, is T_ENUM: PHP's scanner
// wants whitespace or comments, then a name other than extends or implements.
func (l *lexer) enumKeywordAt(next int) bool {
	if next == l.pos || next >= len(l.src) || !isIdentStart(l.src[next]) {
		return false
	}
	rest := l.src[next:]
	for _, kw := range []string{"extends", "implements"} {
		if len(rest) >= len(kw) && strings.EqualFold(rest[:len(kw)], kw) {
			return false
		}
	}
	return true
}

// skipTrivia returns the offset of the first byte at or after p that is not
// whitespace or part of a comment.
func (l *lexer) skipTrivia(p int) int {
	src := l.src
	for p < len(src) {
		switch {
		case isSpace(src[p]):
			p++
		case strings.HasPrefix(src[p:], "/*"):
			end := strings.Index(src[p+2:], "*/")
			if end < 0 {
				return len(src)
			}
			p += 2 + end + 2
		case strings.HasPrefix(src[p:], "//") || (src[p] == '#' && !strings.HasPrefix(src[p:], "#[")):
			for p < len(src) && src[p] != '\n' && !strings.HasPrefix(src[p:], "?>") {
				p++
			}
		default:
			return p
		}
	}
	return p
}

// byteAt returns the source byte at p, or 0 past the end.
func (l *lexer) byteAt(p int) byte {
	if p < 0 || p >= len(l.src) {
		return 0
	}
	return l.src[p]
}

// atSetModifierSuffix reports whether the visibility keyword word is directly
// followed by "(set)": PHP 8.4 lexes "private(set)" as one token (T_PRIVATE_SET).
func (l *lexer) atSetModifierSuffix(word string) bool {
	switch strings.ToLower(word) {
	case "public", "protected", "private":
		return l.pos+len("(set)") <= len(l.src) && strings.EqualFold(l.src[l.pos:l.pos+len("(set)")], "(set)")
	}
	return false
}

// lastSignificant returns the value of the last non-trivia token emitted so far.
func (l *lexer) lastSignificant() string { return l.significantBefore(0) }

// matchOperator returns the byte length of the longest known multi-char operator
// at the current position, or 0 when none matches (single-char punctuation).
func (l *lexer) matchOperator() int {
	if l.pos >= len(l.src) {
		return 0
	}
	// only the operators starting with the current byte, longest first
	for _, op := range operatorsByByte[l.src[l.pos]] {
		if l.hasPrefix(op) {
			return len(op)
		}
	}
	return 0
}

func (l *lexer) lexLineComment(start int) {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		// a ?> ends a line comment in PHP; stop before it
		if l.hasPrefix("?>") {
			break
		}
		l.pos++
	}
	l.emit(token.Comment, start)
}

func (l *lexer) lexBlockComment(start int) {
	l.pos += 2 // consume /*
	for l.pos < len(l.src) && !l.hasPrefix("*/") {
		l.pos++
	}
	if l.hasPrefix("*/") {
		l.pos += 2
	}
	val := l.src[start:l.pos]
	kind := token.Comment
	// PHP's scanner reads "/**" followed by whitespace as a doc comment; "/**/",
	// "/***/" and "/**@var x*/" are plain comments
	if strings.HasPrefix(val, "/**") && len(val) > 3 && isSpace(val[3]) {
		kind = token.DocComment
	}
	l.toks = append(l.toks, token.Token{Kind: kind, Value: val, Pos: start})
}

func (l *lexer) lexString(start int, quote byte) {
	// double-quoted strings may carry a complex "{$...}" interpolation whose inner
	// expression is split out as real tokens (so array fixers can reflow it); the
	// literal chunks keep the quote, the "{" and the "}" so Render stays lossless.
	if quote == '"' {
		if l.lexInterpolatedString(start) {
			return
		}
	}
	l.pos++ // opening quote
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			l.pos += 2
			continue
		}
		if c == quote {
			l.pos++
			break
		}
		l.pos++
	}
	l.emit(token.String, start)
}

// lexInterpolatedString splits a double-quoted string that contains a complex
// "{$...}" interpolation into: a literal chunk ending with "{", the inner
// expression's real tokens, then a chunk starting with "}", repeated, then the
// closing chunk. It returns false (emitting nothing) when the string has no "{$"
// so the caller lexes it as one opaque token.
func (l *lexer) lexInterpolatedString(start int) bool {
	// first pass: does the string contain a top-level "{$" before its close?
	p := l.pos + 1
	hasInterp := false
	for p < len(l.src) {
		c := l.src[p]
		if c == '\\' && p+1 < len(l.src) {
			p += 2
			continue
		}
		if c == '"' {
			break
		}
		if c == '{' && p+1 < len(l.src) && l.src[p+1] == '$' {
			hasInterp = true
			break
		}
		p++
	}
	if !hasInterp {
		return false
	}

	l.pos++ // opening quote
	chunkStart := start
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			l.pos += 2
			continue
		}
		if c == '"' {
			l.pos++ // closing quote
			break
		}
		if c == '{' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '$' {
			l.pos++ // consume "{", keeping it in this chunk
			l.emit(token.String, chunkStart)
			// lex the inner expression as real tokens up to the matching "}"
			depth := 1
			for l.pos < len(l.src) {
				if l.src[l.pos] == '}' && depth == 1 {
					break // interpolation close; starts the next chunk
				}
				before := len(l.toks)
				l.lexPHP()
				if len(l.toks) == before {
					break // defensive: no progress
				}
				tk := l.toks[len(l.toks)-1]
				if tk.Kind == token.Punct {
					switch tk.Value {
					case "{":
						depth++
					case "}":
						depth--
					}
				}
			}
			chunkStart = l.pos // the "}" begins the next literal chunk
			continue
		}
		l.pos++
	}
	if l.pos > chunkStart {
		l.emit(token.String, chunkStart)
	}
	return true
}

func (l *lexer) hasPrefix(s string) bool {
	return strings.HasPrefix(l.src[l.pos:], s)
}

// lexAttribute emits the "#[" that opens an attribute as one token; its contents
// (and the closing "]") then lex as ordinary PHP tokens, so fixers see the arrays,
// named arguments and operators inside, matching php-cs-fixer's T_ATTRIBUTE.
func (l *lexer) lexAttribute(start int) {
	l.pos += 2 // consume "#["
	l.emit(token.AttributeStart, start)
}

// scanHeredoc returns the end offset of a heredoc/nowdoc starting at "<<<", or
// -1 when the "<<<" is not a valid heredoc opener. The whole construct (through
// the closing label) is treated as one opaque token so fixers never touch it.
func (l *lexer) scanHeredoc() int {
	src := l.src
	p := l.pos + 3
	for p < len(src) && (src[p] == ' ' || src[p] == '\t') {
		p++
	}
	var quote byte
	if p < len(src) && (src[p] == '\'' || src[p] == '"') {
		quote = src[p]
		p++
	}
	labelStart := p
	for p < len(src) && isIdent(src[p]) {
		p++
	}
	if p == labelStart {
		return -1 // no label -> not a heredoc
	}
	label := src[labelStart:p]
	if quote != 0 {
		if p >= len(src) || src[p] != quote {
			return -1
		}
		p++
	}
	// opener must end the line
	for p < len(src) && src[p] != '\n' {
		p++
	}
	if p >= len(src) {
		return -1
	}
	// scan body lines for the closing label (PHP 7.3+ allows it indented)
	for p < len(src) {
		p++ // step past the newline to the start of the next line
		lineStart := p
		for lineStart < len(src) && (src[lineStart] == ' ' || src[lineStart] == '\t') {
			lineStart++
		}
		if strings.HasPrefix(src[lineStart:], label) {
			after := lineStart + len(label)
			if after >= len(src) || !isIdent(src[after]) {
				return after // closing marker found
			}
		}
		for p < len(src) && src[p] != '\n' {
			p++
		}
		if p >= len(src) {
			return len(src) // unterminated; consume the rest
		}
	}
	return len(src)
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isNumber(c byte) bool {
	return isDigit(c) || c == '.' || c == '_' || c == 'x' || c == 'X' || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}
func isIdent(c byte) bool { return isIdentStart(c) || isDigit(c) }

// operators lists multi-char operators, longest first, so matchOperator emits
// each as a single Punct token (=> -> :: == === etc.) instead of splitting it.
var operators = []string{
	"<=>", "===", "!==", "**=", "...", "<<=", ">>=", "??=", "?->",
	"->", "=>", "==", "!=", "<>", "<=", ">=", "&&", "||", "++", "--",
	"+=", "-=", "*=", "/=", ".=", "%=", "&=", "|=", "^=", "::", "??",
	"**", "<<", ">>", "|>",
}

// operatorsByByte groups operators by their first byte, preserving the
// longest-first order of `operators`, so matchOperator only tests the few
// candidates for the current byte instead of all of them.
var operatorsByByte = func() [256][]string {
	var m [256][]string
	for _, op := range operators {
		m[op[0]] = append(m[op[0]], op)
	}
	return m
}()

// keywords are PHP reserved words (case-insensitive). Type names (int, string,
// ...) and constants (true, false, null) are T_STRING in PHP and stay Ident.
var keywords = map[string]bool{
	"abstract": true, "and": true, "array": true, "as": true, "break": true,
	"callable": true, "case": true, "catch": true, "class": true, "clone": true,
	"const": true, "continue": true, "declare": true, "default": true, "do": true,
	"echo": true, "else": true, "elseif": true, "empty": true, "enddeclare": true,
	"endfor": true, "endforeach": true, "endif": true, "endswitch": true,
	"endwhile": true, "enum": true, "extends": true, "final": true, "finally": true,
	"fn": true, "for": true, "foreach": true, "function": true, "global": true,
	"goto": true, "if": true, "implements": true, "include": true,
	"include_once": true, "instanceof": true, "insteadof": true, "interface": true,
	"isset": true, "list": true, "match": true, "namespace": true, "new": true,
	"or": true, "print": true, "private": true, "protected": true, "public": true,
	"readonly": true, "require": true, "require_once": true, "return": true,
	"static": true, "switch": true, "throw": true, "trait": true, "try": true,
	"unset": true, "use": true, "var": true, "while": true, "xor": true,
	"yield": true,
}
