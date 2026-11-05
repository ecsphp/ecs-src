// Package fixer defines the Fixer contract. A fixer inspects a token stream and
// mutates it in place, mirroring ECS/PHP-CS-Fixer fixers.
package fixer

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// SourceBase is the GitHub location of the original PHP-CS-Fixer fixers.
const SourceBase = "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/"

// SymplifySourceBase is the GitHub location of symplify's coding-standard fixers,
// for the few rules ported from there rather than PHP-CS-Fixer.
const SymplifySourceBase = "https://github.com/symplify/coding-standard/blob/main/src/Fixer/"

// EcsSrcSourceBase is the ecsphp/ecs-src monorepo location, for Symplify fixers
// that live there but have not been split out to symplify/coding-standard.
const EcsSrcSourceBase = "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/"

// PhpCodeSnifferSourceBase is the PHP_CodeSniffer location, for the few rules
// ported from a PHP_CodeSniffer sniff rather than a PHP-CS-Fixer fixer.
const PhpCodeSnifferSourceBase = "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/"

// ecsSrcOnlyFixers are Symplify fixers sourced from ecsphp/ecs-src rather than
// the split symplify/coding-standard repository.
var ecsSrcOnlyFixers = map[string]bool{
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLineRequiredParamFixer`:              true,
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer`:      true,
	`Symplify\CodingStandard\Fixer\Spacing\MethodChainingNewlineFixer`:                    true,
	`Symplify\CodingStandard\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer`:      true,
	`Symplify\CodingStandard\Fixer\Spacing\NoBlankLineBetweenImportsFixer`:                true,
	`Symplify\CodingStandard\Fixer\Spacing\SpaceAfterCommaHereNowDocFixer`:                true,
	`Symplify\CodingStandard\Fixer\ArrayNotation\ArrayOpenerAndCloserNewlineFixer`:        true,
	`Symplify\CodingStandard\Fixer\LineLength\LineLengthFixer`:                            true,
	`Symplify\CodingStandard\Fixer\Commenting\FixTagTypoFixer`:                            true,
	`Symplify\CodingStandard\Fixer\Commenting\TypeToVarTagFixer`:                          true,
	`Symplify\CodingStandard\Fixer\Commenting\MergeDocBlockStartFixer`:                    true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveDeadParamFixer`:                       true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveDeadVarThisFixer`:                     true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveParamDescriptionDuplicateNameFixer`:   true,
	`Symplify\CodingStandard\Fixer\Commenting\RemoveParamNameReferenceFixer`:              true,
	`Symplify\CodingStandard\Fixer\Commenting\SwitchedTypeAndNameFixer`:                   true,
	`Symplify\CodingStandard\Fixer\Annotation\RemovePHPStormAnnotationFixer`:              true,
	`Symplify\CodingStandard\Fixer\Annotation\RemovePropertyVariableNameDescriptionFixer`: true,
	`Symplify\CodingStandard\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer`:  true,
	`Symplify\CodingStandard\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer`:      true,
	`Symplify\CodingStandard\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer`:   true,
}

// Fixer inspects a token stream and mutates it in place. It is the core contract
// every rule implements, mirroring a PHP-CS-Fixer fixer.
type Fixer interface {
	// Name is the checker identifier (the PHP-CS-Fixer FQCN) shown in reports.
	Name() string
	// SourceURL links to the original PHP-CS-Fixer rule on GitHub. Required for
	// every fixer and verified by the source checker.
	SourceURL() string
	// Fix mutates the stream and reports whether it changed anything.
	Fix(s *tokens.Stream) bool
}

// Candidate is a Fixer that can tell, mostly from the stream's Index, that a
// file cannot concern it, as PHP-CS-Fixer's isCandidate. The runner calls Fix
// (and Errors) only when IsCandidate holds, so IsCandidate must never be false
// for a file the rule would change or report an error on.
type Candidate interface {
	IsCandidate(s *tokens.Stream) bool
}

// ConfigurableFixer is a Fixer that honors ECS rule options (from the dumped
// config). WithConfig returns a fixer configured for the given options.
type ConfigurableFixer interface {
	Fixer
	WithConfig(config map[string]any) Fixer
}

// RawConfigurableFixer is a ConfigurableFixer that needs the order of its
// config's keys, which a map loses. The ECS loader calls WithRawConfig instead.
type RawConfigurableFixer interface {
	ConfigurableFixer
	WithRawConfig(raw []byte) Fixer
}

// SourceURLFor derives the canonical PHP-CS-Fixer source URL from a fixer Name
// (an FQCN like `PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`).
func SourceURLFor(name string) string {
	// a PHP_CodeSniffer sniff: "PHP_CodeSniffer\Standards\...\FooSniff" maps
	// onto "src/Standards/.../FooSniff.php" in the PHPCSStandards repo.
	if rest, ok := strings.CutPrefix(name, `PHP_CodeSniffer\`); ok {
		return PhpCodeSnifferSourceBase + strings.ReplaceAll(rest, `\`, "/") + ".php"
	}

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
