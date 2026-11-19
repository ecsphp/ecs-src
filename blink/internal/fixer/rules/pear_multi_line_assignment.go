package rules

import (
	"strconv"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Formatting/MultiLineAssignmentSniff.php
//
// PEARMultiLineAssignment reports a multi-line assignment whose "=" ends a line,
// or starts a line with the wrong indent. It fixes nothing.
type PEARMultiLineAssignment struct {
	Indent int
}

func (PEARMultiLineAssignment) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Formatting\MultiLineAssignmentSniff`
}

func (PEARMultiLineAssignment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Formatting/MultiLineAssignmentSniff.php"
}

func (r PEARMultiLineAssignment) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (PEARMultiLineAssignment) Fix(*tokens.Stream) bool { return false }

func (r PEARMultiLineAssignment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	toks := f.toks
	ws := func(i int) bool { return toks[i].kind == token.Whitespace }
	var errs []fixer.CodingStandardError
	add := func(i int, code, msg string) {
		errs = append(errs, fixer.CodingStandardError{Line: toks[i].line, Message: msg, Checker: r.Name() + "." + code})
	}
	for i := range toks {
		if !f.is(i, "=") {
			continue
		}
		next := f.nextNot(ws, i+1, len(toks))
		if next < 0 {
			continue
		}
		if toks[next].line != toks[i].line {
			add(i, "EqualSignLine", "Multi-line assignments must have the equal sign on the second line")
			continue
		}
		prev := f.prevNot(ws, i-1, -1)
		if prev < 0 || toks[prev].line == toks[i].line {
			continue
		}
		j := prev - 1
		for ; j >= 0; j-- {
			if toks[j].line != toks[prev].line {
				break
			}
		}
		j++
		assignmentIndent := 0
		if ws(j) {
			assignmentIndent = toks[j].length
		}
		found := 0
		for k := i - 1; k >= 0; k-- {
			if ws(k) {
				found = toks[k].length
				break
			}
		}
		if expected := assignmentIndent + r.Indent; found != expected {
			add(i, "Indent", "Multi-line assignment not indented correctly; expected "+strconv.Itoa(expected)+" spaces but found "+strconv.Itoa(found))
		}
	}
	return errs
}
