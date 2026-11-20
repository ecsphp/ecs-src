package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnusedFunctionParameterSniff.php
//
// UnusedFunctionParameter reports a function parameter its body never uses.
// PHP_CodeSniffer raises it as a warning, which ECS escalates to an error.
// It fixes nothing.
type UnusedFunctionParameter struct{}

func (UnusedFunctionParameter) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\UnusedFunctionParameterSniff`
}

func (UnusedFunctionParameter) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnusedFunctionParameterSniff.php"
}

func (UnusedFunctionParameter) Fix(*tokens.Stream) bool { return false }

func (r UnusedFunctionParameter) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	sc := newPhpcsScope(newSlvFile(s.Tokens()))
	f := sc.f
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if sc.code[i] != "function" && sc.code[i] != "fn" {
			continue
		}
		errs = append(errs, unusedFunctionParameterCheck(sc, i, r.Name())...)
	}
	return errs
}

var unusedFunctionParameterMagic = phpcsSet("__destruct", "__call", "__callstatic", "__get", "__set", "__isset",
	"__unset", "__sleep", "__wakeup", "__serialize", "__unserialize", "__tostring", "__set_state", "__clone",
	"__debuginfo")

// unusedFunctionParameterDollarBrace finds the "${name" PHP lexes as
// T_DOLLAR_OPEN_CURLY_BRACES + T_STRING_VARNAME in an interpolating string.
var unusedFunctionParameterDollarBrace = regexp.MustCompile(`(\\*)\$\{([a-zA-Z_\x80-\xff][a-zA-Z0-9_\x80-\xff]*)`)

// unusedFunctionParameterCheck is the sniff's process for the function,
// closure or arrow function at fn.
func unusedFunctionParameterCheck(sc *phpcsScope, fn int, sniff string) []fixer.CodingStandardError {
	f := sc.f
	open := -1
	if sc.pCloser[fn] >= 0 {
		open = f.toks[sc.pCloser[fn]].closer
	} else if sc.code[fn] == "fn" {
		// the tokenizer gives an arrow function its parentheses after the scope map
		open = f.nextEffective(fn+1, len(f.toks))
		if f.is(open, "&") {
			open = f.nextEffective(open+1, len(f.toks))
		}
		if !f.is(open, "(") || f.toks[open].closer < 0 {
			open = -1
		}
	}
	if sc.opener[fn] < 0 || open < 0 {
		return nil
	}
	code, implements := "Found", false
	if sc.code[fn] == "function" && !f.isClosure(fn) {
		class := -1
		for _, c := range sc.conditions(fn) {
			if sc.code[c] == "class" && !f.isAnonClass(c) {
				class = c
				break
			}
		}
		if class >= 0 {
			name := f.nextEffective(fn+1, len(f.toks))
			if f.is(name, "&") {
				name = f.nextEffective(name+1, len(f.toks))
			}
			if name >= 0 && unusedFunctionParameterMagic[strings.ToLower(f.toks[name].orig)] {
				return nil
			}
			implements = unusedFunctionParameterHasName(sc, class, "implements")
			switch {
			case unusedFunctionParameterHasName(sc, class, "extends"):
				code += "InExtendedClass"
			case implements:
				code += "InImplementedInterface"
			}
		}
	}

	params := phpcsMethodParameters(f, open)
	if len(params) == 0 {
		return nil
	}
	unused := map[string]bool{}
	for _, p := range params {
		if p.token >= 0 && p.visibilityToken < 0 && p.setVisibilityToken < 0 && p.readonlyToken < 0 {
			unused[f.toks[p.token].orig] = true
		}
	}

	end := sc.closer[fn] - 1
	if sc.code[fn] == "fn" {
		end++
	}
	foundContent := false
	for i := sc.opener[fn] + 1; i <= end && i < len(f.toks); i++ {
		if f.isEmpty(i) {
			continue
		}
		if !foundContent && implements {
			if f.isKeyword(i, "throw") {
				return nil
			}
			if f.isKeyword(i, "return") {
				n := f.nextEffective(i+1, len(f.toks))
				if f.is(n, ";") {
					return nil
				}
				if n >= 0 && f.is(f.nextEffective(n+1, len(f.toks)), ";") {
					return nil
				}
			}
		}
		foundContent = true
		t := f.toks[i]
		switch {
		case t.kind == token.Variable && t.orig == "$":
			// T_DOLLAR: "${name}"
			if n := f.nextNonWhitespace(i+1, len(f.toks)); f.is(n, "{") {
				if n = f.nextNonWhitespace(n+1, len(f.toks)); n >= 0 && f.toks[n].kind == token.Ident {
					delete(unused, "$"+f.toks[n].orig)
				}
			}
		case t.kind == token.Variable:
			delete(unused, t.orig)
		case t.kind == token.String && (t.interp || t.heredoc || strings.HasPrefix(t.orig, `"`)):
			var b strings.Builder
			for ; i <= end && f.toks[i].kind == token.String && (f.toks[i].interp || f.toks[i].heredoc); i++ {
				b.WriteString(f.toks[i].orig)
			}
			if b.Len() == 0 {
				b.WriteString(t.orig)
			} else {
				i--
			}
			for _, v := range unusedFunctionParameterStringVars(b.String()) {
				delete(unused, v)
			}
		}
	}
	if !foundContent || len(unused) == 0 {
		return nil
	}

	report := func(name, code string) fixer.CodingStandardError {
		return fixer.CodingStandardError{
			Line:    f.toks[fn].line,
			Message: "The method parameter " + name + " is never used",
			Checker: sniff + "." + code,
		}
	}
	if len(params) == 1 {
		var errs []fixer.CodingStandardError
		for name := range unused {
			errs = append(errs, report(name, code))
		}
		return errs
	}
	var rev []fixer.CodingStandardError
	lastUsed := false
	for _, p := range slices.Backward(params) {
		if p.token < 0 {
			continue
		}
		name := f.toks[p.token].orig
		switch {
		case lastUsed && unused[name]:
			rev = append(rev, report(name, code+"BeforeLastUsed"))
		case !lastUsed && !unused[name]:
			lastUsed = true
		case !lastUsed:
			rev = append(rev, report(name, code+"AfterLastUsed"))
		}
	}
	slices.Reverse(rev)
	return rev
}

