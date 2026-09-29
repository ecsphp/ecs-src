package rules

import (
	"slices"
	"strings"
)

func phpAlignIsHSpace(c byte) bool { return c == ' ' || c == '\t' }

// phpAlignGetMatches parses a docblock line, mirroring PhpdocAlignFixer::getMatches
// for the laravel config (default tags). Returns nil when the line is not a tag
// line (or, when matchCommentOnly, not a description continuation line).
func phpAlignGetMatches(rawLine string, matchCommentOnly bool) *alignMatch {
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
		if m := phpAlignParseTag(indent, line, j+1); m != nil {
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

// phpAlignParseTag parses from just after '@'. off points at the first char of the tag name.
func phpAlignParseTag(indent, line string, off int) *alignMatch {
	// read tag name = [a-zA-Z0-9_-]+
	e := off
	for e < len(line) && phpAlignTagNameChar(line[e]) {
		e++
	}
	tag := line[off:e]
	if tag == "" {
		return nil
	}

	if phpAlignInList(tag, phpAlignNameTags) {
		return phpAlignParseNameTag(indent, tag, line, e)
	}
	if phpAlignInList(tag, phpAlignNoNameTags) {
		return phpAlignParseNoNameTag(indent, tag, line, e)
	}
	// method tags not aligned in this config path unless present; DEFAULT_TAGS has "method"
	if phpAlignInList(tag, phpAlignMethodTags) {
		return phpAlignParseMethodTag(indent, tag, line, e)
	}
	return nil
}

func phpAlignTagNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

func phpAlignInList(tag string, list []string) bool {
	return slices.Contains(list, tag)
}

// name tag: @tag \s+ (hint)? \s* (var=(?:&|\.{3})?\$\S+) (?:\s+ desc)? \h*$
func phpAlignParseNameTag(indent, tag, line string, pos int) *alignMatch {
	// require \s+
	p := pos
	for p < len(line) && phpAlignIsWS(line[p]) {
		p++
	}
	if p == pos {
		return nil // needed \s+
	}
	// find the var: a '$' at bracket depth 0 whose token is separated from the
	// hint by whitespace. A callable/Closure carries `$param` inside parentheses
	// (depth > 0) and a union member like `Foo|$this` attaches with no space, so
	// both stay in the hint; the documented variable follows a space.
	dollar := -1
	depth := 0
	for k := p; k < len(line); k++ {
		switch line[k] {
		case '<', '[', '(', '{':
			depth++
		case '>', ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case '$':
			if depth == 0 {
				ps := k
				if k-1 >= p && line[k-1] == '&' {
					ps = k - 1
				} else if k-3 >= p && line[k-3:k] == "..." {
					ps = k - 3
				}
				if ps == p || phpAlignIsWS(line[ps-1]) {
					dollar = k
				}
			}
		}
		if dollar >= 0 {
			break
		}
	}
	if dollar < 0 {
		return nil
	}
	varStart := dollar
	if varStart-1 >= p && line[varStart-1] == '&' {
		varStart--
	} else if varStart-3 >= p && line[varStart-3:varStart] == "..." {
		varStart -= 3
	}
	hint := strings.TrimRight(line[p:varStart], " \t")
	hint = strings.TrimSpace(hint)
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

// no-name tag: @tag \s+ (hint=REGEX_TYPES)? (?:\s+ desc)? \h*$
func phpAlignParseNoNameTag(indent, tag, line string, pos int) *alignMatch {
	p := pos
	for p < len(line) && phpAlignIsWS(line[p]) {
		p++
	}
	if p == pos {
		return nil
	}
	hintEnd, balanced := phpAlignScanType(line, p)
	if !balanced {
		// REGEX_TYPES cannot match an unbalanced type, so the hint is empty and
		// desc needs its own \s+; that only exists when 2+ leading spaces let the
		// hint's \s+ give one back. A single space leaves no split -> no match.
		if p-pos < 2 {
			return nil
		}
		return &alignMatch{indent: indent, tag: tag, isTag: true, hint: "", varName: "", desc: line[p:]}
	}
	hint := strings.TrimSpace(line[p:hintEnd])
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
