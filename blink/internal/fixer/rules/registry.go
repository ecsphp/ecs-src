package rules

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// SpacingFixers are the whitespace/operator spacing rules, safest first. This is
// the ordered "spaces" set that gradual levels slice.
func SpacingFixers() []fixer.Fixer {
	return []fixer.Fixer{
		NoLeadingNamespaceWhitespace{},
		NoSinglelineWhitespaceBeforeSemicolons{},
		MultilineWhitespaceBeforeSemicolons{},
		NoWhitespaceInBlankLine{},
		SuperfluousWhitespace{},
		SpaceAfterSemicolon{},
		BinaryOperatorSpaces{},
		TernaryOperatorSpaces{},
		ConcatSpace{},
		CastSpaces{},
		LinebreakAfterOpeningTag{},
		BlankLineAfterOpeningTag{},
		BlankLineAfterStrictTypes{},
		NoTrailingWhitespace{},
		NoTrailingWhitespaceInComment{},
		SingleBlankLineAtEndOfFile{},
		BlankLineBeforeStatement{},
	}
}

// CasingFixers normalize keyword, constant and cast casing.
func CasingFixers() []fixer.Fixer {
	return []fixer.Fixer{
		LowercaseKeywords{},
		ConstantCase{},
		LowercaseStaticReference{},
		LowercaseCast{},
		ShortScalarCast{},
		MagicConstantCasing{},
		MagicMethodCasing{},
		NativeFunctionCasing{},
		IntegerLiteralCase{},
		NativeTypeDeclarationCasing{},
		NativeFunctionTypeDeclarationCasing{},
		ClassReferenceNameCasing{},
	}
}

// CommonFixers are non-PSR-12 rules from ECS's common/clean-code sets that are
// token-safe here.
func CommonFixers() []fixer.Fixer {
	return []fixer.Fixer{
		LineEnding{},
		IsNull{},
		YodaStyle{},
		ArraySyntax{},
		ListSyntax{},
		ReturnToYieldFrom{},
		YieldFromArrayToYields{},
		NoWhitespaceBeforeCommaInArray{},
		WhitespaceAfterCommaInArray{},
		TrailingCommaInMultiline{},
		NoTrailingCommaInSingleline{},
		NoSpacesAroundOffset{},
		ObjectOperatorWithoutWhitespace{},
		NoUselessNullsafeOperator{},
		StandardizeNotEquals{},
		TernaryToNullCoalescing{},
		NoEmptyStatement{},
		NoEmptyComment{},
		SingleLineCommentSpacing{},
		SingleQuote{},
		TrimArraySpaces{},
		NoSpaceAroundDoubleColon{},
		AttributeBlockNoSpaces{},
		HeredocToNowdoc{},
		NoBinaryString{},
		NoUselessConcatOperator{},
		NoShortBoolCast{},
		NoUnsetCast{},
		NoWhitespaceInEmptyArray{},
		NormalizeIndexBrace{},
		NoMultilineWhitespaceAroundDoubleArrow{},
		StandardizeIncrement{},
		IncrementStyle{},
		LongToShorthandOperator{},
		SwitchContinueToBreak{},
		NoAlternativeSyntax{},
		NoUnneededBraces{},
		NoMixedEchoPrint{},
		NoAliasFunctions{},
		NoUnneededImportAlias{},
		CleanNamespace{},
		MultilineCommentOpeningClosing{},
		Encoding{},
		DeclareParentheses{},
		TypeDeclarationSpaces{},
		CompactNullableTypeDeclaration{},
		NullableTypeDeclaration{},
		OrderedTypes{},
		TypesSpaces{},
		AlignMultilineComment{},
		AssignNullCoalescingToCoalesceEqual{},
		NullableTypeDeclarationForDefaultNullValue{},
		SingleLineCommentStyle{},
		ExplicitIndirectVariable{},
		ExplicitStringVariable{},
		NoNullPropertyInitialization{},
		Include{},
		EmptyLoopBody{},
		EmptyLoopCondition{},
		NoUselessReturn{},
		ReturnAssignment{},
		SimplifiedNullReturn{},
		FunctionToConstant{},
		NativeConstantInvocation{},
		GlobalNamespaceImport{},
		SelfAccessor{},
		RemoveUselessDefaultComment{},
		ProtectedToPrivate{},
		AddMissingParamName{},
		NoAliasLanguageConstructCall{},
		NoUnreachableDefaultArgumentValue{},
		DirConstant{},
		CombineNestedDirname{},
		RegularCallableCall{},
		ImplodeCall{},
		NoUselessSprintf{},
		NoUselessPrintf{},
		DateTimeCreateFromFormatCall{},
		DateTimeImmutable{},
		ModernizeTypesCasting{},
		FopenFlagOrder{},
		FopenFlags{},
		SimpleToComplexStringVariable{},
		StringLengthToEmpty{},
		StringLineEnding{},
		NoTrailingWhitespaceInString{},
		HeredocClosingMarker{},
		MultilineStringToHeredoc{},
		StringImplicitBackslashes{},
		HeredocIndentation{},
		NonPrintableCharacter{},
		NoHomoglyphNames{},
		ModernSerializationMethods{},
		ArrayPush{},
		SetTypeToCast{},
		PowToExponentiation{},
		EregToPreg{},
		ModernizeStrpos{},
		BacktickToShellExec{},
		MbStrFunctions{},
		RandomApiMigration{},
		StrictComparison{},
		StrictParam{},
		OctalNotation{},
		NumericLiteralSeparator{},
		AttributeEmptyParentheses{},
	}
}

