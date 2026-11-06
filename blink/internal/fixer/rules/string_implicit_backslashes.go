package rules

import (
	"cmp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringImplicitBackslashesFixer.php
//
// StringImplicitBackslashes escapes or unescapes the backslashes that do not
// start an escape sequence, separately for single-quoted strings,
// double-quoted strings and heredocs. Each option is "escape", "unescape" or
// "ignore".
type StringImplicitBackslashes struct {
	SingleQuoted string
	DoubleQuoted string
	Heredoc      string
}

func (StringImplicitBackslashes) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\StringImplicitBackslashesFixer`
}

func (StringImplicitBackslashes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringImplicitBackslashesFixer.php"
}

func (f StringImplicitBackslashes) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["single_quoted"].(string); ok {
		f.SingleQuoted = v
	}
	if v, ok := config["double_quoted"].(string); ok {
		f.DoubleQuoted = v
	}
	if v, ok := config["heredoc"].(string); ok {
		f.Heredoc = v
	}
	return f
}

func (StringImplicitBackslashes) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasKind(token.String)
}

// implicitBackslashSyntax is the kind of literal a backslash is in, which sets
// the characters a backslash escapes there.
type implicitBackslashSyntax int

const (
	backslashSingleQuoted implicitBackslashSyntax = iota
	backslashDoubleQuoted
	backslashHeredoc
)

func (f StringImplicitBackslashes) Fix(s *tokens.Stream) bool {
	modes := [...]string{
		backslashSingleQuoted: cmp.Or(f.SingleQuoted, "unescape"),
		backslashDoubleQuoted: cmp.Or(f.DoubleQuoted, "escape"),
		backslashHeredoc:      cmp.Or(f.Heredoc, "escape"),
	}

	// collected first, so the chunks are walked in token order: PHP-CS-Fixer
	// flips doubleQuoteOpened on every quote token, nested literals included
	chunks := map[int][]encPart{}
	eachStringChunk(s, func(i int, parts []encPart) { chunks[i] = parts })

	changed := false
	doubleQuoteOpened := false
	for i := range s.Len() {
		parts, ok := chunks[i]
		if !ok {
			continue
		}
		v := s.At(i).Value
		repl := map[int]string{}
		for k, p := range parts {
			content := v[p.start:p.end]
			if p.kind == encOther && (content == `"` || content == `b"` || content == `B"`) {
				doubleQuoteOpened = !doubleQuoteOpened
				continue
			}
			if (p.kind != encConstString && p.kind != encText) || !strings.Contains(content, `\`) {
				continue
			}
			if p.kind == encText && k > 0 && strings.HasSuffix(strings.TrimRight(v[parts[k-1].start:parts[k-1].end], " \t\n\r\x00\v"), "'") {
				continue // nowdoc
			}

			syntax := backslashHeredoc
			if p.kind == encConstString {
				_, rest := token.SplitBinaryPrefix(content)
				if rest[0] == '\'' {
					syntax = backslashSingleQuoted
				} else {
					syntax = backslashDoubleQuoted
				}
			} else if doubleQuoteOpened {
				syntax = backslashDoubleQuoted
			}
			if modes[syntax] == "ignore" {
				continue
			}
			if r := fixImplicitBackslashes(content, syntax, modes[syntax] == "escape"); r != content {
				repl[k] = r
			}
		}
		if len(repl) > 0 {
			s.SetValue(i, encRebuild(v, parts, repl))
			changed = true
		}
	}
	return changed
}

// fixImplicitBackslashes ports the fixer's two regexes over each run of
// backslashes: escape doubles an odd run, unescape halves an even one, when
// the character after the run would not make it an escape sequence.
func fixImplicitBackslashes(content string, syntax implicitBackslashSyntax, escape bool) string {
	var b strings.Builder
	last := 0
	for i := 0; i < len(content); {
		if content[i] != '\\' {
			i++
			continue
		}
		start := i
		for i < len(content) && content[i] == '\\' {
			i++
		}
		n := i - start
		if (escape && n%2 == 0) || (!escape && n%2 == 1) || backslashReserved(content, i, syntax) {
			continue
		}
		b.WriteString(content[last:start])
		if escape {
			b.WriteString(content[start:i])
			b.WriteByte('\\')
		} else {
			b.WriteString(content[start : i-1])
		}
		last = i
	}
	if last == 0 {
		return content
	}
	b.WriteString(content[last:])
	return b.String()
}

// backslashReserved reports whether the character at p, right after a run of
// backslashes, is one the fixer's reserved regex matches for syntax. "$" in
// that regex is PCRE's end of subject, which also matches before a final "\n".
func backslashReserved(content string, p int, syntax implicitBackslashSyntax) bool {
	if syntax == backslashSingleQuoted {
		return p < len(content) && content[p] == '\''
	}
	if p == len(content) || (p == len(content)-1 && content[p] == '\n') {
		return true
	}
	switch c := content[p]; {
	case strings.IndexByte("efnrtv$01234567", c) >= 0:
		return true
	case c == '"':
		return syntax == backslashDoubleQuoted
	case c == 'x':
		return p+1 < len(content) && strings.IndexByte("0123456789abcdefABCDEF", content[p+1]) >= 0
	case c == 'u':
		return p+1 < len(content) && content[p+1] == '{'
	}
	return false
}
