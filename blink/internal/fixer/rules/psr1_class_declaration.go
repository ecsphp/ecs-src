package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Classes/ClassDeclarationSniff.php
//
// PSR1ClassDeclaration reports a second class, interface, trait or enum in a
// file and one declared outside a namespace. It fixes nothing.
type PSR1ClassDeclaration struct{}

func (PSR1ClassDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR1\Sniffs\Classes\ClassDeclarationSniff`
}

func (PSR1ClassDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Classes/ClassDeclarationSniff.php"
}

func (PSR1ClassDeclaration) Fix(*tokens.Stream) bool { return false }

func (r PSR1ClassDeclaration) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	sc := newPhpcsScope(f)
	// T_CLASS, T_INTERFACE, T_TRAIT, T_ENUM; an anonymous class is T_ANON_CLASS
	isOO := func(i int) bool { return phpcsIsOOKeyword(f, i) && !f.isAnonClass(i) }
	firstNamespace, firstOO := -1, -1
	for i := range f.toks {
		if firstOO < 0 && isOO(i) {
			firstOO = i
		}
		if firstNamespace < 0 && sc.code[i] == "namespace" {
			firstNamespace = i
		}
	}
	missingNamespace := firstNamespace < 0 || firstOO >= 0 && firstOO < firstNamespace
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !isOO(i) || sc.cond[i] != i || sc.closer[i] < 0 {
			continue
		}
		kind := strings.ToLower(f.toks[i].orig)
		for next := sc.closer[i] + 1; next < len(f.toks); next++ {
			if isOO(next) {
				errs = append(errs, fixer.CodingStandardError{
					Line:    f.toks[next].line,
					Message: "Each " + kind + " must be in a file by itself",
					Checker: r.Name() + ".MultipleClasses",
				})
				break
			}
		}
		if missingNamespace {
			errs = append(errs, fixer.CodingStandardError{
				Line:    f.toks[i].line,
				Message: "Each " + kind + " must be in a namespace of at least one level (a top-level vendor name)",
				Checker: r.Name() + ".MissingNamespace",
			})
		}
	}
	return errs
}
