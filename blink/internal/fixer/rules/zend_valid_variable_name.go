package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Zend/Sniffs/NamingConventions/ValidVariableNameSniff.php
//
// ZendValidVariableName reports variables and properties not in camel caps,
// and properties whose leading underscore does not match their visibility. Its
// "contains numbers" warnings never reach ECS's report. It fixes nothing.
type ZendValidVariableName struct{}

const zendValidVariableNameName = `PHP_CodeSniffer\Standards\Zend\Sniffs\NamingConventions\ValidVariableNameSniff`

func (ZendValidVariableName) Name() string { return zendValidVariableNameName }

func (ZendValidVariableName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Zend/Sniffs/NamingConventions/ValidVariableNameSniff.php"
}

func (ZendValidVariableName) Fix(*tokens.Stream) bool { return false }

func (ZendValidVariableName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	const n = zendValidVariableNameName
	return abstractVariableErrors(s, true, true, func(f *slvFile, kind abstractVariableKind, i int) []fixer.CodingStandardError {
		var errs []fixer.CodingStandardError
		add := func(at int, code, format string, args ...any) {
			errs = append(errs, abstractVariableError(f, at, n+"."+code, fmt.Sprintf(format, args...)))
		}
		switch kind {
		case abstractVariableString:
			for _, name := range abstractVariableStringNames(f.toks[i].orig, false) {
				if !abstractVariableReserved[name] && !abstractVariableIsCamelCaps(name) {
					add(i, "StringVarNotCamelCaps", `Variable "%s" is not in valid camel caps format`, name)
				}
			}
		case abstractVariableMember:
			props, ok := abstractVariableMemberProps(f, i)
			if !ok {
				return nil
			}
			name := strings.TrimLeft(f.toks[i].orig, "$")
			if props.scope == "public" {
				if strings.HasPrefix(name, "_") {
					add(i, "PublicHasUnderscore", `Public member variable "%s" must not contain a leading underscore`, name)
				}
			} else if !strings.HasPrefix(name, "_") {
				add(i, "PrivateNoUnderscore", `%s member variable "%s" must contain a leading underscore`, abstractVariableUcfirst(props.scope), name)
			}
			if bare := strings.TrimLeft(name, "_"); !abstractVariableIsCamelCaps(bare) {
				add(i, "MemberVarNotCamelCaps", `Member variable "%s" is not in valid camel caps format`, bare)
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
							add(v, "NotCamelCaps", `Variable "%s" is not in valid camel caps format`, prop)
						}
					}
				}
			}
			check := name
			if strings.HasPrefix(name, "_") && (f.is(abstractVariablePrev(f, i-1), "::") || abstractVariableHasOOCondition(f, i)) {
				check = name[1:]
			}
			if !abstractVariableIsCamelCaps(check) {
				add(i, "NotCamelCaps", `Variable "%s" is not in valid camel caps format`, name)
			}
		}
		return errs
	})
}
