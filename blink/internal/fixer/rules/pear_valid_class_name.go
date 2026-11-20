package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidClassNameSniff.php
//
// PEARValidClassName reports class, interface, trait and enum names that are
// not Capitalised_Words_With_Underscores. It fixes nothing.
type PEARValidClassName struct{}

func (PEARValidClassName) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\NamingConventions\ValidClassNameSniff`
}

func (PEARValidClassName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidClassNameSniff.php"
}

func (PEARValidClassName) Fix(*tokens.Stream) bool { return false }

var pearValidClassNameCapital = regexp.MustCompile(`^[A-Z]`)

func (r PEARValidClassName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	m := newSiModel(newSlvFile(lexer.Lex(s.Render())))
	var errs []fixer.CodingStandardError
	for i := range m.f.toks {
		if !pearOODeclaration(m, i) {
			continue
		}
		name := ""
		for j := i; j < len(m.code); j++ {
			if m.code[j] == "string" {
				name = strings.TrimSpace(m.f.toks[j].orig)
				break
			}
		}
		kind := strings.ToLower(m.f.toks[i].orig)
		kind = strings.ToUpper(kind[:1]) + kind[1:]
		add := func(code, msg string) {
			errs = append(errs, fixer.CodingStandardError{Line: m.f.toks[i].line, Message: msg, Checker: r.Name() + "." + code})
		}
		if !pearValidClassNameCapital.MatchString(name) {
			add("StartWithCapital", kind+" name must begin with a capital letter")
		}
		valid := true
		for _, bit := range strings.Split(name, "_")[1:] {
			if bit == "" || bit[:1] != strings.ToUpper(bit[:1]) {
				valid = false
				break
			}
		}
		if valid {
			continue
		}
		bits := strings.Split(strings.Trim(name, "_"), "_")
		if bits[0] == "" {
			add("Invalid", kind+" name is not valid")
			continue
		}
		add("Invalid", kind+" name is not valid; consider "+pearSuggestName(strings.Trim(name, "_"))+" instead")
	}
	return errs
}

// pearOODeclaration reports whether token i is PHP_CodeSniffer's T_CLASS,
// T_INTERFACE, T_TRAIT or T_ENUM: a declaration keyword, not "::class" or an
// anonymous class.
func pearOODeclaration(m *siModel, i int) bool {
	switch m.code[i] {
	case "class", "interface", "trait", "enum":
	default:
		return false
	}
	p := m.f.prevEffective(i - 1)
	if m.f.is(p, "::") || m.f.is(p, "->") || m.f.is(p, "?->") {
		return false
	}
	if m.code[i] == "enum" {
		n := m.f.nextEffective(i+1, len(m.f.toks))
		return n >= 0 && m.code[n] == "string"
	}
	return true
}
