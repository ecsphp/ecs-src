package rules

import (
	"slices"
	"strings"
)

func phpAlignIsHSpace(c byte) bool { return c == ' ' || c == '\t' }

// getMatches parses a docblock line, mirroring PhpdocAlignFixer::getMatches. Returns nil when the line is not a tag
// line (or, when matchCommentOnly, not a description continuation line).
func (f PhpdocAlign) getMatches(rawLine string, matchCommentOnly bool) *alignMatch {
	line := strings.TrimRight(rawLine, "\r\n")
	// ^(?P<indent>(?:\ {2}|\t)*)\ ?\*
	i := 0
	indentEnd := 0
	for i < len(line) {
		if strings.HasPrefix(line[i:], "  ") {
			i += 2
			indentEnd = i
		} else if line[i] == '\t' {
			i++
			indentEnd = i
		} else {
			break
		}
	}
	indent := line[:indentEnd]
	if i < len(line) && line[i] == ' ' {
		i++
	}
	if i >= len(line) || line[i] != '*' {
		return nil
	}
	i++ // past '*'
	starPos := i

	// try the tag regex: \h*@tag...
	j := i
	for j < len(line) && phpAlignIsHSpace(line[j]) {
		j++
	}
	if j < len(line) && line[j] == '@' {
		if m := f.parseTag(indent, line, j+1); m != nil {
			return m
		}
	}

	if !matchCommentOnly {
		return nil
	}
	// regexCommentLine: \*(?!\h?+@)(?:\s+(?P<desc>\V+))(?<!\*\/)\r?$
	// after '*': must NOT be (optional single hspace then '@')
	k := starPos
	nb := k
	if nb < len(line) && phpAlignIsHSpace(line[nb]) {
		nb++
	}
	if nb < len(line) && line[nb] == '@' {
		return nil // negative lookahead fails
	}
	// \s+ then desc = \V+ (one or more non-vertical-ws)
	d := starPos
	sawWS := false
	for d < len(line) && phpAlignIsHSpace(line[d]) {
		d++
		sawWS = true
	}
	if !sawWS || d >= len(line) {
		return nil
	}
	desc := line[d:]
	if desc == "" {
		return nil
	}
	// (?<!\*\/): the line must not end with "*/"
	if strings.HasSuffix(line, "*/") {
		return nil
	}
	return &alignMatch{indent: indent, tag: "", isTag: false, desc: desc}
}

// parseTag parses from just after '@'. off points at the first char of the tag name.
func (f PhpdocAlign) parseTag(indent, line string, off int) *alignMatch {
	e := off
	for e < len(line) && !phpAlignIsWS(line[e]) {
		e++
	}
	tag := line[off:e]
	if tag == "" {
		return nil
	}

	tags := f.Tags
	if tags == nil {
		tags = phpAlignDefaultTags
	}
	if !slices.Contains(tags, tag) {
		return nil
	}
	if slices.Contains(phpAlignNameTags, tag) {
		return phpAlignParseNameTag(indent, tag, line, e)
	}
	if slices.Contains(phpAlignMethodTags, tag) {
		return phpAlignParseMethodTag(indent, tag, line, e)
	}
	return phpAlignParseNoNameTag(indent, tag, line, e)
}

// name tag: @tag \s+ (hint=REGEX_TYPES)? \s* (var=(?:&|\.{3})?\$\S+) (?:\s+ desc)? \h*$
func phpAlignParseNameTag(indent, tag, line string, pos int) *alignMatch {
	// require \s+
	p := pos
	for p < len(line) && phpAlignIsWS(line[p]) {
		p++
	}
	if p == pos {
		return nil // needed \s+
	}
	hintEnd := phpAlignHintEnd(line, p, func(end int) bool { return phpAlignVarStart(line, end) >= 0 })
	varStart := phpAlignVarStart(line, hintEnd)
	if varStart < 0 {
		return nil
	}
	hint := line[p:hintEnd]
	// var = \S+ from varStart
	ve := varStart
	for ve < len(line) && !phpAlignIsWS(line[ve]) {
		ve++
	}
	varName := line[varStart:ve]
	// desc: (?:\s+ (\V*))? then \h*$
	desc := ""
	q := ve
	for q < len(line) && phpAlignIsWS(line[q]) {
		q++
	}
	if q > ve && q < len(line) {
		desc = line[q:]
	}
	return &alignMatch{indent: indent, tag: tag, isTag: true, hint: hint, varName: varName, desc: desc}
}

// no-name tag: @tag \s+ (hint=REGEX_TYPES)? \s+ desc \h*$, the line break counting as \s
func phpAlignParseNoNameTag(indent, tag, line string, pos int) *alignMatch {
	p := pos
	for p < len(line) && phpAlignIsWS(line[p]) {
		p++
	}
	if p == pos {
		return nil
	}
	hintEnd := phpAlignHintEnd(line, p, func(end int) bool { return end == len(line) || phpAlignIsWS(line[end]) })
	if hintEnd == p && p < len(line) && p-pos < 2 {
		// with no hint, desc needs its own \s+; that only exists when 2+ leading
		// spaces let the hint's \s+ give one back. A single space -> no match.
		return nil
	}
	hint := line[p:hintEnd]
	desc := ""
	q := hintEnd
	for q < len(line) && phpAlignIsWS(line[q]) {
		q++
	}
	if q < len(line) {
		desc = line[q:]
	}
	return &alignMatch{indent: indent, tag: tag, isTag: true, hint: hint, varName: "", desc: desc}
}

