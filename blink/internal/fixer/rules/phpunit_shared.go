package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Helpers shared by the AbstractPhpUnitFixer ports (PhpUnitConstruct,
// PhpUnitFqcnAnnotation, PhpUnitMockShortWillReturn, PhpUnitTestAnnotation).

// puClassesLastFirst mirrors PhpUnitTestCaseAnalyzer::findPhpUnitClasses: the
// [open brace, close brace] of every PHPUnit test class, last class first, so a
// fixer that inserts tokens never shifts a class it has yet to visit.
func puClassesLastFirst(s *tokens.Stream) [][2]int {
	classes := phpUnitTestClasses(s)
	for i, j := 0, len(classes)-1; i < j; i, j = i+1, j-1 {
		classes[i], classes[j] = classes[j], classes[i]
	}
	return classes
}

// puIsObjectOperator mirrors Token::isObjectOperator ("->" or "?->").
func puIsObjectOperator(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "->") || isPunctVal(s, i, "?->")
}

// puIsTheSameClassCall mirrors FunctionsAnalyzer::isTheSameClassCall: the name
// at index is called as $this->name(, self::name( or static::name( (the
// receiver compared case-insensitively).
func puIsTheSameClassCall(s *tokens.Stream, index int) bool {
	if index < 0 || index >= s.Len() {
		return false
	}
	op := sigPrev(s, index)
	if op < 0 || !puIsObjectOperator(s, op) && !isPunctVal(s, op, "::") {
		return false
	}
	ref := sigPrev(s, op)
	if ref < 0 {
		return false
	}
	t := s.At(ref)
	switch {
	case t.Kind == token.Variable && strings.EqualFold(t.Value, "$this"):
	case t.Kind == token.Ident && strings.EqualFold(t.Value, "self"):
	case t.Kind == token.Keyword && strings.EqualFold(t.Value, "static"):
	default:
		return false
	}
	return isPunctVal(s, sigNext(s, index), "(")
}

// puIsNativeConstant mirrors Token::isNativeConstant: true, false or null in any
// case (blink lexes them as identifiers, PHP as T_STRING).
func puIsNativeConstant(t token.Token) bool {
	if t.Kind != token.Ident {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "true", "false", "null":
		return true
	}
	return false
}

// puAssertCall is one $this->/self::/static:: assert* call: the name-token
// index and the lowercased name.
type puAssertCall struct {
	index   int
	lowered string
}

// puPreviousAssertCalls mirrors AbstractPhpUnitFixer::getPreviousAssertCall: the
// assert* same-class calls in [startIndex, endIndex], yielded last first.
func puPreviousAssertCalls(s *tokens.Stream, startIndex, endIndex int) []puAssertCall {
	var out []puAssertCall
	for index := endIndex; index > startIndex; index-- {
		prev := puPrevIdent(s, index)
		if prev <= startIndex {
			break
		}
		index = prev
		lowered := strings.ToLower(s.At(index).Value)
		if !strings.HasPrefix(lowered, "assert") {
			continue
		}
		if !isPunctVal(s, sigNext(s, index), "(") {
			continue
		}
		if !puIsTheSameClassCall(s, index) {
			continue
		}
		out = append(out, puAssertCall{index: index, lowered: lowered})
	}
	return out
}

// phpUnitTargetFulfills mirrors PhpUnitTargetVersion::fulfills for the simple
// MAJOR.MINOR targets used by the PHPUnit fixers. An empty or "newest"
// candidate always fulfills.
func phpUnitTargetFulfills(candidate, target string) bool {
	if candidate == "" || candidate == "newest" {
		return true
	}
	cMajor, cMinor := phpUnitParseVersion(candidate)
	tMajor, tMinor := phpUnitParseVersion(target)
	if cMajor != tMajor {
		return cMajor > tMajor
	}
	return cMinor >= tMinor
}

func phpUnitParseVersion(v string) (int, int) {
	major, minor := 0, 0
	if before, after, ok := strings.Cut(v, "."); ok {
		major = atoiSafe(before)
		minor = atoiSafe(after)
	} else {
		major = atoiSafe(v)
	}
	return major, minor
}

func atoiSafe(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// puPrevIdent returns the index of the nearest Ident strictly before i, or -1,
// mirroring getPrevTokenOfKind with a [T_STRING] target.
func puPrevIdent(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind == token.Ident {
			return j
		}
	}
	return -1
}

