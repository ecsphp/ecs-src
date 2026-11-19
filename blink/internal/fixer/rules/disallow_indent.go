package rules

import (
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsTabStop is the tab width ECS gives PHP_CodeSniffer.
const phpcsTabStop = 4

// phpcsIndentLine is one line piece of a token, as PHP_CodeSniffer splits every
// multi-line token into one token per line.
type phpcsIndentLine struct {
	tok      int
	start    int // byte offset of the piece in the token value
	text     string
	col      int  // PHPCS column, tabs of tab-replaced tokens expanded
	docSpace bool // leading whitespace of a doc comment line (T_DOC_COMMENT_WHITESPACE)
	keySpace bool // leading whitespace of a line inside "yield\nfrom" (T_WHITESPACE)
}

// phpcsTabsExpanded reports whether PHP_CodeSniffer's tokenizer replaces tabs
// in a token of this kind with spaces (Tokenizer::TOKENS_WITH_TABS).
func phpcsTabsExpanded(k token.Kind) bool {
	switch k {
	case token.Whitespace, token.Comment, token.DocComment, token.String, token.Keyword:
		return true
	}
	return false
}

// phpcsExpandTabs mirrors Tokenizer::replaceTabsInToken: each tab moves to the
// next tab stop; it returns the expanded text and the column after it.
func phpcsExpandTabs(v string, col int) (string, int) {
	var b strings.Builder
	for {
		i := strings.IndexByte(v, '\t')
		if i < 0 {
			b.WriteString(v)
			return b.String(), col + utf8.RuneCountInString(v)
		}
		b.WriteString(v[:i])
		col += utf8.RuneCountInString(v[:i])
		pad := phpcsTabStop - (col+phpcsTabStop-1)%phpcsTabStop
		b.WriteString(strings.Repeat(" ", pad))
		col += pad
		v = v[i+1:]
	}
}

// phpcsIndentLines splits the stream into PHPCS line pieces with their columns.
func phpcsIndentLines(s *tokens.Stream) []phpcsIndentLine {
	var lines []phpcsIndentLine
	col := 1
	for i := range s.Len() {
		t := s.At(i)
		v := t.Value
		for start := 0; start < len(v); {
			end := len(v)
			if j := strings.IndexByte(v[start:], '\n'); j >= 0 {
				end = start + j + 1
			}
			piece := v[start:end]
			lines = append(lines, phpcsIndentLine{tok: i, start: start, text: piece, col: col})
			if strings.HasSuffix(piece, "\n") {
				col = 1
			} else if phpcsTabsExpanded(t.Kind) {
				_, col = phpcsExpandTabs(piece, col)
			} else {
				col += utf8.RuneCountInString(piece)
			}
			start = end
		}
	}
	return lines
}

// phpcsReplaceLines rebuilds every token whose pieces changed.
func phpcsReplaceLines(s *tokens.Stream, lines []phpcsIndentLine, repl map[int]string) bool {
	if len(repl) == 0 {
		return false
	}
	byTok := map[int]*strings.Builder{}
	var order []int
	for n, l := range lines {
		if _, ok := byTok[l.tok]; !ok {
			if !phpcsTokenTouched(lines, n, repl) {
				continue
			}
			byTok[l.tok] = &strings.Builder{}
			order = append(order, l.tok)
		}
		if r, ok := repl[n]; ok {
			byTok[l.tok].WriteString(r)
		} else {
			byTok[l.tok].WriteString(l.text)
		}
	}
	for _, i := range order {
		s.SetValue(i, byTok[i].String())
	}
	return true
}

func phpcsTokenTouched(lines []phpcsIndentLine, n int, repl map[int]string) bool {
	for m := n; m < len(lines) && lines[m].tok == lines[n].tok; m++ {
		if _, ok := repl[m]; ok {
			return true
		}
	}
	return false
}

// phpcsLeadingSpace splits a line piece into its leading whitespace and the rest.
func phpcsLeadingSpace(v string) (string, string) {
	rest := strings.TrimLeft(v, " \t\r\n\f\v")
	return v[:len(v)-len(rest)], rest
}

// phpcsSplitDocLines splits each doc comment or keyword line piece at column 1
// into its leading whitespace and the rest, as PHPCS tokenizes them apart.
func phpcsSplitDocLines(s *tokens.Stream, lines []phpcsIndentLine) []phpcsIndentLine {
	out := make([]phpcsIndentLine, 0, len(lines))
	for _, l := range lines {
		kind := s.At(l.tok).Kind
		if (kind != token.DocComment && kind != token.Keyword) || l.col != 1 {
			out = append(out, l)
			continue
		}
		ws := l.text[:len(l.text)-len(strings.TrimLeft(l.text, " \t"))]
		if ws == "" {
			out = append(out, l)
			continue
		}
		_, after := phpcsExpandTabs(ws, 1)
		out = append(out,
			phpcsIndentLine{tok: l.tok, start: l.start, text: ws, col: 1, docSpace: kind == token.DocComment, keySpace: kind == token.Keyword},
			phpcsIndentLine{tok: l.tok, start: l.start + len(ws), text: l.text[len(ws):], col: after},
		)
	}
	return out
}

// phpcsRunUntilStable re-runs one sniff pass like PHP_CodeSniffer's Fixer
// loop, which re-tokenizes the file between passes.
func phpcsRunUntilStable(s *tokens.Stream, pass func(*tokens.Stream) bool) bool {
	changed := false
	for range 50 {
		if !pass(s) {
			break
		}
		changed = true
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(s.Render()))
	}
	return changed
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/DisallowTabIndentSniff.php
//
// DisallowTabIndent replaces tabs in whitespace, comments and inline HTML with
// spaces, up to the next tab stop of width 4.
type DisallowTabIndent struct{}

func (DisallowTabIndent) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\DisallowTabIndentSniff`
}

func (DisallowTabIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/DisallowTabIndentSniff.php"
}

func (DisallowTabIndent) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, true) {
		return false
	}
	return phpcsRunUntilStable(s, disallowTabIndentPass)
}

func disallowTabIndentPass(s *tokens.Stream) bool {
	lines := phpcsIndentLines(s)
	repl := map[int]string{}
	for n, l := range lines {
		if !strings.Contains(l.text, "\t") {
			continue
		}
		switch s.At(l.tok).Kind {
		case token.InlineHTML:
			repl[n] = strings.ReplaceAll(l.text, "\t", strings.Repeat(" ", phpcsTabStop))
		case token.Comment:
			if lead, rest := phpcsLeadingSpace(l.text); rest != "" && lead == " " {
				continue
			}
			repl[n], _ = phpcsExpandTabs(l.text, l.col)
		case token.Whitespace, token.DocComment, token.Keyword:
			repl[n], _ = phpcsExpandTabs(l.text, l.col)
		}
	}
	return phpcsReplaceLines(s, lines, repl)
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/DisallowSpaceIndentSniff.php
//
// DisallowSpaceIndent indents lines with tabs (width 4), keeping fewer than 4
// trailing precision spaces.
type DisallowSpaceIndent struct{}

func (DisallowSpaceIndent) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\DisallowSpaceIndentSniff`
}