// PhpdocFixers normalize doc comments.
func PhpdocFixers() []fixer.Fixer {
	return []fixer.Fixer{
		DoctrineAnnotationSpaces{},
		DoctrineAnnotationArrayAssignment{},
		DoctrineAnnotationIndentation{},
		DoctrineAnnotationBraces{},
		NoSuperfluousPhpdocTags{},
		PhpdocNoUselessInheritdoc{},
		PhpdocAddMissingParamAnnotation{},
		PhpdocAnnotationWithoutDot{},
		PhpdocArrayType{},
		PhpdocListType{},
		PhpdocParamOrder{},
		PhpdocTypesNoDuplicates{},
		PhpdocScalar{},
		GeneralPhpdocAnnotationRemove{},
		PhpdocTypes{},
		PhpdocNoAliasTag{},
		PhpdocNoPackage{},
		PhpdocNoAccess{},
		PhpdocSingleLineVarSpacing{},
		PhpdocNoEmptyReturn{},
		PhpdocTrim{},
		PhpdocTrimConsecutiveBlankLineSeparation{},
		NoEmptyPhpdoc{},
		NoBlankLinesAfterPhpdoc{},
		PhpdocTagCasing{},
		PhpdocInlineTagNormalizer{},
		PhpdocNoDuplicateTypes{},
		PhpdocVarWithoutName{},
		PhpdocIndent{},
		PhpdocOrderByValue{},
		PhpdocLineSpan{},
		PhpdocTypesOrder{},
		PhpdocVarAnnotationCorrectOrder{},
		PhpdocReturnSelfReference{},
		PhpdocSummary{},
		PhpdocOrder{},
		PhpdocTagType{},
	}
}

