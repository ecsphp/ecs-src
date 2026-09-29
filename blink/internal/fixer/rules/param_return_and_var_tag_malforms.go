package rules

import "blink/internal/tokens"

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/ParamReturnAndVarTagMalformsFixer.php
//
// ParamReturnAndVarTagMalforms is deprecated upstream and a no-op: its
// isCandidate returns false and fix does nothing (it was split into dedicated
// single-task rules). It is kept as a registered no-op so the default set
// matches ECS.
type ParamReturnAndVarTagMalforms struct{}

func (ParamReturnAndVarTagMalforms) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\ParamReturnAndVarTagMalformsFixer`
}

func (ParamReturnAndVarTagMalforms) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/ParamReturnAndVarTagMalformsFixer.php"
}

func (ParamReturnAndVarTagMalforms) Fix(*tokens.Stream) bool {
	return false
}
