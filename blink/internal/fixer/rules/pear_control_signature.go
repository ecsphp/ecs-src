package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/ControlStructures/ControlSignatureSniff.php
//
// PEARControlSignature reports control structures whose signature does not
// match the PEAR layout ("if (...) {EOL" and the like), comments ignored. It
// fixes nothing.
type PEARControlSignature struct{}

func (PEARControlSignature) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\ControlStructures\ControlSignatureSniff`
}

func (PEARControlSignature) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/ControlStructures/ControlSignatureSniff.php"
}

func (PEARControlSignature) Fix(*tokens.Stream) bool { return false }

var pearControlSignaturePatterns = func() []abstractPattern {
	var ps []abstractPattern
	for _, c := range []string{
		"do {EOL...} while (...);EOL", "while (...) {EOL", "for (...) {EOL", "if (...) {EOL", "foreach (...) {EOL",
		"} else if (...) {EOL", "} elseif (...) {EOL", "} else {EOL", "do {EOL", "match (...) {EOL",
	} {
		ps = append(ps, abstractPatternParse(c))
	}
	return ps
}()

func (r PEARControlSignature) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return abstractPatternErrors(s, r.Name(), pearControlSignaturePatterns, true, func(m *siModel, i int) bool {
		switch m.code[i] {
		case "do", "while", "for", "if", "foreach", "else", "elseif", "match":
			return true
		}
		return false
	})
}