// puNextNonWhitespace mirrors Tokens::getNextNonWhitespace (cleared placeholder
// tokens count as whitespace, as PHP's empty tokens do).
func puNextNonWhitespace(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

// Helpers for the AbstractPhpUnitFixer ports that add a class-level annotation
// (PhpUnitInternalClass, PhpUnitSizeClass, PhpUnitTestClassRequiresCovers),
// mirroring DocBlockAnnotationTrait.

// puClassKeywordBefore returns the index of the `class` keyword owning the body
// that opens at openBrace, mirroring getPrevTokenOfKind($startIndex, [[T_CLASS]]).
func puClassKeywordBefore(s *tokens.Stream, openBrace int) int {
	for j := openBrace - 1; j >= 0; j-- {
		if kwIs(s.At(j), "class") {
			return j
		}
	}
	return -1
}

// puClassModifiers mirrors TokensAnalyzer::getClassyModifiers: the final/abstract/
// readonly keywords in the modifier run before the class keyword.
func puClassModifiers(s *tokens.Stream, classIdx int) map[string]bool {
	mods := map[string]bool{}
	idx := classIdx
	for {
		p := sigPrev(s, idx)
		if p < 0 || s.At(p).Kind != token.Keyword {
			break
		}
		lo := strings.ToLower(s.At(p).Value)
		if lo != "final" && lo != "abstract" && lo != "readonly" {
			break
		}
		mods[lo] = true
		idx = p
	}
	return mods
}

// puEnsureClassDocBlockAnnotation mirrors
// DocBlockAnnotationTrait::ensureIsDocBlockWithAnnotation. preventingAttrShortNames
// are matched against the lowercased last segment of each attribute name before
// the class, standing in for upstream's fully-qualified attribute resolution.
func puEnsureClassDocBlockAnnotation(s *tokens.Stream, classIdx int, annotation string, preventingAnnotations, preventingAttrShortNames []string) bool {
	if puIsPreventedByAttribute(s, classIdx, preventingAttrShortNames) {
		return false
	}
	docBlockIndex := puAnnotationDocBlockIndex(s, classIdx)
	if docBlockIndex >= 0 && s.At(docBlockIndex).Kind == token.DocComment {
		return puUpdateDocBlockIfNeeded(s, docBlockIndex, annotation, preventingAnnotations)
	}
	ptaCreateDocBlock(s, docBlockIndex, annotation)
	return docBlockIndex >= 0
}

// puUpdateDocBlockIfNeeded mirrors DocBlockAnnotationTrait::updateDocBlockIfNeeded.
func puUpdateDocBlockIfNeeded(s *tokens.Stream, docBlockIndex int, annotation string, preventingAnnotations []string) bool {
	lines := splitDocLines(s.At(docBlockIndex).Value)
	if puDocHasAnnotationOfType(lines, preventingAnnotations...) {
		return false
	}
	if len(lines) == 1 && !puDocHasAnnotationOfType(lines, annotation) {
		lines = puMakeMultiLine(lines, detectIndent(s, puNextNonWhitespace(s, docBlockIndex)))
	}
	originalIndent := detectIndent(s, docBlockIndex)
	out := append([]string{}, lines[:len(lines)-1]...)
	out = append(out, originalIndent+" * @"+annotation+"\n")
	out = append(out, lines[len(lines)-1])
	s.SetValue(docBlockIndex, strings.Join(out, ""))
	return true
}

// puMakeMultiLine mirrors DocBlock::makeMultiLine for a single-line docblock.
func puMakeMultiLine(lines []string, indent string) []string {
	content := ptaSingleLineDocBlockEntry(lines[0])
	if content == "" {
		return []string{"/**\n", indent + " *\n", indent + " */"}
	}
	return []string{"/**\n", indent + " * " + content + "\n", indent + " */"}
}

// puDocHasAnnotationOfType mirrors DocBlock::getAnnotationsOfType being non-empty.
func puDocHasAnnotationOfType(lines []string, names ...string) bool {
	for _, ann := range docAnnotations(lines) {
		if slices.Contains(names, ann.name) {
			return true
		}
	}
	return false
}

// puIsPreventedByAttribute mirrors DocBlockAnnotationTrait::isPreventedByAttribute,
// matching the lowercased short name of each attribute before the class.
func puIsPreventedByAttribute(s *tokens.Stream, classIdx int, shortNames []string) bool {
	if len(shortNames) == 0 {
		return false
	}
	idx := classIdx
	for {
		idx = sigPrev(s, idx)
		if idx < 0 {
			return false
		}
		if t := s.At(idx); t.Kind == token.Keyword {
			if lo := strings.ToLower(t.Value); lo == "final" || lo == "readonly" {
				continue
			}
		}
		break
	}
	if s.At(idx).Kind != token.AttributeEnd {
		return false
	}
	start := s.AttributeStartOf(idx)
	if start < 0 {
		return false
	}
	for _, name := range puAttributeShortNames(attributeText(s, start, idx)) {
		if slices.Contains(shortNames, name) {
			return true
		}
	}
	return false
}

// puAttributeShortNames returns the lowercased last-segment name of each attribute
// in a folded "#[...]" attribute token.
func puAttributeShortNames(v string) []string {
	inner := strings.TrimSuffix(strings.TrimPrefix(v, "#["), "]")
	var out []string
	for part := range strings.SplitSeq(inner, ",") {
		part = strings.TrimSpace(part)
		if i := strings.IndexByte(part, '('); i >= 0 {
			part = part[:i]
		}
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.LastIndex(part, `\`); i >= 0 {
			part = part[i+1:]
		}
		out = append(out, strings.ToLower(part))
	}
	return out
}
