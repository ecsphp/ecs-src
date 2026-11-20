package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// squizReportError is a CodingStandardError of the sniff named name.
func squizReportError(name, code string, line int, msg string) fixer.CodingStandardError {
	return fixer.CodingStandardError{Line: line, Message: msg, Checker: name + "." + code}
}

// squizReportOO returns the lowercased keyword when PHP_CodeSniffer tokenizes
// i as T_CLASS, T_INTERFACE, T_TRAIT or T_ENUM, else "".
func squizReportOO(f *slvFile, i int) string {
	if f.toks[i].kind != token.Keyword {
		return ""
	}
	kw := strings.ToLower(f.toks[i].orig)
	switch kw {
	case "class", "interface", "trait", "enum":
	default:
		return ""
	}
	if p := f.prevEffective(i - 1); f.is(p, "::") || f.is(p, "->") || f.is(p, "?->") {
		return ""
	}
	if kw == "class" && f.isAnonClass(i) {
		return ""
	}
	return kw
}

// squizReportIsName tells Tokens::NAME_TOKENS: a T_STRING or a namespaced name.
func squizReportIsName(f *slvFile, i int) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "self", "parent", "true", "false", "null", "exit", "die":
		return false
	}
	return true
}

// squizReportIsString tells T_STRING: a name token without a namespace separator.
func squizReportIsString(f *slvFile, i int) bool {
	return squizReportIsName(f, i) && !strings.Contains(f.toks[i].orig, `\`)
}

// squizReportDeclarationName is File::getDeclarationName: the first T_STRING
// from i up to the declaration's "(" or scope opener.
func squizReportDeclarationName(f *slvFile, sc *phpcsScope, i int) string {
	stop := len(f.toks)
	if pc := sc.pCloser[i]; pc >= 0 {
		stop = f.toks[pc].closer
	} else if sc.opener[i] >= 0 && sc.cond[i] == i {
		stop = sc.opener[i]
	}
	for j := i; j < stop; j++ {
		if squizReportIsString(f, j) {
			return f.toks[j].orig
		}
	}
	return ""
}
