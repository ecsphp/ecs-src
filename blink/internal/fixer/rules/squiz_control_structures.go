package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsNewline marks a PHP_CodeSniffer spacing count: a number of spaces, or -1
// for "newline".
const phpcsNewline = -1

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ControlSignatureSniff.php
//
// SquizControlSignature puts one space after a control keyword and after its
// condition, the body on the line after the opening brace and one space
// between a closing brace and else, elseif, catch, finally or do-while.
type SquizControlSignature struct {
	RequiredSpacesBeforeColon int
}

func (SquizControlSignature) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\ControlSignatureSniff`
}

func (SquizControlSignature) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ControlSignatureSniff.php"
}

func (r SquizControlSignature) WithConfig(config map[string]any) fixer.Fixer {
	r.RequiredSpacesBeforeColon = configInt(config, "requiredSpacesBeforeColon", r.RequiredSpacesBeforeColon)
	return r
}

var squizControlSignatureCodes = phpcsSet("try", "catch", "finally", "do", "while", "for", "if", "foreach",
	"else", "elseif", "switch", "match")

func (r SquizControlSignature) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if squizControlSignatureCodes[sc.code[i]] {
				r.process(sc, i)
			}
		}
	})
}

// phpcsWhitespaceFound counts the whitespace token at i the way the sniffs
// report it: 0 when it is not whitespace, -1 when it holds a newline.
func phpcsWhitespaceFound(f *slvFile, i int) int {
	if i >= len(f.toks) || f.toks[i].kind != token.Whitespace {
		return 0
	}
	if strings.Contains(f.toks[i].content, f.eol) {
		return phpcsNewline
	}
	return f.toks[i].length
}

func (r SquizControlSignature) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	if f.nextEffective(stackPtr+1, len(f.toks)) < 0 {
		return
	}
	opener := sc.opener[stackPtr]
	isAlternative := opener >= 0 && sc.code[opener] == ":"
	parenCloser := sc.pCloser[stackPtr]

	expected := 1
	if parenCloser < 0 && isAlternative {
		expected = r.RequiredSpacesBeforeColon
	}
	found := 1
	if stackPtr+1 < len(f.toks) && (f.toks[stackPtr+1].kind != token.Whitespace || f.toks[stackPtr+1].content != " ") {
		found = phpcsWhitespaceFound(f, stackPtr+1)
	}
	if found != expected {
		if found == 0 {
			f.addContent(stackPtr, strings.Repeat(" ", expected))
		} else {
			f.replaceToken(stackPtr+1, strings.Repeat(" ", expected))
		}
	}

	if parenCloser >= 0 && opener >= 0 {
		r.afterCloseParenthesis(f, parenCloser, opener, isAlternative)
	}

	if opener >= 0 {
		squizNewlineAfterOpener(sc, opener)
	} else if sc.code[stackPtr] == "while" && parenCloser >= 0 {
		if next := f.nextEffective(parenCloser+1, len(f.toks)); next >= 0 && sc.code[next] == ";" {
			if phpcsWhitespaceFound(f, parenCloser+1) != 0 {
				f.replaceToken(parenCloser+1, "")
			}
		}
	}

	var closer int
	switch sc.code[stackPtr] {
	case "while":
		if sc.closer[stackPtr] >= 0 {
			return
		}
		closer = f.prevEffective(stackPtr - 1)
		if closer < 0 || sc.code[closer] != "}" || sc.cond[closer] < 0 || sc.code[sc.cond[closer]] != "do" {
			return
		}
	case "else", "elseif", "catch", "finally":
		if isAlternative {
			return
		}
		closer = f.prevEffective(stackPtr - 1)
		if closer < 0 || sc.code[closer] != "}" {
			return
		}
	default:
		return
	}

	found = 1
	switch {
	case f.toks[closer+1].kind != token.Whitespace:
		found = 0
	case f.toks[closer].line != f.toks[stackPtr].line:
		found = phpcsNewline
	case f.toks[closer+1].content != " ":
		found = f.toks[closer+1].length
	}
	if found == 1 {
		return
	}
	for i := closer + 1; i < stackPtr; i++ {
		if f.isPhpcsComment(i) {
			return
		}
	}
	if found == 0 {
		f.addContent(closer, " ")
	} else {
		f.replaceToken(closer+1, " ")
	}
}

func (r SquizControlSignature) afterCloseParenthesis(f *slvFile, closer, opener int, isAlternative bool) {
	expected := 1
	if isAlternative {
		expected = r.RequiredSpacesBeforeColon
	}
	var b strings.Builder
	for i := closer + 1; i < opener; i++ {
		b.WriteString(f.toks[i].content)
	}
	content := b.String()
	blank := strings.TrimSpace(content) == ""
	found := -2
	if blank {
		found = len(content)
		if strings.Contains(content, f.eol) {
			found = phpcsNewline
		}
	}
	if found == expected {
		return
	}
	padding := strings.Repeat(" ", expected)
	if closer == opener-1 {
		f.addContent(closer, padding)
		return
	}
	f.beginChangeset()
	if blank {
		f.addContent(closer, padding)
		if found != 0 {
			for i := closer + 1; i < opener; i++ {
				f.replaceToken(i, "")
			}
		}
	} else {
		f.addContent(closer, padding+f.toks[opener].content)
		f.replaceToken(opener, "")
		if f.toks[opener].line != f.toks[closer].line {
			if next := f.nextNonWhitespace(opener+1, len(f.toks)); next >= 0 && f.toks[next].line != f.toks[opener].line {
				for i := opener + 1; i < next; i++ {
					f.replaceToken(i, "")
				}
			}
		}
	}
	f.endChangeset()
}

func squizNewlineAfterOpener(sc *phpcsScope, opener int) {
	f := sc.f
	next := opener + 1
	for ; next < len(f.toks); next++ {
		t := f.toks[next]
		if t.kind == token.Whitespace || t.kind == token.InlineHTML && strings.TrimSpace(t.content) == "" {
			continue
		}
		if t.line == f.toks[opener].line && (f.isEmpty(next) || t.kind == token.CloseTag) {
			continue
		}
		break
	}
	if next >= len(f.toks) {
		next = len(f.toks) - 1
	}
	if f.toks[next].line != f.toks[opener].line {
		return
	}
	f.beginChangeset()
	for i := opener + 1; i < next; i++ {
		if strings.TrimSpace(f.toks[i].content) != "" {
			break
		}
		f.replaceToken(i, "")
	}
	f.addContent(opener, f.eol)
	f.endChangeset()
}

// phpcsPrevNonWS is File::findPrevious(T_WHITESPACE, start, end, true), stopping
// above end.
func phpcsPrevNonWS(f *slvFile, start, end int) int {
	for i := start; i > end; i-- {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return 0
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/SwitchDeclarationSniff.php
//
// SquizSwitchDeclaration lays out a switch: lowercase CASE and DEFAULT on
// their own lines Indent spaces in, no space before the colon, the breaking
// statement aligned with the CASE and followed by one blank line.
type SquizSwitchDeclaration struct {
	Indent int
}

func (SquizSwitchDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\SwitchDeclarationSniff`
}