// ConstructFixers cover keyword/parenthesis/operator spacing and import cleanups
// from the PSR-12 set.
func ConstructFixers() []fixer.Fixer {
	return []fixer.Fixer{
		DeclareEqualNormalize{},
		SingleSpaceAroundConstruct{},
		NoSpacesAfterFunctionName{},
		LambdaNotUsedImport{},
		NoSpacesInsideParenthesis{},
		UnaryOperatorSpaces{},
		NotOperatorWithSuccessorSpace{},
		NotOperatorWithSpace{},
		NewExpressionParentheses{},
		SimplifiedIfReturn{},
		NoLeadingImportSlash{},
		Elseif{},
		NoSuperfluousElseif{},
		ControlStructureBraces{},
		ControlStructureContinuationPosition{},
		NoUnneededControlParentheses{},
		SwitchCaseSemicolonToColon{},
		SwitchCaseSpace{},
		NoMultipleStatementsPerLine{},
		MethodArgumentSpace{},
		MultilinePromotedProperties{},
		StandaloneLinePromotedProperty{},
		StandaloneLineConstructorParam{},
		StandaloneLinePlainConstructorParam{},
		StandaloneLineRequiredParam{},
		StandaloneLineSymfonyAttributeParam{},
		NoBreakComment{},
		ReturnTypeDeclaration{},
		NewWithParentheses{},
		FunctionDeclaration{},
		SingleLineThrow{},
		UseArrowFunctions{},
		RemoveDeadParam{},
		RemoveDeadVarThis{},
		RemoveParamNameReference{},
		SwitchedTypeAndName{},
		RemovePHPStormAnnotation{},
		RemoveEventSubscriberDescription{},
		RemoveMethodNameDuplicateDescription{},
		RemovePropertyVariableNameDescription{},
		OrderedInterfaces{},
		OrderedTraits{},
		PhpUnitMethodCasing{},
		PhpUnitSetUpTearDownVisibility{},
		PhpUnitConstruct{},
		PhpUnitFqcnAnnotation{},
		PhpUnitMockShortWillReturn{},
		PhpUnitTestAnnotation{},
		PhpUnitStrict{},
		PhpUnitAssertNewNames{},
		PhpUnitMock{},
		PhpUnitDedicateAssertInternalType{},
		PhpUnitNamespaced{},
		PhpUnitAttributes{},
		PhpUnitDedicateAssert{},
		PhpUnitTestCaseStaticMethodCalls{},
		PhpUnitDataProviderMethodOrder{},
		PhpUnitDataProviderName{},
		PhpUnitDataProviderReturnType{},
		PhpUnitDataProviderStatic{},
		PhpUnitInternalClass{},
		PhpUnitSizeClass{},
		PhpUnitTestClassRequiresCovers{},
		PhpUnitExpectation{},
		PhpUnitNoExpectationAnnotation{},
		GlobalKeyword{},
		OperatorLinebreak{},
		PhpdocSeparation{},
		PhpdocToComment{},
		CommentToPhpdoc{},
		PhpdocAlign{},
		ParamReturnAndVarTagMalforms{},
		GeneralPhpdocTagRename{},
		DeclareStrictTypes{},
		VoidReturn{},
		StaticLambda{},
		NoUselessElse{},
		NoUnneededFinalMethod{},
		NoPhp4Constructor{},
		ErrorSuppression{},
		CombineConsecutiveIssets{},
		CombineConsecutiveUnsets{},
		NoUnsetOnProperty{},
		GetClassToClassKeyword{},
		ClassKeywordRemove{},
		ClassKeyword{},
		LogicalOperators{},
		TernaryToElvisOperator{},
		EchoTagSyntax{},
		SemicolonAfterInstruction{},
	}
}

// StructuralFixers reflow imports, namespace/class blank lines and indentation.
func StructuralFixers() []fixer.Fixer {
	return []fixer.Fixer{
		IndentationType{},
		ClassDefinition{},
		BracesPosition{},
		SingleLineEmptyBody{},
		VisibilityRequired{},
		ModifierKeywords{},
		PsrAutoloading{},
		SingleTraitInsertPerStatement{},
		SingleClassElementPerStatement{},
		OrderedClassElements{},
		SelfStaticAccessor{},
		ClassAttributesSeparation{},
		BlankLinesBeforeNamespace{},
		NoBlankLinesBeforeNamespace{},
		SingleBlankLineBeforeNamespace{},
		BlankLineAfterNamespace{},
		NoUnusedImports{},
		RemoveSlashFromNativeConstant{},
		FullyQualifiedStrictTypes{},
		SingleImportPerStatement{},
		GroupImport{},
		OrderedImports{},
		BlankLineBetweenImportGroups{},
		SingleLineAfterImports{},
		NoBlankLineBetweenImports{},
		SpaceAfterCommaHereNowDoc{},
		ArrayOpenerAndCloserNewline{},
		NoBlankLinesAfterClassOpening{},
		StatementIndentation{},
		MethodChainingNewline{},
		MethodChainingIndentation{},
		ArrayListItemNewline{},
		StandaloneLineInMultilineArray{},
		LineLength{},
		ArrayIndentation{},
		FinalInternalClass{},
		FinalClass{},
		FinalPublicMethodForAbstractClass{},
		StaticPrivateMethod{},
		StringableForToString{},
		NoRedundantReadonlyProperty{},
		PhpdocReadonlyClassCommentToKeyword{},
		NoExtraBlankLines{}, // after import removal, which can leave extra blanks
	}
}

