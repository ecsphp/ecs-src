package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/ArrowFunctionDeclarationSniff.php
//
// SlevomatArrowFunctionDeclaration sets the spaces after "fn" and around "=>"
// of an arrow function (spacesCountAfterKeyword, spacesCountBeforeArrow,
// spacesCountAfterArrow, 1 each), leaves line breaks with allowMultiLine, and
// with disallowReturnTypeHint removes the return type.
type SlevomatArrowFunctionDeclaration struct {
	afterKeyword, beforeArrow, afterArrow *int
	allowMultiLine                        bool
	disallowReturnTypeHint                bool
}

func (SlevomatArrowFunctionDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\ArrowFunctionDeclarationSniff`
}

func (SlevomatArrowFunctionDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatArrowFunctionDeclaration{}.Name())
}

func (r SlevomatArrowFunctionDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	for key, field := range map[string]**int{
		"spacesCountAfterKeyword": &r.afterKeyword,
		"spacesCountBeforeArrow":  &r.beforeArrow,
		"spacesCountAfterArrow":   &r.afterArrow,
	} {
		if _, ok := config[key]; ok {
			*field = new(configInt(config, key, 1))
		}
	}
	if v, ok := config["allowMultiLine"].(bool); ok {
		r.allowMultiLine = v
	}
	if v, ok := config["disallowReturnTypeHint"].(bool); ok {
		r.disallowReturnTypeHint = v
	}
	return r
}

func miscSpaces(n *int) int {
	if n == nil {
		return 1
	}
	return *n
}

func (r SlevomatArrowFunctionDeclaration) Fix(s *tokens.Stream) bool {
	afterKeyword, beforeArrow, afterArrow := miscSpaces(r.afterKeyword), miscSpaces(r.beforeArrow), miscSpaces(r.afterArrow)
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "fn") }, func(fn int) int {
			after := f.nextNonWhitespace(fn+1, len(f.toks))
			r.miscCheckSpaces(f, fn, after, afterKeyword)

			arrow := f.slvmArrowOf(fn)
			if arrow < 0 {
				return 0
			}
			if r.disallowReturnTypeHint {
				if start := f.slvmReturnTypeStart(fn); start >= 0 {
					colon := f.prevEffective(start - 1)
					beforeColon := f.prevEffective(colon - 1)
					f.beginChangeset()
					f.slvmRemoveBetween(beforeColon, arrow)
					if beforeArrow > 0 {
						f.addBefore(arrow, strings.Repeat(" ", beforeArrow))
					}
					f.endChangeset()
				}
			}
			r.miscCheckSpaces(f, f.prevNonWhitespace(arrow-1), arrow, beforeArrow)
			r.miscCheckSpaces(f, arrow, f.nextNonWhitespace(arrow+1, len(f.toks)), afterArrow)
			return 0
		})
	})
}

// miscCheckSpaces requires exactly count spaces between before and after.
func (r SlevomatArrowFunctionDeclaration) miscCheckSpaces(f *slvFile, before, after, count int) {
	spaces := f.content(before+1, after-1)
	if r.allowMultiLine && strings.HasPrefix(spaces, f.eol) {
		return
	}
	if len(spaces) == count && (count == 0 || strings.Trim(spaces, " ") == "" && spaces != "") {
		return
	}
	f.beginChangeset()
	if count > 0 {
		f.add(before, strings.Repeat(" ", count))
	}
	f.slvmRemoveBetween(before, after)
	f.endChangeset()
}

// slvmRemoveBetween is FixerHelper::removeBetween.
func (f *slvFile) slvmRemoveBetween(start, end int) {
	for i := start + 1; i < end; i++ {
		f.replaceToken(i, "")
	}
}

// slvmArrowOf returns the T_FN_ARROW of the arrow function at fn, or -1.
func (f *slvFile) slvmArrowOf(fn int) int {
	for j := fn + 1; j < len(f.toks); j++ {
		if f.is(j, "=>") && f.fnOfArrow(j) == fn {
			return j
		}
		if f.is(j, ";") || f.is(j, "{") {
			return -1
		}
	}
	return -1
}

// slvmParenOpener returns the "(" opening the parameters of the function,
// closure or arrow function keyword at fn, or -1.
func (f *slvFile) slvmParenOpener(fn int) int {
	for j := fn + 1; j < len(f.toks); j++ {
		if f.is(j, "(") {
			return j
		}
		if f.is(j, "{") || f.is(j, ";") || f.is(j, "=>") {
			return -1
		}
	}
	return -1
}

// slvmReturnTypeStart is the start of FunctionHelper::findReturnTypeHint, or -1.
func (f *slvFile) slvmReturnTypeStart(fn int) int {
	open := f.slvmParenOpener(fn)
	if open < 0 || f.toks[open].closer < 0 {
		return -1
	}
	next := f.nextEffective(f.toks[open].closer+1, len(f.toks))
	if f.isKeyword(next, "use") {
		if o := f.nextEffective(next+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
			next = f.nextEffective(f.toks[o].closer+1, len(f.toks))
		}
	}
	if !f.is(next, ":") {
		return -1
	}
	return f.nextEffective(next+1, len(f.toks))
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/NamedArgumentSpacingSniff.php
//
// SlevomatNamedArgumentSpacing removes the whitespace between a named
// argument and its colon and puts one space after the colon.
type SlevomatNamedArgumentSpacing struct{}

func (SlevomatNamedArgumentSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\NamedArgumentSpacingSniff`
}

func (SlevomatNamedArgumentSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNamedArgumentSpacing{}.Name())
}

func (SlevomatNamedArgumentSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.slvmIsParamName, func(name int) int {
			colon := -1
			for j := name + 1; j < len(f.toks); j++ {
				if f.is(j, ":") {
					colon = j
					break
				}
			}
			if colon < 0 {
				return 0
			}
			if colon != name+1 {
				f.replace(colon-1, "")
			}
			ws := colon + 1
			if ws < len(f.toks) && f.toks[ws].kind == token.Whitespace && f.toks[ws].content == " " {
				return 0
			}
			if ws < len(f.toks) && f.toks[ws].kind == token.Whitespace {
				f.replace(ws, " ")
			} else {
				f.add(colon, " ")
			}
			return 0
		})
	})
}

// slvmIsParamName tells a named argument label PHP_CodeSniffer retokenizes
// as T_PARAM_NAME: a name between "(" or "," and ":".
func (f *slvFile) slvmIsParamName(i int) bool {
	k := f.toks[i].kind
	if k != token.Ident && k != token.Keyword || strings.Contains(f.toks[i].orig, "\\") {
		return false
	}
	next := f.nextEffective(i+1, len(f.toks))
	if !f.is(next, ":") {
		return false
	}
	prev := f.prevEffective(i - 1)
	return f.is(prev, "(") || f.is(prev, ",")
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/UselessParameterDefaultValueSniff.php
//
// SlevomatUselessParameterDefaultValue removes the default value of a
// parameter followed by a required one (a null default only when the type
// is nullable with "?").
type SlevomatUselessParameterDefaultValue struct{}

func (SlevomatUselessParameterDefaultValue) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\UselessParameterDefaultValueSniff`
}

func (SlevomatUselessParameterDefaultValue) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessParameterDefaultValue{}.Name())
}

func (SlevomatUselessParameterDefaultValue) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		isFunction := func(i int) bool { return f.isKeyword(i, "function") || f.isKeyword(i, "fn") }
		f.eachToken(isFunction, func(fn int) int {
			params := f.slvmMethodParameters(fn)
			for i, p := range params {
				if !p.hasDefault || strings.ToLower(p.defaultValue) == "null" && !p.nullable {
					continue
				}
				for _, next := range params[i+1:] {
					if next.hasDefault {
						continue
					}
					if next.variadic {
						break
					}
					comma := -1
					for j := params[i+1].token - 1; j >= 0; j-- {
						if f.is(j, ",") {
							comma = j
							break
						}
					}
					f.beginChangeset()
					for k := p.token + 1; k < comma; k++ {
						f.replace(k, "")
					}
					f.endChangeset()
					break
				}
			}
			return 0
		})
	})
}

// slvmParam is the part of a File::getMethodParameters entry the ports read.
type slvmParam struct {
	token        int
	defaultValue string
	hasDefault   bool
	nullable     bool
	variadic     bool
}

// slvmMethodParameters is PHP_CodeSniffer's File::getMethodParameters.
func (f *slvFile) slvmMethodParameters(fn int) []slvmParam {
	open := f.slvmParenOpener(fn)
	if open < 0 || f.toks[open].closer < 0 {
		return nil
	}
	closeIdx := f.toks[open].closer
	var params []slvmParam
	cur := slvmParam{token: -1}
	defaultStart := -1
	for i := open + 1; i <= closeIdx; i++ {
		t := f.toks[i]
		switch {
		case t.kind == token.AttributeStart && t.closer >= 0:
			i = t.closer
		case i != closeIdx && (f.is(i, "(") || f.is(i, "[")) && t.closer > i && !f.slvmIsTypeParen(i):
			i = t.closer
		case f.is(i, "&") && defaultStart < 0:
		case t.kind == token.Variable:
			cur.token = i
		case f.is(i, "..."):
			cur.variadic = true
		case f.is(i, "?") && defaultStart < 0:
			cur.nullable = true
		case f.is(i, "="):
			defaultStart = f.nextEffective(i+1, len(f.toks))
		case i == closeIdx || f.is(i, ","):
			if cur.token < 0 {
				continue
			}
			if defaultStart >= 0 {
				cur.hasDefault = true
				cur.defaultValue = strings.TrimSpace(f.content(defaultStart, i-1))
			}
			params = append(params, cur)
			cur = slvmParam{token: -1}
			defaultStart = -1
		}
	}
	return params
}

// slvmIsTypeParen tells the "(" of a DNF type (T_TYPE_OPEN_PARENTHESIS).
func (f *slvFile) slvmIsTypeParen(open int) bool {
	p := f.prevEffective(open - 1)
	if f.is(p, "|") || f.is(p, "(") || f.is(p, ",") || f.is(p, "?") {
		n := f.nextEffective(open+1, len(f.toks))
		return n >= 0 && f.toks[n].kind == token.Ident
	}
	return false
}
