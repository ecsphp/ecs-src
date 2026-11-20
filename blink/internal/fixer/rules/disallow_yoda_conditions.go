package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/ControlStructures/DisallowYodaConditionsSniff.php
//
// DisallowYodaConditions reports a comparison with a literal, a static array or
// a variable-free parenthesis on its left. It fixes nothing.
type DisallowYodaConditions struct{}

func (DisallowYodaConditions) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\ControlStructures\DisallowYodaConditionsSniff`
}

func (DisallowYodaConditions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/ControlStructures/DisallowYodaConditionsSniff.php"
}

func (DisallowYodaConditions) Fix(*tokens.Stream) bool { return false }

// yodaComparisons is Tokens::COMPARISON_TOKENS without T_COALESCE.
var yodaComparisons = map[string]bool{
	"==": true, "!=": true, "===": true, "!==": true, "<>": true, "<=": true, ">=": true,
	"<": true, ">": true, "<=>": true,
}

var yodaArithmetic = map[string]bool{"+": true, "-": true, "*": true, "/": true, "%": true, "**": true}

// yodaStaticPuncts are the punctuation tokens isArrayStatic accepts: the
// assignment, comparison, arithmetic, operator and boolean tokens, the plain
// brackets (short array brackets are not), "," and "=>".
var yodaStaticPuncts = map[string]bool{
	"=": true, "&=": true, "|=": true, ".=": true, "/=": true, "-=": true, "**=": true, "%=": true,
	"*=": true, "+=": true, "^=": true, "=>": true, "<<=": true, ">>=": true, "??=": true,
	"==": true, "!=": true, "===": true, "!==": true, "<>": true, "<=": true, ">=": true,
	"<": true, ">": true, "<=>": true, "??": true,
	"+": true, "-": true, "*": true, "/": true, "%": true, "**": true,
	"&": true, "|": true, "^": true, "<<": true, ">>": true, "&&": true, "||": true,
	"{": true, "}": true, "(": true, ")": true, ",": true,
}

func (y DisallowYodaConditions) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	var errs []fixer.CodingStandardError
	for i, t := range f.toks {
		if t.kind == token.Punct && yodaComparisons[t.orig] && yodaDisallowed(f, i) {
			errs = append(errs, fixer.CodingStandardError{
				Line:    t.line,
				Message: "Usage of Yoda conditions is not allowed; switch the expression order",
				Checker: y.Name() + ".Found",
			})
		}
	}
	return errs
}

func yodaDisallowed(f *slvFile, i int) bool {
	prev := max(f.prevEffective(i-1), 0)
	t := f.toks[prev]
	switch {
	case f.is(prev, ")"), yodaLiteral(t), t.kind == token.Number,
		t.kind == token.String && !t.interp && !t.heredoc:
	case f.is(prev, "]") && t.closer >= 0 && f.isShortArrayOpen(t.closer):
		prev = t.closer
		if !yodaArrayStatic(f, prev) {
			return false
		}
	default:
		return false
	}
	before := max(f.prevEffective(prev-1), 0)
	if f.toks[before].kind == token.Punct && (yodaArithmetic[f.toks[before].orig] || f.toks[before].orig == ".") {
		return false
	}
	if !f.is(prev, ")") {
		return true
	}
	opener := f.toks[prev].closer
	if opener < 0 {
		return false
	}
	owner := f.prevEffective(opener - 1)
	if owner >= 0 && f.isKeyword(owner, "array") {
		return yodaArrayStatic(f, owner)
	}
	if owner >= 0 && yodaName(f.toks[owner]) {
		return false
	}
	for j := prev - 1; j > opener; j-- {
		if f.toks[j].kind == token.Variable {
			return false
		}
	}
	return f.nextEffective(opener+1, prev) >= 0
}

// yodaLiteral reports T_TRUE, T_FALSE and T_NULL, fully qualified or not.
func yodaLiteral(t slvTok) bool {
	if t.kind != token.Ident {
		return false
	}
	switch strings.ToLower(strings.TrimPrefix(t.orig, `\`)) {
	case "true", "false", "null":
		return true
	}
	return false
}

// yodaName reports Tokens::NAME_TOKENS: a name that is no keyword PHP_CodeSniffer
// tokenizes on its own.
func yodaName(t slvTok) bool {
	if t.kind != token.Ident || yodaLiteral(t) {
		return false
	}
	switch strings.ToLower(t.orig) {
	case "exit", "die", "eval", "self", "parent":
		return false
	}
	return true
}

// yodaArrayStatic is isArrayStatic: the array at open ("[" or "array") holds
// only literals and operators, skipping whatever has a scope closer.
func yodaArrayStatic(f *slvFile, open int) bool {
	start := open
	if f.isKeyword(open, "array") {
		start = f.nextEffective(open+1, len(f.toks))
		if !f.is(start, "(") {
			return true
		}
	}
	end := f.toks[start].closer
	for i := start + 1; i < end; i++ {
		if c := phpcsScopeCloser(f, i); c >= 0 {
			i = c
			continue
		}
		t := f.toks[i]
		switch {
		case f.isEmpty(i), yodaLiteral(t), t.kind == token.InlineHTML:
		case t.kind == token.String && !t.heredoc:
		case t.kind == token.Punct && (yodaStaticPuncts[t.orig] || len(t.orig) > 2 && t.orig[0] == '(' && t.orig[len(t.orig)-1] == ')'):
		case t.kind == token.Punct && (t.orig == "[" || t.orig == "]") && !f.isShortArrayOpen(i) && (t.orig != "]" || !f.isShortArrayOpen(t.closer)):
		case t.kind == token.Keyword && yodaBooleanKeyword(t.orig):
		default:
			return false
		}
	}
	return true
}

func yodaBooleanKeyword(v string) bool {
	switch strings.ToLower(v) {
	case "and", "or", "xor":
		return true
	}
	return false
}