// FullOpeningTag runs first (normalize the tag) and NoClosingTag last
// (trailing tag/EOF cleanup).
func builtinFixers() []fixer.Fixer {
	all := []fixer.Fixer{FullOpeningTag{}}
	all = append(all, CommonFixers()...)
	all = append(all, PhpdocFixers()...)
	all = append(all, CasingFixers()...)
	all = append(all, SpacingFixers()...)
	all = append(all, ConstructFixers()...)
	all = append(all, StructuralFixers()...)
	all = append(all,
		DoubleAsteriskInlineVar{},
		FixTagTypo{},
		TypeToVarTag{},
		MergeDocBlockStart{},
		AddMissingVarName{},
		SingleLineInlineVarDocBlock{},
		RemoveSuperfluousReturnName{},
		RemoveSuperfluousVarName{},
		RemoveParamDescriptionDuplicateName{},
		FixParamNameTypo{},
		LanguageConstructSpacing{},
		NativeFunctionInvocation{})
	all = append(all, NoClosingTag{})
	// a rule run without options must behave like PHP-CS-Fixer's default config
	for i, f := range all {
		all[i] = withPHPDefaults(f)
	}
	return all
}

type registration struct {
	fixer    fixer.Fixer
	priority int
}

type registry struct {
	sorted  []fixer.Fixer          // canonical execution order
	builtin []fixer.Fixer          // sorted, without the registered fixers
	byName  map[string]fixer.Fixer // Name -> fixer
	pos     map[string]int         // Name -> index in sorted
}

var (
	registryOnce  sync.Once
	registryBuilt bool
	registrations []registration
	reg           registry
)

// Register adds a fixer that is not built in. It panics when called after the
// registry was built, or when the name is taken.
func Register(f fixer.Fixer, priority int) {
	if registryBuilt {
		panic(fmt.Sprintf("blink: Register(%s) called after the fixer registry was built", f.Name()))
	}
	if nameTaken(f.Name(), builtinFixers(), registrations) {
		panic(fmt.Sprintf("blink: cannot register %s, a fixer of that name already exists", f.Name()))
	}
	registrations = append(registrations, registration{fixer: f, priority: priority})
}

// A deprecated alias counts as taken: ByName resolves it to its successor.
func nameTaken(name string, builtins []fixer.Fixer, registered []registration) bool {
	if _, alias := deprecatedAliases[name]; alias {
		return true
	}
	return slices.ContainsFunc(builtins, func(f fixer.Fixer) bool { return f.Name() == name }) ||
		slices.ContainsFunc(registered, func(r registration) bool { return r.fixer.Name() == name })
}

func buildRegistry() {
	registryBuilt = true
	reg = newRegistry(builtinFixers(), registrations)
}

func newRegistry(builtins []fixer.Fixer, registered []registration) registry {
	all := slices.Clone(builtins)
	priority := make(map[string]int, len(registered))
	for _, r := range registered {
		priority[r.fixer.Name()] = r.priority
		all = append(all, r.fixer)
	}
	priorityOf := func(f fixer.Fixer) int {
		if p, ok := priority[f.Name()]; ok {
			return p
		}
		return fixerPriorityOf(f)
	}
	// run in PHP-CS-Fixer execution order: priority descending, then by rule name
	// ascending - exactly how PHP-CS-Fixer's Utils::sortFixers breaks ties
	slices.SortStableFunc(all, func(a, b fixer.Fixer) int {
		if pa, pb := priorityOf(a), priorityOf(b); pa != pb {
			return cmp.Compare(pb, pa) // priority descending
		}
		return cmp.Compare(fixerSortName(a), fixerSortName(b))
	})

	r := registry{
		sorted: all,
		byName: make(map[string]fixer.Fixer, len(all)),
		pos:    make(map[string]int, len(all)),
	}
	for i, f := range all {
		r.byName[f.Name()] = f
		r.pos[f.Name()] = i
		if _, registered := priority[f.Name()]; !registered {
			r.builtin = append(r.builtin, f)
		}
	}
	return r
}

