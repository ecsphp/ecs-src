package rules

import (
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
