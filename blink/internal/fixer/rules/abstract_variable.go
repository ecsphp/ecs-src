package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// abstractVariableKind is the hook PHP_CodeSniffer's AbstractVariableSniff
// routes a token to.
type abstractVariableKind uint8

const (
	abstractVariableMember abstractVariableKind = iota // processMemberVar
	abstractVariablePlain                              // processVariable
	abstractVariableString                             // processVariableInString
)

// abstractVariableReserved is AbstractVariableSniff::PHP_RESERVED_VARS.
var abstractVariableReserved = map[string]bool{
	"_SERVER": true, "_GET": true, "_POST": true, "_REQUEST": true, "_SESSION": true,
	"_ENV": true, "_COOKIE": true, "_FILES": true, "GLOBALS": true,
	"http_response_header": true, "HTTP_RAW_POST_DATA": true, "php_errormsg": true,
}

// abstractVariableErrors runs AbstractVariableSniff over the stream: visit gets
// every T_VARIABLE and, with strings, every T_DOUBLE_QUOTED_STRING or T_HEREDOC
// line holding a variable, in token order. Without outside, tokens outside any
// OO scope are skipped, as the sniffs that listen only inside OO scopes do.
func abstractVariableErrors(s *tokens.Stream, strs, outside bool, visit func(f *slvFile, kind abstractVariableKind, i int) []fixer.CodingStandardError) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	var errs []fixer.CodingStandardError
	for i, t := range f.toks {
		switch {
		case t.kind == token.Variable && t.orig != "$":
			conds := abstractVariableConditions(f, i)
			oo := -1
			for n, c := range conds {
				if abstractVariableIsOO(f, c) {
					oo = n
					break
				}
			}
			if oo < 0 {
				if outside {
					errs = append(errs, visit(f, abstractVariablePlain, i)...)
				}
				continue
			}
			inFunction := false
			for _, c := range conds[:oo] {
				if f.isKeyword(c, "function") {
					inFunction = true
				}
			}
			if !inFunction {
				for j := f.enclosing(i); j >= 0; j = f.enclosing(j) {
					if f.is(j, "(") && abstractVariableParenOfFunction(f, j) {
						inFunction = true
						break
					}
				}
			}
			if inFunction {
				errs = append(errs, visit(f, abstractVariablePlain, i)...)
			} else {
				errs = append(errs, visit(f, abstractVariableMember, i)...)
			}
		case strs && abstractVariableIsStringToken(f, i) && abstractVariableHasVar(t.orig):
			// a string is handed over once per OO scope holding it, as
			// processTokenWithinScope checks strings before the scope match
			times := 0
			for _, c := range abstractVariableConditions(f, i) {
				if abstractVariableIsOO(f, c) {
					times++
				}
			}
			if times == 0 && outside {
				times = 1
			}
			for range times {
				errs = append(errs, visit(f, abstractVariableString, i)...)
			}
		}
	}
	return errs
}

// abstractVariableConditions lists the owners of the scope braces holding i,
// innermost first: PHP_CodeSniffer's conditions, reversed.
func abstractVariableConditions(f *slvFile, i int) []int {
	var conds []int
	for j := f.enclosing(i); j >= 0; j = f.enclosing(j) {
		if !f.is(j, "{") {
			continue
		}
		if owner := f.scopeCondition(j); owner >= 0 {
			conds = append(conds, owner)
		}
	}
	return conds
}

// abstractVariableIsOO tells Tokens::OO_SCOPE_TOKENS.
func abstractVariableIsOO(f *slvFile, i int) bool {
	return f.isKeyword(i, "class") || f.isKeyword(i, "interface") || f.isKeyword(i, "trait") || f.isKeyword(i, "enum")
}

// abstractVariableHasOOCondition is File::hasCondition(OO_SCOPE_TOKENS).
func abstractVariableHasOOCondition(f *slvFile, i int) bool {
	for _, c := range abstractVariableConditions(f, i) {
		if abstractVariableIsOO(f, c) {
			return true
		}
	}
	return false
}

// abstractVariableParenOfFunction tells a "(" whose parenthesis_owner is a
// T_FUNCTION, T_CLOSURE or T_USE: a parameter list or a closure's use list.
func abstractVariableParenOfFunction(f *slvFile, open int) bool {
	p := f.prevEffective(open - 1)
	if f.isKeyword(p, "use") || f.isKeyword(p, "function") {
		return true
	}
	if f.is(p, "&") {
		return f.isKeyword(f.prevEffective(p-1), "function")
	}
	if p < 0 || f.toks[p].kind != token.Ident && f.toks[p].kind != token.Keyword {
		return false
	}
	q := f.prevEffective(p - 1)
	if f.is(q, "&") {
		q = f.prevEffective(q - 1)
	}
	return f.isKeyword(q, "function")
}

