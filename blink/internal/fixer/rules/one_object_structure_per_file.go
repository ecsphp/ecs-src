package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneObjectStructurePerFileSniff.php
//
// OneObjectStructurePerFile reports every class, interface, trait or enum
// after the first in a file. It fixes nothing.
type OneObjectStructurePerFile struct{}

func (OneObjectStructurePerFile) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\OneObjectStructurePerFileSniff`
}

func (OneObjectStructurePerFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneObjectStructurePerFileSniff.php"
}

func (OneObjectStructurePerFile) Fix(*tokens.Stream) bool { return false }

func (o OneObjectStructurePerFile) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return oneObjectPerFileErrors(s, []string{"class", "interface", "trait", "enum"}, "Only one object structure is allowed in a file", o.Name())
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneClassPerFileSniff.php
//
// OneClassPerFile reports every class after the first in a file.
type OneClassPerFile struct{}

func (OneClassPerFile) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\OneClassPerFileSniff`
}

func (OneClassPerFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneClassPerFileSniff.php"
}

func (OneClassPerFile) Fix(*tokens.Stream) bool { return false }

func (o OneClassPerFile) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return oneObjectPerFileErrors(s, []string{"class"}, "Only one class is allowed in a file", o.Name())
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneInterfacePerFileSniff.php
//
// OneInterfacePerFile reports every interface after the first in a file.
type OneInterfacePerFile struct{}

func (OneInterfacePerFile) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\OneInterfacePerFileSniff`
}

func (OneInterfacePerFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneInterfacePerFileSniff.php"
}

func (OneInterfacePerFile) Fix(*tokens.Stream) bool { return false }

func (o OneInterfacePerFile) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return oneObjectPerFileErrors(s, []string{"interface"}, "Only one interface is allowed in a file", o.Name())
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneTraitPerFileSniff.php
//
// OneTraitPerFile reports every trait after the first in a file.
type OneTraitPerFile struct{}

func (OneTraitPerFile) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\OneTraitPerFileSniff`
}

func (OneTraitPerFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/OneTraitPerFileSniff.php"
}

func (OneTraitPerFile) Fix(*tokens.Stream) bool { return false }

func (o OneTraitPerFile) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return oneObjectPerFileErrors(s, []string{"trait"}, "Only one trait is allowed in a file", o.Name())
}

// oneObjectPerFileErrors reports, for each declaration of the given kinds, the
// next one after its body, as the One*PerFile sniffs do.
func oneObjectPerFileErrors(s *tokens.Stream, kinds []string, msg, sniff string) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	is := func(i int) bool {
		for _, k := range kinds {
			if f.isKeyword(i, k) {
				// PHP_CodeSniffer makes "new class" a T_ANON_CLASS
				return k != "class" || !f.isAnonClass(i)
			}
		}
		return false
	}
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !is(i) {
			continue
		}
		start := i + 1
		for j := i + 1; j < len(f.toks) && !f.is(j, ";"); j++ {
			if f.is(j, "{") {
				if f.toks[j].closer > j {
					start = f.toks[j].closer + 1
				}
				break
			}
		}
		for j := start; j < len(f.toks); j++ {
			if is(j) {
				errs = append(errs, fixer.CodingStandardError{Line: f.toks[j].line, Message: msg, Checker: sniff + ".MultipleFound"})
				break
			}
		}
	}
	return errs
}
