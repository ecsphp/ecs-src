// Package set groups fixers into named, ordered sets, mirroring ECS prepared
// sets and gradual levels.
package set

import (
	"sort"
	"strings"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

// Spaces is the ordered spaces set (safest first), matching ECS SpacesLevel.
func Spaces() []fixer.Fixer { return rules.SpacingFixers() }

// PSR12 is the @PSR-12 rule set as ECS exposes it (withPreparedSets(psr12: true)),
// restricted to the fixers blink implements.
func PSR12() []fixer.Fixer { return selectMembers(psr12Members) }

// Common is ECS's common prepared set (array, casing, cleanup, comments,
// control-structures, docblock, namespaces, spaces, clean-code), restricted to
// the fixers blink implements.
func Common() []fixer.Fixer { return selectMembers(commonMembers) }

// Default is what blink runs with no config: psr12 + common, exactly the pair
// ECS's default prepared sets enable. It is the union of PSR12 and Common.
func Default() []fixer.Fixer { return selectMembers(defaultMembers) }

// PERCS mirrors PHP-CS-Fixer's @PER-CS (as exposed by ECS SetList::PER_CS): the
// full rule set plus single_line_empty_body, which is PER-CS-specific and not
// part of the default/common output.
func PERCS() []fixer.Fixer {
	all := rules.All()
	out := make([]fixer.Fixer, 0, len(all)+1)
	for _, f := range all {
		if percsExcluded[shortName(f.Name())] {
			continue
		}
		out = append(out, f)
	}
	return append(out, rules.SingleLineEmptyBody{})
}

// percsExcluded are implemented fixers that @PER-CS does not enable, so the
// per-cs set (which otherwise runs every fixer) leaves them out. PsrAutoloading
// renames classes from the file path and is opt-in only.
var percsExcluded = map[string]bool{
	"PsrAutoloadingFixer": true,
}

var byName = map[string]func() []fixer.Fixer{
	"spaces": Spaces,
	"casing": rules.CasingFixers,
	"psr12":  PSR12,
	"per-cs": PERCS,
	"common": Common,
}

// selectMembers keeps the built-in fixers whose short class name is in members,
// preserving All()'s priority-sorted execution order.
func selectMembers(members map[string]bool) []fixer.Fixer {
	var out []fixer.Fixer
	for _, f := range rules.All() {
		if members[shortName(f.Name())] {
			out = append(out, f)
		}
	}
	return out
}

// shortName returns the class name without its namespace ("...\FooFixer" -> "FooFixer").
func shortName(name string) string {
	if i := strings.LastIndexByte(name, '\\'); i >= 0 {
		return name[i+1:]
	}
	return name
}

// Get returns the fixers of a named set.
func Get(name string) ([]fixer.Fixer, bool) {
	build, ok := byName[name]
	if !ok {
		return nil, false
	}
	return build(), true
}

// SpacesLevel returns the first n rules of the spaces set, for gradual adoption
// (withSpacesLevel in ECS). n is clamped to the set size.
func SpacesLevel(n int) []fixer.Fixer {
	all := Spaces()
	if n < 0 {
		n = 0
	}
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// Names lists the available set names, sorted.
func Names() []string {
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// defaultMembers is psr12Members ∪ commonMembers.
var defaultMembers = func() map[string]bool {
	m := map[string]bool{}
	for k := range psr12Members {
		m[k] = true
	}
	for k := range commonMembers {
		m[k] = true
	}
	return m
}()

// psr12Members is the @PSR-12 set (from ECS config/set/psr12.php), by short name.
var psr12Members = map[string]bool{
	"BinaryOperatorSpacesFixer":                   true,
	"BlankLineAfterNamespaceFixer":                true,
	"BlankLineAfterOpeningTagFixer":               true,
	"BracesPositionFixer":                         true,
	"ClassDefinitionFixer":                        true,
	"ConcatSpaceFixer":                            true,
	"ConstantCaseFixer":                           true,
	"ControlStructureBracesFixer":                 true,
	"ControlStructureContinuationPositionFixer":   true,
	"DeclareEqualNormalizeFixer":                  true,
	"DeclareParenthesesFixer":                     true,
	"ElseifFixer":                                 true,
	"EncodingFixer":                               true,
	"FullOpeningTagFixer":                         true,
	"FunctionDeclarationFixer":                    true,
	"IndentationTypeFixer":                        true,
	"LineEndingFixer":                             true,
	"LowercaseCastFixer":                          true,
	"LowercaseKeywordsFixer":                      true,
	"MethodArgumentSpaceFixer":                    true,
	"NewWithParenthesesFixer":                     true,
	"NoBlankLinesAfterClassOpeningFixer":          true,
	"NoBreakCommentFixer":                         true,
	"NoClosingTagFixer":                           true,
	"NoExtraBlankLinesFixer":                      true,
	"NoLeadingImportSlashFixer":                   true,
	"NoMultipleStatementsPerLineFixer":            true,
	"NoSinglelineWhitespaceBeforeSemicolonsFixer": true,
	"NoSpacesAfterFunctionNameFixer":              true,
	"NoTrailingWhitespaceFixer":                   true,
	"NoTrailingWhitespaceInCommentFixer":          true,
	"NoWhitespaceBeforeCommaInArrayFixer":         true,
	"OrderedImportsFixer":                         true,
	"ReturnTypeDeclarationFixer":                  true,
	"ShortScalarCastFixer":                        true,
	"SingleBlankLineAtEofFixer":                   true,
	"SingleClassElementPerStatementFixer":         true,
	"SingleImportPerStatementFixer":               true,
	"SingleLineAfterImportsFixer":                 true,
	"SingleSpaceAroundConstructFixer":             true,
	"SpacesInsideParenthesesFixer":                true,
	"StatementIndentationFixer":                   true,
	"SwitchCaseSemicolonToColonFixer":             true,
	"SwitchCaseSpaceFixer":                        true,
	"TernaryOperatorSpacesFixer":                  true,
	"UnaryOperatorSpacesFixer":                    true,
	"VisibilityRequiredFixer":                     true,
	"WhitespaceAfterCommaInArrayFixer":            true,
}

// commonMembers is ECS's common set (array, casing, cleanup, comments,
// control-structures, docblock, namespaces, spaces, clean-code), by short name.
var commonMembers = map[string]bool{
	"AddMissingParamNameFixer":                        true,
	"AddMissingVarNameFixer":                          true,
	"AlignMultilineCommentFixer":                      true,
	"ArrayIndentationFixer":                           true,
	"ArrayListItemNewlineFixer":                       true,
	"ArrayOpenerAndCloserNewlineFixer":                true,
	"ArraySyntaxFixer":                                true,
	"AssignNullCoalescingToCoalesceEqualFixer":        true,
	"BinaryOperatorSpacesFixer":                       true,
	"BlankLineAfterOpeningTagFixer":                   true,
	"BlankLineAfterStrictTypesFixer":                  true,
	"CastSpacesFixer":                                 true,
	"ClassAttributesSeparationFixer":                  true,
	"ClassDefinitionFixer":                            true,
	"ClassReferenceNameCasingFixer":                   true,
	"ConcatSpaceFixer":                                true,
	"DoubleAsteriskInlineVarFixer":                    true,
	"ExplicitIndirectVariableFixer":                   true,
	"ExplicitStringVariableFixer":                     true,
	"FixParamNameTypoFixer":                           true,
	"FixTagTypoFixer":                                 true,
	"FunctionToConstantFixer":                         true,
	"GeneralPhpdocAnnotationRemoveFixer":              true,
	"IncludeFixer":                                    true,
	"IntegerLiteralCaseFixer":                         true,
	"IsNullFixer":                                     true,
	"LambdaNotUsedImportFixer":                        true,
	"ListSyntaxFixer":                                 true,
	"LongToShorthandOperatorFixer":                    true,
	"MagicConstantCasingFixer":                        true,
	"MagicMethodCasingFixer":                          true,
	"MergeDocBlockStartFixer":                         true,
	"MethodArgumentSpaceFixer":                        true,
	"MethodChainingIndentationFixer":                  true,
	"MultilineCommentOpeningClosingFixer":             true,
	"NativeFunctionCasingFixer":                       true,
	"NativeTypeDeclarationCasingFixer":                true,
	"NewWithParenthesesFixer":                         true,
	"NoAlternativeSyntaxFixer":                        true,
	"NoBlankLineBetweenImportsFixer":                  true,
	"NoBlankLinesAfterClassOpeningFixer":              true,
	"NoBlankLinesAfterPhpdocFixer":                    true,
	"NoEmptyCommentFixer":                             true,
	"NoEmptyPhpdocFixer":                              true,
	"NoEmptyStatementFixer":                           true,
	"NoExtraBlankLinesFixer":                          true,
	"NoLeadingNamespaceWhitespaceFixer":               true,
	"NoMultilineWhitespaceAroundDoubleArrowFixer":     true,
	"NoNullPropertyInitializationFixer":               true,
	"NoShortBoolCastFixer":                            true,
	"NoSinglelineWhitespaceBeforeSemicolonsFixer":     true,
	"NoSpacesAroundOffsetFixer":                       true,
	"NoSuperfluousElseifFixer":                        true,
	"NoSuperfluousPhpdocTagsFixer":                    true,
	"NoTrailingCommaInSinglelineFixer":                true,
	"NoTrailingWhitespaceInCommentFixer":              true,
	"NoUnneededControlParenthesesFixer":               true,
	"NoUnneededBracesFixer":                           true,
	"NoUnneededImportAliasFixer":                      true,
	"NoUnusedImportsFixer":                            true,
	"NoUselessConcatOperatorFixer":                    true,
	"NoUselessNullsafeOperatorFixer":                  true,
	"NoUselessReturnFixer":                            true,
	"NoWhitespaceBeforeCommaInArrayFixer":             true,
	"NoWhitespaceInBlankLineFixer":                    true,
	"NoWhitespaceInEmptyArrayFixer":                   true,
	"NotOperatorWithSuccessorSpaceFixer":              true,
	"NullableTypeDeclarationForDefaultNullValueFixer": true,
	"ObjectOperatorWithoutWhitespaceFixer":            true,
	"OrderedImportsFixer":                             true,
	"ParamReturnAndVarTagMalformsFixer":               true,
	"PhpdocIndentFixer":                               true,
	"PhpdocInlineTagNormalizerFixer":                  true,
	"PhpdocLineSpanFixer":                             true,
	"PhpdocNoAccessFixer":                             true,
	"PhpdocNoAliasTagFixer":                           true,
	"PhpdocNoDuplicateTypesFixer":                     true,
	"PhpdocNoEmptyReturnFixer":                        true,
	"PhpdocNoPackageFixer":                            true,
	"PhpdocOrderByValueFixer":                         true,
	"PhpdocReturnSelfReferenceFixer":                  true,
	"PhpdocScalarFixer":                               true,
	"PhpdocSingleLineVarSpacingFixer":                 true,
	"PhpdocTagCasingFixer":                            true,
	"PhpdocTrimConsecutiveBlankLineSeparationFixer":   true,
	"PhpdocTrimFixer":                                 true,
	"PhpdocTypesFixer":                                true,
	"PhpdocVarWithoutNameFixer":                       true,
	"ProtectedToPrivateFixer":                         true,
	"RemoveDeadParamFixer":                            true,
	"RemoveDeadVarThisFixer":                          true,
	"RemoveEventSubscriberDescriptionFixer":           true,
	"RemoveMethodNameDuplicateDescriptionFixer":       true,
	"RemovePHPStormAnnotationFixer":                   true,
	"RemoveParamDescriptionDuplicateNameFixer":        true,
	"RemoveParamNameReferenceFixer":                   true,
	"RemovePropertyVariableNameDescriptionFixer":      true,
	"RemoveSuperfluousReturnNameFixer":                true,
	"RemoveSuperfluousVarNameFixer":                   true,
	"RemoveUselessDefaultCommentFixer":                true,
	"ReturnTypeDeclarationFixer":                      true,
	"SelfAccessorFixer":                               true,
	"SingleBlankLineBeforeNamespaceFixer":             true,
	"SingleClassElementPerStatementFixer":             true,
	"SingleLineCommentSpacingFixer":                   true,
	"SingleLineInlineVarDocBlockFixer":                true,
	"SingleQuoteFixer":                                true,
	"SingleTraitInsertPerStatementFixer":              true,
	"SpaceAfterCommaHereNowDocFixer":                  true,
	"SpaceAfterSemicolonFixer":                        true,
	"StandaloneLinePromotedPropertyFixer":             true,
	"StandaloneLineRequiredParamFixer":                true,
	"StandardizeIncrementFixer":                       true,
	"StandardizeNotEqualsFixer":                       true,
	"SwitchContinueToBreakFixer":                      true,
	"SwitchedTypeAndNameFixer":                        true,
	"TernaryOperatorSpacesFixer":                      true,
	"TernaryToNullCoalescingFixer":                    true,
	"TrailingCommaInMultilineFixer":                   true,
	"TrimArraySpacesFixer":                            true,
	"TypeDeclarationSpacesFixer":                      true,
	"TypeToVarTagFixer":                               true,
	"TypesSpacesFixer":                                true,
	"WhitespaceAfterCommaInArrayFixer":                true,
	"YodaStyleFixer":                                  true,
}