// All returns every fixer in PHP-CS-Fixer execution order, each configured with
// PHP-CS-Fixer's defaults (see phpDefaultsJSON). The registry is built
// once; the returned slice is a fresh copy the caller may append to or reorder.
func All() []fixer.Fixer {
	registryOnce.Do(buildRegistry)
	return slices.Clone(reg.sorted)
}

// Builtin is All without the fixers added with Register.
func Builtin() []fixer.Fixer {
	registryOnce.Do(buildRegistry)
	return slices.Clone(reg.builtin)
}

// fixerSortName returns the name PHP-CS-Fixer sorts a fixer by: a core fixer's
// snake_case rule name (its class short name minus "Fixer"), or the full class
// name for a non-core (e.g. Symplify) fixer, whose getName() returns static::class.
func fixerSortName(f fixer.Fixer) string {
	name := f.Name()
	if !strings.HasPrefix(name, `PhpCsFixer\`) {
		return name
	}
	short := name[strings.LastIndexByte(name, '\\')+1:]
	short = strings.TrimSuffix(short, "Fixer")
	return camelCaseToUnderscore(short)
}

// CanonicalOrder sorts fixers in place to match All()'s execution order
// (PHP-CS-Fixer priority plus the curated tie-break), so the blink/--blink path
// applies rules in the same order as the standalone path. Fixers unknown to
// All() are placed last.
func CanonicalOrder(fixers []fixer.Fixer) {
	registryOnce.Do(buildRegistry)
	const last = 1 << 30
	pos := func(f fixer.Fixer) int {
		if i, ok := reg.pos[f.Name()]; ok {
			return i
		}
		return last
	}
	slices.SortStableFunc(fixers, func(a, b fixer.Fixer) int {
		return cmp.Compare(pos(a), pos(b))
	})
}

// fixerPriorityOf returns a fixer's PHP-CS-Fixer priority (0 when unknown).
func fixerPriorityOf(f fixer.Fixer) int {
	name := f.Name()
	short := name
	if i := strings.LastIndexByte(name, '\\'); i >= 0 {
		short = name[i+1:]
	}
	return fixerPriority[short]
}

// deprecatedAlias is a deprecated PHP-CS-Fixer fixer that only proxies a
// successor blink implements.
type deprecatedAlias struct {
	successor string
	// options maps the deprecated fixer's options to its successor's; nil when
	// they are the same options.
	options func(deprecated map[string]any) map[string]any
}

// fixedOptions is for a deprecated fixer that has no options of its own.
func fixedOptions(successor map[string]any) func(map[string]any) map[string]any {
	return func(map[string]any) map[string]any { return successor }
}

// deprecatedAliases maps a deprecated PHP-CS-Fixer class to its successor, so a
// dumped ECS config that still references the old name resolves to the fixer,
// configured as the deprecated fixer configures it.
var deprecatedAliases = map[string]deprecatedAlias{
	`PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`: {successor: `PhpCsFixer\Fixer\ControlStructure\NoUnneededBracesFixer`},
	`PhpCsFixer\Fixer\Basic\CurlyBracesPositionFixer`:              {successor: `PhpCsFixer\Fixer\Basic\BracesPositionFixer`},
	`PhpCsFixer\Fixer\Operator\NewWithBracesFixer`:                 {successor: `PhpCsFixer\Fixer\Operator\NewWithParenthesesFixer`},
	`PhpCsFixer\Fixer\Whitespace\CompactNullableTypehintFixer`:     {successor: `PhpCsFixer\Fixer\Whitespace\CompactNullableTypeDeclarationFixer`},
	`PhpCsFixer\Fixer\Whitespace\NoSpacesInsideParenthesisFixer`:   {successor: `PhpCsFixer\Fixer\Whitespace\SpacesInsideParenthesesFixer`},
	`PhpCsFixer\Fixer\ArrayNotation\NoTrailingCommaInSinglelineArrayFixer`: {
		successor: `PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`,
		options:   fixedOptions(map[string]any{"elements": []string{"array"}}),
	},
	`PhpCsFixer\Fixer\ControlStructure\NoTrailingCommaInListCallFixer`: {
		successor: `PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`,
		options:   fixedOptions(map[string]any{"elements": []string{"array_destructuring"}}),
	},
	`PhpCsFixer\Fixer\FunctionNotation\NoTrailingCommaInSinglelineFunctionCallFixer`: {
		successor: `PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`,
		options:   fixedOptions(map[string]any{"elements": []string{"arguments", "array_destructuring"}}),
	},
	`PhpCsFixer\Fixer\FunctionNotation\FunctionTypehintSpaceFixer`: {
		successor: `PhpCsFixer\Fixer\Whitespace\TypeDeclarationSpacesFixer`,
		options:   fixedOptions(map[string]any{"elements": []string{"function"}}),
	},
	`PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAfterConstructFixer`: {
		successor: `PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`,
		options:   singleSpaceAfterConstructOptions,
	},
	`PhpCsFixer\Fixer\StringNotation\EscapeImplicitBackslashesFixer`: {
		successor: `PhpCsFixer\Fixer\StringNotation\StringImplicitBackslashesFixer`,
		options:   escapeImplicitBackslashesOptions,
	},
}

// singleSpaceAfterConstructDefault is SingleSpaceAfterConstructFixer's default
// `constructs`: every construct it knows but type_colon.
var singleSpaceAfterConstructDefault = []string{
	"abstract", "as", "attribute", "break", "case", "catch", "class", "clone", "comment",
	"const", "const_import", "continue", "do", "echo", "else", "elseif", "enum", "extends",
	"final", "finally", "for", "foreach", "function", "function_import", "global", "goto",
	"if", "implements", "include", "include_once", "instanceof", "insteadof", "interface",
	"match", "named_argument", "namespace", "new", "open_tag_with_echo", "php_doc",
	"php_open", "print", "private", "protected", "public", "readonly", "require",
	"require_once", "return", "static", "switch", "throw", "trait", "try", "use",
	"use_lambda", "use_trait", "var", "while", "yield", "yield_from",
}

// singleSpaceAfterConstructOptions ports SingleSpaceAfterConstructFixer's
// configurePostNormalisation.
func singleSpaceAfterConstructOptions(deprecated map[string]any) map[string]any {
	constructs, ok := deprecated["constructs"]
	if !ok {
		constructs = singleSpaceAfterConstructDefault
	}
	return map[string]any{
		"constructs_contain_a_single_space":     []string{"yield_from"},
		"constructs_preceded_by_a_single_space": []string{},
		"constructs_followed_by_a_single_space": constructs,
	}
}

// escapeImplicitBackslashesOptions ports EscapeImplicitBackslashesFixer's
// configurePostNormalisation: each enabled option escapes, the others ignore.
func escapeImplicitBackslashesOptions(deprecated map[string]any) map[string]any {
	mode := func(option string, enabled bool) string {
		if v, ok := deprecated[option].(bool); ok {
			enabled = v
		}
		if enabled {
			return "escape"
		}
		return "ignore"
	}
	return map[string]any{
		"single_quoted": mode("single_quoted", false),
		"double_quoted": mode("double_quoted", true),
		"heredoc":       mode("heredoc_syntax", true),
	}
}

// DeprecatedOptions maps the options of a deprecated class to its successor's.
// It reports false when no mapping is needed.
func DeprecatedOptions(class string, options map[string]any) (map[string]any, bool) {
	alias, ok := deprecatedAliases[class]
	if !ok || alias.options == nil {
		return nil, false
	}
	return alias.options(options), true
}

// DeprecatedNames returns the deprecated classes ByName resolves, sorted.
func DeprecatedNames() []string {
	names := make([]string, 0, len(deprecatedAliases))
	for name := range deprecatedAliases {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ByName returns the fixer whose Name matches, if any, configured with
// PHP-CS-Fixer's defaults like every fixer All returns. A deprecated class
// resolves to its successor, configured as the deprecated fixer configures it
// by default.
func ByName(name string) (fixer.Fixer, bool) {
	registryOnce.Do(buildRegistry)
	alias, deprecated := deprecatedAliases[name]
	if !deprecated {
		f, ok := reg.byName[name]
		return f, ok
	}
	f, ok := reg.byName[alias.successor]
	if !ok || alias.options == nil {
		return f, ok
	}
	configurable, ok := f.(fixer.ConfigurableFixer)
	if !ok {
		return nil, false
	}
	return configurable.WithConfig(alias.options(nil)), true
}
