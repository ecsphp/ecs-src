package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/ControlStructures/InlineControlStructureSniff.php
//
// InlineControlStructure wraps the body of a brace-less control structure in
// braces: "if ($a) echo 1;" -> "if ($a) { echo 1;\n}".
type InlineControlStructure struct{}

func (InlineControlStructure) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\ControlStructures\InlineControlStructureSniff`
}

func (InlineControlStructure) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/ControlStructures/InlineControlStructureSniff.php"
}

var inlineControlKeywords = map[string]bool{
	"if": true, "else": true, "elseif": true, "foreach": true, "while": true, "do": true, "for": true,
}

func (InlineControlStructure) Fix(s *tokens.Stream) bool {
	if !s.Index().HasAnyWord("if", "else", "elseif", "foreach", "while", "do", "for") {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool {
			return f.toks[i].kind == token.Keyword && inlineControlKeywords[strings.ToLower(f.toks[i].orig)]
		}, func(i int) int {
			return inlineControlProcess(f, i)
		})
	})
}

func inlineControlProcess(f *slvFile, ptr int) int {
	n := len(f.toks)
	kw := strings.ToLower(f.toks[ptr].orig)
	if _, _, ok := f.icsScope(ptr); ok {
		return ptr + 1
	}
	if kw == "else" {
		if next := f.nextEffective(ptr+1, n); next < 0 || f.isKeyword(next, "if") {
			return ptr + 1
		}
	}

	_, parenCloser := f.icsParens(ptr)
	nextTokenIndex := -1
	if parenCloser >= 0 {
		nextTokenIndex = parenCloser + 1
	} else if kw == "else" || kw == "do" {
		nextTokenIndex = ptr + 1
	}
	if nextTokenIndex >= 0 {
		first := f.nextEffective(nextTokenIndex, n)
		if first < 0 {
			return ptr + 1
		}
		if f.is(first, ";") {
			return ptr + 1
		}
	}

	if parenCloser < 0 && kw != "else" {
		if kw != "do" {
			return ptr + 1
		}
		for w := ptr + 1; w < n; w++ {
			if f.isKeyword(w, "while") {
				if _, c := f.icsParens(w); c < 0 {
					return ptr + 1
				}
				break
			}
		}
	}

	start := ptr
	if parenCloser >= 0 {
		start = parenCloser
	}
	if nn := f.nextEffective(start+1, n); nn >= 0 && (f.is(nn, "{") || f.is(nn, ":")) {
		return n
	}

	f.beginChangeset()
	closer := ptr
	if parenCloser >= 0 {
		closer = parenCloser
	}
	if closer+1 < n && f.toks[closer+1].kind == token.Whitespace {
		f.icsAdd(closer, " {")
	} else {
		f.icsAdd(closer, " { ")
	}

	lastNonEmpty := closer
	end := closer + 1
	for ; end < n; end++ {
		t := f.toks[end]
		if f.is(end, ";") {
			break
		}
		if t.kind == token.CloseTag {
			end = lastNonEmpty
			break
		}
		owner := ""
		if t.kind == token.Keyword {
			owner = strings.ToLower(t.orig)
		}
		_, scopeCloser, hasScope := f.icsScope(end)
		if inlineControlKeywords[owner] && !hasScope {
			f.inChangeset = false
			f.changeset = nil
			return ptr + 1
		}
		if hasScope {
			end = scopeCloser
			switch owner {
			case "do", "if", "elseif", "try", "catch", "finally":
				next := f.nextEffective(end+1, n)
				if next < 0 {
					goto done
				}
				nextKw := ""
				if f.toks[next].kind == token.Keyword {
					nextKw = strings.ToLower(f.toks[next].orig)
				}
				if (owner == "if" || owner == "elseif") && (nextKw == "elseif" || nextKw == "else") {
					continue
				}
				if (owner == "try" || owner == "catch" || owner == "finally") && (nextKw == "catch" || nextKw == "finally") {
					continue
				}
				if owner == "do" && nextKw == "while" {
					end = -1
					for j := next + 1; j < n; j++ {
						if f.is(j, ";") {
							end = j
							break
						}
					}
					if end < 0 {
						end = n
						goto done
					}
				}
			case "function":
				if next := f.nextEffective(end+1, n); next >= 0 && f.is(next, ";") {
					end = next
				}
			}
			break
		}
		if pc := f.icsParenCloserOf(end); pc >= 0 {
			end = pc
			lastNonEmpty = end
			continue
		}
		if t.kind != token.Whitespace {
			lastNonEmpty = end
		}
	}
done:
	if end >= n {
		end = lastNonEmpty
	}

	endLine := end
	nextContent := f.nextEffective(end+1, n)
	if nextContent < 0 || f.toks[nextContent].line != f.toks[end].line {
		for endLine = end; endLine < n; endLine++ {
			if endLine+1 >= n || f.toks[endLine].line != f.toks[endLine+1].line {
				break
			}
		}
		if endLine >= n {
			endLine = n - 1
		}
		isComment := func(k token.Kind) bool { return k == token.Comment || k == token.DocComment }
		if !isComment(f.toks[endLine].kind) &&
			(f.toks[endLine].kind != token.Whitespace || !isComment(f.toks[endLine-1].kind)) {
			endLine = end
		}
	}

	var endToken int
	var added string
	if endLine != end {
		endToken = endLine
	} else {
		endToken = end
		added = f.eol
		if !f.is(end, ";") && !f.is(end, "}") {
			f.icsAdd(end, "; ")
		}
	}

	next := f.nextNonWhitespace(endToken+1, n)
	if next >= 0 && (f.isKeyword(next, "else") || f.isKeyword(next, "elseif")) {
		f.replaceToken(next, "} "+f.tokenContent(next))
	} else {
		indent := ""
		first := ptr
		for ; first > 0; first-- {
			if f.toks[first].column == 1 {
				break
			}
		}
		switch f.toks[first].kind {
		case token.Whitespace:
			indent = f.toks[first].content
		case token.InlineHTML, token.OpenTag:
			added = ""
		}
		added += indent + "}"
		if next >= 0 && f.toks[endToken].kind == token.Comment && !isPhpcsAnnotation(f.toks[endToken].orig) {
			added += f.eol
		}
		f.icsAdd(endToken, added)
	}
	f.endChangeset()
	return ptr + 1
}

// icsAdd is Fixer::addContent.
func (f *slvFile) icsAdd(i int, v string) {
	f.replaceToken(i, f.tokenContent(i)+v)
}

// icsParens returns the condition parentheses PHP_CodeSniffer gives the
// control keyword at i (parenthesis_opener/closer), or -1, -1.
func (f *slvFile) icsParens(i int) (int, int) {
	switch strings.ToLower(f.toks[i].orig) {
	case "if", "elseif", "while", "for", "foreach", "switch", "catch", "declare", "match":
	default:
		return -1, -1
	}
	o := f.nextEffective(i+1, len(f.toks))
	if !f.is(o, "(") || f.toks[o].closer < 0 {
		return -1, -1
	}
	return o, f.toks[o].closer
}

// icsParenCloserOf is parenthesis_closer of the token at i: a "(" or a keyword
// owning parentheses, -1 otherwise.
func (f *slvFile) icsParenCloserOf(i int) int {
	if f.is(i, "(") {
		return f.toks[i].closer
	}
	if f.toks[i].kind != token.Keyword && f.toks[i].kind != token.Ident {
		return -1
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "array", "list", "function", "fn", "use", "isset", "unset", "empty", "exit", "die", "eval",
		"if", "elseif", "while", "for", "foreach", "switch", "catch", "declare", "match":
	default:
		return -1
	}
	o := f.nextEffective(i+1, len(f.toks))
	if !f.is(o, "(") {
		return -1
	}
	return f.toks[o].closer
}

var icsAltEnd = map[string]string{
	"if": "endif", "elseif": "endif", "else": "endif", "while": "endwhile", "for": "endfor",
	"foreach": "endforeach", "switch": "endswitch", "declare": "enddeclare",
}

// icsScope returns the scope opener and closer PHP_CodeSniffer gives the
// keyword at i, if any.
func (f *slvFile) icsScope(i int) (int, int, bool) {
	t := f.toks[i]
	if t.kind != token.Keyword {
		return 0, 0, false
	}
	n := len(f.toks)
	kw := strings.ToLower(t.orig)
	var q int
	switch kw {
	case "if", "elseif", "while", "for", "foreach", "switch", "catch", "declare", "match":
		_, c := f.icsParens(i)
		if c < 0 {
			return 0, 0, false
		}
		q = f.nextEffective(c+1, n)
	case "else", "do", "try", "finally":
		q = f.nextEffective(i+1, n)
	case "function", "class":
		for q = i + 1; q < n; q++ {
			if f.is(q, ";") || f.is(q, "{") {
				break
			}
			if f.is(q, "(") && f.toks[q].closer > q {
				q = f.toks[q].closer
			}
		}
	case "fn":
		return f.icsArrowScope(i)
	default:
		return 0, 0, false
	}
	if q < 0 || q >= n {
		return 0, 0, false
	}
	if f.is(q, "{") && f.toks[q].closer > q {
		return q, f.toks[q].closer, true
	}
	if f.is(q, ":") {
		if endKw, ok := icsAltEnd[kw]; ok {
			if c := f.icsAltCloser(kw, q, endKw); c >= 0 {
				return q, c, true
			}
		}
	}
	return 0, 0, false
}

// icsAltCloser finds the end of an alternative-syntax scope opened by ":" at q:
// the matching end keyword, or for if/elseif the next else/elseif of the same level.
func (f *slvFile) icsAltCloser(kw string, q int, endKw string) int {
	depth := 0
	for j := q + 1; j < len(f.toks); j++ {
		if f.is(j, "{") && f.toks[j].closer > j {
			j = f.toks[j].closer
			continue
		}
		if f.toks[j].kind != token.Keyword {
			continue
		}
		w := strings.ToLower(f.toks[j].orig)
		switch {
		case w == endKw:
			if depth == 0 {
				return j
			}
			depth--
		case icsAltEnd[w] == endKw && (w == kw || endKw != "endif" || w == "if"):
			if w == "else" || w == "elseif" {
				if depth == 0 && (kw == "if" || kw == "elseif") && f.icsIsAlt(j) {
					return j
				}
				continue
			}
			if f.icsIsAlt(j) {
				depth++
			}
		}
	}
	return -1
}

// icsIsAlt reports whether the control keyword at j opens with ":".
func (f *slvFile) icsIsAlt(j int) bool {
	q := j + 1
	if _, c := f.icsParens(j); c >= 0 {
		q = c + 1
	}
	return f.is(f.nextEffective(q, len(f.toks)), ":")
}

// icsArrowScope mirrors PHP_CodeSniffer's arrow function scope: "=>" up to the
// first ";", ",", ")", "]", "}", ":" (outside a ternary) or close tag.
func (f *slvFile) icsArrowScope(i int) (int, int, bool) {
	n := len(f.toks)
	arrow := -1
	for j := i + 1; j < n; j++ {
		if f.is(j, "(") && f.toks[j].closer > j {
			j = f.toks[j].closer
			continue
		}
		if f.is(j, "=>") {
			arrow = j
			break
		}
		if f.is(j, ";") || f.is(j, "{") {
			return 0, 0, false
		}
	}
	if arrow < 0 {
		return 0, 0, false
	}
	inTernary := false
	for j := arrow + 1; j < n; j++ {
		t := f.toks[j]
		if t.kind == token.CloseTag || t.kind == token.OpenTag {
			return arrow, j, true
		}
		if t.kind != token.Punct {
			continue
		}
		switch t.orig {
		case "(", "[", "{":
			if t.closer > j {
				j = t.closer
			}
		case "?":
			inTernary = true
		case ":":
			if inTernary {
				inTernary = false
				continue
			}
			return arrow, j, true
		case ";", ",", ")", "]", "}":
			return arrow, j, true
		}
	}
	return 0, 0, false
}

// isPhpcsAnnotation reports whether PHP_CodeSniffer retokenizes the comment as
// one of its T_PHPCS_* annotation tokens instead of T_COMMENT.
func isPhpcsAnnotation(v string) bool {
	text := strings.ToLower(strings.TrimLeft(v, " \t/*#"))
	return strings.HasPrefix(text, "phpcs:") || strings.HasPrefix(text, "@phpcs:")
}