// phpAlignHintEnd returns where (?:REGEX_TYPES)? at p ends once PCRE settles
// it, or p for no hint. Past the greedy match, PCRE backtracks into the first
// type only (the glue loop is possessive), longest reading first, and only a
// reading that a space follows can fit: "mixed <b>TRUE</b>" ends after mixed.
func phpAlignHintEnd(line string, p int, follows func(end int) bool) int {
	ts, ok := parsePHPTypes(line, p)
	if !ok {
		return p
	}
	if follows(ts.end) {
		return ts.end
	}
	for end := ts.members[0].end - 1; end > p; end-- {
		if !phpAlignIsWS(line[end]) || !follows(end) {
			continue
		}
		if t, ok := parsePHPType(line[:end], p); ok && t.end == end {
			return end
		}
	}
	return p
}

// phpAlignVarStart returns where (?:&|\.{3})?\$\S+ starts after the spaces at i, or -1.
func phpAlignVarStart(line string, i int) int {
	for i < len(line) && phpAlignIsWS(line[i]) {
		i++
	}
	d := i
	if strings.HasPrefix(line[d:], "&") {
		d++
	} else if strings.HasPrefix(line[d:], "...") {
		d += 3
	}
	if d+1 < len(line) && line[d] == '$' && !phpAlignIsWS(line[d+1]) {
		return i
	}
	return -1
}

// method tag: @tag (\s+static)? \s+ (hint)? \s+ (signature=.+\)) — best effort
func phpAlignParseMethodTag(indent, tag, line string, pos int) *alignMatch {
	// Fall back: treat as no-name if it doesn't clearly have a signature "(...)"
	if strings.IndexByte(line[pos:], '(') < 0 {
		return phpAlignParseNoNameTag(indent, tag, line, pos)
	}
	p := pos
	for p < len(line) && phpAlignIsWS(line[p]) {
		p++
	}
	if p == pos {
		return nil
	}
	static := ""
	if strings.HasPrefix(line[p:], "static") && (p+6 >= len(line) || phpAlignIsWS(line[p+6])) {
		static = "static"
		p += 6
		for p < len(line) && phpAlignIsWS(line[p]) {
			p++
		}
	}
	// hint then \s+ signature (ends with ')')
	sigParen := strings.LastIndexByte(line, ')')
	if sigParen < 0 {
		return nil
	}
	if p > sigParen {
		return nil
	}
	// signature start: after hint + \s+. Find the last top-level ws before the signature name.
	hintEnd, _ := phpAlignScanType(line, p)
	var hint, signature string
	if hintEnd > sigParen {
		// no return type: the scanned run is the name+signature itself
		signature = strings.TrimRight(line[p:sigParen+1], " \t")
	} else {
		hint = strings.TrimSpace(line[p:hintEnd])
		q := hintEnd
		for q < len(line) && phpAlignIsWS(line[q]) {
			q++
		}
		if q > sigParen {
			return nil
		}
		signature = strings.TrimRight(line[q:sigParen+1], " \t")
	}
	if !strings.HasSuffix(signature, ")") {
		return nil
	}
	// desc: (?:\s+ \V*) after the signature's closing paren
	desc := ""
	r := sigParen + 1
	for r < len(line) && phpAlignIsWS(line[r]) {
		r++
	}
	if r < len(line) {
		desc = line[r:]
	}
	// hint/static swap: if hint empty and static set, static becomes hint
	if hint == "" && static != "" {
		hint = static
		static = ""
	}
	return &alignMatch{indent: indent, tag: tag, isTag: true, hint: hint, varName: signature, static: static, desc: desc}
}

func phpAlignIsWS(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// phpAlignScanType consumes a PHPDoc type expression starting at pos, returning
// the index just past it. Approximates TypeExpression::REGEX_TYPES: it consumes
// type characters and bracketed groups, treating a top-level space as the end
// unless the next non-space is a union/intersection continuation (| or &).
// phpAlignScanType returns the index just past the type and whether it ended
// balanced. An unbalanced run (depth > 0 at end of line) means REGEX_TYPES would
// not match at all.
func phpAlignScanType(line string, pos int) (int, bool) {
	depth := 0
	i := pos
	last := pos // last index that is definitely part of the type
	for i < len(line) {
		c := line[i]
		switch c {
		case '<', '[', '(', '{':
			depth++
			i++
			last = i
		case '>', ']', ')', '}':
			if depth > 0 {
				depth--
			}
			i++
			last = i
		case ' ', '\t':
			if depth > 0 {
				i++
				continue
			}
			// top-level space: continuation only if next non-space is | or &
			k := i
			for k < len(line) && (line[k] == ' ' || line[k] == '\t') {
				k++
			}
			if k < len(line) && (line[k] == '|' || line[k] == '&') {
				i = k
				continue
			}
			return last, depth == 0
		case '\n', '\r':
			return last, depth == 0
		default:
			i++
			last = i
		}
	}
	return last, depth == 0
}
