package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/RequireExplicitBooleanOperatorPrecedenceSniff.php
//
// RequireExplicitBooleanOperatorPrecedence reports a boolean operator (&&, ||,
// and, or, xor) that follows a different one in the same expression without
// parentheses to make the precedence explicit. It fixes nothing.
type RequireExplicitBooleanOperatorPrecedence struct{}

func (RequireExplicitBooleanOperatorPrecedence) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\RequireExplicitBooleanOperatorPrecedenceSniff`
}

func (RequireExplicitBooleanOperatorPrecedence) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/RequireExplicitBooleanOperatorPrecedenceSniff.php"
}

func (RequireExplicitBooleanOperatorPrecedence) Fix(*tokens.Stream) bool { return false }

func (r RequireExplicitBooleanOperatorPrecedence) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	sc := newPhpcsScope(newSlvFile(s.Tokens()))
	f := sc.f
	inlineThen := requireBooleanPrecedenceInlineThens(s, f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		op := requireBooleanPrecedenceOperator(sc, inlineThen, i)
		if op == "" || op == "?" || op == ":" {
			continue
		}
		prev := requireBooleanPrecedencePrevious(sc, inlineThen, i-1, sc.startOfStatement(i))
		if prev < 0 {
			continue
		}
		if p := requireBooleanPrecedenceOperator(sc, inlineThen, prev); p == op || p == "?" || p == ":" {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line: f.toks[i].line,
			Message: "Mixing different binary boolean operators within an expression" +
				" without using parentheses to clarify precedence is not allowed.",
			Checker: r.Name() + ".MissingParentheses",
		})
	}
	return errs
}

// requireBooleanPrecedenceOperator names the sniff's search target at i: a
// boolean operator ("&&", "||", "and", "or", "xor"), "?" for T_INLINE_THEN,
// ":" for T_INLINE_ELSE, or "" for any other token.
func requireBooleanPrecedenceOperator(sc *phpcsScope, inlineThen map[int]bool, i int) string {
	f := sc.f
	t := f.toks[i]
	switch {
	case f.is(i, "&&"), f.is(i, "||"):
		return t.orig
	case t.kind == token.Keyword:
		switch kw := strings.ToLower(t.orig); kw {
		case "and", "or", "xor":
			return kw
		}
	case inlineThen[i], f.is(i, "?:"):
		return "?"
	case sc.code[i] == "inline_else":
		return ":"
	}
	return ""
}

// requireBooleanPrecedencePrevious is the sniff's local findPrevious: the
// nearest search target from i back to end, skipping bracketed parts and
// scopes and stopping at a ";", or -1.
func requireBooleanPrecedencePrevious(sc *phpcsScope, inlineThen map[int]bool, i, end int) int {
	f := sc.f
	for ; i >= end && i >= 0; i-- {
		if requireBooleanPrecedenceOperator(sc, inlineThen, i) != "" {
			return i
		}
		switch {
		case sc.opener[i] >= 0 && i == sc.closer[i]:
			i = sc.opener[i]
		case (f.is(i, "]") || f.is(i, "}") || f.is(i, ")")) && f.toks[i].closer >= 0 && f.toks[i].closer < i:
			i = f.toks[i].closer
		case f.is(i, ";"):
			return -1
		}
	}
	return -1
}

// requireBooleanPrecedenceInlineThens marks the f tokens that are a ternary
// "?" (T_INLINE_THEN, not T_NULLABLE), matching f to s by byte offset.
func requireBooleanPrecedenceInlineThens(s *tokens.Stream, f *slvFile) map[int]bool {
	ternary := map[int]bool{}
	at := 0
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.Punct && t.Value == "?" && !phpcsIsNullable(s, i) {
			ternary[at] = true
		}
		at += len(s.At(i).Value)
	}
	out := map[int]bool{}
	at = 0
	for i, t := range f.toks {
		if ternary[at] && f.is(i, "?") {
			out[i] = true
		}
		at += len(t.orig)
	}
	return out
}
