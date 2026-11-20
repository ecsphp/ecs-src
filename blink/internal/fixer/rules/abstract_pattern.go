package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// abstractPatternElem is one parsed element of a PHP_CodeSniffer
// AbstractPatternSniff pattern: a token, a skip to the closing parenthesis or
// brace, a name ("abc") or a newline ("EOL").
type abstractPatternElem struct {
	kind  byte // 't' token, 'k' skip, 's' string, 'n' newline
	code  string
	value string
	to    string // skip: "parenthesis_closer" or "scope_closer"
}

type abstractPattern struct {
	code   string // the pattern as the sniff writes it
	listen int    // listen_pos: the element the sniff registers for
	elems  []abstractPatternElem
}

// abstractPatternWeights is the part of Tokens::WEIGHTINGS the patterns use.
var abstractPatternWeights = map[string]int{
	"function": 100, "while": 50, "for": 50, "foreach": 50, "if": 50, "else": 50, "elseif": 50, "do": 50, "match": 50,
}

// abstractPatternParse is AbstractPatternSniff::parse for the patterns blink
// ports: ' ' is a whitespace token, "abc" a name, "EOL" a newline and "..."
// a skip to the closer of the "(" or "{" before it, whose closer the skip
// swallows. Any other run of characters up to a space is one token.
func abstractPatternParse(code string) abstractPattern {
	p := abstractPattern{code: code}
	for i := 0; i < len(code); {
		switch {
		case strings.HasPrefix(code[i:], "..."):
			to := abstractPatternSkipTo(code, i-1)
			p.elems = append(p.elems, abstractPatternElem{kind: 'k', to: to})
			i += 4
		case strings.HasPrefix(code[i:], "abc"):
			p.elems = append(p.elems, abstractPatternElem{kind: 's'})
			i += 3
		case strings.HasPrefix(code[i:], "EOL"):
			p.elems = append(p.elems, abstractPatternElem{kind: 'n'})
			i += 3
		case code[i] == ' ':
			p.elems = append(p.elems, abstractPatternElem{kind: 't', code: "whitespace", value: " "})
			i++
		case strings.ContainsRune("(){};", rune(code[i])):
			p.elems = append(p.elems, abstractPatternElem{kind: 't', code: code[i : i+1], value: code[i : i+1]})
			i++
		default:
			e := i
			for e < len(code) && code[e] >= 'a' && code[e] <= 'z' && !strings.HasPrefix(code[e:], "abc") {
				e++
			}
			p.elems = append(p.elems, abstractPatternElem{kind: 't', code: code[i:e], value: code[i:e]})
			i = e
		}
	}
	highest := -1
	for n, e := range p.elems {
		if e.kind == 't' && abstractPatternWeights[e.code] > highest {
			highest = abstractPatternWeights[e.code]
			p.listen = n
		}
	}
	return p
}

// abstractPatternSkipTo is createSkipPattern: the closer of the first unclosed
// "(" or "{" at or before from.
func abstractPatternSkipTo(code string, from int) string {
	parens, braces := 0, 0
	for i := from; i >= 0; i-- {
		switch code[i] {
		case '(':
			if parens == 0 {
				return "parenthesis_closer"
			}
			parens--
		case '{':
			if braces == 0 {
				return "scope_closer"
			}
			braces--
		case ')':
			parens++
		case '}':
			braces++
		}
	}
	return "unknown"
}

// abstractPatternFile is the PHP_CodeSniffer token data the patterns read.
type abstractPatternFile struct {
	m              *siModel
	ignoreComments bool
}

func (a *abstractPatternFile) code(i int) string {
	if a.isComment(i) {
		return "comment"
	}
	return a.m.code[i]
}

func (a *abstractPatternFile) content(i int) string { return a.m.f.toks[i].content }

func (a *abstractPatternFile) line(i int) int { return a.m.f.toks[i].line }

func (a *abstractPatternFile) isComment(i int) bool {
	k := a.m.f.toks[i].kind
	return k == token.Comment || k == token.DocComment
}

func (a *abstractPatternFile) ignored(i int) bool {
	return a.m.f.toks[i].kind == token.Whitespace || a.ignoreComments && a.isComment(i)
}

func (a *abstractPatternFile) str(start, end int) string {
	var b strings.Builder
	for j := max(start, 0); j < end && j < len(a.m.f.toks); j++ {
		b.WriteString(a.content(j))
	}
	return b.String()
}

