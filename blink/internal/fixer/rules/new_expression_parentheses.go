package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NewExpressionParenthesesFixer.php
//
// NewExpressionParentheses controls whether a PHP 8.4 "new" expression with a
// further call is wrapped in parentheses. With use_parentheses off (default) it
// unwraps "(new Foo())->bar()" to "new Foo()->bar()"; with it on it wraps the
// bare form back up.
type NewExpressionParentheses struct {
	UseParentheses bool
}

func (f NewExpressionParentheses) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["use_parentheses"].(bool); ok {
		f.UseParentheses = v
	}
	return f
}

func (NewExpressionParentheses) Name() string {
	return `PhpCsFixer\Fixer\Operator\NewExpressionParenthesesFixer`
}

func (NewExpressionParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NewExpressionParenthesesFixer.php"
}

func (f NewExpressionParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 3; index > 0; index-- {
		if !kwIs(s.At(index), "new") {
			continue
		}

		classStart := aliasNextMeaningful(s, index)
		if classStart < 0 {
			return changed
		}

		// anonymous class
		if kwIs(s.At(classStart), "class") {
			nextIndex := aliasNextMeaningful(s, classStart)
			if isPunctVal(s, nextIndex, "(") {
				nextIndex = s.MatchForward(nextIndex)
			} else {
				nextIndex = classStart
			}
			bodyStart := nextPunctOfKind(s, nextIndex, "{")
			if bodyStart < 0 {
				continue
			}
			bodyEnd := s.MatchForward(bodyStart)
			if bodyEnd < 0 {
				continue
			}
			if f.applyDirection(s, index, bodyEnd) {
				changed = true
			}
			continue
		}

		// named class or expression
		classEnd := nepFindClassEndIndex(s, classStart)
		if classEnd < 0 {
			continue
		}
		nextIndex := aliasNextMeaningful(s, classEnd)
		if !isPunctVal(s, nextIndex, "(") {
			continue
		}
		argsEnd := s.MatchForward(nextIndex)
		if argsEnd < 0 {
			continue
		}
		if f.applyDirection(s, index, argsEnd) {
			changed = true
		}
	}
	return changed
}

func (f NewExpressionParentheses) applyDirection(s *tokens.Stream, exprStart, exprEnd int) bool {
	if f.UseParentheses {
		return nepEnsureWrapped(s, exprStart, exprEnd)
	}
	return nepEnsureNotWrapped(s, exprStart, exprEnd)
}

// nepEnsureWrapped wraps the new-expression in instantiation parentheses when a
// "->"/"?->"/"::" follows it and it is not already wrapped.
func nepEnsureWrapped(s *tokens.Stream, exprStart, exprEnd int) bool {
	prevIndex := aliasPrevMeaningful(s, exprStart)
	nextIndex := aliasNextMeaningful(s, exprEnd)

	if nepIsWrappingPair(s, prevIndex, nextIndex) {
		return false
	}
	if !nepFollowedByCall(s, nextIndex) {
		return false
	}

	s.InsertAt(exprEnd+1, token.Token{Kind: token.Punct, Value: ")"})
	s.InsertAt(exprStart, token.Token{Kind: token.Punct, Value: "("})
	return true
}

// nepEnsureNotWrapped drops the instantiation parentheses around the
// new-expression when it is wrapped and a "->"/"?->"/"::" follows.
func nepEnsureNotWrapped(s *tokens.Stream, exprStart, exprEnd int) bool {
	prevIndex := aliasPrevMeaningful(s, exprStart)
	nextIndex := aliasNextMeaningful(s, exprEnd)

	if !nepIsWrappingPair(s, prevIndex, nextIndex) {
		return false
	}
	operatorIndex := aliasNextMeaningful(s, nextIndex)
	if !nepFollowedByCall(s, operatorIndex) {
		return false
	}

	clearTokenAndMergeWS(s, prevIndex)
	clearTokenAndMergeWS(s, nextIndex)
	return true
}

// nepIsWrappingPair reports whether openIndex is the instantiation "(" directly
// before "new" and closeIndex is its matching ")".
func nepIsWrappingPair(s *tokens.Stream, openIndex, closeIndex int) bool {
	return isPunctVal(s, openIndex, "(") && isPunctVal(s, closeIndex, ")") &&
		s.MatchForward(openIndex) == closeIndex
}

// nepFollowedByCall reports whether the token at i is "->", "?->" or "::".
func nepFollowedByCall(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "->") || isPunctVal(s, i, "?->") || isPunctVal(s, i, "::")
}

// nepFindClassEndIndex ports NewExpressionParenthesesFixer::findClassEndIndex: it
// walks the class reference starting at index and returns the index just before
// the token that follows it, or -1 when index does not start a class reference.
func nepFindClassEndIndex(s *tokens.Stream, index int) int {
	// (expression) class name
	if isPunctVal(s, index, "(") {
		return s.MatchForward(index)
	}
	if !nepIsClassRefToken(s, index) {
		return -1
	}
	for nepIsClassRefToken(s, index) {
		if blockEnd := aliasBlockEnd(s, index); blockEnd >= 0 {
			index = blockEnd
		}
		index = aliasNextMeaningful(s, index)
		if index < 0 {
			return -1
		}
	}
	return index - 1
}

// nepIsClassRefToken reports the tokens a class reference is built from: names,
// the namespace separator, variables, index brackets, the dynamic-variable brace
// and the member operators.
func nepIsClassRefToken(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	switch t.Kind {
	case token.Ident, token.Variable:
		return true
	case token.Punct:
		switch t.Value {
		case `\`, "[", "->", "?->", "::":
			return true
		case "{":
			// CT::T_DYNAMIC_VAR_BRACE_OPEN "${"
			p := aliasPrevMeaningful(s, i)
			return p >= 0 && s.At(p).Kind == token.Variable && s.At(p).Value == "$"
		}
	}
	return false
}