// abstractVariableIsStringToken tells a double-quoted string or heredoc line,
// which is a T_DOUBLE_QUOTED_STRING or T_HEREDOC once it holds a variable
// (blink leaves "${name}" unsplit, so its interp flag is no guide); a nowdoc
// and a backtick string are none.
func abstractVariableIsStringToken(f *slvFile, i int) bool {
	t := f.toks[i]
	if t.kind != token.String {
		return false
	}
	start := i
	for start > 0 && f.toks[start-1].kind == token.String && f.toks[start-1].heredoc == t.heredoc &&
		strings.HasSuffix(f.toks[start-1].orig, "\n") {
		start--
	}
	head := f.toks[start].orig
	if t.heredoc {
		label := strings.TrimLeft(strings.TrimPrefix(head, "<<<"), " \t")
		return !strings.HasPrefix(label, "'")
	}
	head = strings.TrimPrefix(strings.TrimPrefix(head, "b"), "B")
	return strings.HasPrefix(head, `"`)
}

// abstractVariableHasVar is the |(?<!\\)(?:\\{2})*\${?[a-zA-Z0-9_]+}?| check:
// an unescaped "$" followed by a name character, optionally after "{".
func abstractVariableHasVar(v string) bool {
	for k := 0; k < len(v); k++ {
		if v[k] != '$' {
			continue
		}
		slashes := 0
		for b := k - 1; b >= 0 && v[b] == '\\'; b-- {
			slashes++
		}
		if slashes%2 != 0 {
			continue
		}
		n := k + 1
		if n < len(v) && v[n] == '{' {
			n++
		}
		if n < len(v) && (abstractVariableIsAlnum(v[n]) || v[n] == '_') {
			return true
		}
	}
	return false
}

// abstractVariableStringNames is preg_match_all of |[^\\]\${?(name)| (with
// brace) or |[^\\]\$(name)| over a string token, names in the PHP byte sense.
func abstractVariableStringNames(v string, brace bool) []string {
	var names []string
	for k := 0; k+1 < len(v); k++ {
		if v[k] == '\\' || v[k+1] != '$' {
			continue
		}
		n := k + 2
		if brace && n < len(v) && v[n] == '{' {
			n++
		}
		if n >= len(v) || !abstractVariableNameStart(v[n]) {
			continue
		}
		end := n + 1
		for end < len(v) && (abstractVariableNameStart(v[end]) || v[end] >= '0' && v[end] <= '9') {
			end++
		}
		names = append(names, v[n:end])
		k = end - 1
	}
	return names
}

func abstractVariableNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= 0x7f
}

func abstractVariableIsAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// abstractVariableIsCamelCaps is Common::isCamelCaps($name, false, true, false).
func abstractVariableIsCamelCaps(name string) bool {
	if name == "" {
		return false
	}
	upper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	lower := name[0] >= 'a' && name[0] <= 'z'
	acronym := len(name) >= 2 && upper(name[0]) && upper(name[1])
	if !lower && !acronym {
		return false
	}
	for k := 1; k < len(name); k++ {
		if !abstractVariableIsAlnum(name[k]) {
			return false
		}
	}
	return true
}

// abstractVariableProps is the part of File::getMemberProperties the naming
// sniffs read; ok is false where it throws (a variable in an enum).
type abstractVariableProps struct {
	scope          string
	scopeSpecified bool
	setScope       string
}

func abstractVariableMemberProps(f *slvFile, i int) (abstractVariableProps, bool) {
	conds := abstractVariableConditions(f, i)
	if len(conds) == 0 || !abstractVariableIsOO(f, conds[0]) || f.isKeyword(conds[0], "enum") {
		return abstractVariableProps{}, false
	}
	props := abstractVariableProps{scope: "public"}
	start := -1
	for j := i - 1; j >= 0; j-- {
		if f.toks[j].kind == token.AttributeEnd || f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			start = j
			break
		}
	}
	for k := start + 1; k < i; k++ {
		if f.isEmpty(k) {
			continue
		}
		if f.toks[k].kind != token.Keyword {
			break
		}
		switch v := strings.ToLower(f.toks[k].orig); v {
		case "public", "private", "protected":
			props.scope, props.scopeSpecified = v, true
		case "public(set)", "protected(set)", "private(set)":
			props.setScope = strings.TrimSuffix(v, "(set)")
		case "static", "var", "readonly", "final", "abstract":
		default:
			return props, true
		}
	}
	return props, true
}

// abstractVariableNext is findNext(T_WHITESPACE, i, null, true), or -1.
func abstractVariableNext(f *slvFile, i int) int {
	for ; i < len(f.toks); i++ {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return -1
}

// abstractVariablePrev is findPrevious(T_WHITESPACE, i, null, true), or -1.
func abstractVariablePrev(f *slvFile, i int) int {
	for ; i >= 0; i-- {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return -1
}

func abstractVariableError(f *slvFile, i int, checker, msg string) fixer.CodingStandardError {
	return fixer.CodingStandardError{Line: f.toks[i].line, Message: msg, Checker: checker}
}

// abstractVariableUcfirst is PHP's ucfirst for a visibility keyword.
func abstractVariableUcfirst(v string) string {
	if v == "" {
		return v
	}
	return strings.ToUpper(v[:1]) + v[1:]
}
