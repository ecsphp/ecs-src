package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// squizReport runs a report-only sniff over the stream as PHP_CodeSniffer
// tokenizes it; report adds an error on token i with the sniff's error code.
func squizReport(s *tokens.Stream, sniff string, process func(f *slvFile, sc *phpcsScope, report func(i int, code, msg string))) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	process(f, sc, func(i int, code, msg string) {
		errs = append(errs, fixer.CodingStandardError{Line: f.toks[i].line, Message: msg, Checker: sniff + "." + code})
	})
	return errs
}

// squizReportParenOwner is the parenthesis_owner of the "(" at open, or -1;
// beside the owners of phpcsScope it knows arrow functions and anonymous classes.
func squizReportParenOwner(f *slvFile, sc *phpcsScope, open int) int {
	if sc.pOwner[open] >= 0 {
		return sc.pOwner[open]
	}
	p := f.prevEffective(open - 1)
	if f.isKeyword(p, "fn") || f.isKeyword(p, "class") && f.isKeyword(f.prevEffective(p-1), "new") {
		return p
	}
	return -1
}

// squizReportNested lists the "(" holding i, outermost first, as the keys of
// PHP_CodeSniffer's nested_parenthesis.
func squizReportNested(sc *phpcsScope, i int) []int {
	var out []int
	for o := sc.nested[i]; o >= 0; o = sc.nested[o] {
		out = append([]int{o}, out...)
	}
	return out
}

// squizReportIsInlineThen tells a ternary "?" (T_INLINE_THEN) from a nullable type.
func squizReportIsInlineThen(f *slvFile, sc *phpcsScope, i int) bool {
	return f.is(i, "?") && sc.isInlineThen(i)
}

// squizReportIsBool is T_TRUE / T_FALSE.
func squizReportIsBool(f *slvFile, i int) bool {
	v := strings.ToLower(strings.TrimPrefix(f.toks[i].orig, `\`))
	return (v == "true" || v == "false") && f.toks[i].kind == token.Ident && !squizReportIsMember(f, i)
}

// squizReportIsMember is a name after "->", "?->", "::" or "function".
func squizReportIsMember(f *slvFile, i int) bool {
	p := f.prevEffective(i - 1)
	return f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") || f.isKeyword(p, "function") || f.isKeyword(p, "const")
}

// squizReportIsPlainName is a name token (Tokens::NAME_TOKENS): blink lexes some
// keywords PHP_CodeSniffer gives their own token as names.
func squizReportIsPlainName(f *slvFile, i int) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(strings.TrimPrefix(f.toks[i].orig, `\`)) {
	case "true", "false", "null", "self", "parent", "exit", "die", "eval":
		return squizReportIsMember(f, i)
	}
	return true
}

// squizReportNextNonEmpty is findNext(EMPTY_TOKENS, i, null, true), or -1.
func squizReportNextNonEmpty(f *slvFile, i int) int { return f.nextEffective(i, len(f.toks)) }

// squizReportPrevNonEmpty is findPrevious(EMPTY_TOKENS, i, null, true), or -1.
func squizReportPrevNonEmpty(f *slvFile, i int) int {
	if i >= len(f.toks) {
		i = len(f.toks) - 1
	}
	return f.prevEffective(i)
}

// squizReportPrevLocal is findPrevious(..., $local = true): it searches back
// from start over whole scopes, brackets and parentheses, stopping at a ";".
func squizReportPrevLocal(f *slvFile, sc *phpcsScope, start int, match func(i int) bool) int {
	for i := start; i >= 0; i-- {
		if match(i) {
			return i
		}
		t := f.toks[i]
		switch {
		case sc.opener[i] >= 0 && sc.closer[i] == i:
			i = sc.opener[i]
		case (f.is(i, "]") || f.is(i, "}") || f.is(i, ")")) && t.closer >= 0 && t.closer < i:
			i = t.closer
		case f.is(i, ";"):
			return -1
		}
	}
	return -1
}

// squizReportIsComparison is Tokens::COMPARISON_TOKENS.
func squizReportIsComparison(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Punct {
		return false
	}
	switch f.toks[i].orig {
	case "==", "!=", "<>", "===", "!==", "<=", ">=", "<", ">", "<=>", "??":
		return true
	}
	return false
}

// squizReportIsArithmetic is Tokens::ARITHMETIC_TOKENS.
func squizReportIsArithmetic(f *slvFile, i int) bool {
	if i < 0 || f.toks[i].kind != token.Punct {
		return false
	}
	switch f.toks[i].orig {
	case "+", "-", "*", "/", "%", "**":
		return true
	}
	return false
}

// squizReportIsBooleanOperator is Tokens::BOOLEAN_OPERATORS.
func squizReportIsBooleanOperator(f *slvFile, i int) bool {
	return f.is(i, "&&") || f.is(i, "||") || f.isKeyword(i, "and") || f.isKeyword(i, "or") || f.isKeyword(i, "xor")
}
