package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/DisallowDirectMagicInvokeCallSniff.php
//
// SlevomatDisallowDirectMagicInvokeCall turns "$a->__invoke()" into "$a()".
type SlevomatDisallowDirectMagicInvokeCall struct{}

func (SlevomatDisallowDirectMagicInvokeCall) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\DisallowDirectMagicInvokeCallSniff`
}

func (SlevomatDisallowDirectMagicInvokeCall) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowDirectMagicInvokeCall{}.Name())
}

func (SlevomatDisallowDirectMagicInvokeCall) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool {
			return f.toks[i].kind == token.Ident && strings.EqualFold(f.toks[i].orig, "__invoke")
		}, func(i int) int {
			open := f.nextEffective(i+1, len(f.toks))
			if !f.is(open, "(") {
				return 0
			}
			op := f.prevEffective(i - 1)
			if !f.is(op, "->") {
				return 0
			}
			f.beginChangeset()
			f.removeBetweenIncluding(op, open-1)
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/ReferenceSpacingSniff.php
//
// SlevomatReferenceSpacing sets the spaces after a reference "&"
// (spacesCountAfterReference, 0 by default).
type SlevomatReferenceSpacing struct {
	spaces int
}

func (SlevomatReferenceSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\ReferenceSpacingSniff`
}

func (SlevomatReferenceSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReferenceSpacing{}.Name())
}

func (r SlevomatReferenceSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.spaces = configInt(config, "spacesCountAfterReference", r.spaces)
	return r
}

func (r SlevomatReferenceSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.is(i, "&") }, func(i int) int {
			if !f.isReference(i) {
				return 0
			}
			after := f.nextNonWhitespace(i+1, len(f.toks))
			if after < 0 || len(f.content(i+1, after-1)) == r.spaces {
				return 0
			}
			f.beginChangeset()
			f.add(i, strings.Repeat(" ", r.spaces))
			f.removeBetweenIncluding(i+1, after-1)
			f.endChangeset()
			return 0
		})
	})
}

// isReference is ReferenceSpacingSniff::isReference.
func (f *slvFile) isReference(i int) bool {
	prev := f.prevEffective(i - 1)
	if f.isKeyword(prev, "function") || f.isKeyword(prev, "fn") {
		return true
	}
	open := -1
	for j := i - 1; j >= 0; j-- {
		if f.is(j, "(") {
			open = j
			break
		}
	}
	if open >= 0 && f.toks[open].closer > i {
		if f.isFunctionParams(open) {
			return true
		}
		if f.isKeyword(f.prevEffective(open-1), "use") {
			return true
		}
	}
	if f.identEnd(f.nextEffective(i+1, len(f.toks))) < 0 {
		return false
	}
	return f.is(prev, "=") || f.is(prev, "=>") || f.isShortArrayOpen(prev) || f.is(prev, ",") || f.isKeyword(prev, "as")
}

