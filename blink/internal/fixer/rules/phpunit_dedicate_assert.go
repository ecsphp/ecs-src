package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDedicateAssertFixer.php
//
// PhpUnitDedicateAssert turns assertions wrapping a native predicate into the
// dedicated PHPUnit assertion: assertTrue(is_null($x)) -> assertNull($x),
// assertTrue(empty($x)) -> assertEmpty($x), assertSame(count($x), ...) ->
// assertCount(...), assertTrue($x instanceof Foo) -> assertInstanceOf(...), and
// the generic is_* predicates to assertInternalType('type', ...). The default
// target (newest) enables every mapping. Risky: assumes PHPUnit's own methods.
type PhpUnitDedicateAssert struct{}

// dedicateFix is one FIX_MAP entry: the dedicated assertions for the positive
// and negative forms (negative "" means there is none), the required argument
// count (0 means 1) and whether the two arguments are swapped.
type dedicateFix struct {
	positive string
	negative string
	argCount int
	swap     bool
}

// dedicateFixMap ports the array entries of FIX_MAP (the non-generic ones).
var dedicateFixMap = map[string]dedicateFix{
	"array_key_exists": {positive: "assertArrayHasKey", negative: "assertArrayNotHasKey", argCount: 2},
	"empty":            {positive: "assertEmpty", negative: "assertNotEmpty"},
	"file_exists":      {positive: "assertFileExists", negative: "assertFileNotExists"},
	"is_dir":           {positive: "assertDirectoryExists", negative: "assertDirectoryNotExists"},
	"is_infinite":      {positive: "assertInfinite", negative: "assertFinite"},
	"is_nan":           {positive: "assertNan", negative: ""},
	"is_null":          {positive: "assertNull", negative: "assertNotNull"},
	"is_readable":      {positive: "assertIsReadable", negative: "assertNotIsReadable"},
	"is_writable":      {positive: "assertIsWritable", negative: "assertNotIsWritable"},
	"str_contains":     {positive: "assertStringContainsString", negative: "assertStringNotContainsString", argCount: 2, swap: true},
	"str_ends_with":    {positive: "assertStringEndsWith", negative: "assertStringEndsNotWith", argCount: 2, swap: true},
	"str_starts_with":  {positive: "assertStringStartsWith", negative: "assertStringStartsNotWith", argCount: 2, swap: true},
}

// dedicateFunctions is $this->functions for the default target (newest): every
// predicate the fixer rewrites. is_resource is deliberately absent upstream.
var dedicateFunctions = map[string]bool{
	"array_key_exists": true,
	"file_exists":      true,
	"is_null":          true,
	"str_ends_with":    true,
	"str_starts_with":  true,
	"empty":            true,
	"is_array":         true,
	"is_bool":          true,
	"is_boolean":       true,
	"is_callable":      true,
	"is_double":        true,
	"is_float":         true,
	"is_int":           true,
	"is_integer":       true,
	"is_long":          true,
	"is_numeric":       true,
	"is_object":        true,
	"is_real":          true,
	"is_scalar":        true,
	"is_string":        true,
	"is_infinite":      true,
	"is_nan":           true,
	"is_dir":           true,
	"is_readable":      true,
	"is_writable":      true,
	"str_contains":     true,
}

