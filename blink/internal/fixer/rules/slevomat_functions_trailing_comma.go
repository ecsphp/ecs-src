package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireTrailingCommaInCallSniff.php
//
// SlevomatRequireTrailingCommaInCall adds a comma after the last argument of a
// multi-line call whose ")" is on its own line.
type SlevomatRequireTrailingCommaInCall struct {
	disabled bool
}

func (SlevomatRequireTrailingCommaInCall) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireTrailingCommaInCallSniff`
}

func (SlevomatRequireTrailingCommaInCall) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireTrailingCommaInCall{}.Name())
}

func (r SlevomatRequireTrailingCommaInCall) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireTrailingCommaInCall) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isCallOpen, func(open int) int {
			closeIdx := f.toks[open].closer
			if f.toks[open].line == f.toks[closeIdx].line {
				return 0
			}
			last := f.prevEffective(closeIdx - 1)
			if last == open || f.toks[closeIdx].line == f.toks[last].line || f.is(last, ",") {
				return 0
			}
			f.beginChangeset()
			f.add(last, ",")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/DisallowTrailingCommaInCallSniff.php
//
// SlevomatDisallowTrailingCommaInCall removes the comma after the last
// argument of a call, with onlySingleLine only in single-line calls.
type SlevomatDisallowTrailingCommaInCall struct {
	onlySingleLine bool
}

func (SlevomatDisallowTrailingCommaInCall) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\DisallowTrailingCommaInCallSniff`
}

func (SlevomatDisallowTrailingCommaInCall) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowTrailingCommaInCall{}.Name())
}

func (r SlevomatDisallowTrailingCommaInCall) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["onlySingleLine"].(bool); ok {
		r.onlySingleLine = v
	}
	return r
}

func (r SlevomatDisallowTrailingCommaInCall) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isCallOpen, func(open int) int {
			closeIdx := f.toks[open].closer
			f.removeTrailingComma(f.prevEffective(closeIdx-1), open, closeIdx, r.onlySingleLine)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireTrailingCommaInDeclarationSniff.php
//
// SlevomatRequireTrailingCommaInDeclaration adds a comma after the last
// parameter of a multi-line function, closure or arrow function declaration.
type SlevomatRequireTrailingCommaInDeclaration struct {
	disabled bool
}

func (SlevomatRequireTrailingCommaInDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireTrailingCommaInDeclarationSniff`
}

func (SlevomatRequireTrailingCommaInDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireTrailingCommaInDeclaration{}.Name())
}

func (r SlevomatRequireTrailingCommaInDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireTrailingCommaInDeclaration) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isFunctionToken, func(i int) int {
			open := f.functionParenOpen(i)
			if open < 0 || f.toks[open].line == f.toks[f.toks[open].closer].line {
				return 0
			}
			last := f.prevEffective(f.toks[open].closer - 1)
			if last == open || f.is(last, ",") {
				return 0
			}
			f.beginChangeset()
			f.add(last, ",")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/DisallowTrailingCommaInDeclarationSniff.php
//
// SlevomatDisallowTrailingCommaInDeclaration removes the comma after the last
// parameter of a declaration, with onlySingleLine only in single-line ones.
type SlevomatDisallowTrailingCommaInDeclaration struct {
	onlySingleLine bool
}

func (SlevomatDisallowTrailingCommaInDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\DisallowTrailingCommaInDeclarationSniff`
}

func (SlevomatDisallowTrailingCommaInDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowTrailingCommaInDeclaration{}.Name())
}

func (r SlevomatDisallowTrailingCommaInDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["onlySingleLine"].(bool); ok {
		r.onlySingleLine = v
	}
	return r
}

func (r SlevomatDisallowTrailingCommaInDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isFunctionToken, func(i int) int {
			open := f.functionParenOpen(i)
			if open < 0 {
				return 0
			}
			closeIdx := f.toks[open].closer
			f.removeTrailingComma(f.prevNonWhitespaceFrom(closeIdx-1, open), open, closeIdx, r.onlySingleLine)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireTrailingCommaInClosureUseSniff.php
//
// SlevomatRequireTrailingCommaInClosureUse adds a comma after the last
// variable of a multi-line closure "use".
type SlevomatRequireTrailingCommaInClosureUse struct {
	disabled bool
}

func (SlevomatRequireTrailingCommaInClosureUse) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireTrailingCommaInClosureUseSniff`
}

func (SlevomatRequireTrailingCommaInClosureUse) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireTrailingCommaInClosureUse{}.Name())
}

func (r SlevomatRequireTrailingCommaInClosureUse) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireTrailingCommaInClosureUse) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isClosureToken, func(i int) int {
			open := f.closureUseOpen(i)
			if open < 0 || f.toks[open].line == f.toks[f.toks[open].closer].line {
				return 0
			}
			last := f.prevNonWhitespaceFrom(f.toks[open].closer-1, open)
			if f.is(last, ",") {
				return 0
			}
			f.beginChangeset()
			f.add(last, ",")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/DisallowTrailingCommaInClosureUseSniff.php
//
// SlevomatDisallowTrailingCommaInClosureUse removes the comma after the last
// variable of a closure "use", with onlySingleLine only in single-line ones.
type SlevomatDisallowTrailingCommaInClosureUse struct {
	onlySingleLine bool
}

func (SlevomatDisallowTrailingCommaInClosureUse) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\DisallowTrailingCommaInClosureUseSniff`
}

func (SlevomatDisallowTrailingCommaInClosureUse) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowTrailingCommaInClosureUse{}.Name())
}

