package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/GetClassToClassKeywordFixer.php
//
// GetClassToClassKeyword replaces get_class($object) with $object::class.
type GetClassToClassKeyword struct{}

func (GetClassToClassKeyword) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\GetClassToClassKeywordFixer`
}

func (GetClassToClassKeyword) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/GetClassToClassKeywordFixer.php"
}

// IsCandidate rules out a file without the trigger: only a "get_class" call is replaced.
func (GetClassToClassKeyword) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("get_class")
}

func (GetClassToClassKeyword) Fix(s *tokens.Stream) bool {
	// indices below assume no cleared placeholders, as after PHP-CS-Fixer's
	// clearEmptyTokens between fixers
	s.ClearEmptyTokens()
	type call struct{ index, open, variable, close int }
	var calls []call
	var analysis gfcAnalysis
outer:
	for index := s.Len() - 1; index > 0; index-- {
		if s.At(index).Kind != token.Ident || !strings.EqualFold(s.At(index).Value, "get_class") {
			continue
		}
		if !analysis.isGlobalFunctionCall(s, index) {
			continue
		}
		open := sigNext(s, index)
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || closeIdx == sigNext(s, open) {
			continue // get_class with no arguments
		}
		meaningful := 0
		var variables []int
		for i := open + 1; i < closeIdx; i++ {
			t := s.At(i)
			switch {
			case t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment:
			case t.Kind == token.Punct && (t.Value == "(" || t.Value == ")"):
			default:
				meaningful++
			}
			if !gfcIsTVariable(s, i) {
				continue
			}
			if strings.ToLower(t.Value) == "$this" {
				continue outer // get_class($this)
			}
			variables = append(variables, i)
		}
		if meaningful > 1 || len(variables) != 1 {
			continue // more logic, more arguments, or no variable argument
		}
		calls = append(calls, call{index, open, variables[0], closeIdx})
	}
	if len(calls) == 0 {
		return false
	}

	// like PHP-CS-Fixer: clear every call first, then insert the replacements
	replacements := make([][]token.Token, len(calls))
	for k, c := range calls {
		replacements[k] = []token.Token{
			{Kind: token.Variable, Value: s.At(c.variable).Value},
			{Kind: token.Punct, Value: "::"},
			{Kind: token.Ident, Value: "class"},
		}
		for i := c.open; i <= c.close; i++ {
			// a token cleared by an earlier merge is no longer T_WHITESPACE in
			// PHP-CS-Fixer, so it goes through the merge again
			if t := s.At(i); (t.Kind == token.Whitespace && t.Value != "") || t.Kind == token.Comment || t.Kind == token.DocComment {
				continue
			}
			clearTokenAndMergeWS(s, i)
		}
		if prev := sigPrev(s, c.index); isPunctVal(s, prev, `\`) {
			clearTokenAt(s, prev)
		}
		clearTokenAt(s, c.index)
	}
	for k, c := range calls {
		s.InsertSliceAt(c.index, replacements[k])
	}
	s.ClearEmptyTokens()
	return true
}