func (a *abstractPatternFile) prev(i int, skip func(int) bool) int {
	for ; i >= 0; i-- {
		if !skip(i) {
			return i
		}
	}
	return -1
}

func (a *abstractPatternFile) next(i int, skip func(int) bool) int {
	for ; i < len(a.m.f.toks); i++ {
		if i >= 0 && !skip(i) {
			return i
		}
	}
	return -1
}

// hasParenOpener is isset($tokens[$i]['parenthesis_opener']).
func (a *abstractPatternFile) hasParenOpener(i int) bool {
	f := a.m.f
	if f.is(i, "(") || f.is(i, ")") {
		return f.toks[i].closer >= 0
	}
	return a.m.sc.pCloser[i] >= 0
}

func (a *abstractPatternFile) parenOpener(i int) int {
	f := a.m.f
	switch {
	case f.is(i, "("):
		return i
	case f.is(i, ")"):
		return f.toks[i].closer
	}
	return f.toks[a.m.sc.pCloser[i]].closer
}

// process is AbstractPatternSniff::processPattern: ok is false when the pattern
// does not apply, else msg is the error, or "" when the code matches.
func (a *abstractPatternFile) process(p abstractPattern, stackPtr int) (msg string, ok bool) {
	toks := a.m.f.toks
	sc := a.m.sc
	eol := a.m.f.eol
	n := len(toks)
	pat := p.elems
	found := ""
	hasError := false
	orig := stackPtr
	if p.listen > 0 {
		stackPtr--
		for i := p.listen - 1; i >= 0; i-- {
			if stackPtr < 0 {
				return "", false
			}
			switch pat[i].kind {
			case 't':
				if pat[i].code == "whitespace" {
					if a.code(stackPtr) == "whitespace" {
						found = a.content(stackPtr) + found
					}
					if i != 0 && a.content(stackPtr) != pat[i].value {
						hasError = true
					}
					continue
				}
				prev := a.prev(stackPtr, a.ignored)
				if prev < 0 || a.code(prev) != pat[i].code {
					return "", false
				}
				found = a.content(prev) + a.str(prev+1, stackPtr) + found
				if i > 0 && pat[i-1].kind == 'k' {
					stackPtr = prev
				} else {
					stackPtr = prev - 1
				}
			case 'k':
				next := a.prev(stackPtr, a.ignored)
				if pat[i].to == "parenthesis_closer" {
					if next < 0 || !a.hasParenOpener(next) {
						return "", false
					}
					found = "...{" + found
					stackPtr = a.parenOpener(next) - 1
				} else {
					if next < 0 || sc.opener[next] < 0 {
						return "", false
					}
					found = "...(" + found
					stackPtr = sc.opener[next] - 1
				}
			case 's':
				found = "abc"
			case 'n':
				if a.ignoreComments && a.isComment(stackPtr) {
					start := a.prev(stackPtr-1, a.isComment)
					if a.line(start) != a.line(start+1) {
						start++
					}
					found = a.str(start, stackPtr+1) + found
					stackPtr = start - 1
				}
				if a.code(stackPtr) == "whitespace" {
					if a.content(stackPtr) != eol {
						found = a.content(stackPtr) + found
						if stackPtr > 0 && a.content(stackPtr-1) != eol && a.ignoreComments && !a.isComment(stackPtr-1) {
							hasError = true
						} else {
							stackPtr--
						}
					} else {
						found = "EOL" + found
					}
				} else {
					found = a.content(stackPtr) + found
					hasError = true
				}
				if !hasError && (i == 0 || pat[i-1].kind != 'n') {
					if prev := a.prev(stackPtr-1, a.ignored); prev >= 0 && a.line(prev) != a.line(stackPtr) {
						hasError = true
					}
				}
			}
		}
	}

	stackPtr = orig
	lastAdded := -1
	if stackPtr+len(pat)-p.listen > n {
		return "", false
	}
	for i := p.listen; i < len(pat); i++ {
		if stackPtr >= n {
			break
		}
		nextIsSkip := i+1 < len(pat) && pat[i+1].kind == 'k'
		switch pat[i].kind {
		case 't':
			if pat[i].code == "whitespace" {
				if a.ignoreComments && (a.isComment(stackPtr) || stackPtr+1 < n && a.isComment(stackPtr+1)) {
					continue
				}
				tokenContent := ""
				if a.code(stackPtr) == "whitespace" {
					if i+1 == len(pat) {
						tokenContent = a.content(stackPtr)
					} else {
						next := a.next(stackPtr, func(j int) bool { return toks[j].kind == token.Whitespace || a.isComment(j) })
						if next < 0 {
							next = n
						}
						tokenContent = a.str(stackPtr, next)
						lastAdded = stackPtr
						stackPtr = next
					}
					if stackPtr != lastAdded {
						found += tokenContent
					}
				} else if stackPtr != lastAdded {
					found += a.content(stackPtr)
					lastAdded = stackPtr
				}
				if nextIsSkip && !strings.HasPrefix(tokenContent, pat[i].value) || !nextIsSkip && tokenContent != pat[i].value {
					hasError = true
				}
				continue
			}
			next := a.next(stackPtr, a.ignored)
			if next < 0 || a.code(next) != pat[i].code {
				return "", false
			}
			if lastAdded >= 0 {
				c := a.code(next)
				if (c == "{" || c == "}") && sc.cond[next] >= 0 && sc.cond[next] > lastAdded {
					return "", false
				}
				if (c == "(" || c == ")") && a.m.pOwner[next] >= 0 && a.m.pOwner[next] > lastAdded {
					return "", false
				}
			}
			if next > stackPtr {
				hasComment := false
				for j := stackPtr; j < next; j++ {
					found += a.content(j)
					if a.isComment(j) {
						hasComment = true
					}
				}
				if !a.ignoreComments || !hasComment {
					hasError = true
				}
				if a.line(next) != a.line(stackPtr) {
					hasError = true
				}
			}
			if next != lastAdded {
				found += a.content(next)
				lastAdded = next
			}
			if nextIsSkip {
				stackPtr = next
			} else {
				stackPtr = next + 1
			}
		case 'k':
			next := a.prev(stackPtr, func(j int) bool { c := a.code(j); return c != "{" && c != "[" && c != "(" })
			if next < 0 {
				return "", false
			}
			if pat[i].to == "parenthesis_closer" {
				if !a.m.f.is(next, "(") || toks[next].closer < 0 {
					return "", false
				}
				found += "...)"
				stackPtr = toks[next].closer + 1
			} else {
				if !a.m.f.is(next, "{") || sc.closer[next] < 0 {
					return "", false
				}
				found += "...}"
				stackPtr = sc.closer[next] + 1
			}
		case 's':
			if a.code(stackPtr) != "string" {
				hasError = true
			}
			if stackPtr != lastAdded {
				found += "abc"
				lastAdded = stackPtr
			}
			stackPtr++
		case 'n':
			newline := 0
			for j := stackPtr; j < n; j++ {
				if strings.Contains(a.content(j), eol) {
					newline = j
					break
				}
			}
			var next int
			if newline == 0 {
				next = n - 1
				hasError = true
			} else {
				if !a.ignoreComments && a.isComment(newline) {
					hasError = true
				}
				if newline == stackPtr {
					next = stackPtr + 1
				} else {
					next = a.next(stackPtr, a.ignored)
					if next < newline {
						hasError = true
					} else {
						next = newline + 1
					}
				}
			}
			if stackPtr != lastAdded {
				found += a.str(stackPtr, next)
				lastAdded = next - 1
			}
			stackPtr = next
		}
	}
	if !hasError {
		return "", true
	}
	found = strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\n`, "\t", `\t`).Replace(found)
	found = strings.ReplaceAll(found, "EOL", `\n`)
	return `Expected "` + strings.ReplaceAll(p.code, "EOL", `\n`) + `"; found "` + found + `"`, true
}

// abstractPatternErrors runs the patterns over every token they listen for, as
// AbstractPatternSniff::process. listens tells which tokens a code registers.
func abstractPatternErrors(s *tokens.Stream, checker string, patterns []abstractPattern, ignoreComments bool, listens func(m *siModel, i int) bool) []fixer.CodingStandardError {
	m := newSiModel(newSlvFile(lexer.Lex(s.Render())))
	a := &abstractPatternFile{m: m, ignoreComments: ignoreComments}
	var errs []fixer.CodingStandardError
	for i := range m.f.toks {
		if !listens(m, i) {
			continue
		}
		msg := ""
		for _, p := range patterns {
			if p.elems[p.listen].code != m.code[i] {
				continue
			}
			e, ok := a.process(p, i)
			if !ok {
				continue
			}
			if e == "" {
				break
			}
			if msg == "" {
				msg = e
			}
		}
		if msg != "" {
			errs = append(errs, fixer.CodingStandardError{Line: m.f.toks[i].line, Message: msg, Checker: checker + ".Found"})
		}
	}
	return errs
}
