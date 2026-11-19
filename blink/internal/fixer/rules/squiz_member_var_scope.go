package rules

import (
	"fmt"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/MemberVarScopeSniff.php
//
// MemberVarScope reports a property declared without a (read) visibility. It
// fixes nothing.
type MemberVarScope struct{}

const memberVarScopeName = `PHP_CodeSniffer\Standards\Squiz\Sniffs\Scope\MemberVarScopeSniff`

func (MemberVarScope) Name() string { return memberVarScopeName }

func (MemberVarScope) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/MemberVarScopeSniff.php"
}

func (MemberVarScope) Fix(*tokens.Stream) bool { return false }

func (MemberVarScope) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return abstractVariableErrors(s, false, false, func(f *slvFile, kind abstractVariableKind, i int) []fixer.CodingStandardError {
		if kind != abstractVariableMember {
			return nil
		}
		props, ok := abstractVariableMemberProps(f, i)
		if !ok || props.scopeSpecified {
			return nil
		}
		if props.setScope == "" {
			return []fixer.CodingStandardError{abstractVariableError(f, i, memberVarScopeName+".Missing",
				fmt.Sprintf(`Scope modifier not specified for member variable "%s"`, f.toks[i].orig))}
		}
		return []fixer.CodingStandardError{abstractVariableError(f, i, memberVarScopeName+".AsymReadMissing",
			fmt.Sprintf(`Read scope modifier not specified for member variable "%s"`, f.toks[i].orig))}
	})
}
