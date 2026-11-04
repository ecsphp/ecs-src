package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NoUnsetOnPropertyFixer.php
//
// NoUnsetOnProperty rewrites unset on an object or static property to a null
// assignment ("unset($this->a)" -> "$this->a = null"). Non-property arguments of
// the same unset call are kept as their own unset statements (risky: unset and
// null are not equivalent for typed or referenced properties).
type NoUnsetOnProperty struct{}

func (NoUnsetOnProperty) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\NoUnsetOnPropertyFixer`
}

func (NoUnsetOnProperty) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NoUnsetOnPropertyFixer.php"
}

// noUnsetInfo mirrors the fixer's _UnsetInfo: one argument of an unset call.
type noUnsetInfo struct {
	startIndex, endIndex int
	isToTransform        bool
	isFirst              bool
}

func (NoUnsetOnProperty) Fix(s *tokens.Stream) bool {
	before := s.Render()
	for index := s.Len() - 1; index >= 0; index-- {
		if !kwIs(s.At(index), "unset") {
			continue
		}

		unsetsInfo := noUnsetGatherInfo(s, index)
		if !noUnsetAnyToTransform(unsetsInfo) {
			continue
		}

		// process last to first so earlier indices stay valid as we insert
		isLast := true
		for _, u := range slices.Backward(unsetsInfo) {
			noUnsetUpdateTokens(s, u, isLast)
			isLast = false
		}
	}
	s.ClearEmptyTokens()
	return s.Render() != before
}

func noUnsetGatherInfo(s *tokens.Stream, index int) []noUnsetInfo {
	start := nextPunctOfKind(s, index, "(")
	if start < 0 {
		return nil
	}
	end := s.MatchForward(start)
	if end < 0 {
		return nil
	}

	var out []noUnsetInfo
	isFirst := true
	for _, arg := range noUnsetArguments(s, start, end) {
		startIndex := sigNext(s, arg[0]-1)
		endIndex := sigPrev(s, arg[1]+1)
		out = append(out, noUnsetInfo{
			startIndex:    startIndex,
			endIndex:      endIndex,
			isToTransform: noUnsetIsProperty(s, startIndex, endIndex),
			isFirst:       isFirst,
		})
		isFirst = false
	}
	return out
}

// noUnsetArguments ports ArgumentsAnalyzer::getArguments for the parentheses at
// [open, close]: the start and end index of each argument, skipping the ',' and
// a PHP 7.3 trailing comma, and the '(' ')' themselves.
func noUnsetArguments(s *tokens.Stream, open, close int) [][2]int {
	first := sigNext(s, open)
	if first < 0 || isPunctVal(s, first, ")") {
		return nil
	}

	var args [][2]int
	start := open + 1
	i := open + 1
	for ; i < close; i++ {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
			if end := s.MatchForward(i); end >= 0 {
				i = end
			}
			continue
		}
		if isPunctVal(s, i, ",") {
			if sigNext(s, i) == close {
				break // trailing ',' in call (PHP 7.3)
			}
			args = append(args, [2]int{start, i - 1})
			start = i + 1
		}
	}
	return append(args, [2]int{start, i - 1})
}

// noUnsetIsProperty reports whether the argument [index, endIndex] is a single
// object property ("$a->b") or static property ("A::$b"), the two forms the
// fixer rewrites.
func noUnsetIsProperty(s *tokens.Stream, index, endIndex int) bool {
	if index < 0 {
		return false
	}
	t := s.At(index)
	if t.Kind == token.Variable && t.Value != "$" {
		next := sigNext(s, index)
		if next < 0 || !isPunctVal(s, next, "->") {
			return false
		}
		next = sigNext(s, next)
		nextNext := sigNext(s, next)
		if nextNext >= 0 && nextNext < endIndex {
			return false
		}
		return next >= 0 && s.At(next).Kind == token.Ident
	}
	if (t.Kind == token.Punct && t.Value == `\`) || t.Kind == token.Ident {
		next := noUnsetAfterClassName(s, index)
		nextNext := sigNext(s, next)
		if nextNext >= 0 && nextNext < endIndex {
			return false
		}
		return next >= 0 && s.At(next).Kind == token.Variable && s.At(next).Value != "$"
	}
	return false
}

// noUnsetAfterClassName is getTokenNotOfKindsSibling forward over a class name:
// it skips "::", "\" and name tokens, returning the first token past them.
func noUnsetAfterClassName(s *tokens.Stream, index int) int {
	for j := index + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.Whitespace || isComment(t) {
			continue
		}
		if t.Kind == token.Ident || (t.Kind == token.Punct && (t.Value == "::" || t.Value == `\`)) {
			continue
		}
		return j
	}
	return -1
}

func noUnsetAnyToTransform(unsetsInfo []noUnsetInfo) bool {
	for _, u := range unsetsInfo {
		if u.isToTransform {
			return true
		}
	}
	return false
}

func noUnsetUpdateTokens(s *tokens.Stream, u noUnsetInfo, isLast bool) {
	// first entry to transform: drop the leading "unset("
	if u.isFirst && u.isToTransform {
		braceIndex := noUnsetPrevOfKind(s, u.startIndex, token.Punct, "(")
		unsetIndex := noUnsetPrevOfKind(s, braceIndex, token.Keyword, "unset")
		clearTokenAndMergeWS(s, braceIndex)
		clearTokenAndMergeWS(s, unsetIndex)
	}

	// last entry to transform: drop the trailing ")" (and a trailing ",")
	if isLast && u.isToTransform {
		braceIndex := nextPunctOfKind(s, u.endIndex, ")")
		if previousIndex := sigPrev(s, braceIndex); isPunctVal(s, previousIndex, ",") {
			clearTokenAndMergeWS(s, previousIndex)
		}
		clearTokenAndMergeWS(s, braceIndex)
	}

	// not last: the separating "," becomes a ";"
	if !isLast {
		if commaIndex := nextPunctOfKind(s, u.endIndex, ","); commaIndex >= 0 {
			s.Set(commaIndex, token.Token{Kind: token.Punct, Value: ";"})
		}
	}

	// a kept (non-property) entry that is not last gains a trailing ")"
	if !u.isToTransform && !isLast {
		s.InsertAt(u.endIndex+1, token.Token{Kind: token.Punct, Value: ")"})
	}

	// a kept entry that is not first gains a leading "unset("
	if !u.isToTransform && !u.isFirst {
		s.InsertSliceAt(u.startIndex, []token.Token{
			{Kind: token.Keyword, Value: "unset"},
			{Kind: token.Punct, Value: "("},
		})
	}

	// a transformed entry gains a trailing " = null"
	if u.isToTransform {
		s.InsertSliceAt(u.endIndex+1, []token.Token{
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: "="},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Ident, Value: "null"},
		})
	}
}

// noUnsetPrevOfKind scans backward from idx for a token of the given kind and
// value, returning its index or -1.
func noUnsetPrevOfKind(s *tokens.Stream, idx int, kind token.Kind, value string) int {
	for j := idx - 1; j >= 0; j-- {
		if t := s.At(j); t.Kind == kind && t.Value == value {
			return j
		}
	}
	return -1
}
