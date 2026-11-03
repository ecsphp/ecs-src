package rules

import (
	"strings"

	"blink/internal/lexer"
	"blink/internal/token"
)

// The lexer keeps a string or heredoc as one String token. The string fixers
// need the PHP tokens token_get_all sees inside it, so encapsedParts re-scans it.

// encPartKind is the PHP token kind of a span inside a String token.
type encPartKind int

const (
	encOther        encPartKind = iota // quotes, braces, heredoc end, code punctuation and whitespace
	encConstString                     // T_CONSTANT_ENCAPSED_STRING
	encText                            // T_ENCAPSED_AND_WHITESPACE
	encStartHeredoc                    // T_START_HEREDOC
	encVariable                        // T_VARIABLE
	encName                            // T_STRING
	encVarname                         // T_STRING_VARNAME: the name in "${name}"
	encDollarOpen                      // T_DOLLAR_OPEN_CURLY_BRACES: "${"
	encComment                         // T_COMMENT
	encDocComment                      // T_DOC_COMMENT
)

// encPart is one PHP token inside a String token value: v[start:end].
type encPart struct {
	kind       encPartKind
	start, end int
}

// encapsedParts splits a String token value (a whole string, one of the lexer's
// chunks around a "{$...}", or a heredoc) into the PHP tokens it holds, or
// returns nil for a value it does not recognize.
func encapsedParts(v string) []encPart {
	var out []encPart
	if !appendEncapsedParts(v, 0, &out) {
		return nil
	}
	return out
}

// encAdder returns a function that appends a non-empty part, shifted by base.
func encAdder(out *[]encPart, base int) func(encPartKind, int, int) {
	return func(k encPartKind, start, end int) {
		if end > start {
			*out = append(*out, encPart{kind: k, start: base + start, end: base + end})
		}
	}
}

// appendEncapsedParts appends the parts of v, shifted by base, and reports
// whether v was recognized.
func appendEncapsedParts(v string, base int, out *[]encPart) bool {
	if v == "" {
		return false
	}
	add := encAdder(out, base)
	switch {
	case v[0] == '\'':
		add(encConstString, 0, len(v))
		return true
	case strings.HasPrefix(v, "<<<"):
		nl := strings.IndexByte(v, '\n')
		if nl < 0 {
			return false
		}
		startEnd := nl + 1
		endStart := max(strings.LastIndexByte(v, '\n')+1, startEnd)
		add(encStartHeredoc, 0, startEnd)
		if strings.TrimLeft(v[3:], " \t")[0] == '\'' {
			add(encText, startEnd, endStart) // nowdoc: no interpolation
		} else {
			scanEncapsed(v, startEnd, endStart, base, out)
		}
		add(encOther, endStart, len(v))
		return true
	case v[0] == '"' || v[0] == '}':
		bodyEnd := len(v)
		if last := v[len(v)-1]; len(v) > 1 && (last == '"' || last == '{') {
			bodyEnd--
		}
		mark := len(*out)
		add(encOther, 0, 1)
		interpolated := scanEncapsed(v, 1, bodyEnd, base, out)
		add(encOther, bodyEnd, len(v))
		if !interpolated && v[0] == '"' && bodyEnd < len(v) && v[bodyEnd] == '"' {
			*out = (*out)[:mark]
			add(encConstString, 0, len(v))
		}
		return true
	}
	return false
}