// unusedFunctionParameterHasName is findExtendedClassName or
// findImplementedInterfaceNames returning a name: the keyword sits between the
// class and its "{", followed (after whitespace only) by a name.
func unusedFunctionParameterHasName(sc *phpcsScope, class int, keyword string) bool {
	f := sc.f
	for j := class + 1; j < sc.opener[class] && j < len(f.toks); j++ {
		if !f.isKeyword(j, keyword) {
			continue
		}
		n := f.nextNonWhitespace(j+1, len(f.toks))
		return n >= 0 && n <= sc.opener[class] && f.toks[n].kind == token.Ident
	}
	return false
}

// unusedFunctionParameterStringVars lists the variables PHP's tokenizer finds
// in a string the sniff re-tokenizes: a double-quoted string or a heredoc.
func unusedFunctionParameterStringVars(content string) []string {
	if strings.HasPrefix(content, "'") || strings.HasPrefix(strings.TrimLeft(strings.TrimPrefix(content, "<<<"), " \t"), "'") {
		return nil
	}
	var vars []string
	for _, t := range lexer.Lex("<?php " + content + ";") {
		if t.Kind == token.Variable {
			vars = append(vars, t.Value)
		}
	}
	for _, m := range unusedFunctionParameterDollarBrace.FindAllStringSubmatch(content, -1) {
		if len(m[1])%2 == 0 {
			vars = append(vars, "$"+m[2])
		}
	}
	return vars
}
