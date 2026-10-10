package rules

import (
	"cmp"
	"slices"
	"strings"
	"sync"

	"blink/internal/fixer"
)

// SpacingFixers are the whitespace/operator spacing rules, safest first. This is
// the ordered "spaces" set that gradual levels slice.
func SpacingFixers() []fixer.Fixer {
	return []fixer.Fixer{
		NoLeadingNamespaceWhitespace{},
		NoSinglelineWhitespaceBeforeSemicolons{},
		MultilineWhitespaceBeforeSemicolons{},
		NoWhitespaceInBlankLine{},
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
		SimplifiedNullReturn{},
		FunctionToConstant{},
		SelfAccessor{},
		RemoveUselessDefaultComment{},
		ProtectedToPrivate{},
		AddMissingParamName{},
		NoAliasLanguageConstructCall{},
		NoUnreachableDefaultArgumentValue{},
	}
}

// PhpdocFixers normalize doc comments.
func PhpdocFixers() []fixer.Fixer {
	return []fixer.Fixer{
		DoctrineAnnotationSpaces{},
		DoctrineAnnotationArrayAssignment{},
		DoctrineAnnotationIndentation{},
		NoSuperfluousPhpdocTags{},
		PhpdocNoUselessInheritdoc{},
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
		StandaloneLinePromotedProperty{},
		StandaloneLinePlainConstructorParam{},
		StandaloneLineRequiredParam{},
		StandaloneLineSymfonyAttributeParam{},
		NoBreakComment{},
		ReturnTypeDeclaration{},
		NewWithParentheses{},
		FunctionDeclaration{},
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
		OperatorLinebreak{},
		PhpdocSeparation{},
		PhpdocToComment{},
		PhpdocAlign{},
		ParamReturnAndVarTagMalforms{},
		GeneralPhpdocTagRename{},
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
		SingleBlankLineBeforeNamespace{},
		BlankLineAfterNamespace{},
		NoUnusedImports{},
		FullyQualifiedStrictTypes{},
		SingleImportPerStatement{},
		OrderedImports{},
		BlankLineBetweenImportGroups{},
		SingleLineAfterImports{},
		NoBlankLineBetweenImports{},
		SpaceAfterCommaHereNowDoc{},
		ArrayOpenerAndCloserNewline{},
		NoBlankLinesAfterClassOpening{},
		StatementIndentation{},
		MethodChainingIndentation{},
		ArrayListItemNewline{},
		StandaloneLineInMultilineArray{},
		ArrayIndentation{},
		NoExtraBlankLines{}, // after import removal, which can leave extra blanks
	}
}

// All returns every built-in fixer in execution order. FullOpeningTag runs first
// (normalize the tag) and NoClosingTag last (trailing tag/EOF cleanup).
var (
	registryOnce sync.Once
	sortedFixers []fixer.Fixer          // canonical execution order
	fixerByName  map[string]fixer.Fixer // Name -> fixer
	canonicalPos map[string]int         // Name -> index in sortedFixers
)

func buildRegistry() {
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
		FixParamNameTypo{})
	all = append(all, NoClosingTag{})
	// run in PHP-CS-Fixer execution order: priority descending, then by rule name
	// ascending - exactly how PHP-CS-Fixer's Utils::sortFixers breaks ties
	slices.SortStableFunc(all, func(a, b fixer.Fixer) int {
		if pa, pb := fixerPriorityOf(a), fixerPriorityOf(b); pa != pb {
			return cmp.Compare(pb, pa) // priority descending
		}
		return cmp.Compare(fixerSortName(a), fixerSortName(b))
	})

	sortedFixers = all
	fixerByName = make(map[string]fixer.Fixer, len(all))
	canonicalPos = make(map[string]int, len(all))
	for i, f := range all {
		fixerByName[f.Name()] = f
		canonicalPos[f.Name()] = i
	}
}

// All returns every fixer in PHP-CS-Fixer execution order. The registry is built
// once; the returned slice is a fresh copy the caller may append to or reorder.
func All() []fixer.Fixer {
	registryOnce.Do(buildRegistry)
	return slices.Clone(sortedFixers)
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
		if i, ok := canonicalPos[f.Name()]; ok {
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

// deprecatedAliases maps a deprecated PHP-CS-Fixer class to its successor, so a
// dumped ECS config that still references the old name resolves to the fixer.
var deprecatedAliases = map[string]string{
	`PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`: `PhpCsFixer\Fixer\ControlStructure\NoUnneededBracesFixer`,
}

// ByName returns the fixer whose Name matches, if any.
func ByName(name string) (fixer.Fixer, bool) {
	registryOnce.Do(buildRegistry)
	if canonical, ok := deprecatedAliases[name]; ok {
		name = canonical
	}
	f, ok := fixerByName[name]
	return f, ok
}
