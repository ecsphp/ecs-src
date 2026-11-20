package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FunctionCommentThrowTagSniff.php
//
// FunctionCommentThrowTag reports a function doc comment whose @throws tags do
// not match the exceptions the function body throws.
type FunctionCommentThrowTag struct{}

func (FunctionCommentThrowTag) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\FunctionCommentThrowTagSniff`
}

func (FunctionCommentThrowTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FunctionCommentThrowTagSniff.php"
}

func (FunctionCommentThrowTag) Fix(*tokens.Stream) bool { return false }

func (r FunctionCommentThrowTag) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !f.isNamedFunction(i) || sc.cond[i] != i || sc.closer[i] < 0 {
			continue
		}
		errs = append(errs, r.function(f, sc, i)...)
	}
	return errs
}

func (r FunctionCommentThrowTag) function(f *slvFile, sc *phpcsScope, fn int) []fixer.CodingStandardError {
	end := fn - 1
	for ; end >= 0; end-- {
		t := f.toks[end]
		if t.kind == token.Whitespace || t.kind == token.Keyword && functionCommentThrowTagModifiers[strings.ToLower(t.orig)] {
			continue
		}
		if t.kind == token.AttributeEnd && t.closer >= 0 {
			end = t.closer
			continue
		}
		break
	}
	if end < 0 || f.toks[end].doc != slvDocClose {
		return nil
	}
	scopeEnd := sc.closer[fn]
	var thrown []string
	found := false
	unknown := 0
	for cur := fn + 1; cur < scopeEnd; cur++ {
		switch {
		case f.isKeyword(cur, "function") && f.isClosure(cur) || f.isKeyword(cur, "class") && f.isAnonClass(cur):
			if c := sc.closer[cur]; c > cur {
				cur = c - 1 // the loop's cur++ lands on the closer, as findNext from closer+1
			}
			continue
		case !f.isKeyword(cur, "throw"):
			continue
		}
		found = true
		next := f.nextEffective(cur+1, len(f.toks))
		switch {
		case f.isKeyword(next, "new") || squizReportIsName(f, next):
			ex := next
			if f.isKeyword(next, "new") {
				ex = f.nextEffective(next+1, scopeEnd)
			}
			if ex >= 0 && squizReportIsName(f, ex) {
				name := f.toks[ex].orig
				if len(name) > 10 && strings.EqualFold(name[:10], `namespace\`) {
					name = name[10:]
				}
				thrown = append(thrown, name)
			}
		case next >= 0 && f.toks[next].kind == token.Variable:
			thrown = append(thrown, functionCommentThrowTagRethrown(f, sc, fn, cur, next)...)
		default:
			unknown++
		}
	}
	if !found {
		return nil
	}
	var unique []string
	for _, t := range thrown {
		if !slices.Contains(unique, t) {
			unique = append(unique, t)
		}
	}
	var tags []string
	for j := f.docOpener(end); j < end; j++ {
		if f.toks[j].doc != slvDocTag || f.toks[j].orig != "@throws" {
			continue
		}
		if j+2 < len(f.toks) && f.toks[j+2].doc == slvDocString {
			ex := f.toks[j+2].orig
			if sp := strings.IndexByte(ex, ' '); sp >= 0 {
				ex = ex[:sp]
			}
			if !slices.Contains(tags, ex) {
				tags = append(tags, ex)
			}
		}
	}
	line := f.toks[end].line
	if len(tags) == 0 {
		return []fixer.CodingStandardError{squizReportError(r.Name(), "Missing", line, "Missing @throws tag in function comment")}
	}
	if len(unique) == 0 {
		return nil
	}
	if n := len(unique) + unknown; n != len(tags) {
		return []fixer.CodingStandardError{squizReportError(r.Name(), "WrongNumber", line,
			fmt.Sprintf("Expected %d @throws tag(s) in function comment; %d found", n, len(tags)))}
	}
	var errs []fixer.CodingStandardError
	for _, t := range unique {
		ok := false
		for _, tag := range tags {
			if strings.HasSuffix(tag, t) {
				ok = true
				break
			}
		}
		if !ok {
			errs = append(errs, squizReportError(r.Name(), "Missing", line, `Missing @throws tag for "`+t+`" exception`))
		}
	}
	return errs
}

// functionCommentThrowTagModifiers is Tokens::METHOD_MODIFIERS.
var functionCommentThrowTagModifiers = map[string]bool{
	"private": true, "public": true, "protected": true, "abstract": true, "static": true, "final": true,
}

// functionCommentThrowTagRethrown returns the types caught by the nearest catch
// before the throw at cur when it rethrows that catch's variable.
func functionCommentThrowTagRethrown(f *slvFile, sc *phpcsScope, fn, cur, variable int) []string {
	catch := -1
	for j := cur; j >= sc.opener[fn]; j-- {
		if f.isKeyword(j, "catch") {
			catch = j
			break
		}
	}
	if catch < 0 || sc.pCloser[catch] < 0 {
		return nil
	}
	pc := sc.pCloser[catch]
	po := f.toks[pc].closer
	caught := 0 // findPrevious returning false reads token 0
	for j := pc - 1; j >= po; j-- {
		if f.toks[j].kind == token.Variable {
			caught = j
			break
		}
	}
	if f.toks[caught].content != f.toks[variable].content {
		return nil
	}
	var out []string
	for e := range strings.SplitSeq(f.content(po+1, caught-1), "|") {
		out = append(out, strings.Trim(e, " \t\n\r\x00\x0B"))
	}
	return out
}
