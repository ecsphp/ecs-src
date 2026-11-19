package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Files/IncludingFileSniff.php
//
// PEARIncludingFile drops the parentheses after include/require and uses
// include(_once) for conditional includes, require(_once) for the rest.
type PEARIncludingFile struct{}

func (PEARIncludingFile) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Files\IncludingFileSniff`
}

func (PEARIncludingFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Files/IncludingFileSniff.php"
}

var phpcsParenOwnerKeywords = map[string]bool{
	"array": true, "list": true, "function": true, "use": true, "while": true,
	"for": true, "foreach": true, "switch": true, "if": true, "elseif": true,
	"catch": true, "declare": true, "match": true, "isset": true, "empty": true,
	"unset": true, "eval": true, "exit": true, "die": true, "fn": true,
}

var phpcsAssignmentOps = map[string]bool{
	"=": true, "&=": true, "|=": true, ".=": true, "/=": true, "-=": true,
	"**=": true, "%=": true, "*=": true, "+=": true, "^=": true, "<<=": true,
	">>=": true, "??=": true,
}

func (PEARIncludingFile) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i, t := range f.toks {
			if t.kind != token.Keyword {
				continue
			}
			switch strings.ToLower(t.orig) {
			case "include", "include_once", "require", "require_once":
				pearIncludingFile(f, i)
			}
		}
	})
}

func pearIncludingFile(f *slvFile, i int) {
	if next := f.nextEffective(i+1, len(f.toks)); f.is(next, "(") && f.toks[next].closer >= 0 {
		f.beginChangeset()
		f.replaceToken(f.toks[next].closer, "")
		if f.toks[next-1].kind != token.Whitespace {
			f.replaceToken(next, " ")
		} else {
			f.replaceToken(next, "")
		}
		f.endChangeset()
	}

	inCondition := phpcsHasCondition(f, i)
	for o := f.enclosing(i); o >= 0 && !inCondition; o = f.enclosing(o) {
		if f.is(o, "(") && phpcsParenHasOwner(f, o) {
			inCondition = true
		}
	}
	if p := f.prevEffective(i - 1); p >= 0 && f.toks[p].kind == token.Punct &&
		(phpcsAssignmentOps[f.toks[p].orig] || f.toks[p].orig == "=>" && f.isDoubleArrow(p)) {
		inCondition = true
	}

	want := map[string]string{"require_once": "include_once", "require": "include"}
	if !inCondition {
		want = map[string]string{"include_once": "require_once", "include": "require"}
	}
	if v, ok := want[strings.ToLower(f.toks[i].orig)]; ok {
		f.replaceToken(i, v)
	}
}

// phpcsHasCondition reports whether PHP_CodeSniffer gives the token at i any
// condition: a scope owner (class, function, if, ...) whose braces hold it.
func phpcsHasCondition(f *slvFile, i int) bool {
	for o := f.enclosingScope(i); o >= 0; o = f.enclosingScope(o) {
		if f.braceOwner(o) >= 0 {
			return true
		}
	}
	for k := i - 1; k >= 0; k-- {
		if f.toks[k].kind != token.Keyword {
			continue
		}
		kw := strings.ToLower(f.toks[k].orig)
		endKw, ok := icsAltEnd[kw]
		if !ok || !f.icsIsAlt(k) {
			continue
		}
		q := k + 1
		if _, c := f.icsParens(k); c >= 0 {
			q = c + 1
		}
		q = f.nextEffective(q, len(f.toks))
		if closer := f.icsAltCloser(kw, q, endKw); closer > i {
			return true
		}
	}
	return false
}

// phpcsParenHasOwner reports whether the "(" at o has a parenthesis_owner.
func phpcsParenHasOwner(f *slvFile, o int) bool {
	p := f.prevEffective(o - 1)
	if p < 0 {
		return false
	}
	if f.toks[p].kind == token.Keyword && phpcsParenOwnerKeywords[strings.ToLower(f.toks[p].orig)] {
		return true
	}
	if f.is(p, "&") {
		p = f.prevEffective(p - 1)
	}
	if pp := f.prevEffective(p - 1); pp >= 0 && f.isKeyword(pp, "function") {
		return true
	}
	return f.isKeyword(p, "class") && f.isKeyword(f.prevEffective(p-1), "new")
}
