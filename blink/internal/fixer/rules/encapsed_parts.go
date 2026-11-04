package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// The lexer splits an interpolated string, heredoc or backtick literal like
// token_get_all does, with its code lexed as ordinary tokens, but keeps each run
// of the other PHP tokens (quotes, T_ENCAPSED_AND_WHITESPACE, T_CURLY_OPEN,
// "${name}", ...) together as one String token. The string fixers need those
// PHP tokens, so encapsedParts splits a String token's value into them.

// encPartKind is the PHP token kind of a span inside a String token.
type encPartKind int

const (
	encOther        encPartKind = iota // quotes, "}", T_END_HEREDOC
	encConstString                     // T_CONSTANT_ENCAPSED_STRING
	encText                            // T_ENCAPSED_AND_WHITESPACE
	encStartHeredoc                    // T_START_HEREDOC
	encVarname                         // T_STRING_VARNAME: the name in "${name}"
	encDollarOpen                      // T_DOLLAR_OPEN_CURLY_BRACES: "${"
	encCurlyOpen                       // T_CURLY_OPEN: the "{" of "{$"
)

// encPart is one PHP token inside a String token value: v[start:end].
type encPart struct {
	kind       encPartKind
	start, end int
}

// eachStringChunk calls fn with the index and the PHP tokens of every String
// token of s whose value it recognizes. The chunks of an interpolated literal
// are visited in order from its Head; a Mid or Tail without its Head is skipped.
// fn may change the value of the token it gets, not the number of tokens.
func eachStringChunk(s *tokens.Stream, fn func(i int, parts []encPart)) {
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String {
			continue
		}
		switch t.Part {
		case token.Whole:
			if parts := encapsedParts(t, false); parts != nil {
				fn(i, parts)
			}
		case token.Head:
			chunks := literalChunks(s, i)
			if chunks == nil {
				chunks = []int{i}
			}
			afterExpr := false
			for _, j := range chunks {
				parts := encapsedParts(s.At(j), afterExpr)
				if parts == nil {
					break
				}
				afterExpr = encOpensExpr(parts)
				fn(j, parts)
			}
		}
	}
}

// encOpensExpr reports whether a chunk with these parts ends by opening a
// "{$...}" or "${...}" expression, closed by the "}" the next chunk starts with.
func encOpensExpr(parts []encPart) bool {
	if len(parts) == 0 {
		return false
	}
	switch parts[len(parts)-1].kind {
	case encCurlyOpen, encDollarOpen, encVarname:
		return true
	}
	return false
}

// encapsedParts splits the value of the String token t into the PHP tokens it
// holds, or returns nil for a value it does not recognize. afterExpr tells that
// t is a Mid or Tail chunk starting with the "}" that closes an expression.
func encapsedParts(t token.Token, afterExpr bool) []encPart {
	v := t.Value
	if v == "" {
		return nil
	}
	var out []encPart
	add := func(k encPartKind, start, end int) {
		if end > start {
			out = append(out, encPart{kind: k, start: start, end: end})
		}
	}
	p, to := 0, len(v)
	opens := t.Part == token.Whole || t.Part == token.Head
	closes := t.Part == token.Whole || t.Part == token.Tail
	heredoc, quote := false, byte(0)
	switch {
	case opens:
		prefix, rest := token.SplitBinaryPrefix(v)
		switch {
		case strings.HasPrefix(rest, "<<<"):
			if p = encLineEnd(v, len(prefix)); p < 0 {
				return nil
			}
			add(encStartHeredoc, 0, p)
			heredoc = true
			if t.Part == token.Whole && strings.TrimLeft(rest[3:], " \t")[0] == '\'' {
				end := encHeredocEnd(v, p)
				add(encText, p, end) // nowdoc: no interpolation
				add(encOther, end, len(v))
				return out
			}
		case rest[0] == '\'':
			if t.Part != token.Whole {
				return nil
			}
			add(encConstString, 0, len(v))
			return out
		case rest[0] == '"' || rest[0] == '`':
			quote = rest[0]
			p = len(prefix) + 1
			add(encOther, 0, p)
		default:
			return nil
		}
	case afterExpr && v[0] == '}':
		add(encOther, 0, 1)
		p = 1
	}
	if closes {
		switch last := v[len(v)-1]; {
		case !heredoc && (last == '"' || last == '`') && len(v)-1 >= p:
			to = len(v) - 1
		case heredoc || t.Part == token.Tail:
			to = encHeredocEnd(v, p)
		default:
			return nil
		}
	}
	interpolated := encScanBody(v, p, to, !closes, add)
	if !interpolated && t.Part == token.Whole && quote == '"' {
		return []encPart{{kind: encConstString, start: 0, end: len(v)}}
	}
	add(encOther, to, len(v))
	return out
}

// encScanBody adds the parts of the literal body v[from:to] and reports whether
// it interpolates. codeFollows tells that the chunk ends where code begins.
func encScanBody(v string, from, to int, codeFollows bool, add func(encPartKind, int, int)) bool {
	interpolated := false
	textStart, p := from, from
	for p < to {
		switch c := v[p]; {
		case c == '\\':
			p += 2
			continue
		case c == '$' && p+1 < to && v[p+1] == '{':
			add(encText, textStart, p)
			interpolated = true
			add(encDollarOpen, p, p+2)
			p += 2
			if p < to && isNameStart(v[p]) {
				if q := encLabelEnd(v, p, to); q == to || v[q] == '}' || v[q] == '[' {
					add(encVarname, p, q)
					p = q
				}
			}
			if p < to && v[p] == '}' {
				add(encOther, p, p+1)
				p++
			}
			textStart = p
			continue
		case c == '{' && codeFollows && p+1 == to:
			add(encText, textStart, p)
			interpolated = true
			add(encCurlyOpen, p, p+1)
			p++
			textStart = p
			continue
		}
		p++
	}
	add(encText, textStart, min(p, to))
	return interpolated
}

// encLineEnd returns the end of the first line break at or after from, or -1.
func encLineEnd(v string, from int) int {
	nl := strings.IndexAny(v[from:], "\r\n")
	if nl < 0 {
		return -1
	}
	nl += from
	if v[nl] == '\r' && nl+1 < len(v) && v[nl+1] == '\n' {
		nl++
	}
	return nl + 1
}

// encHeredocEnd returns where the T_END_HEREDOC of the heredoc value v starts:
// after its last line break, and not before from.
func encHeredocEnd(v string, from int) int {
	return max(strings.LastIndexAny(v, "\r\n")+1, from)
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