// isFunctionParams tells the "(" of a function, closure or arrow function's
// parameters, the parentheses PHP_CodeSniffer gives such an owner.
func (f *slvFile) isFunctionParams(open int) bool {
	p := f.prevEffective(open - 1)
	if f.is(p, "&") {
		p = f.prevEffective(p - 1)
	}
	if f.isKeyword(p, "function") || f.isKeyword(p, "fn") {
		return true
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

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/RequireNowdocSniff.php
//
// SlevomatRequireNowdoc turns a heredoc without variables or escapes into a
// nowdoc.
type SlevomatRequireNowdoc struct{}

func (SlevomatRequireNowdoc) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\RequireNowdocSniff`
}

func (SlevomatRequireNowdoc) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireNowdoc{}.Name())
}

var (
	slvHeredocPlain    = regexp.MustCompile(`^([^\\$]|\\[^nrtvef0-7xu])*$`)
	slvHeredocStart    = regexp.MustCompile(`^<<<"?(\w+)"?`)
	slvHeredocUnescape = regexp.MustCompile(`\\(\\[nrtvef]|\$|\\|\\[0-7]{1,3}|\\x[0-9A-Fa-f]{1,2}|\\u\{[0-9A-Fa-f]+\})`)
)

func (SlevomatRequireNowdoc) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool {
			t := f.toks[i]
			return t.kind == token.String && t.heredoc && strings.HasPrefix(t.orig, "<<<") &&
				!strings.HasPrefix(strings.TrimLeft(t.orig[3:], " \t"), "'")
		}, func(start int) int {
			end := start + 1
			for end < len(f.toks) && f.toks[end].heredoc && strings.HasSuffix(f.toks[end].orig, "\n") {
				end++
			}
			if end >= len(f.toks) || !f.toks[end].heredoc {
				return 0
			}
			for i := start + 1; i < end; i++ {
				if !slvHeredocPlain.MatchString(strings.TrimSuffix(f.toks[i].content, "\n")) {
					return 0
				}
			}
			f.beginChangeset()
			f.replace(start, slvHeredocStart.ReplaceAllString(f.toks[start].content, "<<<'$1'"))
			for i := start + 1; i < end; i++ {
				f.replace(i, slvHeredocUnescape.ReplaceAllString(f.toks[i].content, "$1"))
			}
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/ShortListSniff.php
//
// SlevomatShortList turns "list($a, $b)" into "[$a, $b]".
type SlevomatShortList struct{}

func (SlevomatShortList) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\ShortListSniff`
}

func (SlevomatShortList) SourceURL() string {
	return fixer.SourceURLFor(SlevomatShortList{}.Name())
}

func (SlevomatShortList) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "list") }, func(i int) int {
			open := -1
			for j := i + 1; j < len(f.toks); j++ {
				if f.is(j, "(") {
					open = j
					break
				}
			}
			if open < 0 || f.toks[open].closer < 0 {
				return 0
			}
			f.beginChangeset()
			f.removeBetweenIncluding(i, open-1)
			f.replace(open, "[")
			f.replace(f.toks[open].closer, "]")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/TypeCastSniff.php
//
// SlevomatTypeCast replaces the (boolean), (double), (integer) and (real)
// casts and drops (binary).
type SlevomatTypeCast struct{}

func (SlevomatTypeCast) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\TypeCastSniff`
}

func (SlevomatTypeCast) SourceURL() string {
	return fixer.SourceURLFor(SlevomatTypeCast{}.Name())
}

var (
	slvCastName     = regexp.MustCompile(`(?i)^\(\s*(\S+)\s*\)$`)
	slvInvalidCasts = map[string]string{"boolean": "bool", "double": "float", "integer": "int", "real": "float", "binary": ""}
)

func (SlevomatTypeCast) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool {
			t := f.toks[i]
			return t.kind == token.Punct && len(t.orig) > 2 && t.orig[0] == '(' && t.orig[len(t.orig)-1] == ')'
		}, func(i int) int {
			m := slvCastName.FindStringSubmatch(f.toks[i].content)
			if m == nil {
				return 0
			}
			name := strings.ToLower(m[1])
			replacement, ok := slvInvalidCasts[name]
			if !ok {
				return 0
			}
			f.beginChangeset()
			if name == "binary" {
				f.removeBetweenIncluding(i, f.nextEffective(i+1, len(f.toks))-1)
			} else {
				f.replace(i, "("+replacement+")")
			}
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/UselessSemicolonSniff.php
//
// SlevomatUselessSemicolon removes doubled semicolons, ones opening a scope
// and ones after a block.
type SlevomatUselessSemicolon struct{}

func (SlevomatUselessSemicolon) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\UselessSemicolonSniff`
}

func (SlevomatUselessSemicolon) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessSemicolon{}.Name())
}

func (SlevomatUselessSemicolon) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.is(i, ";") }, func(i int) int {
			prev := f.prevEffective(i - 1)
			if prev < 0 {
				return 0
			}
			if f.is(prev, ";") && !f.inForHeader(i) {
				f.removeUselessSemicolon(i)
			}
			if f.toks[prev].kind == token.OpenTag || f.is(prev, "{") {
				f.removeUselessSemicolon(i)
			}
			if f.is(prev, "}") && f.toks[prev].closer >= 0 {
				owner := f.scopeCondition(prev)
				if owner >= 0 && !f.isExpressionScope(owner) {
					f.removeUselessSemicolon(i)
				}
			}
			return 0
		})
	})
}

// isExpressionScope tells a closure, anonymous class or match, whose block
// ends an expression rather than a statement.
func (f *slvFile) isExpressionScope(owner int) bool {
	switch {
	case f.isKeyword(owner, "match"):
		return true
	case f.isKeyword(owner, "function"):
		return f.isClosure(owner)
	case f.isKeyword(owner, "class"):
		return f.isKeyword(f.prevEffective(owner-1), "new")
	}
	return false
}

// inForHeader tells a ";" inside the parentheses of a "for".
func (f *slvFile) inForHeader(semicolon int) bool {
	for j := semicolon + 1; j < len(f.toks) && !f.is(j, ";"); j++ {
		if f.is(j, ")") {
			open := f.toks[j].closer
			return open >= 0 && open < semicolon && f.isKeyword(f.prevEffective(open-1), "for")
		}
	}
	return false
}

func (f *slvFile) removeUselessSemicolon(semicolon int) {
	start := semicolon
	for start > 0 && f.toks[start-1].kind == token.Whitespace {
		start--
		if f.toks[start].content == f.eol {
			break
		}
	}
	end := semicolon
	for end < len(f.toks)-1 && f.toks[end+1].kind == token.Whitespace && f.toks[end+1].content != f.eol {
		end++
	}
	f.beginChangeset()
	f.removeBetweenIncluding(start, end)
	f.endChangeset()
}
