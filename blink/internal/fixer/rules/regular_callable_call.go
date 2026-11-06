package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/RegularCallableCallFixer.php
//
// RegularCallableCall rewrites "call_user_func"/"call_user_func_array" calls as
// direct calls when the callback is a string literal naming a function, a
// closure, or a variable: call_user_func("var_dump", 1, 2) becomes
// var_dump(1, 2), and call_user_func_array($cb, [1, 2]) becomes $cb(...[1, 2]).
// Risky: the two functions may be overridden, and arguments may have side
// effects the rewrite reorders.
type RegularCallableCall struct{}

func (RegularCallableCall) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\RegularCallableCallFixer`
}

func (RegularCallableCall) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/RegularCallableCallFixer.php"
}

// IsCandidate rules out a file without one of the two call names.
func (RegularCallableCall) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("call_user_func", "call_user_func_array")
}

func (RegularCallableCall) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		if !regularCallableIsCallName(s, index) {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue // redeclared or overridden
		}
		open := aliasNextMeaningful(s, index)
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		args := aliasArguments(s, open, closeIdx)
		if len(args) < 1 {
			return changed // no arguments, mirrors the upstream early return
		}
		if regularCallableProcessCall(s, index, args) {
			changed = true
		}
	}
	return changed
}

// regularCallableIsCallName reports whether the token at i is the T_STRING
// "call_user_func" or "call_user_func_array", case-insensitively.
func regularCallableIsCallName(s *tokens.Stream, i int) bool {
	if s.At(i).Kind != token.Ident {
		return false
	}
	v := s.At(i).Value
	return strings.EqualFold(v, "call_user_func") || strings.EqualFold(v, "call_user_func_array")
}

// regularCallableProcessCall ports RegularCallableCallFixer::processCall.
func regularCallableProcessCall(s *tokens.Stream, index int, args [][2]int) bool {
	firstArgIndex := aliasNextMeaningful(s, aliasNextMeaningful(s, index))
	if firstArgIndex < 0 {
		return false
	}
	firstArg := s.At(firstArgIndex)

	switch {
	case aliasIsConstantString(firstArg):
		after := aliasNextMeaningful(s, firstArgIndex)
		if !isPunctVal(s, after, ",") && !isPunctVal(s, after, ")") {
			return false // an expression like call_user_func("foo"."bar", ...)
		}
		content := firstArg.Value
		if !regularCallableIsValidInvoke(content) {
			return false
		}
		newCall := regularCallableNameTokens(content)
		if len(newCall) == 0 {
			return false
		}
		regularCallableReplace(s, index, newCall, firstArgIndex, firstArgIndex)
		return true

	case kwIs(firstArg, "function") || (kwIs(firstArg, "static") && kwIs(s.At(aliasNextMeaningful(s, firstArgIndex)), "function")):
		brace := regularCallableNextBrace(s, firstArgIndex)
		if brace < 0 {
			return false
		}
		end := s.MatchForward(brace)
		if end < 0 {
			return false
		}
		newCall := regularCallableSubcollection(s, firstArgIndex, end)
		newCall = append([]token.Token{aliasPunct("(")}, newCall...)
		newCall = append(newCall, aliasPunct(")"))
		regularCallableReplace(s, index, newCall, firstArgIndex, end)
		return true

	case aliasIsVariable(s, firstArgIndex):
		end := args[0][1] // end of the first argument
		// the same variable used again in a later argument is unsafe to reorder
		for _, arg := range args {
			if arg[1] == end {
				continue
			}
			for i := arg[0]; i <= arg[1]; i++ {
				if regularCallableTokenEquals(s.At(i), firstArg) {
					return false
				}
			}
		}
		newCall := regularCallableSubcollection(s, firstArgIndex, end)
		if regularCallableIsComplex(s, firstArgIndex, end) {
			newCall = append([]token.Token{aliasPunct("(")}, newCall...)
			newCall = append(newCall, aliasPunct(")"))
		}
		regularCallableReplace(s, index, newCall, firstArgIndex, end)
		return true
	}
	return false
}

// regularCallableReplace ports RegularCallableCallFixer::replaceCallUserFuncWithCallback.
func regularCallableReplace(s *tokens.Stream, callIndex int, newCall []token.Token, firstArgStart, firstArgEnd int) {
	useEllipsis := s.At(callIndex).Kind == token.Ident && strings.EqualFold(s.At(callIndex).Value, "call_user_func_array")

	for i := firstArgStart; i <= firstArgEnd; i++ {
		clearTokenAt(s, i)
	}

	after := aliasNextMeaningful(s, firstArgEnd)
	if isPunctVal(s, after, ",") {
		if useEllipsis {
			if second := aliasNextMeaningful(s, after); second >= 0 {
				s.InsertAt(second, aliasPunct("..."))
			}
		}
		clearTokenAt(s, after)
		removeWhitespaceSafely(s, after, 1)
	}

	s.ReplaceRange(callIndex, callIndex, newCall)

	if prev := aliasPrevMeaningful(s, callIndex); isPunctVal(s, prev, `\`) {
		clearTokenAndMergeWS(s, prev)
	}
}

// regularCallableNextBrace returns the index of the first "{" at or after i.
func regularCallableNextBrace(s *tokens.Stream, i int) int {
	for j := i; j < s.Len(); j++ {
		if isPunctVal(s, j, "{") {
			return j
		}
	}
	return -1
}

// regularCallableSubcollection clones the tokens in [start, end], dropping byte
// offsets so the clones can be spliced in elsewhere.
func regularCallableSubcollection(s *tokens.Stream, start, end int) []token.Token {
	out := make([]token.Token, 0, end-start+1)
	for i := start; i <= end; i++ {
		t := s.At(i)
		out = append(out, token.Token{Kind: t.Kind, Part: t.Part, Value: t.Value})
	}
	return out
}

// regularCallableIsComplex reports whether the variable callback in [start, end]
// is more than a variable with plain index accesses, so the call must be wrapped
// in parentheses. It mirrors the backward scan of processCall, skipping
// whitespace, comments, variables and "[...]" index brackets.
func regularCallableIsComplex(s *tokens.Stream, start, end int) bool {
	for i := end; i >= start; i-- {
		switch s.At(i).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		if s.At(i).Kind == token.Variable && s.At(i).Value != "$" {
			continue
		}
		if isPunctVal(s, i, "]") {
			if bs := s.MatchBackward(i); bs >= 0 && aliasIsIndexBracket(s, bs) {
				i = bs
				continue
			}
		}
		return true
	}
	return false
}

// regularCallableTokenEquals is Token::equals for the repeated-variable scan:
// same kind and value.
func regularCallableTokenEquals(a, b token.Token) bool {
	return a.Kind == b.Kind && a.Value == b.Value
}

// regularCallableIsValidInvoke ports isValidFunctionInvoke: the quoted string
// content names a function (no binary prefix, no surrounding whitespace).
func regularCallableIsValidInvoke(content string) bool {
	if len(content) < 3 || content[0] == 'b' || content[0] == 'B' {
		return false
	}
	inner := content[1 : len(content)-1]
	return inner == strings.Trim(inner, " \t\n\r\x00\x0b")
}

// regularCallableNameTokens tokenizes the function name inside the quoted string
// content (unescaping "\\" to "\"), as the callback to splice in place of the
// call_user_func call.
func regularCallableNameTokens(content string) []token.Token {
	inner := strings.ReplaceAll(content, `\\`, `\`)
	inner = inner[1 : len(inner)-1] // strip the quotes
	raw := lexer.Lex("<?php " + inner + "();")
	start := 1
	for start < len(raw) && raw[start].Kind == token.Whitespace {
		start++
	}
	end := len(raw) - 3 // drop the trailing "(", ")" and ";"
	var out []token.Token
	for i := start; i < end; i++ {
		out = append(out, token.Token{Kind: raw[i].Kind, Value: raw[i].Value})
	}
	return out
}