func (DisallowSpaceIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/DisallowSpaceIndentSniff.php"
}

func (DisallowSpaceIndent) Fix(s *tokens.Stream) bool {
	if !phpcsHasOpenTag(s, true) {
		return false
	}
	return phpcsRunUntilStable(s, disallowSpaceIndentPass)
}

func disallowSpaceIndentPass(s *tokens.Stream) bool {
	eol := phpcsEolChar(s)
	lines := phpcsSplitDocLines(s, phpcsIndentLines(s))
	repl := map[int]string{}
	for n, l := range lines {
		if l.col != 1 {
			continue
		}
		kind := s.At(l.tok).Kind
		comment := kind == token.Comment || l.docSpace
		if l.keySpace {
			kind = token.Whitespace
		}
		if kind != token.Whitespace && kind != token.InlineHTML && kind != token.Comment && !l.docSpace {
			continue
		}
		content := l.text
		trimmed := strings.TrimRight(content, "\r\n")
		size := utf8.RuneCountInString(trimmed)
		if kind != token.InlineHTML {
			expanded, _ := phpcsExpandTabs(trimmed, 1)
			size = len(expanded)
		}
		nonWhitespace := ""
		lead, rest := phpcsLeadingSpace(content)
		switch {
		case (kind == token.InlineHTML || kind == token.Comment) && rest != "":
			content = lead
			size = len(strings.ReplaceAll(lead, "\t", strings.Repeat(" ", phpcsTabStop)))
			nonWhitespace = rest
		case strings.HasSuffix(content, "\n"):
			content = trimmed
			nonWhitespace = eol
		}
		spaces := strings.Count(content, " ")
		tabs := strings.Count(content, "\t")
		if spaces == 0 || (comment && content == " ") {
			continue
		}
		tabAfterSpaces := strings.Contains(content[strings.IndexByte(content, ' '):], "\t")
		wantTabs, wantSpaces := size/phpcsTabStop, size%phpcsTabStop
		if tabs == wantTabs && spaces == wantSpaces && (tabs == 0 || !tabAfterSpaces) {
			continue
		}
		repl[n] = strings.Repeat("\t", wantTabs) + strings.Repeat(" ", wantSpaces) + nonWhitespace
	}
	return phpcsReplaceLines(s, lines, repl)
}
