package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/InterfaceNameSuffixSniff.php
//
// InterfaceNameSuffix reports an interface whose name does not end in
// "Interface". It fixes nothing.
type InterfaceNameSuffix struct{}

func (InterfaceNameSuffix) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\NamingConventions\InterfaceNameSuffixSniff`
}

func (InterfaceNameSuffix) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/InterfaceNameSuffixSniff.php"
}

func (InterfaceNameSuffix) Fix(*tokens.Stream) bool { return false }

func (r InterfaceNameSuffix) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return nameSuffixErrors(s, "interface", "Interface", r.Name())
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/TraitNameSuffixSniff.php
//
// TraitNameSuffix reports a trait whose name does not end in "Trait". It fixes
// nothing.
type TraitNameSuffix struct{}

func (TraitNameSuffix) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\NamingConventions\TraitNameSuffixSniff`
}

func (TraitNameSuffix) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/TraitNameSuffixSniff.php"
}

func (TraitNameSuffix) Fix(*tokens.Stream) bool { return false }

func (r TraitNameSuffix) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return nameSuffixErrors(s, "trait", "Trait", r.Name())
}

// nameSuffixErrors reports every keyword declaration whose name, from
// File::getDeclarationName, does not end in suffix, case-insensitively.
func nameSuffixErrors(s *tokens.Stream, keyword, suffix, sniff string) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !f.isKeyword(i, keyword) || !phpcsIsOOKeyword(f, i) {
			continue
		}
		name := f.nextEffective(i+1, len(f.toks))
		if name < 0 || f.toks[name].kind != token.Ident {
			continue
		}
		n := f.toks[name].orig
		if strings.EqualFold(n[max(len(n)-len(suffix), 0):], suffix) {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    f.toks[i].line,
			Message: suffix + ` names must be suffixed with "` + suffix + `"; found "` + n + `"`,
			Checker: sniff + ".Missing",
		})
	}
	return errs
}
