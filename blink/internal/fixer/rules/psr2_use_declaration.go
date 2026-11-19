package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Namespaces/UseDeclarationSniff.php
//
// PSR2UseDeclaration splits multi-name and grouped imports into one "use" per
// name, puts a single space after "use" and one blank line after the last one.
type PSR2UseDeclaration struct{}

func (PSR2UseDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Namespaces\UseDeclarationSniff`
}

func (PSR2UseDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Namespaces/UseDeclarationSniff.php"
}

func (PSR2UseDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if phpcsIsUse(f, i) {
				psr2UseDeclaration(f, i)
			}
		}
	})
}

func psr2UseDeclaration(f *slvFile, ptr int) {
	if psr2IgnoreUse(f, ptr) {
		return
	}
	n := len(f.toks)
	if ptr+1 < n && f.toks[ptr+1].content != " " {
		f.replaceToken(ptr+1, " ")
	}

	next := -1
	for j := ptr + 1; j < n; j++ {
		if f.is(j, ",") || f.is(j, ";") || f.toks[j].kind == token.CloseTag || phpcsIsUseGroupOpen(f, j) {
			next = j
			break
		}
	}
	if next >= 0 && f.is(next, ",") {
		baseUse := "use"
		if ptr+2 < n {
			switch f.toks[ptr+2].content {
			case "const":
				baseUse = "use const"
			case "function":
				baseUse = "use function"
			}
		}
		if next+1 < n && f.toks[next+1].kind != token.Whitespace {
			baseUse += " "
		}
		f.replaceToken(next, ";"+f.eol+baseUse)
	} else if next >= 0 && f.is(next, "{") {
		psr2SplitUseGroup(f, ptr, next)
	}

	nextUse := psr2NextUse(f, ptr+1)
	for nextUse >= 0 && psr2IgnoreUse(f, nextUse) {
		nextUse = psr2NextUse(f, nextUse+1)
	}
	if nextUse >= 0 {
		return
	}

	end := -1
	for j := ptr + 1; j < n; j++ {
		if f.is(j, ";") || f.toks[j].kind == token.CloseTag || phpcsIsUseGroupClose(f, j) {
			end = j
			break
		}
	}
	if end < 0 {
		return
	}
	if f.is(end, "}") {
		if ne := f.nextEffective(end+1, n); ne >= 0 && f.is(ne, ";") {
			end = ne
		}
	}
	for end++; end < n; end++ {
		if !f.isEmpty(end) || f.toks[end].column == 1 {
			break
		}
	}
	end--
	if t := f.toks[end]; t.kind == token.Comment && strings.HasPrefix(t.content, "/*") && !strings.HasSuffix(t.content, "*/") {
		end--
	}
	next = f.nextNonWhitespace(end+1, n)
	if next < 0 || f.toks[next].kind == token.CloseTag {
		return
	}
	diff := max(f.toks[next].line-f.toks[end].line-1, 0)
	if f.toks[next].line-f.toks[end].line-1 == 1 {
		return
	}
	if diff == 0 {
		f.addContent(end, f.eol)
		return
	}
	f.beginChangeset()
	for i := end + 1; i < next; i++ {
		if f.toks[i].line == f.toks[next].line {
			break
		}
		f.replaceToken(i, "")
	}
	f.addContent(end, f.eol)
	f.endChangeset()
}

func psr2SplitUseGroup(f *slvFile, ptr, next int) {
	n := len(f.toks)
	closingCurly := -1
	for j := next + 1; j < n; j++ {
		if phpcsIsUseGroupClose(f, j) {
			closingCurly = j
			break
		}
	}
	if closingCurly < 0 {
		return
	}
	baseUse := strings.TrimRight(f.content(ptr, next-1), " \t\n\r\x00\x0B")
	lastNonWhitespace := f.prevNonWhitespace(closingCurly - 1)

	f.beginChangeset()
	for i := ptr; i <= next; i++ {
		f.replaceToken(i, "")
	}
	if next+1 < n && psr2OnlyNewlines(f.toks[next+1].content) {
		f.replaceToken(next+1, "")
	}
	for next >= 0 {
		next = f.nextEffective(next+1, closingCurly)
		if next < 0 {
			break
		}
		nonWhitespace := f.prevNonWhitespace(next - 1)
		for i := nonWhitespace + 1; i < next; i++ {
			if psr2OnlyNewlines(f.toks[i].content) {
				continue
			}
			f.replaceToken(i, "")
		}
		if c := f.toks[next].content; c == "const" || c == "function" {
			f.addContentBefore(next, "use ")
			next = f.nextEffective(next+1, closingCurly)
			f.addContentBefore(next, strings.ReplaceAll(baseUse, "use ", ""))
		} else {
			f.addContentBefore(next, baseUse)
		}

		comma := -1
		for j := next + 1; j < closingCurly; j++ {
			if f.is(j, ",") {
				comma = j
				break
			}
		}
		next = comma
		if next < 0 {
			f.addContent(lastNonWhitespace, ";")
			break
		}
		nextNonEmpty := f.nextEffective(next+1, closingCurly)
		if nextNonEmpty >= 0 && f.toks[nextNonEmpty].line == f.toks[next].line {
			prevNonWhitespace := next
			for j := nextNonEmpty - 1; j > next; j-- {
				if f.toks[j].kind != token.Whitespace {
					prevNonWhitespace = j
					break
				}
			}
			if prevNonWhitespace == next {
				f.replaceToken(next, ";"+f.eol)
			} else {
				f.replaceToken(next, ";")
				f.addContent(prevNonWhitespace, f.eol)
			}
		} else {
			f.replaceToken(next, ";")
		}
	}

	next = f.nextEffective(closingCurly+1, n)
	if next < 0 || !f.is(next, ";") {
		next = closingCurly
	}
	for i := lastNonWhitespace + 1; i <= next; i++ {
		f.replaceToken(i, "")
	}
	f.endChangeset()
}

func psr2OnlyNewlines(v string) bool {
	return v != "" && strings.Trim(v, "\r\n") == ""
}

func psr2NextUse(f *slvFile, from int) int {
	for j := from; j < len(f.toks); j++ {
		if phpcsIsUse(f, j) {
			return j
		}
	}
	return -1
}

// phpcsIsUse tells T_USE from a "use" member or method name.
func phpcsIsUse(f *slvFile, i int) bool {
	if !f.isKeyword(i, "use") {
		return false
	}
	p := f.prevEffective(i - 1)
	return !f.is(p, "->") && !f.is(p, "?->") && !f.is(p, "::") && !f.isKeyword(p, "function")
}

// psr2IgnoreUse skips closure "use", a "use" at the end of the file and a
// trait import in a class, trait or enum (an anonymous class is no T_CLASS).
func psr2IgnoreUse(f *slvFile, ptr int) bool {
	next := f.nextEffective(ptr+1, len(f.toks))
	if next < 0 || f.is(next, "(") {
		return true
	}
	for scope := f.enclosingScope(ptr); scope >= 0; scope = f.enclosingScope(scope) {
		owner := f.braceOwner(scope)
		if owner < 0 {
			continue
		}
		switch strings.ToLower(f.toks[owner].orig) {
		case "trait", "enum":
			return true
		case "class":
			if !f.isKeyword(f.prevEffective(owner-1), "new") {
				return true
			}
		}
	}
	return false
}

// phpcsIsUseGroupOpen tells T_OPEN_USE_GROUP, the "{" after the "\" of a group
// import, from other braces.
func phpcsIsUseGroupOpen(f *slvFile, i int) bool {
	return f.is(i, "{") && f.is(f.prevEffective(i-1), "\\")
}

func phpcsIsUseGroupClose(f *slvFile, i int) bool {
	return f.is(i, "}") && f.toks[i].closer >= 0 && phpcsIsUseGroupOpen(f, f.toks[i].closer)
}
