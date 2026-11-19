package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDeclarationArgumentSpacingSniff.php
//
// FunctionDeclarationArgumentSpacing fixes the spacing inside a function's
// parameter list: around "=", after "&", "..." and type hints, around commas
// and inside the parentheses.
type FunctionDeclarationArgumentSpacing struct {
	EqualsSpacing             int
	RequiredSpacesAfterOpen   int
	RequiredSpacesBeforeClose int
}

func (FunctionDeclarationArgumentSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\FunctionDeclarationArgumentSpacingSniff`
}

func (FunctionDeclarationArgumentSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDeclarationArgumentSpacingSniff.php"
}

func (r FunctionDeclarationArgumentSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.EqualsSpacing = configInt(config, "equalsSpacing", r.EqualsSpacing)
	r.RequiredSpacesAfterOpen = configInt(config, "requiredSpacesAfterOpen", r.RequiredSpacesAfterOpen)
	r.RequiredSpacesBeforeClose = configInt(config, "requiredSpacesBeforeClose", r.RequiredSpacesBeforeClose)
	return r
}

func (r FunctionDeclarationArgumentSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.isKeyword(i, "function") {
				p := f.prevEffective(i - 1)
				if f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") {
					continue
				}
				fn, ok := phpcsFunctionAt(f, i)
				if !ok {
					continue
				}
				r.processBracket(f, fn.openParen)
				if fn.closure {
					if open, _ := fn.useParens(f); open >= 0 {
						r.processBracket(f, open)
					}
				}
				continue
			}
			if open := phpcsFnParen(f, i); open >= 0 {
				r.processBracket(f, open)
			}
		}
	})
}

// phpcsFnParen returns the "(" of the arrow function whose "fn" is at i, or -1.
func phpcsFnParen(f *slvFile, i int) int {
	if !f.isKeyword(i, "fn") {
		return -1
	}
	if p := f.prevEffective(i - 1); f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") || f.isKeyword(p, "function") {
		return -1
	}
	o := f.nextEffective(i+1, len(f.toks))
	if f.is(o, "&") {
		o = f.nextEffective(o+1, len(f.toks))
	}
	if !f.is(o, "(") || f.toks[o].closer < 0 {
		return -1
	}
	return o
}

func (r FunctionDeclarationArgumentSpacing) gapTo(f *slvFile, from, to int, ws int) (int, bool) {
	if f.toks[from].line != f.toks[to].line {
		return 0, true
	}
	if f.isWS(ws) {
		return f.toks[ws].length, false
	}
	return 0, false
}

// clearAfter replaces the whitespace tokens after i with nothing.
func clearWhitespaceAfter(f *slvFile, i int) {
	for j := i + 1; j < len(f.toks) && f.isWS(j); j++ {
		f.replaceToken(j, "")
	}
}

func (r FunctionDeclarationArgumentSpacing) processBracket(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	multiLine := f.toks[open].line != f.toks[closeIdx].line
	params := phpcsMethodParameters(f, open)

	if len(params) == 0 {
		if f.nextNot(f.isWS, open+1, closeIdx) < 0 {
			if closeIdx-open != 1 {
				f.beginChangeset()
				for j := open + 1; f.isWS(j); j++ {
					f.replaceToken(j, "")
				}
				f.endChangeset()
			}
			return
		}
	}

	for n, p := range params {
		if p.referenceToken >= 0 && f.isWS(p.referenceToken+1) {
			f.beginChangeset()
			clearWhitespaceAfter(f, p.referenceToken)
			f.endChangeset()
		}
		if p.variadicToken >= 0 && f.isWS(p.variadicToken+1) {
			f.beginChangeset()
			clearWhitespaceAfter(f, p.variadicToken)
			f.endChangeset()
		}
		if p.equalToken >= 0 {
			r.equalsSide(f, p.token, p.equalToken, p.token+1, p.equalToken)
			r.equalsSide(f, p.equalToken, p.defaultToken, p.equalToken+1, p.defaultToken)
		}
		if p.typeHintToken >= 0 {
			end := p.typeHintEndToken
			var gap strings.Builder
			for j := end + 1; f.isWS(j); j++ {
				gap.WriteString(f.toks[j].content)
			}
			if gap.String() != " " {
				f.beginChangeset()
				f.addContent(end, " ")
				clearWhitespaceAfter(f, end)
				f.endChangeset()
			}
		}
		for _, mod := range []int{p.visibilityToken, p.setVisibilityToken, p.readonlyToken} {
			if mod < 0 {
				continue
			}
			after := f.nextNot(f.isWS, mod+1, p.token)
			spaces := 0
			if after >= 0 && f.toks[mod].line != f.toks[after].line {
				spaces = -1
			} else if f.isWS(mod + 1) {
				spaces = f.toks[mod+1].length
			}
			if spaces != 1 {
				f.beginChangeset()
				f.addContent(mod, " ")
				clearWhitespaceAfter(f, mod)
				f.endChangeset()
			}
		}

		if n == 0 || params[n-1].commaToken < 0 {
			continue
		}
		comma := params[n-1].commaToken
		endOfPrev := f.prevEffective(comma - 1)
		spaceBefore := 0
		if f.toks[endOfPrev].line != f.toks[comma].line {
			spaceBefore = -1
		} else if f.isWS(comma - 1) {
			spaceBefore = f.toks[comma-1].length
		}
		if spaceBefore != 0 {
			startOfCurr := f.nextEffective(comma+1, len(f.toks))
			f.beginChangeset()
			f.addContent(endOfPrev, ",")
			f.replaceToken(comma, "")
			if f.toks[comma].line == f.toks[startOfCurr].line {
				clearWhitespaceAfter(f, comma)
			} else {
				for j := comma - 1; f.isWS(j) && f.toks[endOfPrev].line != f.toks[j].line; j-- {
					f.replaceToken(j, "")
				}
				for j := comma + 1; f.isWS(j) && f.toks[comma].line == f.toks[j].line; j++ {
					f.replaceToken(j, "")
				}
			}
			f.endChangeset()
		}

		if multiLine {
			next := f.nextEffective(comma+1, closeIdx)
			nextLine := f.toks[0].line
			if next >= 0 {
				nextLine = f.toks[next].line
			}
			if nextLine != f.toks[comma].line {
				continue
			}
		}
		spacesAfter := 0
		if f.isWS(comma + 1) {
			spacesAfter = f.toks[comma+1].length
		}
		switch {
		case spacesAfter == 0:
			f.addContent(comma, " ")
		case spacesAfter != 1:
			f.replaceToken(comma+1, " ")
		}
	}

	if multiLine {
		return
	}
	gap := 0
	if f.isWS(closeIdx - 1) {
		gap = f.toks[closeIdx-1].length
	}
	if gap != r.RequiredSpacesBeforeClose {
		padding := strings.Repeat(" ", r.RequiredSpacesBeforeClose)
		if gap == 0 {
			f.addContentBefore(closeIdx, padding)
		} else {
			f.replaceToken(closeIdx-1, padding)
		}
	}
	gap = 0
	if f.isWS(open + 1) {
		gap = f.toks[open+1].length
	}
	if gap != r.RequiredSpacesAfterOpen {
		padding := strings.Repeat(" ", r.RequiredSpacesAfterOpen)
		if gap == 0 {
			f.addContent(open, padding)
		} else {
			f.replaceToken(open+1, padding)
		}
	}
}

// equalsSide checks the spacing between from and to (the parameter and its
// "=", or the "=" and the default value), fixing it when only whitespace sits
// between them.
func (r FunctionDeclarationArgumentSpacing) equalsSide(f *slvFile, from, to, ws, searchEnd int) {
	spaces, newline := r.gapTo(f, from, to, ws)
	if !newline && spaces == r.EqualsSpacing {
		return
	}
	if f.nextNot(f.isWS, from+1, searchEnd) >= 0 {
		return
	}
	f.beginChangeset()
	f.addContent(from, strings.Repeat(" ", r.EqualsSpacing))
	clearWhitespaceAfter(f, from)
	f.endChangeset()
}

// phpcsParam is one entry of File::getMethodParameters; token indexes are -1
// when unset.
type phpcsParam struct {
	token, defaultToken, equalToken, referenceToken, variadicToken int
	typeHintToken, typeHintEndToken                                int
	visibilityToken, setVisibilityToken, readonlyToken, commaToken int
}

// phpcsTypeParen reports whether the "(" or ")" at i belongs to a DNF type in
// a parameter list rather than to an expression.
func phpcsTypeParen(f *slvFile, i, open int) bool {
	if f.is(i, ")") {
		i = f.toks[i].closer
		if i < 0 {
			return false
		}
	}
	p := f.prevEffective(i - 1)
	if p == open || f.is(p, ",") || f.is(p, "|") || p >= 0 && f.toks[p].kind == token.AttributeEnd {
		return true
	}
	if p < 0 || f.toks[p].kind != token.Keyword {
		return false
	}
	switch kw := strings.ToLower(f.toks[p].orig); {
	case kw == "public" || kw == "protected" || kw == "private" || kw == "readonly" || kw == "final",
		strings.HasSuffix(kw, "(set)"):
		return true
	}
	return false
}

// phpcsMethodParameters is File::getMethodParameters for the parameter list
// opening at open.
func phpcsMethodParameters(f *slvFile, open int) []phpcsParam {
	closeIdx := f.toks[open].closer
	if closeIdx < 0 {
		return nil
	}
	var vars []phpcsParam
	reset := func() phpcsParam {
		return phpcsParam{token: -1, defaultToken: -1, equalToken: -1, referenceToken: -1, variadicToken: -1,
			typeHintToken: -1, typeHintEndToken: -1, visibilityToken: -1, setVisibilityToken: -1, readonlyToken: -1,
			commaToken: -1}
	}
	cur := reset()
	defaultStart := -1
	typeTok := func(i int) {
		if defaultStart >= 0 {
			return
		}
		if cur.typeHintToken < 0 {
			cur.typeHintToken = i
		}
		cur.typeHintEndToken = i
	}
	for i := open + 1; i <= closeIdx; i++ {
		t := f.toks[i]
		if (f.is(i, "(") || f.is(i, ")")) && t.closer >= 0 && i != closeIdx {
			if defaultStart < 0 && phpcsTypeParen(f, i, open) {
				typeTok(i)
				continue
			}
			if f.is(i, "(") {
				i = t.closer
			}
			continue
		}
		if (f.is(i, "[") || f.is(i, "{")) && t.closer > i {
			i = t.closer
			continue
		}
		if f.isParamName(i) {
			continue
		}
		switch t.kind {
		case token.AttributeStart:
			if t.closer > i {
				i = t.closer
			}
		case token.Variable:
			cur.token = i
		case token.Ident:
			lower := strings.ToLower(t.orig)
			switch lower {
			case "self", "parent":
				typeTok(i)
				continue
			case "true", "false", "null":
				typeTok(i)
				continue
			}
			prevComma := -1
			for k := i; k >= open; k-- {
				if f.is(k, ",") {
					prevComma = k
					break
				}
			}
			if prevComma >= 0 {
				eq := false
				for k := prevComma; k < i; k++ {
					if f.is(k, "=") {
						eq = true
						break
					}
				}
				if eq {
					continue
				}
			}
			typeTok(i)
		case token.Keyword:
			kw := strings.ToLower(t.orig)
			switch {
			case kw == "callable":
				if cur.typeHintToken < 0 {
					cur.typeHintToken = i
				}
				cur.typeHintEndToken = i
			case kw == "static" || kw == "namespace":
				typeTok(i)
			case kw == "array" && !f.is(f.nextEffective(i+1, len(f.toks)), "("):
				typeTok(i)
			case kw == "public" || kw == "protected" || kw == "private":
				if defaultStart < 0 {
					cur.visibilityToken = i
				}
			case strings.HasSuffix(kw, "(set)"):
				if defaultStart < 0 {
					cur.setVisibilityToken = i
				}
			case kw == "readonly":
				if defaultStart < 0 {
					cur.readonlyToken = i
				}
			}
		case token.Punct:
			switch t.orig {
			case "&":
				if n := f.nextEffective(i+1, len(f.toks)); n >= 0 && f.toks[n].kind == token.Variable || f.is(n, "...") {
					if defaultStart < 0 {
						cur.referenceToken = i
					}
				} else {
					typeTok(i)
				}
			case "|", "\\":
				typeTok(i)
			case "?":
				if defaultStart < 0 {
					cur.typeHintEndToken = i
				}
			case "...":
				cur.variadicToken = i
			case "=":
				defaultStart = f.nextEffective(i+1, len(f.toks))
				cur.equalToken = i
			case ")", ",":
				if cur.token < 0 {
					continue
				}
				if defaultStart >= 0 {
					cur.defaultToken = defaultStart
				} else {
					cur.equalToken = -1
				}
				if t.orig == "," {
					cur.commaToken = i
				}
				vars = append(vars, cur)
				cur = reset()
				defaultStart = -1
			}
		}
	}
	return vars
}

// isParamName reports whether the word at i is a named-argument label, which
// PHP_CodeSniffer's tokenizer turns into T_PARAM_NAME: a name followed by ":"
// right after "(" or ",".
func (f *slvFile) isParamName(i int) bool {
	k := f.toks[i].kind
	if k != token.Ident && k != token.Keyword || strings.Contains(f.toks[i].orig, `\`) {
		return false
	}
	p := f.prevEffective(i - 1)
	return f.is(f.nextEffective(i+1, len(f.toks)), ":") && (f.is(p, "(") || f.is(p, ","))
}