func (r SlevomatDisallowTrailingCommaInClosureUse) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["onlySingleLine"].(bool); ok {
		r.onlySingleLine = v
	}
	return r
}

func (r SlevomatDisallowTrailingCommaInClosureUse) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isClosureToken, func(i int) int {
			open := f.closureUseOpen(i)
			if open < 0 {
				return 0
			}
			closeIdx := f.toks[open].closer
			f.removeTrailingComma(f.prevNonWhitespaceFrom(closeIdx-1, open), open, closeIdx, r.onlySingleLine)
			return 0
		})
	})
}

// removeTrailingComma drops the comma at last before ")" at closeIdx, and the
// whitespace after it when ")" is on the comma's line.
func (f *slvFile) removeTrailingComma(last, open, closeIdx int, onlySingleLine bool) {
	if !f.is(last, ",") {
		return
	}
	if onlySingleLine && f.toks[open].line != f.toks[closeIdx].line {
		return
	}
	f.beginChangeset()
	f.replace(last, "")
	if f.toks[last].line == f.toks[closeIdx].line {
		for i := last + 1; i < closeIdx; i++ {
			f.replace(i, "")
		}
	}
	f.endChangeset()
}

// prevNonWhitespaceFrom is TokenHelper::findPreviousExcluding(T_WHITESPACE)
// searching back from i down to end inclusive, or -1.
func (f *slvFile) prevNonWhitespaceFrom(i, end int) int {
	for ; i >= end && i >= 0; i-- {
		if f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return -1
}

// isCallOpen tells the "(" the call sniffs register: one after a name, a
// variable, isset, unset, ")" or a class keyword, not after a declaration's
// function name.
func (f *slvFile) isCallOpen(i int) bool {
	if !f.is(i, "(") || f.toks[i].closer < 0 {
		return false
	}
	p := f.prevEffective(i - 1)
	if p < 0 {
		return false
	}
	t := f.toks[p]
	switch {
	case t.kind == token.Ident, t.kind == token.Variable, f.is(p, ")"),
		f.isKeyword(p, "isset"), f.isKeyword(p, "unset"), f.isKeyword(p, "static"):
	default:
		return false
	}
	return !f.isFunctionToken(f.prevEffective(p - 1))
}

// isFunctionToken tells T_FUNCTION, T_CLOSURE and T_FN (an "fn" opening an
// arrow function).
func (f *slvFile) isFunctionToken(i int) bool {
	if f.isKeyword(i, "function") {
		return true
	}
	return f.isKeyword(i, "fn") && f.functionParenOpen(i) >= 0
}

func (f *slvFile) isClosureToken(i int) bool {
	return f.isKeyword(i, "function") && f.isClosure(i)
}

// functionParenOpen returns the "(" of the parameters of the function token at
// i, or -1.
func (f *slvFile) functionParenOpen(i int) int {
	n := f.nextEffective(i+1, len(f.toks))
	if f.is(n, "&") {
		n = f.nextEffective(n+1, len(f.toks))
	}
	if f.isKeyword(i, "function") && n >= 0 && (f.toks[n].kind == token.Ident || f.toks[n].kind == token.Keyword) {
		n = f.nextEffective(n+1, len(f.toks))
	}
	if !f.is(n, "(") || f.toks[n].closer < 0 {
		return -1
	}
	return n
}

// closureUseOpen returns the "(" of the "use" of the closure at i, or -1.
func (f *slvFile) closureUseOpen(i int) int {
	open := f.functionParenOpen(i)
	if open < 0 {
		return -1
	}
	use := f.nextEffective(f.toks[open].closer+1, len(f.toks))
	if !f.isKeyword(use, "use") {
		return -1
	}
	n := f.nextEffective(use+1, len(f.toks))
	if !f.is(n, "(") || f.toks[n].closer < 0 {
		return -1
	}
	return n
}
