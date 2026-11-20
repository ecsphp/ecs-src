package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/FunctionCallSignatureSniff.php
//
// PEARFunctionCallSignature fixes the spacing around a call's parentheses and,
// for a multi-line call, puts each argument line Indent spaces in from the call
// and the closing parenthesis on its own line.
type PEARFunctionCallSignature struct {
	Indent                    int
	AllowMultipleArguments    bool
	RequiredSpacesAfterOpen   int
	RequiredSpacesBeforeClose int
	// psr2MultiLine swaps in PSR2's isMultiLineCall: a call is multi-line once
	// any argument or the closing parenthesis starts a new line.
	psr2MultiLine bool
}

func (PEARFunctionCallSignature) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Functions\FunctionCallSignatureSniff`
}

func (PEARFunctionCallSignature) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/FunctionCallSignatureSniff.php"
}

func (c PEARFunctionCallSignature) WithConfig(config map[string]any) fixer.Fixer {
	c.Indent = configInt(config, "indent", c.Indent)
	c.RequiredSpacesAfterOpen = configInt(config, "requiredSpacesAfterOpen", c.RequiredSpacesAfterOpen)
	c.RequiredSpacesBeforeClose = configInt(config, "requiredSpacesBeforeClose", c.RequiredSpacesBeforeClose)
	if v, ok := config["allowMultipleArguments"].(bool); ok {
		c.AllowMultipleArguments = v
	}
	return c
}

var pearCallNameKeywords = map[string]bool{
	"include": true, "include_once": true, "require": true, "require_once": true,
	"isset": true, "unset": true, "empty": true, "static": true,
}

func (c PEARFunctionCallSignature) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if c.registered(f, i) {
				c.process(f, i)
			}
		}
	})
}

// registered mirrors the sniff's register(): FUNCTION_NAME_TOKENS, T_VARIABLE
// and the closing brackets.
func (PEARFunctionCallSignature) registered(f *slvFile, i int) bool {
	t := f.toks[i]
	switch t.kind {
	case token.Ident, token.Variable:
		return true
	case token.Keyword:
		lw := strings.ToLower(t.orig)
		return pearCallNameKeywords[lw] || lw == "class" && f.isAnonClass(i)
	case token.Punct:
		return t.orig == ")" || t.orig == "]" || t.orig == "}"
	}
	return false
}

func (f *slvFile) isAnonClass(i int) bool {
	p := f.prevEffective(i - 1)
	for p >= 0 && (f.isKeyword(p, "readonly") || f.toks[p].kind == token.AttributeEnd) {
		if f.toks[p].kind == token.AttributeEnd {
			p = f.toks[p].closer
		}
		p = f.prevEffective(p - 1)
	}
	return f.isKeyword(p, "new")
}

// phpcsScopeCloser returns the scope_closer PHP_CodeSniffer sets on the token
// at i (a scope owner, its opener or its closer), or -1.
func phpcsScopeCloser(f *slvFile, i int) int {
	if t := f.toks[i]; t.heredoc {
		// T_START_HEREDOC / T_START_NOWDOC open a scope closed by the end marker
		if !strings.HasPrefix(t.orig, "<<<") {
			if i+1 == len(f.toks) || !f.toks[i+1].heredoc {
				return i
			}
			return -1
		}
		for i+1 < len(f.toks) && f.toks[i+1].heredoc {
			i++
		}
		return i
	}
	if f.is(i, "}") {
		if o := f.toks[i].closer; o >= 0 && f.closesScope(i) && f.braceOwner(o) >= 0 {
			return i
		}
		return -1
	}
	if f.is(i, "{") && f.braceOwner(i) < 0 {
		return -1
	}
	return f.scopeCloser(i)
}

func (c PEARFunctionCallSignature) process(f *slvFile, stackPtr int) {
	if f.is(stackPtr, "}") && phpcsScopeCloser(f, stackPtr) >= 0 {
		return
	}
	openBracket := f.nextEffective(stackPtr+1, len(f.toks))
	if !f.is(openBracket, "(") || f.toks[openBracket].closer < 0 {
		return
	}
	previous := stackPtr - 1
	for previous >= 0 && (f.isEmpty(previous) || f.is(previous, "&")) {
		previous--
	}
	if f.isKeyword(previous, "function") {
		return
	}
	closeBracket := f.toks[openBracket].closer

	if !f.isKeyword(stackPtr, "class") && stackPtr+1 != openBracket {
		f.beginChangeset()
		for i := stackPtr + 1; i < openBracket; i++ {
			f.replaceToken(i, "")
		}
		f.replaceToken(openBracket, "(")
		f.endChangeset()
	}

	next := closeBracket + 1
	for next < len(f.toks) && f.toks[next].kind == token.Whitespace {
		next++
	}
	if f.is(next, ";") && f.isEmpty(closeBracket+1) {
		f.beginChangeset()
		for i := closeBracket + 1; i < next; i++ {
			f.replaceToken(i, "")
		}
		f.replaceToken(closeBracket, ")")
		f.endChangeset()
	}

	if c.isMultiLineCall(f, stackPtr, openBracket) {
		c.processMultiLine(f, stackPtr, openBracket)
		return
	}
	c.processSingleLine(f, openBracket)
}

func (c PEARFunctionCallSignature) isMultiLineCall(f *slvFile, stackPtr, openBracket int) bool {
	closeBracket := f.toks[openBracket].closer
	if !c.psr2MultiLine {
		return f.toks[openBracket].line != f.toks[closeBracket].line
	}
	next := f.nextEffective(openBracket+1, len(f.toks))
	if f.toks[next].line != f.toks[stackPtr].line {
		return true
	}
	end := phpcsEndOfStatement(f, openBracket+1)
	for f.is(end, ",") {
		next = f.nextEffective(end+1, closeBracket)
		if next < 0 {
			return false
		}
		if f.toks[next].line != f.toks[end].line {
			return true
		}
		end = phpcsEndOfStatement(f, next)
	}
	next = f.nextEffective(end+1, closeBracket)
	return next >= 0 && f.toks[next].line != f.toks[end].line
}

func (c PEARFunctionCallSignature) processSingleLine(f *slvFile, openBracket int) {
	closer := f.toks[openBracket].closer
	if openBracket == closer-1 {
		return
	}
	afterOpen, beforeClose := 0, 0
	for j := openBracket + 1; j < closer; j++ {
		if f.toks[j].kind != token.Whitespace {
			afterOpen, beforeClose = c.RequiredSpacesAfterOpen, c.RequiredSpacesBeforeClose
			break
		}
	}

	if afterOpen == 0 && f.toks[openBracket+1].kind == token.Whitespace {
		f.replaceToken(openBracket+1, "")
	} else if afterOpen > 0 {
		found := 0
		if f.toks[openBracket+1].kind == token.Whitespace {
			found = f.toks[openBracket+1].length
		}
		if found != afterOpen {
			padding := strings.Repeat(" ", afterOpen)
			if found == 0 {
				f.replaceToken(openBracket, f.tokenContent(openBracket)+padding)
			} else {
				f.replaceToken(openBracket+1, padding)
			}
		}
	}

	prev := closer - 1
	for prev > openBracket && f.toks[prev].kind == token.Whitespace {
		prev--
	}
	if f.toks[prev].heredoc {
		return
	}
	const newline = -1
	spaceBeforeClose := 0
	if f.toks[prev].line != f.toks[closer].line {
		spaceBeforeClose = newline
	} else if f.toks[closer-1].kind == token.Whitespace {
		spaceBeforeClose = f.toks[closer-1].length
	}
	if spaceBeforeClose == beforeClose {
		return
	}
	padding := strings.Repeat(" ", beforeClose)
	switch spaceBeforeClose {
	case 0:
		f.replaceToken(closer, padding+f.tokenContent(closer))
	case newline:
		f.beginChangeset()
		closingContent := ")"
		next := closer + 1
		for next < len(f.toks) && f.toks[next].kind == token.Whitespace {
			next++
		}
		if f.is(next, ";") {
			closingContent += ";"
			for i := closer + 1; i <= next; i++ {
				f.replaceToken(i, "")
			}
		}
		p := closer - 1
		for f.isEmpty(p) {
			if f.toks[p].kind == token.Comment && strings.Contains(f.toks[p].content, "*/") {
				break
			}
			p--
		}
		f.replaceToken(p, f.tokenContent(p)+padding+closingContent)
		prevNonWs := closer - 1
		for prevNonWs >= 0 && f.toks[prevNonWs].kind == token.Whitespace {
			prevNonWs--
		}
		for i := prevNonWs + 1; i <= closer; i++ {
			f.replaceToken(i, "")
		}
		f.endChangeset()
	default:
		f.replaceToken(closer-1, padding)
	}
}

// firstNonWhitespaceOnLineOf is File::findFirstOnLine(T_WHITESPACE, i, true).
func firstNonWhitespaceOnLineOf(f *slvFile, i int) int {
	found := -1
	for j := i; j >= 0 && f.toks[j].line == f.toks[i].line; j-- {
		if f.toks[j].kind != token.Whitespace {
			found = j
		}
	}
	return found
}

// isConstantString reports whether the token at i is T_CONSTANT_ENCAPSED_STRING.
func isConstantString(f *slvFile, i int) bool {
	if i < 0 {
		return false
	}
	t := f.toks[i]
	return t.kind == token.String && !t.heredoc && !t.interp
}

func (c PEARFunctionCallSignature) processMultiLine(f *slvFile, stackPtr, openBracket int) {
	first := firstNonWhitespaceOnLineOf(f, stackPtr)
	if first > 0 && isConstantString(f, first) && isConstantString(f, first-1) {
		prev := first - 2
		for prev >= 0 && isConstantString(f, prev) {
			prev--
		}
		first = -1
		for j := prev; j >= 0 && f.toks[j].line == f.toks[prev].line; j-- {
			if !f.isEmpty(j) {
				first = j
			}
		}
		if first < 0 {
			first = prev + 1
		}
	}

	foundFunctionIndent := 0
	if first >= 0 {
		if f.toks[first].kind == token.InlineHTML || first > 0 && isConstantString(f, first) && isConstantString(f, first-1) {
			content := f.toks[first].content
			trimmed := strings.TrimLeft(content, " \t\n\r\x00\x0B")
			if trimmed == "" {
				foundFunctionIndent = len(content)
			} else {
				foundFunctionIndent = len(content) - len(trimmed)
			}
		} else {
			foundFunctionIndent = f.toks[first].column - 1
		}
	}

	functionIndent := foundFunctionIndent / c.Indent * c.Indent
	adjustment := 0
	if foundFunctionIndent != functionIndent {
		adjustment = functionIndent - foundFunctionIndent
		padding := strings.Repeat(" ", functionIndent)
		switch {
		case foundFunctionIndent == 0:
			f.replaceToken(first, padding+f.tokenContent(first))
		case f.toks[first].kind == token.InlineHTML:
			f.replaceToken(first, padding+strings.TrimLeft(f.toks[first].content, " \t\n\r\x00\x0B"))
		default:
			f.replaceToken(first-1, padding)
		}
	}

	argIndent := strings.Repeat(" ", foundFunctionIndent+c.Indent)
	if next := f.nextEffective(openBracket+1, len(f.toks)); next >= 0 && f.toks[next].line == f.toks[openBracket].line {
		f.replaceToken(openBracket, f.tokenContent(openBracket)+f.eol+argIndent)
	}

	closeBracket := f.toks[openBracket].closer
	prev := closeBracket - 1
	for prev >= 0 && f.toks[prev].kind == token.Whitespace {
		prev--
	}
	if f.toks[prev].line == f.toks[closeBracket].line {
		f.replaceToken(closeBracket, f.eol+argIndent+f.tokenContent(closeBracket))
	}

	lastLine := f.toks[openBracket].line - 1
	argStart, argEnd := -1, -1

	i := openBracket + 1
	for i < len(f.toks) && f.toks[i].kind == token.Whitespace {
		i++
	}
	if f.toks[i].line > f.toks[openBracket].line+1 {
		f.beginChangeset()
		for x := openBracket + 1; x < i; x++ {
			if f.toks[x].line == f.toks[openBracket].line {
				continue
			}
			if f.toks[x].line == f.toks[i].line {
				break
			}
			f.replaceToken(x, "")
		}
		f.endChangeset()
	}

	i = f.nextEffective(openBracket+1, len(f.toks))
	if f.toks[i-1].kind == token.Whitespace && f.toks[i-1].line == f.toks[i].line {
		i--
	}
	for ; i < closeBracket; i++ {
		t := f.toks[i]
		inArg := i > argStart && i < argEnd
		if t.line != lastLine {
			lastLine = t.line
			if t.heredoc || t.kind == token.InlineHTML {
				continue
			}
			if t.kind == token.String && f.toks[i-1].kind == token.String && !f.toks[i-1].heredoc && t.interp == f.toks[i-1].interp {
				continue
			}
			var nextCode int
			if t.line != f.toks[openBracket].line {
				if t.kind == token.Whitespace {
					nextCode = i + 1
					for nextCode <= closeBracket && f.toks[nextCode].kind == token.Whitespace {
						nextCode++
					}
					if f.toks[nextCode].line != lastLine {
						if !inArg {
							f.replaceToken(i, "")
						}
						continue
					}
				} else {
					nextCode = i
				}

				expectedIndent := foundFunctionIndent + c.Indent + adjustment
				if f.toks[nextCode].line == f.toks[closeBracket].line {
					inArg = false
					expectedIndent = foundFunctionIndent + adjustment
				}

				foundIndent := 0
				isDocWs := t.kind == token.DocComment && t.doc == slvDocWhitespace
				if t.kind != token.Whitespace && !isDocWs {
					if t.kind == token.Comment && f.toks[i-1].kind == token.Comment {
						trimmed := strings.TrimLeft(t.content, " \t\n\r\x00\x0B")
						if trimmed == "" {
							continue
						}
						foundIndent = len(t.content) - len(trimmed)
					}
				} else {
					foundIndent = t.length
				}

				if foundIndent < expectedIndent || !inArg && expectedIndent != foundIndent {
					f.beginChangeset()
					padding := strings.Repeat(" ", expectedIndent)
					if foundIndent == 0 {
						f.replaceToken(i, padding+f.tokenContent(i))
						if closer := phpcsScopeCloser(f, i); closer >= 0 {
							phpcsChangeCodeBlockIndent(f, i, closer, expectedIndent)
						}
					} else {
						if t.kind == token.Comment {
							f.replaceToken(i, padding+strings.TrimLeft(t.content, " \t\n\r\x00\x0B"))
						} else {
							f.replaceToken(i, padding)
						}
						if closer := phpcsScopeCloser(f, i+1); closer >= 0 {
							phpcsChangeCodeBlockIndent(f, i+1, closer, expectedIndent-foundIndent)
						}
					}
					f.endChangeset()
				}
			} else {
				nextCode = i
			}
			if !inArg {
				argStart = nextCode
				argEnd = phpcsEndOfStatement(f, nextCode)
			}
		}

		if !inArg && f.is(i, ",") {
			next := f.nextEffective(i+1, closeBracket)
			if next < 0 {
				continue
			}
			if !c.AllowMultipleArguments && f.toks[i].line == f.toks[next].line {
				f.beginChangeset()
				for x := next - 1; x > i; x-- {
					if f.toks[x].kind != token.Whitespace {
						break
					}
					f.replaceToken(x, "")
				}
				f.replaceToken(next, f.eol+argIndent+f.tokenContent(next))
				f.endChangeset()
			}
			argStart = next
			argEnd = phpcsEndOfStatement(f, next)
		}
	}
}

// phpcsChangeCodeBlockIndent is Fixer::changeCodeBlockIndent.
func phpcsChangeCodeBlockIndent(f *slvFile, start, end, change int) {
	baseIndent := ""
	if change > 0 {
		baseIndent = strings.Repeat(" ", change)
	}
	own := !f.inChangeset
	if own {
		f.beginChangeset()
	}
	for i := start; i <= end && i+1 < len(f.toks); i++ {
		t := f.toks[i]
		if t.column != 1 || f.toks[i+1].line != t.line {
			continue
		}
		var v string
		if t.kind == token.Whitespace || t.kind == token.DocComment && t.doc == slvDocWhitespace {
			v = strings.Repeat(" ", max(t.length+change, 0)) + strings.TrimLeft(t.content, " \t\n\r\x00\x0B")
		} else {
			v = baseIndent + t.content
		}
		f.replaceToken(i, v)
	}
	if own {
		f.endChangeset()
	}
}

// isMatchArrow reports whether the "=>" at i is a match arm's T_MATCH_ARROW.
func (f *slvFile) isMatchArrow(i int) bool {
	return f.is(i, "=>") && f.fnOfArrow(i) < 0 && !f.isDoubleArrow(i)
}

// phpcsEndOfStatement is File::findEndOfStatement(start, [T_COLON]).
func phpcsEndOfStatement(f *slvFile, start int) int {
	if !f.isMatchArrow(start) {
		if match := phpcsOuterMatch(f, start); match >= 0 {
			opener, closer := match, f.toks[match].closer
			beforeArrow := true
			for p := start - 1; p >= opener; p-- {
				if f.isMatchArrow(p) {
					beforeArrow = false
					for c := p + 1; c < start; c++ {
						if f.is(c, ",") {
							beforeArrow = true
							break
						}
					}
					break
				}
			}
			if beforeArrow {
				for n := start + 1; n <= closer; n++ {
					if f.isMatchArrow(n) {
						start = n
						break
					}
				}
			}
		}
	}

	lastNotEmpty := start
	for i := start; i < len(f.toks); i++ {
		t := f.toks[i]
		if i != start {
			switch {
			case t.kind == token.Punct && (t.orig == "," || t.orig == ";"),
				t.kind == token.Punct && t.orig == "=>" && f.isDoubleArrow(i):
				return i
			case t.kind == token.Punct && (t.orig == ")" || t.orig == "]" || t.orig == "}"),
				t.kind == token.OpenTag, t.kind == token.CloseTag:
				return lastNotEmpty
			}
		}
		if closer := phpcsScopeCloser(f, i); closer >= 0 && closer != i {
			if f.isKeyword(i, "fn") {
				lastNotEmpty = closer
				i = closer - 1
				continue
			}
			if i == start && t.kind == token.Keyword {
				return closer
			}
			i = closer
		} else if (f.is(i, "[") || f.is(i, "(") || f.is(i, "{")) && t.closer >= 0 {
			i = t.closer
		}
		if !f.isEmpty(i) {
			lastNotEmpty = i
		}
	}
	return len(f.toks) - 1
}

// phpcsOuterMatch returns the scope opener of the outermost match whose body
// holds i, or -1.
func phpcsOuterMatch(f *slvFile, i int) int {
	found := -1
	for o := f.enclosingScope(i); o >= 0; o = f.enclosingScope(o) {
		if owner := f.braceOwner(o); owner >= 0 && f.isKeyword(owner, "match") {
			found = o
		}
	}
	return found
}
