package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/UpperCaseConstantNameSniff.php
//
// UpperCaseConstantName reports a "const" or define() constant whose name is
// not upper case. It fixes nothing.
type UpperCaseConstantName struct{}

func (UpperCaseConstantName) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\NamingConventions\UpperCaseConstantNameSniff`
}

func (UpperCaseConstantName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/UpperCaseConstantNameSniff.php"
}

func (UpperCaseConstantName) Fix(*tokens.Stream) bool { return false }

func (r UpperCaseConstantName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	inAttribute := make([]bool, len(f.toks))
	for i, t := range f.toks {
		if t.kind == token.AttributeStart && t.closer > i {
			for j := i + 1; j < t.closer; j++ {
				inAttribute[j] = true
			}
		}
	}
	var errs []fixer.CodingStandardError
	add := func(i int, code, msg string) {
		errs = append(errs, fixer.CodingStandardError{Line: f.toks[i].line, Message: msg, Checker: r.Name() + "." + code})
	}
	for i, t := range f.toks {
		if f.isKeyword(i, "const") {
			assign := -1
			for j := i + 1; j < len(f.toks); j++ {
				if f.is(j, "=") || f.is(j, ";") {
					assign = j
					break
				}
			}
			if assign < 0 || !f.is(assign, "=") {
				continue
			}
			constant := f.prevEffective(assign - 1)
			if constant <= i {
				continue
			}
			name := f.toks[constant].content
			if upper := upperCaseConstantNameUpper(name); upper != name {
				add(constant, "ClassConstantNotUpperCase", "Class constants must be uppercase; expected "+upper+" but found "+name)
			}
			continue
		}
		// T_STRING and T_NAME_FULLY_QUALIFIED
		if t.kind != token.Ident || strings.Contains(strings.TrimPrefix(t.orig, `\`), `\`) ||
			!strings.EqualFold(strings.TrimLeft(t.orig, `\`), "define") {
			continue
		}
		if p := f.prevEffective(i - 1); f.is(p, "->") || f.is(p, "::") || f.is(p, "?->") || f.isKeyword(p, "new") {
			continue
		}
		if inAttribute[i] {
			continue
		}
		open := f.nextEffective(i+1, len(f.toks))
		if !f.is(open, "(") {
			continue
		}
		c := f.nextEffective(open+1, len(f.toks))
		if c < 0 || !upperCaseConstantNameIsLiteral(f, c) {
			continue
		}
		name, prefix := f.toks[c].content, ""
		if at := strings.LastIndexByte(name, '\\'); at >= 0 {
			prefix, name = name[:at+1], name[at+1:]
		}
		if upper := upperCaseConstantNameUpper(name); upper != name {
			add(c, "ConstantNotUpperCase", "Constants must be uppercase; expected "+prefix+upper+" but found "+prefix+name)
		}
	}
	return errs
}

// upperCaseConstantNameIsLiteral reports whether i is a T_CONSTANT_ENCAPSED_STRING:
// a quoted string without interpolation.
func upperCaseConstantNameIsLiteral(f *slvFile, i int) bool {
	t := f.toks[i]
	return t.kind == token.String && !t.heredoc && !t.interp
}

// upperCaseConstantNameUpper is PHP 8's strtoupper: ASCII letters only.
func upperCaseConstantNameUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}
