package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidVariableNameSniff.php
//
// PEARValidVariableName reports a private property without a leading
// underscore and a public or protected one with it. It fixes nothing.
type PEARValidVariableName struct{}

const pearValidVariableNameName = `PHP_CodeSniffer\Standards\PEAR\Sniffs\NamingConventions\ValidVariableNameSniff`

func (PEARValidVariableName) Name() string { return pearValidVariableNameName }

func (PEARValidVariableName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidVariableNameSniff.php"
}

func (PEARValidVariableName) Fix(*tokens.Stream) bool { return false }

func (PEARValidVariableName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return abstractVariableErrors(s, false, false, func(f *slvFile, kind abstractVariableKind, i int) []fixer.CodingStandardError {
		if kind != abstractVariableMember {
			return nil
		}
		props, ok := abstractVariableMemberProps(f, i)
		if !ok {
			return nil
		}
		name := strings.TrimLeft(f.toks[i].orig, "$")
		specified := props.scopeSpecified || props.setScope != ""
		switch {
		case props.scope == "private" && name[0] != '_':
			return []fixer.CodingStandardError{abstractVariableError(f, i, pearValidVariableNameName+".PrivateNoUnderscore",
				fmt.Sprintf(`Private member variable "%s" must be prefixed with an underscore`, name))}
		case props.scope != "private" && specified && name[0] == '_':
			return []fixer.CodingStandardError{abstractVariableError(f, i, pearValidVariableNameName+".PublicUnderscore",
				fmt.Sprintf(`%s member variable "%s" must not be prefixed with an underscore`, abstractVariableUcfirst(props.scope), name))}
		}
		return nil
	})
}
