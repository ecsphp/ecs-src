package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/SwitchDeclarationSniff.php
//
// PSR2SwitchDeclaration formats the CASE and DEFAULT statements of a switch:
// lowercase keyword, one space after "case", a colon right after the value,
// the body on the next line and the terminating statement on its own line,
// Indent spaces in from the CASE.
type PSR2SwitchDeclaration struct {
	Indent int
}

func (PSR2SwitchDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\ControlStructures\SwitchDeclarationSniff`
}

func (PSR2SwitchDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/SwitchDeclarationSniff.php"
}

func (r PSR2SwitchDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r PSR2SwitchDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if sc.code[i] == "switch" {
				r.process(sc, i)
			}
		}
	})
}

func (r PSR2SwitchDeclaration) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	if sc.opener[stackPtr] < 0 || sc.closer[stackPtr] < 0 {
		return
	}
	end := sc.closer[stackPtr]
	caseAlignment := f.toks[stackPtr].column + r.Indent
	for nextCase := phpcsNextCase(sc, stackPtr+1, end); nextCase >= 0; nextCase = phpcsNextCase(sc, nextCase+1, end) {
		isCase := sc.code[nextCase] == "case"
		content := f.toks[nextCase].content
		if lower := strings.ToLower(content); content != lower {
			f.replaceToken(nextCase, lower)
		}
		if isCase && (sc.code[nextCase+1] != "whitespace" || f.toks[nextCase+1].content != " ") {
			if sc.code[nextCase+1] != "whitespace" {
				f.replaceToken(nextCase, f.tokenContent(nextCase)+" ")
			} else {
				f.replaceToken(nextCase+1, " ")
			}
		}
		opener, nextCloser := sc.opener[nextCase], sc.closer[nextCase]
		if opener < 0 || nextCloser < 0 {
			continue
		}
		if sc.code[opener] == ":" {
			r.colonCase(sc, nextCase, opener, nextCloser, caseAlignment)
			continue
		}
		switch sc.code[opener] {
		case ";":
			f.replaceToken(opener, ":")
		case "close_tag":
			prev := opener - 1
			for prev >= 0 && sc.code[prev] == "whitespace" {
				prev--
			}
			f.replaceToken(prev, f.tokenContent(prev)+":")
		}
	}
}

func (r PSR2SwitchDeclaration) colonCase(sc *phpcsScope, nextCase, opener, nextCloser, caseAlignment int) {
	f := sc.f
	if sc.code[opener-1] == "whitespace" {
		f.replaceToken(opener-1, "")
	}
	next := opener + 1
	for ; next < nextCloser; next++ {
		if !f.isEmpty(next) || sc.code[next] == "comment" && f.toks[next].line != f.toks[opener].line {
			break
		}
	}
	if f.toks[next].line != f.toks[opener].line+1 {
		if f.toks[next].line == f.toks[opener].line {
			f.replaceToken(next, f.eol+strings.Repeat(" ", caseAlignment+r.Indent-1)+f.tokenContent(next))
		} else {
			f.beginChangeset()
			for i := opener + 1; i < next; i++ {
				if f.toks[i].line == f.toks[opener].line {
					continue
				}
				if f.toks[i].line == f.toks[next].line {
					break
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
	}
	if sc.cond[nextCloser] != nextCase {
		return
	}
	prev := nextCloser - 1
	for prev > nextCase && sc.code[prev] == "whitespace" {
		prev--
	}
	if f.toks[prev].line == f.toks[nextCloser].line {
		f.replaceToken(prev, f.tokenContent(prev)+f.eol)
		f.replaceToken(nextCloser, strings.TrimSpace(f.toks[nextCloser].content))
		return
	}
	diff := f.toks[nextCase].column + r.Indent - f.toks[nextCloser].column
	switch {
	case diff > 0:
		f.replaceToken(nextCloser, strings.Repeat(" ", diff)+f.tokenContent(nextCloser))
	case diff < 0:
		v := f.tokenContent(nextCloser - 1)
		f.replaceToken(nextCloser-1, v[:max(0, len(v)+diff)])
	}
}

// phpcsNextCase is SwitchDeclarationSniff::findNextCase: the next CASE or
// DEFAULT from start before end, skipping nested switches, or -1.
func phpcsNextCase(sc *phpcsScope, start, end int) int {
	for i := start; i < end; i++ {
		switch sc.code[i] {
		case "switch":
			if sc.closer[i] < 0 {
				return -1
			}
			i = sc.closer[i]
		case "case", "default":
			return i
		}
	}
	return -1
}