// scanEncapsed appends the parts of the body v[from:to] and reports whether it
// interpolates.
func scanEncapsed(v string, from, to, base int, out *[]encPart) bool {
	add := encAdder(out, base)
	interpolated := false
	textStart := from
	p := from
	for p < to {
		c := v[p]
		switch {
		case c == '\\':
			p += 2
			continue
		case c == '$' && p+1 < to && isNameStart(v[p+1]):
			add(encText, textStart, p)
			interpolated = true
			q := encLabelEnd(v, p+1, to)
			add(encVariable, p, q)
			p = q
			switch {
			case p < to && v[p] == '[':
				p = scanVarOffset(v, p, to, add)
			case strings.HasPrefix(v[p:to], "->") && p+2 < to && isNameStart(v[p+2]):
				q = encLabelEnd(v, p+2, to)
				add(encOther, p, p+2)
				add(encName, p+2, q)
				p = q
			case strings.HasPrefix(v[p:to], "?->") && p+3 < to && isNameStart(v[p+3]):
				q = encLabelEnd(v, p+3, to)
				add(encOther, p, p+3)
				add(encName, p+3, q)
				p = q
			}
			textStart = p
			continue
		case c == '$' && p+1 < to && v[p+1] == '{':
			add(encText, textStart, p)
			interpolated = true
			add(encDollarOpen, p, p+2)
			p += 2
			if p < to && isNameStart(v[p]) {
				if q := encLabelEnd(v, p, to); q < to && (v[q] == '[' || v[q] == '}') {
					add(encVarname, p, q)
					p = q
				}
			}
			p = scanEncapsedCode(v, p, to, base, out)
			textStart = p
			continue
		case c == '{' && p+1 < to && v[p+1] == '$':
			add(encText, textStart, p)
			interpolated = true
			add(encOther, p, p+1)
			p = scanEncapsedCode(v, p+1, to, base, out)
			textStart = p
			continue
		}
		p++
	}
	add(encText, textStart, min(p, to))
	return interpolated
}

// scanVarOffset scans the "[offset]" after a simple interpolated variable
// (PHP's ST_VAR_OFFSET state) starting at the "[" at p, and returns where it ends.
func scanVarOffset(v string, p, to int, add func(encPartKind, int, int)) int {
	add(encOther, p, p+1)
	p++
	for p < to {
		c := v[p]
		switch {
		case c == ']':
			add(encOther, p, p+1)
			return p + 1
		case c >= '0' && c <= '9':
			q := encLabelEnd(v, p, to)
			add(encOther, p, q) // T_NUM_STRING
			p = q
		case c == '$' && p+1 < to && isNameStart(v[p+1]):
			q := encLabelEnd(v, p+1, to)
			add(encVariable, p, q)
			p = q
		case isNameStart(c):
			q := encLabelEnd(v, p, to)
			add(encName, p, q)
			p = q
		case c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '\\' || c == '\'' || c == '#':
			return p // invalid offset: PHP leaves the state here
		default:
			add(encOther, p, p+1)
			p++
		}
	}
	return p
}

// scanEncapsedCode lexes the expression of a "{$...}" or "${...}" from p (just
// after the opening brace) through its matching "}", appends its tokens and
// returns where it ends.
func scanEncapsedCode(v string, p, to, base int, out *[]encPart) int {
	const prefix = "<?php"
	toks := lexer.Lex(prefix + v[p:to])
	depth := 1
	for _, t := range toks {
		start := t.Pos - len(prefix)
		if start < 0 {
			continue
		}
		start += p
		end := start + len(t.Value)
		kind := encOther
		switch t.Kind {
		case token.Punct:
			switch t.Value {
			case "{":
				depth++
			case "}":
				depth--
			}
		case token.String:
			if appendEncapsedParts(t.Value, base+start, out) {
				continue
			}
		case token.Variable:
			if t.Value != "$" {
				kind = encVariable
			}
		case token.Ident:
			kind = encName
		case token.Comment:
			kind = encComment
		case token.DocComment:
			kind = encDocComment
		}
		*out = append(*out, encPart{kind: kind, start: base + start, end: base + end})
		if depth == 0 {
			return end
		}
	}
	return to
}

// encLabelEnd returns the end of the label starting at p.
func encLabelEnd(v string, p, to int) int {
	for p < to && isNameByte(v[p]) {
		p++
	}
	return p
}

// encRebuild renders v from its parts, replacing those in repl.
func encRebuild(v string, parts []encPart, repl map[int]string) string {
	var b strings.Builder
	b.Grow(len(v) + 8)
	for k, p := range parts {
		if r, ok := repl[k]; ok {
			b.WriteString(r)
			continue
		}
		b.WriteString(v[p.start:p.end])
	}
	return b.String()
}
