package rules

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/HeredocIndentationFixer.php
//
// HeredocIndentation indents a heredoc or nowdoc body and its closing marker
// one level deeper than the line it starts on ("start_plus_one", the default)
// or at the same level ("same_as_start"), using PHP 7.3 flexible heredocs.
type HeredocIndentation struct {
	SameAsStart bool
}

func (HeredocIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\HeredocIndentationFixer`
}

func (HeredocIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/HeredocIndentationFixer.php"
}

func (f HeredocIndentation) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["indentation"].(string); ok {
		f.SameAsStart = v == "same_as_start"
	}
	return f
}

func (HeredocIndentation) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasKind(token.String)
}

// heredocPiece is one PHP token between T_START_HEREDOC and T_END_HEREDOC:
// part k of String token tok, or the whole token tok when k is -1.
type heredocPiece struct {
	tok, k int
	kind   encPartKind
}

func (f HeredocIndentation) Fix(s *tokens.Stream) bool {
	chunks := map[int][]encPart{}
	eachStringChunk(s, func(i int, parts []encPart) { chunks[i] = parts })

	changed := false
	for end := s.Len() - 1; end >= 0; end-- {
		if !s.At(end).EndsHeredoc() {
			continue
		}
		start := end
		for start >= 0 && !s.At(start).StartsHeredoc() {
			start--
		}
		if start < 0 {
			break
		}
		chain := []int{start}
		if start != end {
			if s.At(start).Part != token.Head {
				end = start
				continue
			}
			chain = literalChunks(s, start)
			if len(chain) == 0 || chain[len(chain)-1] != end {
				end = start
				continue
			}
		}
		if f.fixHeredoc(s, chunks, chain) {
			changed = true
		}
		end = start
	}
	return changed
}

func (f HeredocIndentation) fixHeredoc(s *tokens.Stream, chunks map[int][]encPart, chain []int) bool {
	first, last := chain[0], chain[len(chain)-1]
	for _, j := range chain {
		if chunks[j] == nil {
			return false
		}
	}
	endParts := chunks[last]
	if chunks[first][0].kind != encStartHeredoc {
		return false
	}

	indent := detectIndent(s, first)
	if !f.SameAsStart {
		indent += "    "
	}

	// repl holds the new value of each piece, keyed by token then part (-1 for
	// a whole token); the T_START_HEREDOC and T_END_HEREDOC parts included
	repl := map[int]map[int]string{}
	set := func(tok, k int, v string) {
		if repl[tok] == nil {
			repl[tok] = map[int]string{}
		}
		repl[tok][k] = v
	}
	value := func(p heredocPiece) string {
		if v, ok := repl[p.tok][p.k]; ok {
			return v
		}
		if p.k < 0 {
			return s.At(p.tok).Value
		}
		part := chunks[p.tok][p.k]
		return s.At(p.tok).Value[part.start:part.end]
	}

	endValue := value(heredocPiece{tok: last, k: len(endParts) - 1})
	currentIndent := endValue[:len(endValue)-len(strings.TrimLeft(endValue, " \t"))]
	set(last, len(endParts)-1, indent+endValue[len(currentIndent):])

	inChain := map[int]bool{}
	for _, j := range chain {
		inChain[j] = true
	}
	var pieces []heredocPiece
	for j := first; j <= last; j++ {
		if !inChain[j] {
			kind := encOther
			if s.At(j).Kind == token.Whitespace {
				kind = encText
			}
			pieces = append(pieces, heredocPiece{tok: j, k: -1, kind: kind})
			continue
		}
		for k, part := range chunks[j] {
			if (j == first && k == 0) || (j == last && k == len(chunks[j])-1) {
				continue
			}
			pieces = append(pieces, heredocPiece{tok: j, k: k, kind: part.kind})
		}
	}
	if len(pieces) > 0 {
		for n, p := range slices.Backward(pieces) {
			if p.kind == encText {
				set(p.tok, p.k, heredocReindentLines(value(p), currentIndent, indent, n == len(pieces)-1))
			}
		}
		if p := pieces[0]; p.kind != encText || s.At(p.tok).Kind == token.Whitespace {
			set(first, 0, value(heredocPiece{tok: first, k: 0})+indent)
		} else {
			content := value(p)
			if content[0] != '\r' && content[0] != '\n' && strings.HasPrefix(content, currentIndent) {
				content = indent + content[len(currentIndent):]
			} else if currentIndent != "" {
				content = content[pcreHorizontalRun(content, 0, utf8.ValidString(content)):]
			}
			set(p.tok, p.k, content)
		}
	}

	changed := false
	for tok, byPart := range repl {
		v := s.At(tok).Value
		var nv string
		if r, ok := byPart[-1]; ok {
			nv = r
		} else {
			nv = encRebuild(v, chunks[tok], byPart)
		}
		if nv != v {
			s.SetValue(tok, nv)
			changed = true
		}
	}
	return changed
}

// heredocReindentLines ports the fixer's two replacements on a piece of the
// body: drop the indentation of lines not indented by the closing marker's
// indentation, then swap that indentation for the new one. The last piece
// keeps its final line break unindented when the closing marker had none.
func heredocReindentLines(content, currentIndent, indent string, last bool) string {
	if currentIndent != "" {
		var b strings.Builder
		from := 0
		valid := utf8.ValidString(content)
		for _, p := range pcreLineStarts(content, valid) {
			if strings.HasPrefix(content[p:], currentIndent) {
				continue
			}
			if n := pcreHorizontalRun(content, p, valid); n > p {
				b.WriteString(content[from:p])
				from = n
			}
		}
		if from > 0 {
			b.WriteString(content[from:])
			content = b.String()
		}
	}

	var b strings.Builder
	from := 0
	valid := utf8.ValidString(content)
	for _, p := range pcreLineStarts(content, valid) {
		if !strings.HasPrefix(content[p:], currentIndent) {
			continue
		}
		q := p + len(currentIndent)
		if pcreVerticalLen(content, q, valid) > 0 || (last && currentIndent == "" && q == len(content)) {
			continue
		}
		b.WriteString(content[from:p])
		b.WriteString(indent)
		from = q
	}
	if from == 0 {
		return content
	}
	b.WriteString(content[from:])
	return b.String()
}

// The fixer's regexes run with the "u" modifier, and without it on a subject
// that is not valid UTF-8, which changes what "\h" and "\v" match: valid
// tells which of the two applies.

// pcreLineStarts returns the offsets of content right after a "\v".
func pcreLineStarts(content string, valid bool) []int {
	var starts []int
	for i := 0; i < len(content); {
		if n := pcreVerticalLen(content, i, valid); n > 0 {
			i += n
			starts = append(starts, i)
			continue
		}
		i++
	}
	return starts
}

// pcreVerticalLen returns the byte length of the "\v" character at i, or 0.
func pcreVerticalLen(content string, i int, valid bool) int {
	if i >= len(content) {
		return 0
	}
	if c := content[i]; c >= '\n' && c <= '\r' {
		return 1
	}
	if !valid {
		if content[i] == 0x85 {
			return 1
		}
		return 0
	}
	switch r, n := utf8.DecodeRuneInString(content[i:]); r {
	case 0x85, 0x2028, 0x2029:
		return n
	}
	return 0
}

// pcreHorizontalRun returns the end of the run of "\h" characters at i.
func pcreHorizontalRun(content string, i int, valid bool) int {
	for i < len(content) {
		if c := content[i]; c == ' ' || c == '\t' {
			i++
			continue
		}
		if !valid {
			if content[i] != 0xA0 {
				break
			}
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(content[i:])
		if r != 0xA0 && r != 0x1680 && r != 0x180E && (r < 0x2000 || r > 0x200A) && r != 0x202F && r != 0x205F && r != 0x3000 {
			break
		}
		i += n
	}
	return i
}
