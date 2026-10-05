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
	return l.toks
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
	if l.hasPrefix("<?php") || l.hasPrefix("<?=") {
		return true
	}
	rest := l.src[l.pos+2:]
	if len(rest) >= 3 && strings.EqualFold(rest[:3], "xml") {
		return false
	}
	return true
}

func (l *lexer) lexOpenTag() {
	start := l.pos
	switch {
	case l.hasPrefix("<?php"):
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
		l.emit(l.identKind(l.src[start:l.pos]), start)
	default:
		if op := l.matchOperator(); op > 0 {
			l.pos += op
		} else {
			l.pos++
		}
		l.emit(token.Punct, start)
	}
}

// identKind classifies an identifier run as Keyword or Ident. A name that
// directly follows an object/static access operator (-> ?-> ::) is always a
// property or method name, never a keyword.
func (l *lexer) identKind(word string) token.Kind {
	prev := l.lastSignificant()
	if prev == "->" || prev == "?->" || prev == "::" || prev == `\` {
		return token.Ident // property/method name or namespace segment
	}
	// a name right after "function" or "const" (e.g. a method named "match") is
	// an identifier, not the keyword it happens to spell
	if lp := strings.ToLower(prev); lp == "function" || lp == "const" {
		return token.Ident
	}
	// a namespace segment like "Enum\Action" is an identifier, not a keyword
	if l.pos < len(l.src) && l.src[l.pos] == '\\' {
		return token.Ident
	}
	if keywords[strings.ToLower(word)] {
		return token.Keyword
	}
	return token.Ident
}

// lastSignificant returns the value of the last non-trivia token emitted so far.
func (l *lexer) lastSignificant() string {
	for _, tk := range slices.Backward(l.toks) {
		switch tk.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		default:
			return tk.Value
		}
	}
	return ""
}

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
	// /** ... */ is a doc comment, but /**/ is not
	if strings.HasPrefix(val, "/**") && val != "/**/" {
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

// lexAttribute consumes a "#[ ... ]" attribute as one opaque token, matching
// brackets across lines and skipping string and heredoc/nowdoc content inside
// (so a multiline nowdoc in an attribute never leaks out as code).
func (l *lexer) lexAttribute(start int) {
	l.pos += 2 // consume "#["
	depth := 1
	for l.pos < len(l.src) && depth > 0 {
		switch c := l.src[l.pos]; {
		case c == '[':
			depth++
			l.pos++
		case c == ']':
			depth--
			l.pos++
		case c == '\'' || c == '"':
			l.skipString(c)
		case l.hasPrefix("<<<"):
			if end := l.scanHeredoc(); end > l.pos {
				l.pos = end
			} else {
				l.pos++
			}
		default:
			l.pos++
		}
	}
	l.emit(token.Comment, start)
}

// skipString advances past a single- or double-quoted string, honoring escapes.
func (l *lexer) skipString(quote byte) {
	l.pos++ // opening quote
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			l.pos += 2
			continue
		}
		l.pos++
		if c == quote {
			return
		}
	}
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
	"**", "<<", ">>",
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