func (SquizSwitchDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/SwitchDeclarationSniff.php"
}

func (r SquizSwitchDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r SquizSwitchDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if sc.code[i] == "switch" {
				r.process(sc, i)
			}
		}
	})
}

var squizCaseBreakers = phpcsSet("break", "return", "continue", "throw", "exit", "goto")

// phpcsLineBreakAfter starts a new line at padding after prev, replacing the
// whitespace after it or appending to it.
func phpcsLineBreakAfter(f *slvFile, prev int, padding string) {
	if f.toks[prev+1].kind == token.Whitespace {
		f.replaceToken(prev+1, f.eol+padding)
	} else {
		f.addContent(prev, f.eol+padding)
	}
}

// phpcsAlign indents tok, the first on its line, to column-1 spaces.
func phpcsAlign(f *slvFile, tok int, padding string) {
	if f.toks[tok].column == 1 || f.toks[tok-1].kind != token.Whitespace {
		f.addContentBefore(tok, padding)
	} else {
		f.replaceToken(tok-1, padding)
	}
}

func (r SquizSwitchDeclaration) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	if sc.opener[stackPtr] < 0 || sc.closer[stackPtr] < 0 {
		return
	}
	switchCloser := sc.closer[stackPtr]
	caseAlignment := f.toks[stackPtr].column + r.Indent
	padding := strings.Repeat(" ", caseAlignment-1)
	for nextCase := phpcsNextCase(sc, stackPtr+1, switchCloser); nextCase >= 0; nextCase = phpcsNextCase(sc, nextCase+1, switchCloser) {
		isCase := sc.code[nextCase] == "case"
		if content := f.toks[nextCase].content; content != strings.ToLower(content) {
			f.replaceToken(nextCase, strings.ToLower(content))
		}
		if isCase && (f.toks[nextCase+1].kind != token.Whitespace || f.toks[nextCase+1].content != " ") {
			if f.toks[nextCase+1].kind != token.Whitespace {
				f.addContent(nextCase, " ")
			} else {
				f.replaceToken(nextCase+1, " ")
			}
		}

		beforeCase := phpcsPrevNonWS(f, nextCase-1, -1)
		if f.toks[beforeCase].line == f.toks[nextCase].line {
			phpcsLineBreakAfter(f, beforeCase, padding)
		} else if f.toks[nextCase].column != caseAlignment {
			phpcsAlign(f, nextCase, padding)
		}

		opener := sc.opener[nextCase]
		if opener < 0 {
			continue
		}
		if sc.code[opener] == "close_tag" {
			f.addContent(phpcsPrevNonWS(f, opener-1, -1), ":")
		}
		if f.toks[opener-1].kind == token.Whitespace {
			f.replaceToken(opener-1, "")
		}

		nextBreak := sc.closer[nextCase]
		if nextBreak < 0 || !squizCaseBreakers[sc.code[nextBreak]] {
			continue
		}
		if sc.cond[nextBreak] == nextCase {
			r.breakLayout(sc, stackPtr, nextCase, opener, nextBreak, caseAlignment, isCase)
		}
	}
}

