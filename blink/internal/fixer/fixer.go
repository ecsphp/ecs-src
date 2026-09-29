// Package fixer defines the Fixer contract. A fixer inspects a token stream and
// mutates it in place, mirroring ECS/PHP-CS-Fixer fixers.
package fixer

import (
	"strings"

	"blink/internal/tokens"
)

// SourceBase is the GitHub location of the original PHP-CS-Fixer fixers.
const SourceBase = "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/"

// SymplifySourceBase is the GitHub location of symplify's coding-standard fixers,
// for the few rules ported from there rather than PHP-CS-Fixer.
const SymplifySourceBase = "https://github.com/symplify/coding-standard/blob/main/src/Fixer/"

// EcsSrcSourceBase is the ecsphp/ecs-src monorepo location, for Symplify fixers
// that live there but have not been split out to symplify/coding-standard.
const EcsSrcSourceBase = "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/"

// ecsSrcOnlyFixers are Symplify fixers sourced from ecsphp/ecs-src rather than
// the split symplify/coding-standard repository.
var ecsSrcOnlyFixers = map[string]bool{
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLineRequiredParamFixer`:              true,
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer`:      true,
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer`:      true,
	`Symplify\CodingStandard\Fixer\Spacing\NoBlankLineBetweenImportsFixer`:                true,
	`Symplify\CodingStandard\Fixer\Spacing\SpaceAfterCommaHereNowDocFixer`:                true,
	`Symplify\CodingStandard\Fixer\ArrayNotation\ArrayOpenerAndCloserNewlineFixer`:        true,
	`Symplify\CodingStandard\Fixer\Commenting\FixTagTypoFixer`:                            true,
	`Symplify\CodingStandard\Fixer\Commenting\TypeToVarTagFixer`:                          true,
	`Symplify\CodingStandard\Fixer\Commenting\MergeDocBlockStartFixer`:                    true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveDeadParamFixer`:                       true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveDeadVarThisFixer`:                     true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveParamNameReferenceFixer`:              true,
	`Symplify\CodingStandard\Fixer\Commenting\SwitchedTypeAndNameFixer`:                   true,
	`Symplify\CodingStandard\Fixer\Annotation\RemovePHPStormAnnotationFixer`:              true,
	`Symplify\CodingStandard\Fixer\Annotation\RemovePropertyVariableNameDescriptionFixer`: true,
	`Symplify\CodingStandard\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer`:  true,
	`Symplify\CodingStandard\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer`:      true,
}

type Fixer interface {
	// Name is the checker identifier (the PHP-CS-Fixer FQCN) shown in reports.
	Name() string
	// SourceURL links to the original PHP-CS-Fixer rule on GitHub. Required for
	// every fixer and verified by the source checker.
	SourceURL() string
	// Fix mutates the stream and reports whether it changed anything.
	Fix(s *tokens.Stream) bool
}

// SourceURLFor derives the canonical PHP-CS-Fixer source URL from a fixer Name
// (an FQCN like `PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`).
func SourceURLFor(name string) string {
	parts := strings.Split(name, `\`)
	if len(parts) < 2 {
		return ""
	}
	category := parts[len(parts)-2]
	class := parts[len(parts)-1]
	if ecsSrcOnlyFixers[name] {
		return EcsSrcSourceBase + category + "/" + class + ".php"
	}
	if strings.HasPrefix(name, `Symplify\CodingStandard\`) {
		return SymplifySourceBase + category + "/" + class + ".php"
	}
	return SourceBase + category + "/" + class + ".php"
}
