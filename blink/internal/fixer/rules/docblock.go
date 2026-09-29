package rules

import "strings"

// A docblock is modeled as its opening line ("/**"), inner content lines and
// closing line ("*/"). Each inner line keeps its "<indent>* " prefix separate
// from the content so fixers can rewrite content and rebuild verbatim.

type docLine struct {
	prefix  string // indentation + "*" (+ " " when content follows)
	content string
}

type docblock struct {
	open   string
	inner  []docLine
	close  string
	single bool // "/** ... */" on one line
}

// parseDoc splits a /** ... */ value into content lines. Malformed input returns
// ok=false. Single-line docblocks are supported.
func parseDoc(v string) (docblock, bool) {
	if !strings.HasPrefix(v, "/**") || !strings.HasSuffix(v, "*/") || len(v) < 5 {
		return docblock{}, false
	}
	if !strings.Contains(v, "\n") {
		body := strings.TrimSpace(v[3 : len(v)-2])
		return docblock{open: "/**", inner: []docLine{{content: body}}, close: "*/", single: true}, true
	}
	lines := strings.Split(v, "\n")
	if len(lines) < 3 {
		return docblock{}, false
	}
	d := docblock{open: lines[0], close: lines[len(lines)-1]}
	for _, line := range lines[1 : len(lines)-1] {
		star := strings.IndexByte(line, '*')
		if star < 0 {
			return docblock{}, false
		}
		prefix := line[:star+1]
		rest := line[star+1:]
		if strings.HasPrefix(rest, " ") {
			prefix += " "
			rest = rest[1:]
		}
		d.inner = append(d.inner, docLine{prefix: prefix, content: rest})
	}
	return d, true
}

func (d docblock) render() string {
	if d.single {
		if len(d.inner) == 0 || d.inner[0].content == "" {
			return "/** */"
		}
		return "/** " + d.inner[0].content + " */"
	}
	var b strings.Builder
	b.WriteString(d.open)
	for _, l := range d.inner {
		b.WriteString("\n")
		if l.content == "" {
			b.WriteString(strings.TrimRight(l.prefix, " "))
		} else {
			b.WriteString(l.prefix + l.content)
		}
	}
	b.WriteString("\n" + d.close)
	return b.String()
}
