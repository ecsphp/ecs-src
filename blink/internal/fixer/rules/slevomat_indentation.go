package rules

import "strings"

// slvRemoveIndentation is IndentationHelper::removeIndentation over start..end.
func (sc *phpcsScope) slvRemoveIndentation(start, end int, defaultIndentation string) string {
	f := sc.f
	const one = "    "
	var b strings.Builder
	inHeredoc := false
	for p := start; p <= end; p++ {
		content := f.toks[p].content
		if !inHeredoc && (p == start || strings.HasSuffix(f.toks[p-1].content, f.eol)) {
			switch {
			case content == f.eol:
			case strings.HasPrefix(content, one):
				content = content[len(one):]
			default:
				content = defaultIndentation + strings.TrimLeft(content, " \t\n\r\x00\x0B")
			}
		}
		switch sc.code[p] {
		case "start_heredoc":
			inHeredoc = true
		case "end_heredoc":
			inHeredoc = false
		}
		b.WriteString(content)
	}
	return strings.TrimRight(b.String(), " \t\n\r\x00\x0B")
}