func (PhpUnitDedicateAssert) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDedicateAssertFixer`
}

func (PhpUnitDedicateAssert) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDedicateAssertFixer.php"
}

// WithConfig accepts the target option; the default (newest) enables every
// mapping, so a lower target is the only thing it could narrow. Kept as a no-op:
// blink models PHP-CS-Fixer's default configuration.
func (f PhpUnitDedicateAssert) WithConfig(map[string]any) fixer.Fixer {
	return f
}

func (PhpUnitDedicateAssert) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		for _, call := range puPreviousAssertCalls(s, cls[0], cls[1]) {
			if dedicateAssertCall(s, call) {
				changed = true
			}
		}
	}
	return changed
}

func dedicateAssertCall(s *tokens.Stream, call puAssertCall) bool {
	switch call.lowered {
	case "asserttrue", "assertfalse":
		return fixAssertTrueFalse(s, call)
	case "assertsame", "assertnotsame", "assertequals", "assertnotequals":
		return fixAssertSameEquals(s, call)
	}
	return false
}

// assertBraces returns the "(" and matching ")" of the assert call at index.
func assertBraces(s *tokens.Stream, index int) (open, close int, ok bool) {
	open = sigNext(s, index)
	if !isPunctVal(s, open, "(") {
		return 0, 0, false
	}
	close = s.MatchForward(open)
	if close < 0 {
		return 0, 0, false
	}
	return open, close, true
}

func fixAssertTrueFalse(s *tokens.Stream, call puAssertCall) bool {
	openBraceIndex, _, ok := assertBraces(s, call.index)
	if !ok {
		return false
	}
	testNSIndex := -1
	testIndex := sigNext(s, openBraceIndex)
	if testIndex < 0 {
		return false
	}
	t := s.At(testIndex)
	isEmptyOrString := t.Kind == token.Ident || (t.Kind == token.Keyword && strings.EqualFold(t.Value, "empty"))
	if !isEmptyOrString {
		if fixAssertTrueFalseInstanceof(s, call, testIndex) {
			return true
		}
		if !isPunctVal(s, testIndex, "\\") { // T_NS_SEPARATOR
			return false
		}
		testNSIndex = testIndex
		testIndex = sigNext(s, testIndex)
		if testIndex < 0 {
			return false
		}
	}

	testOpenIndex := sigNext(s, testIndex)
	if !isPunctVal(s, testOpenIndex, "(") {
		return false
	}
	testCloseIndex := s.MatchForward(testOpenIndex)
	if testCloseIndex < 0 {
		return false
	}
	assertCallCloseIndex := sigNext(s, testCloseIndex)
	if !isPunctVal(s, assertCallCloseIndex, ")") && !isPunctVal(s, assertCallCloseIndex, ",") {
		return false
	}

	content := strings.ToLower(s.At(testIndex).Value)
	if !dedicateFunctions[content] {
		return false
	}

	arguments := aliasArguments(s, testOpenIndex, testCloseIndex)
	isPositive := call.lowered == "asserttrue"

	if fixDetails, ok := dedicateFixMap[content]; ok {
		expectedCount := fixDetails.argCount
		if expectedCount == 0 {
			expectedCount = 1
		}
		if expectedCount != len(arguments) {
			return false
		}
		name := fixDetails.positive
		if !isPositive {
			name = fixDetails.negative
		}
		if name == "" {
			return false
		}
		// comment-interleaved swaps are left to the full PHP-CS-Fixer pass
		if fixDetails.swap && !spanClean(s, testOpenIndex, testCloseIndex) {
			return false
		}
		s.SetValue(call.index, name)
		removeFunctionCall(s, testNSIndex, testIndex, testOpenIndex, testCloseIndex)
		if fixDetails.swap {
			swapArguments(s, arguments)
		}
		return true
	}

	if len(arguments) != 1 {
		return false
	}

	typ := content[3:] // drop "is_"
	name := "assertInternalType"
	if !isPositive {
		name = "assertNotInternalType"
	}
	s.SetValue(call.index, name)
	s.Set(testIndex, token.Token{Kind: token.String, Value: "'" + typ + "'"})
	s.Set(testOpenIndex, token.Token{Kind: token.Punct, Value: ","})

	clearTokenAndMergeWS(s, testCloseIndex)
	commaIndex := sigPrev(s, testCloseIndex)
	if isPunctVal(s, commaIndex, ",") {
		puRemoveTrailingWhitespace(s, commaIndex)
		clearTokenAt(s, commaIndex)
	}
	if testOpenIndex+1 < s.Len() && s.At(testOpenIndex+1).Kind != token.Whitespace {
		s.InsertAt(testOpenIndex+1, token.Token{Kind: token.Whitespace, Value: " "})
	}
	if testNSIndex >= 0 {
		clearTokenAndMergeWS(s, testNSIndex)
	}
	return true
}

// fixAssertTrueFalseInstanceof ports the instanceof branch. It returns true once
// the `$var instanceof <name>` shape is recognised (so the caller stops), and
// rewrites only the clean, comment-free case.
func fixAssertTrueFalseInstanceof(s *tokens.Stream, call puAssertCall, testIndex int) bool {
	isPositiveAssertion := call.lowered == "asserttrue"

	variableIndex := testIndex
	isPositiveCondition := true
	if isPunctVal(s, testIndex, "!") {
		variableIndex = sigNext(s, testIndex)
		isPositiveCondition = false
	}
	if variableIndex < 0 || s.At(variableIndex).Kind != token.Variable {
		return false
	}
	instanceOfIndex := sigNext(s, variableIndex)
	if !kwIs(s.At(instanceOfIndex), "instanceof") {
		return false
	}

	classEndIndex := sigNext(s, instanceOfIndex)
	if classEndIndex < 0 {
		return false
	}
	var classPart []token.Token
	for {
		classPart = append(classPart, s.At(classEndIndex))
		classEndIndex = sigNext(s, classEndIndex)
		if classEndIndex < 0 || !dedicateIsClassPart(s.At(classEndIndex)) {
			break
		}
	}

	if !isPunctVal(s, classEndIndex, ",") && !isPunctVal(s, classEndIndex, ")") {
		return true // recognised as instanceof, but not a shape we fix
	}
	// comment-interleaved instanceof is left to the full PHP-CS-Fixer pass
	if !spanClean(s, testIndex, classEndIndex-1) {
		return true
	}

	isInstanceOfVar := classPart[0].Kind == token.Variable

	replacement := make([]token.Token, 0, len(classPart)+4)
	replacement = append(replacement, classPart...)
	if !isInstanceOfVar {
		replacement = append(replacement,
			token.Token{Kind: token.Punct, Value: "::"},
			token.Token{Kind: token.Ident, Value: "class"},
		)
	}
	replacement = append(replacement,
		token.Token{Kind: token.Punct, Value: ","},
		token.Token{Kind: token.Whitespace, Value: " "},
		s.At(variableIndex),
	)

	name := "assertNotInstanceOf"
	if isPositiveAssertion == isPositiveCondition {
		name = "assertInstanceOf"
	}

	s.ReplaceRange(testIndex, classEndIndex-1, replacement)
	s.SetValue(call.index, name)
	return true
}

// dedicateIsClassPart reports whether a token continues a class name
// (T_STRING, T_NS_SEPARATOR or T_VARIABLE).
func dedicateIsClassPart(t token.Token) bool {
	return t.Kind == token.Ident || t.Kind == token.Variable || (t.Kind == token.Punct && t.Value == "\\")
}

func fixAssertSameEquals(s *tokens.Stream, call puAssertCall) bool {
	openBraceIndex, _, ok := assertBraces(s, call.index)
	if !ok {
		return false
	}

	expectedIndex := sigNext(s, openBraceIndex)
	if expectedIndex < 0 {
		return false
	}
	if s.At(expectedIndex).Kind == token.Variable {
		if !isPunctVal(s, sigNext(s, expectedIndex), ",") {
			return false
		}
	} else if s.At(expectedIndex).Kind != token.Number || !lexer.IsIntegerLiteral(s.At(expectedIndex).Value) {
		return false
	}

	commaIndex := sigNext(s, expectedIndex)
	if !isPunctVal(s, commaIndex, ",") {
		return false
	}

	countCallIndex := sigNext(s, commaIndex)
	defaultNSIndex := -1
	if isPunctVal(s, countCallIndex, "\\") {
		defaultNSIndex = countCallIndex
		countCallIndex = sigNext(s, countCallIndex)
	}
	if countCallIndex < 0 || s.At(countCallIndex).Kind != token.Ident {
		return false
	}
	lowerContent := strings.ToLower(s.At(countCallIndex).Value)
	if lowerContent != "count" && lowerContent != "sizeof" {
		return false
	}

	countOpenIndex := sigNext(s, countCallIndex)
	if !isPunctVal(s, countOpenIndex, "(") {
		return false
	}
	countCloseIndex := s.MatchForward(countOpenIndex)
	if countCloseIndex < 0 {
		return false
	}
	afterCloseIndex := sigNext(s, countCloseIndex)
	if !isPunctVal(s, afterCloseIndex, ")") && !isPunctVal(s, afterCloseIndex, ",") {
		return false
	}

	removeFunctionCall(s, defaultNSIndex, countCallIndex, countOpenIndex, countCloseIndex)

	name := "assertCount"
	if strings.Contains(call.lowered[6:], "not") {
		name = "assertNotCount"
	}
	s.SetValue(call.index, name)
	return true
}

// removeFunctionCall ports PhpUnitDedicateAssertFixer::removeFunctionCall: clears
// the call name (and its optional "\" prefix), its "(" and its ")", dropping a
// trailing comma inside.
func removeFunctionCall(s *tokens.Stream, callNSIndex, callIndex, openIndex, closeIndex int) {
	clearTokenAndMergeWS(s, callIndex)
	if callNSIndex >= 0 {
		clearTokenAndMergeWS(s, callNSIndex)
	}
	clearTokenAndMergeWS(s, openIndex)
	commaIndex := sigPrev(s, closeIndex)
	if isPunctVal(s, commaIndex, ",") {
		puRemoveTrailingWhitespace(s, commaIndex)
		clearTokenAt(s, commaIndex)
	}
	clearTokenAndMergeWS(s, closeIndex)
}

// swapArguments ports PhpUnitDedicateAssertFixer::swapArguments for two
// arguments: the first argument gains a leading space when it moves into second
// position, the second loses its leading space when it moves into first.
func swapArguments(s *tokens.Stream, arguments [][2]int) {
	if len(arguments) != 2 {
		return
	}
	first, second := arguments[0], arguments[1]

	firstTokens := copyRange(s, first[0], first[1])
	secondTokens := copyRange(s, second[0], second[1])

	newSecond := firstTokens
	if len(newSecond) == 0 || newSecond[0].Kind != token.Whitespace {
		newSecond = append([]token.Token{{Kind: token.Whitespace, Value: " "}}, newSecond...)
	}
	newFirst := secondTokens
	if len(newFirst) > 0 && newFirst[0].Kind == token.Whitespace {
		newFirst = newFirst[1:]
	}

	// replace the higher range first so the lower range indices stay valid
	s.ReplaceRange(second[0], second[1], newSecond)
	s.ReplaceRange(first[0], first[1], newFirst)
}

func copyRange(s *tokens.Stream, start, end int) []token.Token {
	out := make([]token.Token, 0, end-start+1)
	for i := start; i <= end && i < s.Len(); i++ {
		out = append(out, s.At(i))
	}
	return out
}

// puRemoveTrailingWhitespace clears the spaces and tabs at the start of the
// whitespace token following index, mirroring Tokens::removeTrailingWhitespace.
func puRemoveTrailingWhitespace(s *tokens.Stream, index int) {
	next := index + 1
	if next >= s.Len() || s.At(next).Kind != token.Whitespace {
		return
	}
	s.SetValue(next, strings.TrimLeft(s.At(next).Value, " \t"))
}
