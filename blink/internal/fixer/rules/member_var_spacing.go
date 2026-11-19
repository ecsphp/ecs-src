package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/MemberVarSpacingSniff.php
//
// MemberVarSpacing puts Spacing blank lines before a property (SpacingBeforeFirst
// before the first one in a class) and none between its comment and itself.
type MemberVarSpacing struct {
	Spacing            int
	SpacingBeforeFirst int
}

func (MemberVarSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\MemberVarSpacingSniff`
}

func (MemberVarSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/MemberVarSpacingSniff.php"
}

func (m MemberVarSpacing) WithConfig(config map[string]any) fixer.Fixer {
	m.Spacing = configInt(config, "spacing", m.Spacing)
	m.SpacingBeforeFirst = configInt(config, "spacingBeforeFirst", m.SpacingBeforeFirst)
	return m
}

func (m MemberVarSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.toks[i].kind == token.Variable }, func(i int) int {
			scopes, inFunction := f.ooConditions(i)
			if scopes == 0 {
				return 0
			}
			if !inFunction && f.inFunctionParens(i) {
				inFunction = true
			}
			if inFunction {
				return 0
			}
			end := m.process(f, i)
			if scopes > 1 {
				// AbstractScopeSniff returns min() of every OO scope's result,
				// and the outer ones return null
				return 0
			}
			return end
		})
	})
}

// ooConditions counts the class-like scopes holding i and reports whether a
// function or closure sits between the innermost of them and i.
func (f *slvFile) ooConditions(i int) (int, bool) {
	scopes, inFunction, seenOO := 0, false, false
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		owner := f.braceOwner(open)
		if owner < 0 {
			continue
		}
		switch strings.ToLower(f.toks[owner].orig) {
		case "class", "interface", "trait", "enum":
			scopes++
			seenOO = true
		case "function":
			if !seenOO {
				inFunction = true
			}
		}
	}
	return scopes, inFunction
}

// inFunctionParens reports whether any parentheses holding i belong to a
// function, closure or closure use.
func (f *slvFile) inFunctionParens(i int) bool {
	for open := f.enclosing(i); open >= 0; open = f.enclosing(open) {
		if !f.is(open, "(") {
			continue
		}
		p := f.prevEffective(open - 1)
		if f.isKeyword(p, "use") || f.isKeyword(p, "function") {
			return true
		}
		if p >= 0 && (f.toks[p].kind == token.Ident || f.toks[p].kind == token.Keyword) {
			q := f.prevEffective(p - 1)
			if f.is(q, "&") {
				q = f.prevEffective(q - 1)
			}
			if f.isKeyword(q, "function") {
				return true
			}
		}
	}
	return false
}

func (f *slvFile) isMemberVarPrefix(i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch v := strings.ToLower(f.toks[i].orig); v {
	case "public", "protected", "private", "static", "final", "var", "readonly", "abstract":
		return true
	default:
		return strings.HasSuffix(v, "(set)")
	}
}

// attributeCloser is the "]" of the attribute holding i, -1 when none does.
func (f *slvFile) attributeCloser(i int) int {
	if f.toks[i].kind == token.AttributeEnd {
		return i
	}
	for j := i; j >= 0; j = f.enclosing(j) {
		if f.toks[j].kind == token.AttributeStart && f.toks[j].closer >= i {
			return f.toks[j].closer
		}
	}
	return -1
}

// docBlockOpener is the comment_opener of the docblock closer at i.
func (f *slvFile) docBlockOpener(i int) int {
	for j := i; j >= 0; j-- {
		if f.toks[j].doc == slvDocOpen {
			return j
		}
	}
	return i
}

func (f *slvFile) isDocClose(i int) bool {
	return i >= 0 && f.toks[i].doc == slvDocClose
}

func (f *slvFile) isPhpcsAnnotation(i int) bool {
	return i >= 0 && f.toks[i].kind == token.Comment && isPhpcsAnnotationComment(f.toks[i].orig)
}

func (f *slvFile) nextOfKind(i int, k token.Kind) int {
	for ; i < len(f.toks); i++ {
		if f.toks[i].kind != k {
			return i
		}
	}
	return -1
}

func (m MemberVarSpacing) process(f *slvFile, v int) int {
	toks := f.toks
	eops := -1
	for j := v - 1; j >= 0; j-- {
		if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			eops = j
			break
		}
		if (f.is(j, ")") || f.is(j, "]")) && toks[j].closer >= 0 {
			j = toks[j].closer
		}
	}
	startOfStatement := -1
	for j := eops + 1; j < v; j++ {
		if f.isMemberVarPrefix(j) {
			startOfStatement = j
			break
		}
		if f.is(j, ";") {
			break
		}
	}
	if startOfStatement < 0 {
		return 0
	}
	endOfStatement := 0
	for j := v + 1; j < len(toks); j++ {
		if f.is(j, ";") {
			endOfStatement = j
			break
		}
	}

	start := startOfStatement
	prev := startOfStatement - 1
	for ; prev >= 0; prev-- {
		if toks[prev].kind == token.Whitespace {
			continue
		}
		if toks[prev].kind == token.AttributeEnd && toks[prev].closer >= 0 {
			prev = toks[prev].closer
			start = prev
			continue
		}
		break
	}
	if f.isDocClose(prev) {
		start = prev
	} else if prev >= 0 && (toks[prev].kind == token.Comment || toks[prev].kind == token.DocComment) {
		prevContent := f.prevEffective(prev - 1)
		if prevContent < 0 || toks[prevContent].line != toks[prev].line {
			start = prev
		}
	}

	for i := start + 1; i < startOfStatement; i++ {
		if c := f.attributeCloser(i); c >= 0 {
			i = c
			continue
		}
		if toks[i].column != 1 || toks[i].kind != token.Whitespace || toks[i].line == toks[i+1].line || f.isPhpcsAnnotation(i-1) {
			continue
		}
		next := f.nextOfKind(i+1, token.Whitespace)
		if next < 0 {
			break
		}
		f.beginChangeset()
		for j := i; j < next; j++ {
			if toks[j].line == toks[next].line {
				break
			}
			f.replaceToken(j, "")
		}
		f.endChangeset()
		i = next
	}

	var first int
	switch {
	case start == startOfStatement:
		first = start
		for j := start; j >= 0 && toks[j].line == toks[start].line; j-- {
			if !f.isEmpty(j) {
				first = j
			}
		}
	case f.isDocClose(start):
		first = f.docBlockOpener(start)
	default:
		first = f.prevEffective(start-1) + 1
		for ; first < len(toks); first++ {
			if toks[first].kind == token.Comment || toks[first].kind == token.DocComment || toks[first].kind == token.AttributeStart {
				break
			}
		}
	}

	prev = first - 1
	for prev >= 0 && toks[prev].kind == token.Whitespace {
		prev--
	}
	if prev < 0 {
		return endOfStatement
	}
	if f.is(prev, "}") {
		if owner := f.scopeCondition(prev); owner >= 0 && f.isKeyword(owner, "function") && !f.isClosure(owner) {
			return 0
		}
	}
	spacing := m.Spacing
	if f.is(prev, "{") {
		if owner := f.scopeCondition(prev); owner >= 0 {
			switch strings.ToLower(toks[owner].orig) {
			case "class", "interface", "trait", "enum":
				spacing = m.SpacingBeforeFirst
			}
		}
	}
	if toks[first].line-toks[prev].line-1 == spacing {
		return endOfStatement
	}
	f.beginChangeset()
	for i := prev + 1; i < first; i++ {
		if toks[i].line == toks[prev].line {
			continue
		}
		if toks[i].line == toks[first].line {
			for range spacing {
				f.replaceToken(i, f.eol+f.tokenContent(i))
			}
			break
		}
		f.replaceToken(i, "")
	}
	f.endChangeset()
	return endOfStatement
}
