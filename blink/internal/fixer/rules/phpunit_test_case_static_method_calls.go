package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestCaseStaticMethodCallsFixer.php
//
// PhpUnitTestCaseStaticMethodCalls normalizes how PHPUnit's own assert/helper
// methods are referenced inside a test class, so every call uses the same form:
// static:: (the default), self:: or $this->. Risky: assumes the methods are
// PHPUnit's own. Lambdas and nested/anonymous classes are left alone.
type PhpUnitTestCaseStaticMethodCalls struct {
	callType string // "static" (default), "self" or "this"
}

func (PhpUnitTestCaseStaticMethodCalls) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitTestCaseStaticMethodCallsFixer`
}

func (PhpUnitTestCaseStaticMethodCalls) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestCaseStaticMethodCallsFixer.php"
}

// WithConfig reads the call_type option (self/static/this); any other value
// keeps the default, static.
func (f PhpUnitTestCaseStaticMethodCalls) WithConfig(config map[string]any) fixer.Fixer {
	if c, ok := config["call_type"].(string); ok {
		switch c {
		case "self", "static", "this":
			f.callType = c
		}
	}
	return f
}

func (f PhpUnitTestCaseStaticMethodCalls) Fix(s *tokens.Stream) bool {
	callType := f.callType
	if callType == "" {
		callType = "static"
	}
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		if staticMethodCallsClass(s, cls[0], cls[1], callType) {
			changed = true
		}
	}
	return changed
}

func staticMethodCallsClass(s *tokens.Stream, startIndex, endIndex int, callType string) bool {
	changed := false
	for index := startIndex; index < endIndex; index++ {
		// skip nested/anonymous classes
		if kwIs(s.At(index), "class") {
			index = puFindEndOfNextBlock(s, index, endIndex)
			continue
		}
		if kwIs(s.At(index), "function") {
			// skip lambdas, including their bodies
			if puIsLambda(s, index) {
				index = puFindEndOfNextBlock(s, index, endIndex)
				continue
			}
			// do not turn `self`/`static` into `$this` inside static methods
			if callType == "this" && puIsStaticMethod(s, index) {
				index = puFindEndOfNextBlock(s, index, endIndex)
				continue
			}
		}

		if s.At(index).Kind != token.Ident || !phpUnitStaticMethods[s.At(index).Value] {
			continue
		}

		nextIndex := sigNext(s, index)
		if nextIndex < 0 {
			break
		}
		if !isPunctVal(s, nextIndex, "(") {
			index = nextIndex
			continue
		}
		// first-class callable `name(...)` is left alone
		if isFirstClassCallableAt(s, index) {
			continue
		}

		operatorIndex := sigPrev(s, index)
		referenceIndex := sigPrev(s, operatorIndex)
		if !puIsTheSameClassCall(s, index) || puIsCallType(s.At(referenceIndex), callType) {
			continue
		}

		op, ref := puConversionTokens(callType)
		s.Set(operatorIndex, op)
		s.Set(referenceIndex, ref)
		changed = true
	}
	return changed
}

// puConversionTokens returns the operator and reference tokens for a call type.
func puConversionTokens(callType string) (token.Token, token.Token) {
	switch callType {
	case "this":
		return token.Token{Kind: token.Punct, Value: "->"}, token.Token{Kind: token.Variable, Value: "$this"}
	case "self":
		return token.Token{Kind: token.Punct, Value: "::"}, token.Token{Kind: token.Ident, Value: "self"}
	default: // static
		return token.Token{Kind: token.Punct, Value: "::"}, token.Token{Kind: token.Keyword, Value: "static"}
	}
}

// puIsCallType reports whether the reference token already uses the target form.
func puIsCallType(t token.Token, callType string) bool {
	switch callType {
	case "this":
		return t.Kind == token.Variable && strings.EqualFold(t.Value, "$this")
	case "self":
		return t.Kind == token.Ident && strings.EqualFold(t.Value, "self")
	default: // static
		return t.Kind == token.Keyword && strings.EqualFold(t.Value, "static")
	}
}

// puIsLambda mirrors TokensAnalyzer::isLambda for the "function" at index: the
// next meaningful token (past a return-reference "&") is "(".
func puIsLambda(s *tokens.Stream, index int) bool {
	n := sigNext(s, index)
	if isPunctVal(s, n, "&") {
		n = sigNext(s, n)
	}
	return isPunctVal(s, n, "(")
}

// puIsStaticMethod reports whether the method at the "function" index carries a
// `static` modifier.
func puIsStaticMethod(s *tokens.Stream, index int) bool {
	p := getPrevNonWhitespace(s, index)
	for p >= 0 {
		t := s.At(p)
		if t.Kind == token.AttributeEnd {
			p = getPrevNonWhitespace(s, s.AttributeStartOf(p))
			continue
		}
		if t.Kind == token.Comment {
			p = getPrevNonWhitespace(s, p)
			continue
		}
		if t.Kind == token.Keyword && puModifierKw[strings.ToLower(t.Value)] {
			if strings.EqualFold(t.Value, "static") {
				return true
			}
			p = getPrevNonWhitespace(s, p)
			continue
		}
		break
	}
	return false
}

// puFindEndOfNextBlock mirrors the fixer's findEndOfNextBlock: the matching "}"
// of the next "{", or the next ";". endIndex bounds a malformed scan.
func puFindEndOfNextBlock(s *tokens.Stream, index, endIndex int) int {
	nextIndex := nextPunctOfKind(s, index, ";", "{")
	if nextIndex < 0 {
		return endIndex
	}
	if isPunctVal(s, nextIndex, "{") {
		if end := s.MatchForward(nextIndex); end > nextIndex {
			return end
		}
		return endIndex
	}
	return nextIndex
}
