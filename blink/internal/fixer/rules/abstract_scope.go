package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// abstractScopeFile is a file as PHP_CodeSniffer's report-only sniffs read it:
// its tokens, scope map, and the errors a sniff has added so far.
type abstractScopeFile struct {
	f      *slvFile
	sc     *phpcsScope
	sniff  string
	errors []fixer.CodingStandardError
	levels []int
}

func abstractScopeLoad(s *tokens.Stream, sniff string) *abstractScopeFile {
	f := newSlvFile(lexer.Lex(s.Render()))
	return &abstractScopeFile{f: f, sc: newPhpcsScope(f), sniff: sniff}
}

// addError is File::addError: the error is reported on the line of token i.
func (a *abstractScopeFile) addError(i int, code, format string, args ...any) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	a.errors = append(a.errors, fixer.CodingStandardError{Line: a.f.toks[i].line, Message: msg, Checker: a.sniff + "." + code})
}

// code is the PHP_CodeSniffer token code of i, telling T_CLOSURE from
// T_FUNCTION and T_ANON_CLASS from T_CLASS.
func (a *abstractScopeFile) code(i int) string {
	c := a.sc.code[i]
	switch {
	case c == "function" && a.f.isClosure(i):
		return "closure"
	case c == "class" && a.f.isAnonClassKeyword(i):
		return "anon_class"
	}
	return c
}

// abstractScopeOO is Tokens::OO_SCOPE_TOKENS.
var abstractScopeOO = phpcsSet("class", "anon_class", "interface", "trait", "enum")

// run is AbstractScopeSniff::process over every token whose code is in
// listen: within runs once per enclosing condition in scopes, outermost first;
// outside, when set, runs for a token no such condition holds.
func (a *abstractScopeFile) run(scopes, listen map[string]bool, within func(i, scope int), outside func(i int)) {
	for i := range a.f.toks {
		if !listen[a.code(i)] {
			continue
		}
		found := false
		for _, c := range a.sc.conditions(i) {
			if scopes[a.code(c)] {
				within(i, c)
				found = true
			}
		}
		if !found && outside != nil {
			outside(i)
		}
	}
}

// level is a token's level: the number of scope conditions holding it.
func (a *abstractScopeFile) level(i int) int {
	if a.levels == nil {
		delta := make([]int, len(a.f.toks)+1)
		for _, r := range a.sc.scopes {
			delta[r.opener+1]++
			delta[r.closer]--
		}
		a.levels = make([]int, len(a.f.toks))
		depth := 0
		for j := range a.levels {
			depth += delta[j]
			a.levels[j] = depth
		}
	}
	return a.levels[i]
}

// deepest reports whether scope is the innermost condition holding i.
func (a *abstractScopeFile) deepest(i, scope int) bool {
	conds := a.sc.conditions(i)
	return len(conds) > 0 && conds[len(conds)-1] == scope
}

// isTString tells PHP_CodeSniffer's T_STRING: an unqualified name, or a
// keyword used as the name of a declared function.
func (a *abstractScopeFile) isTString(i int) bool {
	f := a.f
	t := f.toks[i]
	switch t.kind {
	case token.Ident:
		return a.sc.code[i] == "string" && !strings.Contains(t.orig, `\`)
	case token.Keyword:
		p := f.prevEffective(i - 1)
		if f.is(p, "&") {
			p = f.prevEffective(p - 1)
		}
		return f.isKeyword(p, "function")
	}
	return false
}

// declarationName is File::getDeclarationName.
func (a *abstractScopeFile) declarationName(i int) string {
	stop := len(a.f.toks)
	switch {
	case a.sc.pCloser[i] >= 0:
		stop = a.f.toks[a.sc.pCloser[i]].closer
	case a.sc.opener[i] >= 0:
		stop = a.sc.opener[i]
	}
	for j := i; j < stop; j++ {
		if a.isTString(j) {
			return a.f.toks[j].orig
		}
	}
	return ""
}

// className is the declaration name of an OO scope, "[Anonymous Class]" for
// an anonymous class.
func (a *abstractScopeFile) className(scope int) string {
	if a.code(scope) == "anon_class" {
		return "[Anonymous Class]"
	}
	return a.declarationName(scope)
}

// abstractScopeMethodProps is the scope part of File::getMethodProperties.
type abstractScopeMethodProps struct {
	scope          string
	scopeSpecified bool
	isStatic       bool
}

func (a *abstractScopeFile) methodProps(i int) abstractScopeMethodProps {
	f := a.f
	props := abstractScopeMethodProps{scope: "public"}
	named := a.code(i) == "function"
	for j := i - 1; j > 0; j-- {
		t := f.toks[j]
		if t.kind == token.Whitespace || (t.kind == token.Comment && !isPhpcsAnnotationComment(t.orig)) {
			continue
		}
		if t.kind != token.Keyword {
			break
		}
		switch kw := strings.ToLower(t.orig); {
		case kw == "static":
			props.isStatic = true
		case named && (kw == "public" || kw == "private" || kw == "protected"):
			props.scope, props.scopeSpecified = kw, true
		case named && (kw == "final" || kw == "abstract"):
		default:
			return props
		}
	}
	return props
}

// abstractScopeIsCamelCaps is Common::isCamelCaps with classFormat false and
// visibilityPublic true.
func abstractScopeIsCamelCaps(name string, strict bool) bool {
	if name == "" {
		return false
	}
	isUpper := func(b byte) bool { return b >= 'A' && b <= 'Z' }
	isLower := func(b byte) bool { return b >= 'a' && b <= 'z' }
	switch {
	case isLower(name[0]):
	case !strict && len(name) > 1 && isUpper(name[0]) && isUpper(name[1]):
	default:
		return false
	}
	for k := 1; k < len(name); k++ {
		b := name[k]
		if !isUpper(b) && !isLower(b) && (b < '0' || b > '9') {
			return false
		}
	}
	if strict {
		last := false
		for k := 1; k < len(name); k++ {
			caps := isUpper(name[k])
			if caps && last {
				return false
			}
			last = caps
		}
	}
	return true
}

// abstractScopeMagicMethods is the MAGIC_METHODS list of the naming sniffs.
var abstractScopeMagicMethods = phpcsSet("construct", "destruct", "call", "callstatic", "get", "set", "isset",
	"unset", "sleep", "wakeup", "serialize", "unserialize", "tostring", "invoke", "set_state", "clone", "debuginfo")

// abstractScopeDoubleUnderscoreMethods is CamelCapsFunctionNameSniff's
// DOUBLE_UNDERSCORE_METHODS: SoapClient's methods.
var abstractScopeDoubleUnderscoreMethods = phpcsSet("dorequest", "getcookies", "getfunctions", "getlastrequest",
	"getlastrequestheaders", "getlastresponse", "getlastresponseheaders", "gettypes", "setcookie", "setlocation",
	"setsoapheaders", "soapcall")

// abstractScopeMagicPrefixed is preg_match('|^__[^_]|', $name).
func abstractScopeMagicPrefixed(name string) bool {
	return len(name) > 2 && strings.HasPrefix(name, "__") && name[2] != '_'
}

// abstractScopeUcfirst is PHP's ucfirst.
func abstractScopeUcfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
