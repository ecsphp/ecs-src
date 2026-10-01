package rules

import (
	"sort"
	"strings"

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
		MethodChainingNewline{},
		MethodChainingIndentation{},
		ArrayListItemNewline{},
		StandaloneLineInMultilineArray{},
		ArrayIndentation{},
		NoExtraBlankLines{}, // after import removal, which can leave extra blanks
	}
}

// All returns every built-in fixer in execution order. FullOpeningTag runs first
// (normalize the tag) and NoClosingTag last (trailing tag/EOF cleanup).
func All() []fixer.Fixer {
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
	sort.SliceStable(all, func(a, b int) bool {
		pa, pb := fixerPriorityOf(all[a]), fixerPriorityOf(all[b])
		if pa != pb {
			return pa > pb
		}
		return fixerSortName(all[a]) < fixerSortName(all[b])
	})
	return all
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
	idx := map[string]int{}
	for i, f := range All() {
		idx[f.Name()] = i
	}
	const last = 1 << 30
	pos := func(f fixer.Fixer) int {
		if i, ok := idx[f.Name()]; ok {
			return i
		}
		return last
	}
	sort.SliceStable(fixers, func(a, b int) bool {
		return pos(fixers[a]) < pos(fixers[b])
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

// ByName returns the fixer whose Name matches, if any.
func ByName(name string) (fixer.Fixer, bool) {
	for _, f := range All() {
		if f.Name() == name {
			return f, true
		}
	}
	return nil, false
}