func (r SquizSwitchDeclaration) breakLayout(sc *phpcsScope, stackPtr, nextCase, opener, nextBreak, caseAlignment int, isCase bool) {
	f := sc.f
	padding := strings.Repeat(" ", caseAlignment-1)
	beforeBreak := phpcsPrevNonWS(f, nextBreak-1, -1)
	if f.toks[beforeBreak].line == f.toks[nextBreak].line {
		phpcsLineBreakAfter(f, beforeBreak, padding)
	} else if f.toks[nextBreak].column != caseAlignment {
		phpcsAlign(f, nextBreak, padding)
	}

	switchCloser := sc.closer[stackPtr]
	nextLine := f.toks[switchCloser].line
	semicolon := sc.endOfStatement(nextBreak)
	nextRelevant := semicolon + 1
	for ; nextRelevant < switchCloser; nextRelevant++ {
		if !f.isEmpty(nextRelevant) {
			nextLine = f.toks[nextRelevant].line
			break
		}
		if f.toks[nextRelevant].kind != token.Whitespace && f.toks[nextBreak].line != f.toks[nextRelevant].line {
			nextLine = f.toks[nextRelevant].line
			break
		}
	}

	if isCase && nextLine != f.toks[semicolon].line+2 && nextRelevant != switchCloser {
		f.beginChangeset()
		if nextLine == f.toks[semicolon].line {
			replacement := f.eol + f.eol + padding
			if f.toks[nextRelevant-1].kind == token.Whitespace {
				f.replaceToken(nextRelevant-1, replacement)
			} else {
				f.addContentBefore(nextRelevant, replacement)
			}
		} else {
			for i := semicolon + 1; i < nextRelevant; i++ {
				if f.toks[semicolon].line == f.toks[i].line {
					if f.toks[i].line != f.toks[i+1].line {
						f.addContent(i, f.eol)
					}
					continue
				}
				if f.toks[i].line == f.toks[nextRelevant].line {
					break
				}
				f.replaceToken(i, "")
			}
		}
		f.endChangeset()
	}

	caseLine := f.toks[nextCase].line
	nextLine = f.toks[nextBreak].line
	i := opener + 1
	for ; i < nextBreak; i++ {
		if !f.isEmpty(i) {
			nextLine = f.toks[i].line
			break
		}
		if f.toks[i].kind != token.Whitespace && f.toks[opener].line != f.toks[i].line {
			nextLine = f.toks[i].line
			break
		}
	}
	if nextLine == caseLine {
		pad := f.eol + strings.Repeat(" ", caseAlignment-1+r.Indent)
		if f.toks[i-1].kind == token.Whitespace {
			f.replaceToken(i-1, pad)
		} else {
			f.addContentBefore(i, pad)
		}
	}
}
