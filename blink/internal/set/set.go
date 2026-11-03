// Package set groups fixers into named, ordered sets, mirroring ECS prepared
// sets and gradual levels.
package set

import (
	"maps"
	"sort"
	"strings"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

// Spaces is the ordered spaces set (safest first), matching ECS SpacesLevel.
func Spaces() []fixer.Fixer { return fromRegistry(rules.SpacingFixers(), spacesConfig) }

// Casing is the casing set.
func Casing() []fixer.Fixer { return fromRegistry(rules.CasingFixers(), nil) }

// fromRegistry swaps each fixer for the registry's, which holds PHP-CS-Fixer's
// defaults, then applies the options config gives it.
func fromRegistry(fixers []fixer.Fixer, config setConfig) []fixer.Fixer {
	out := make([]fixer.Fixer, 0, len(fixers))
	for _, f := range fixers {
		if registered, ok := rules.ByName(f.Name()); ok {
			f = registered
		}
		out = append(out, configure(f, config[shortName(f.Name())]))
	}
	return out
}

// PSR12 is the @PSR-12 rule set as ECS exposes it (withPreparedSets(psr12: true)),
// restricted to the fixers blink implements.
func PSR12() []fixer.Fixer { return selectMembers(psr12Members, psr12Config) }

// Common is ECS's common prepared set (array, casing, cleanup, comments,
// control-structures, docblock, namespaces, spaces, clean-code), restricted to
// the fixers blink implements.
func Common() []fixer.Fixer { return selectMembers(commonMembers, commonConfig) }

// Default is what blink runs with no config: psr12 + common, exactly the pair
// ECS's default prepared sets enable. It is the union of PSR12 and Common.
func Default() []fixer.Fixer { return selectMembers(defaultMembers, defaultConfig) }

// PERCS is ECS's SetList::PER_CS (config/set/per-cs.php), limited to the fixers
// blink implements.
func PERCS() []fixer.Fixer { return build(perCSRules) }

// setRule is a rule of a prepared set and its configuration (nil for defaults).
type setRule struct {
	class  string
	config map[string]any
}

// build returns the set's implemented fixers, configured, in canonical order.
func build(set []setRule) []fixer.Fixer {
	out := make([]fixer.Fixer, 0, len(set))
	for _, r := range set {
		f, ok := rules.ByName(r.class)
		if !ok {
			continue
		}
		out = append(out, configure(f, r.config))
	}
	rules.CanonicalOrder(out)
	return out
}

// perCSRules copies config/set/per-cs.php, with options in their JSON shapes as
// dump-config sends them. TestPERCSMatchesECS fails when the two drift apart.
var perCSRules = []setRule{
	{`PhpCsFixer\Fixer\Operator\BinaryOperatorSpacesFixer`, map[string]any{
		"default": "at_least_single_space",
	}},
	{`PhpCsFixer\Fixer\Basic\BracesPositionFixer`, map[string]any{
		"allow_single_line_anonymous_functions":     false,
		"allow_single_line_empty_anonymous_classes": true,
	}},
	{`PhpCsFixer\Fixer\ClassNotation\ClassDefinitionFixer`, map[string]any{
		"inline_constructor_arguments": false,
		"space_before_parenthesis":     true,
	}},
	{`PhpCsFixer\Fixer\Operator\ConcatSpaceFixer`, map[string]any{
		"spacing": "one",
	}},
	{`PhpCsFixer\Fixer\FunctionNotation\FunctionDeclarationFixer`, map[string]any{
		"closure_fn_spacing": "none",
	}},
	{`PhpCsFixer\Fixer\FunctionNotation\MethodArgumentSpaceFixer`, map[string]any{
		"after_heredoc": false,
	}},
	{`PhpCsFixer\Fixer\Operator\NewWithParenthesesFixer`, map[string]any{
		"anonymous_class": false,
	}},
	{`PhpCsFixer\Fixer\Whitespace\NoExtraBlankLinesFixer`, map[string]any{
		"tokens": []any{"use"},
	}},
	{`PhpCsFixer\Fixer\ClassNotation\OrderedClassElementsFixer`, map[string]any{
		"order": []any{"use_trait"},
	}},
	{`PhpCsFixer\Fixer\Import\OrderedImportsFixer`, map[string]any{
		"imports_order":  []any{"class", "function", "const"},
		"sort_algorithm": "none",
	}},
	{`PhpCsFixer\Fixer\ClassNotation\OrderedTypesFixer`, map[string]any{
		"null_adjustment": "always_last",
		"sort_algorithm":  "none",
	}},
	{`PhpCsFixer\Fixer\Import\SingleImportPerStatementFixer`, map[string]any{
		"group_to_single_imports": false,
	}},
	{`PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`, map[string]any{
		"constructs_followed_by_a_single_space": []any{
			"abstract", "as", "case", "catch", "class", "const", "const_import", "do", "else", "elseif",
			"enum", "final", "finally", "for", "foreach", "function", "function_import", "if", "insteadof",
			"interface", "match", "named_argument", "namespace", "new", "private", "protected", "public",
			"readonly", "static", "switch", "trait", "try", "type_colon", "use", "use_lambda", "while",
		},
		"constructs_preceded_by_a_single_space": []any{"as", "else", "elseif", "use_lambda"},
	}},
	{`PhpCsFixer\Fixer\ControlStructure\TrailingCommaInMultilineFixer`, map[string]any{
		"after_heredoc": true,
		"elements":      []any{"arguments", "array_destructuring", "arrays", "match", "parameters"},
	}},
	{`PhpCsFixer\Fixer\Operator\UnaryOperatorSpacesFixer`, map[string]any{
		"only_dec_inc": true,
	}},
	{`PhpCsFixer\Fixer\Whitespace\ArrayIndentationFixer`, nil},
	{`PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer`, nil},
	{`PhpCsFixer\Fixer\AttributeNotation\AttributeBlockNoSpacesFixer`, nil},
	{`PhpCsFixer\Fixer\NamespaceNotation\BlankLineAfterNamespaceFixer`, nil},
	{`PhpCsFixer\Fixer\PhpTag\BlankLineAfterOpeningTagFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\BlankLineBetweenImportGroupsFixer`, nil},
	{`PhpCsFixer\Fixer\NamespaceNotation\BlankLinesBeforeNamespaceFixer`, nil},
	{`PhpCsFixer\Fixer\CastNotation\CastSpacesFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\CompactNullableTypeDeclarationFixer`, nil},
	{`PhpCsFixer\Fixer\Casing\ConstantCaseFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\ControlStructureBracesFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\ControlStructureContinuationPositionFixer`, nil},
	{`PhpCsFixer\Fixer\LanguageConstruct\DeclareEqualNormalizeFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\ElseifFixer`, nil},
	{`PhpCsFixer\Fixer\Basic\EncodingFixer`, nil},
	{`PhpCsFixer\Fixer\PhpTag\FullOpeningTagFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\IndentationTypeFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\LineEndingFixer`, nil},
	{`PhpCsFixer\Fixer\CastNotation\LowercaseCastFixer`, nil},
	{`PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer`, nil},
	{`PhpCsFixer\Fixer\Casing\LowercaseStaticReferenceFixer`, nil},
	{`PhpCsFixer\Fixer\ClassNotation\ModifierKeywordsFixer`, nil},
	{`PhpCsFixer\Fixer\ClassNotation\NoBlankLinesAfterClassOpeningFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\NoBreakCommentFixer`, nil},
	{`PhpCsFixer\Fixer\PhpTag\NoClosingTagFixer`, nil},
	{`PhpCsFixer\Fixer\Import\NoLeadingImportSlashFixer`, nil},
	{`PhpCsFixer\Fixer\Basic\NoMultipleStatementsPerLineFixer`, nil},
	{`PhpCsFixer\Fixer\Operator\NoSpaceAroundDoubleColonFixer`, nil},
	{`PhpCsFixer\Fixer\FunctionNotation\NoSpacesAfterFunctionNameFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\NoTrailingWhitespaceFixer`, nil},
	{`PhpCsFixer\Fixer\Comment\NoTrailingWhitespaceInCommentFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\NoWhitespaceInBlankLineFixer`, nil},
	{`PhpCsFixer\Fixer\LanguageConstruct\NullableTypeDeclarationFixer`, nil},
	{`PhpCsFixer\Fixer\Operator\OperatorLinebreakFixer`, nil},
	{`PhpCsFixer\Fixer\FunctionNotation\ReturnTypeDeclarationFixer`, nil},
	{`PhpCsFixer\Fixer\CastNotation\ShortScalarCastFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\SingleBlankLineAtEofFixer`, nil},
	{`PhpCsFixer\Fixer\ClassNotation\SingleClassElementPerStatementFixer`, nil},
	{`PhpCsFixer\Fixer\Import\SingleLineAfterImportsFixer`, nil},
	{`PhpCsFixer\Fixer\Basic\SingleLineEmptyBodyFixer`, nil},
	{`PhpCsFixer\Fixer\ClassNotation\SingleTraitInsertPerStatementFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\SpacesInsideParenthesesFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\StatementIndentationFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\SwitchCaseSemicolonToColonFixer`, nil},
	{`PhpCsFixer\Fixer\ControlStructure\SwitchCaseSpaceFixer`, nil},
	{`PhpCsFixer\Fixer\Operator\TernaryOperatorSpacesFixer`, nil},
	{`PhpCsFixer\Fixer\Whitespace\TypesSpacesFixer`, nil},
}

var byName = map[string]func() []fixer.Fixer{
	"spaces": Spaces,
	"casing": Casing,
	"psr12":  PSR12,
	"per-cs": PERCS,
	"common": Common,
}

// selectMembers keeps the built-in fixers whose short class name is in members,
// preserving All()'s priority-sorted execution order, with the options config
// gives them.
func selectMembers(members map[string]bool, config setConfig) []fixer.Fixer {
	var out []fixer.Fixer
	for _, f := range rules.All() {
		if name := shortName(f.Name()); members[name] {
			out = append(out, configure(f, config[name]))
		}
	}
	return out
}

// configure applies options over f's own (PHP-CS-Fixer's defaults for a
// registry fixer); nil options leave f as it is.
func configure(f fixer.Fixer, options map[string]any) fixer.Fixer {
	if options == nil {
		return f
	}
	return f.(fixer.ConfigurableFixer).WithConfig(options)
}

// setConfig maps a member's short class name to the options a set gives it.
type setConfig map[string]map[string]any

// The options ECS's prepared sets configure, for members whose blink default
// (PHP-CS-Fixer's) differs from what the set wants. commonConfig mirrors the
// RULE_CONFIGURATIONS of src/Config/Level/*Level.php, psr12Config
// config/set/psr12.php and spacesConfig SpacesLevel. per-cs carries its options
// in perCSRules.
var (
	commonConfig = setConfig{
		"ConcatSpaceFixer":                   {"spacing": "one"},
		"GeneralPhpdocAnnotationRemoveFixer": {"annotations": []any{"author", "package", "group", "category"}},
		"NoSuperfluousPhpdocTagsFixer":       {"allow_mixed": true, "remove_inheritdoc": true},
		"OrderedImportsFixer":                {"imports_order": []any{"class", "function", "const"}},
		"YodaStyleFixer":                     {"equal": false, "identical": false, "less_and_greater": false},
	}
	psr12Config = setConfig{
		"ConcatSpaceFixer":    {"spacing": "one"},
		"OrderedImportsFixer": {"imports_order": []any{"class", "function", "const"}},
	}
	spacesConfig = setConfig{
		"ConcatSpaceFixer": {"spacing": "one"},
	}
)

// defaultConfig is psr12Config then commonConfig, a later option winning.
var defaultConfig = func() setConfig {
	m := setConfig{}
	for _, c := range []setConfig{psr12Config, commonConfig} {
		for name, options := range c {
			if m[name] == nil {
				m[name] = map[string]any{}
			}
			maps.Copy(m[name], options)
		}
	}
	return m
}()

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
