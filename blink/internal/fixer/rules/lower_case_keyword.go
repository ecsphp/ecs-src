package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseKeywordSniff.php
//
// LowerCaseKeyword lowercases PHP keywords: "ForEach ($a As $b)" -> "foreach ($a as $b)".
type LowerCaseKeyword struct{}

func (LowerCaseKeyword) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\LowerCaseKeywordSniff`
}

func (LowerCaseKeyword) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseKeywordSniff.php"
}

func (LowerCaseKeyword) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword && (t.Kind != token.Ident || !identKeyword(s, i)) {
			continue
		}
		// "array" is only the T_ARRAY keyword in "array(...)"; as a type it is a name
		if strings.EqualFold(t.Value, "array") && !nextIsOpenParen(s, i) {
			continue
		}
		lower := lowerASCII(t.Value)
		// "yield /* c */ from" is one token: keep the comment's case
		if v := t.Value; isYieldFrom(t) {
			lower = lowerASCII(v[:len("yield")]) + v[len("yield"):len(v)-len("from")] + lowerASCII(v[len(v)-len("from"):])
		}
		if lower != t.Value {
			s.SetValue(i, lower)
			changed = true
		}
	}
	return changed
}

// identKeywords lex as names in blink but are keyword tokens to PHP_CodeSniffer
// (T_EXIT, T_EVAL, T_SELF, T_PARENT) unless used as a member or declared name.
var identKeywords = map[string]bool{"exit": true, "die": true, "eval": true, "self": true, "parent": true}

func identKeyword(s *tokens.Stream, i int) bool {
	if !identKeywords[strings.ToLower(s.At(i).Value)] {
		return false
	}
	prev, ok := prevSignificant(s, i)
	if !ok {
		return true
	}
	switch strings.ToLower(prev.Value) {
	case "->", "?->", "::", "function", "const", "\\":
		return false
	}
	return true
}

func nextIsOpenParen(s *tokens.Stream, i int) bool {
	j := i + 1
	for j < s.Len() && isPhpcsEmpty(s.At(j)) {
		j++
	}
	return j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "("
}
