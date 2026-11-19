package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/NamingConventions/ValidVariableNameSniff.php
//
// SquizValidVariableName reports variables and properties not in camel caps,
// and properties whose leading underscore does not match their visibility. It
// fixes nothing.
type SquizValidVariableName struct{}

const squizValidVariableNameName = `PHP_CodeSniffer\Standards\Squiz\Sniffs\NamingConventions\ValidVariableNameSniff`

func (SquizValidVariableName) Name() string { return squizValidVariableNameName }

func (SquizValidVariableName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/NamingConventions/ValidVariableNameSniff.php"
}

func (SquizValidVariableName) Fix(*tokens.Stream) bool { return false }

func (SquizValidVariableName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	const n = squizValidVariableNameName
	return abstractVariableErrors(s, true, true, func(f *slvFile, kind abstractVariableKind, i int) []fixer.CodingStandardError {
		var errs []fixer.CodingStandardError
		add := func(at int, code, format string, args ...any) {
			errs = append(errs, abstractVariableError(f, at, n+"."+code, fmt.Sprintf(format, args...)))
		}
		switch kind {
		case abstractVariableString:
			for _, name := range abstractVariableStringNames(f.toks[i].orig, true) {
				if !abstractVariableReserved[name] && !abstractVariableIsCamelCaps(name) {
					add(i, "StringNotCamelCaps", `Variable "%s" is not in valid camel caps format`, name)
				}
			}
		case abstractVariableMember:
			props, ok := abstractVariableMemberProps(f, i)
			if !ok {
				return nil
			}
			name := strings.TrimLeft(f.toks[i].orig, "$")
			if props.scope != "private" {
				if strings.HasPrefix(name, "_") {
					add(i, "PublicHasUnderscore", `%s member variable "%s" must not contain a leading underscore`, abstractVariableUcfirst(props.scope), name)
				}
			} else if !strings.HasPrefix(name, "_") {
				add(i, "PrivateNoUnderscore", `Private member variable "%s" must contain a leading underscore`, name)
			}
			if !abstractVariableIsCamelCaps(strings.TrimLeft(name, "_")) {
				add(i, "MemberNotCamelCaps", `Member variable "%s" is not in valid camel caps format`, name)
			}
		case abstractVariablePlain:
			name := strings.TrimLeft(f.toks[i].orig, "$")
			if abstractVariableReserved[name] {
				return nil
			}
			if op := abstractVariableNext(f, i+1); f.is(op, "->") || f.is(op, "?->") {
				if v := abstractVariableNext(f, op+1); v >= 0 && f.toks[v].kind == token.Ident {
					if b := abstractVariableNext(f, v+1); !f.is(b, "(") {
						prop := f.toks[v].orig
						if !abstractVariableIsCamelCaps(strings.TrimPrefix(prop, "_")) {
							add(v, "MemberNotCamelCaps", `Member variable "%s" is not in valid camel caps format`, prop)
						}
					}
				}
			}
			if f.is(abstractVariablePrev(f, i-1), "::") {
				if !abstractVariableIsCamelCaps(strings.TrimPrefix(name, "_")) {
					add(i, "MemberNotCamelCaps", `Member variable "%s" is not in valid camel caps format`, f.toks[i].orig)
				}
				return errs
			}
			check := name
			if strings.HasPrefix(name, "_") && abstractVariableHasOOCondition(f, i) {
				check = name[1:]
			}
			if !abstractVariableIsCamelCaps(check) {
				add(i, "NotCamelCaps", `Variable "%s" is not in valid camel caps format`, name)
			}
		}
		return errs
	})
}
