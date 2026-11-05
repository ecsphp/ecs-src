package rules

import (
	"strings"
	"unicode"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/SuperfluousWhitespaceSniff.php
//
// SuperfluousWhitespace removes four kinds of superfluous whitespace: leading
// whitespace before the file's first "<?php"/"<?" tag, whitespace after the
// last content of the file, whitespace at the end of any line (inside comments
// and docblocks too), and runs of more than one blank line inside a function or
// closure body (anonymous-class bodies keep their own spacing).
type SuperfluousWhitespace struct{}

func (SuperfluousWhitespace) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\SuperfluousWhitespaceSniff`
}

func (SuperfluousWhitespace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/SuperfluousWhitespaceSniff.php"
}

func (SuperfluousWhitespace) Fix(s *tokens.Stream) bool {
	// end-of-line trimming first so the blank-line collapse sees clean newlines
	changed := superfluousFixEndOfLines(s)
	if superfluousFixEmptyLines(s) {
		changed = true
	}
	if superfluousFixStartFile(s) {
		changed = true
	}
	if superfluousFixEndFile(s) {
		changed = true
	}
	s.ClearEmptyTokens()
	return changed
}

// superfluousFixStartFile drops whitespace-only inline HTML before the file's
// first "<?php"/"<?" tag (not "<?=", which the sniff leaves alone).
func superfluousFixStartFile(s *tokens.Stream) bool {
	open := -1
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.OpenTag {
			open = i
			break
		}
	}
	if open <= 0 || s.At(open).Value == "<?=" {
		return false
	}
	var b strings.Builder
	for i := 0; i < open; i++ {
		if s.At(i).Kind != token.InlineHTML {
			return false
		}
		b.WriteString(s.At(i).Value)
	}
	if !superfluousAllSpace(b.String()) {
		return false
	}
	for i := 0; i < open; i++ {
		s.SetValue(i, "")
	}
	return true
}

// superfluousFixEndFile drops whitespace-only inline HTML after the file's last
// close tag. The close tag keeps the single line break right after it, as PHP's
// tokenizer attaches that newline to the "?>" token.
func superfluousFixEndFile(s *tokens.Stream) bool {
	closeIdx := -1
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind == token.CloseTag {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 || closeIdx == s.Len()-1 {
		return false
	}
	var b strings.Builder
	for i := closeIdx + 1; i < s.Len(); i++ {
		if s.At(i).Kind != token.InlineHTML {
			return false
		}
		b.WriteString(s.At(i).Value)
	}
	after := b.String()
	if after == "" || !superfluousAllSpace(after) {
		return false
	}
	kept := ""
	switch {
	case strings.HasPrefix(after, "\r\n"):
		kept = "\r\n"
	case strings.HasPrefix(after, "\n"), strings.HasPrefix(after, "\r"):
		kept = after[:1]
	}
	if after == kept {
		return false
	}
	s.SetValue(closeIdx+1, kept)
	for i := closeIdx + 2; i < s.Len(); i++ {
		s.SetValue(i, "")
	}
	return true
}

// superfluousFixEndOfLines trims horizontal whitespace before every line break
// in whitespace, comment and docblock tokens, and turns trailing horizontal
// whitespace at the very end of the file into a single line break.
func superfluousFixEndOfLines(s *tokens.Stream) bool {
	changed := false
	eol := superfluousEol(s)
	last := s.Len() - 1
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Whitespace && t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		if (t.Kind == token.Comment || t.Kind == token.DocComment) && superfluousIsAnnotation(t.Value) {
			// PHP_CodeSniffer tokenizes phpcs:/@phpcs: annotations as their own
			// token types, not T_COMMENT, so the sniff never trims them
			continue
		}
		v := superfluousTrimBeforeBreaks(t.Value)
		switch {
		case i == last:
			// trailing whitespace at end of file becomes a single line break
			v = superfluousTrailingToEol(v, eol)
		case superfluousIsLineComment(t):
			// a "//" or "#" comment runs to the line's end, but blink keeps the
			// line break in the next token, so trim the comment's own tail
			v = strings.TrimRight(v, " \t\x00\x0B")
		}
		if v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// superfluousFixEmptyLines collapses a run of more than one blank line to a
// single blank line, but only inside a function or closure whose innermost
// scope is not an anonymous class - mirroring the sniff's hasCondition and
// T_ANON_CLASS checks.
func superfluousFixEmptyLines(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Whitespace || strings.Count(t.Value, "\n") < 3 {
			continue
		}
		if !superfluousInFunctionNotAnon(s, i) {
			continue
		}
		if v := superfluousCollapseBlankLines(t.Value); v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// superfluousInFunctionNotAnon reports whether the token at i sits inside a
// function or closure body whose innermost enclosing scope is not an anonymous
// class, walking the enclosing braces outward.
func superfluousInFunctionNotAnon(s *tokens.Stream, i int) bool {
	depth := 0
	foundInner, innerAnon := false, false
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			kind, kw := classifyBrace(s, j)
			if !foundInner {
				foundInner = true
				innerAnon = kind == braceClassLike && superfluousIsAnonClass(s, kw)
			}
			if kind == braceFunctionDecl || kind == braceClosure {
				return !innerAnon
			}
		}
	}
	return false
}

// superfluousIsAnonClass reports whether the class-like keyword at kw opens an
// anonymous class ("new class ...").
func superfluousIsAnonClass(s *tokens.Stream, kw int) bool {
	if kw < 0 || !kwIs(s.At(kw), "class") {
		return false
	}
	p := prevMeaningfulIndex(s, kw)
	return p >= 0 && kwIs(s.At(p), "new")
}

// superfluousCollapseBlankLines keeps the first two line breaks (the end of the
// previous code line and one blank line) and the indentation after the last
// break, dropping the extra blank lines.
func superfluousCollapseBlankLines(v string) string {
	eol := "\n"
	if strings.Contains(v, "\r\n") {
		eol = "\r\n"
	}
	trailing := ""
	if nl := strings.LastIndexByte(v, '\n'); nl >= 0 {
		trailing = v[nl+1:]
	}
	return eol + eol + trailing
}

// superfluousTrimBeforeBreaks drops a run of horizontal whitespace that sits
// directly before a line break.
func superfluousTrimBeforeBreaks(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); {
		if superfluousHorizontal(v[i]) {
			j := i
			for j < len(v) && superfluousHorizontal(v[j]) {
				j++
			}
			if j < len(v) && (v[j] == '\n' || v[j] == '\r') {
				i = j // drop the run
				continue
			}
			b.WriteString(v[i:j])
			i = j
			continue
		}
		b.WriteByte(v[i])
		i++
	}
	if b.Len() == len(v) {
		return v
	}
	return b.String()
}

// superfluousTrailingToEol replaces a trailing run of horizontal whitespace with
// a single line break, matching the sniff's rtrim-plus-eolChar at end of file.
func superfluousTrailingToEol(v, eol string) string {
	j := len(v)
	for j > 0 && superfluousHorizontal(v[j-1]) {
		j--
	}
	if j == len(v) {
		return v
	}
	return v[:j] + eol
}

// superfluousIsAnnotation reports whether a comment is a phpcs:/@phpcs:
// annotation, which PHP_CodeSniffer tokenizes specially (not as T_COMMENT),
// mirroring Tokenizer::createPositionMap's leading/trailing strip.
func superfluousIsAnnotation(v string) bool {
	text := strings.TrimRight(strings.TrimLeft(v, " \t/*#"), " */\t\r\n")
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "phpcs:") || strings.HasPrefix(lower, "@phpcs:")
}

// superfluousIsLineComment reports whether t is a "//" or "#" line comment.
func superfluousIsLineComment(t token.Token) bool {
	return t.Kind == token.Comment && (strings.HasPrefix(t.Value, "//") || strings.HasPrefix(t.Value, "#"))
}

// superfluousHorizontal reports whether b is whitespace PHP's rtrim strips but
// that is not a line break.
func superfluousHorizontal(b byte) bool {
	return b == ' ' || b == '\t' || b == '\x00' || b == '\x0B'
}

// superfluousAllSpace reports whether v is non-empty and entirely whitespace,
// mirroring the sniff's `^[\pZ\s]+$` unicode check.
func superfluousAllSpace(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// superfluousEol returns the file's line ending, defaulting to "\n".
func superfluousEol(s *tokens.Stream) string {
	for i := 0; i < s.Len(); i++ {
		if v := s.At(i).Value; strings.Contains(v, "\n") {
			if strings.Contains(v, "\r\n") {
				return "\r\n"
			}
			return "\n"
		}
	}
	return "\n"
}
