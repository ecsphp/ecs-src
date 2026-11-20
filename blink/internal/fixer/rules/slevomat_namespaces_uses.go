package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/AlphabeticallySortedUsesSniff.php
// SlevomatAlphabeticallySortedUses sorts the use statements of each namespace.
type SlevomatAlphabeticallySortedUses struct {
	notPsr12      bool
	caseSensitive bool
}

func (SlevomatAlphabeticallySortedUses) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\AlphabeticallySortedUsesSniff`
}

func (SlevomatAlphabeticallySortedUses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAlphabeticallySortedUses{}.Name())
}

func (r SlevomatAlphabeticallySortedUses) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["psr12Compatible"].(bool); ok {
		r.notPsr12 = !v
	}
	if v, ok := config["caseSensitive"].(bool); ok {
		r.caseSensitive = v
	}
	return r
}

func (r SlevomatAlphabeticallySortedUses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.nsuIsUseGroupOpen(i) {
				return
			}
		}
		for _, g := range f.nsuFileUses() {
			for n := 1; n < len(g.uses); n++ {
				if r.compare(g.uses[n], g.uses[n-1]) >= 0 {
					continue
				}
				for _, u := range g.uses {
					if f.is(f.nsuNextOf(u.pointer, ";", ","), ",") {
						return
					}
				}
				r.fix(f, g.uses)
				return
			}
		}
	})
}

func (r SlevomatAlphabeticallySortedUses) fix(f *slvFile, uses []*nsuUse) {
	first := uses[0].pointer
	lastSemicolon := f.nsuNextOf(uses[len(uses)-1].pointer, ";")
	comments := map[int]string{}
	for _, u := range uses {
		before := f.prevNonWhitespace(u.pointer - 1)
		if !f.nsuIsComment(before) {
			continue
		}
		if strings.HasSuffix(f.content(before, u.pointer-1), f.eol+f.eol) {
			continue
		}
		start := f.nsuCommentStart(before)
		comments[u.pointer] = f.content(start, before)
		if first == u.pointer {
			first = start
		}
	}
	sorted := slices.Clone(uses)
	slices.SortStableFunc(sorted, r.compare)
	lines := make([]string, len(sorted))
	for n, u := range sorted {
		comment := comments[u.pointer]
		if comment != "" && !strings.HasSuffix(comment, f.eol) {
			comment += f.eol
		}
		kind := ""
		if u.kind != "" {
			kind = u.kind + " "
		}
		if nsuLastPart(u.fqn) == u.name {
			lines[n] = comment + "use " + kind + u.fqn + ";"
		} else {
			lines[n] = comment + "use " + kind + u.fqn + " as " + u.name + ";"
		}
	}
	f.beginChangeset()
	f.removeBetweenIncluding(first, lastSemicolon)
	f.add(first, strings.Join(lines, f.eol))
	f.endChangeset()
}

func (r SlevomatAlphabeticallySortedUses) compare(a, b *nsuUse) int {
	if a.kind != b.kind {
		order := func(kind string) int {
			switch kind {
			case "function":
				if r.notPsr12 {
					return 3
				}
				return 2
			case "const":
				if r.notPsr12 {
					return 2
				}
				return 3
			}
			return 1
		}
		return order(a.kind) - order(b.kind)
	}
	ap := strings.Split(strings.TrimLeft(a.fqn, `\`), `\`)
	bp := strings.Split(strings.TrimLeft(b.fqn, `\`), `\`)
	for n := range min(len(ap), len(bp)) {
		c := 0
		if r.caseSensitive {
			c = strings.Compare(ap[n], bp[n])
		} else {
			c = nsuStrcasecmp(ap[n], bp[n])
		}
		if c != 0 {
			return c
		}
	}
	return len(ap) - len(bp)
}

// nsuStrcasecmp is PHP's strcasecmp: bytes compared after ASCII lowercasing.
func nsuStrcasecmp(a, b string) int {
	lower := func(c byte) byte {
		if c >= 'A' && c <= 'Z' {
			return c + 'a' - 'A'
		}
		return c
	}
	for n := 0; n < len(a) && n < len(b); n++ {
		if ca, cb := lower(a[n]), lower(b[n]); ca != cb {
			return int(ca) - int(cb)
		}
	}
	return len(a) - len(b)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/NamespaceDeclarationSniff.php
// SlevomatNamespaceDeclaration wants one space after "namespace", nothing
// between the name and ";", and no bracketed namespaces.
type SlevomatNamespaceDeclaration struct{}

func (SlevomatNamespaceDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\NamespaceDeclarationSniff`
}

func (SlevomatNamespaceDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNamespaceDeclaration{}.Name())
}

func (SlevomatNamespaceDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "namespace") }, func(ns int) int {
			if ws := ns + 1; ws < len(f.toks) && f.toks[ws].kind == token.Whitespace && f.toks[ws].content != " " {
				f.beginChangeset()
				f.replace(ws, " ")
				f.endChangeset()
			}
			opener := f.nsuNamespaceOpener(ns)
			if opener < 0 {
				start := f.nextEffective(ns+1, len(f.toks))
				if start >= 0 {
					end := start
					for f.isName(end + 1) {
						end++
					}
					semicolon := f.nsuNextOf(end+1, ";")
					if semicolon >= 0 && end+1 != semicolon {
						f.beginChangeset()
						f.removeBetweenIncluding(end+1, semicolon-1)
						f.endChangeset()
					}
				}
				return 0
			}
			f.beginChangeset()
			f.replace(opener, ";")
			f.replace(f.toks[opener].closer, "")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/NamespaceSpacingSniff.php
// SlevomatNamespaceSpacing sets the blank lines before and after a namespace
// declaration.
type SlevomatNamespaceSpacing struct {
	before, after int
	configured    bool
}

func (SlevomatNamespaceSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\NamespaceSpacingSniff`
}

func (SlevomatNamespaceSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNamespaceSpacing{}.Name())
}

func (r SlevomatNamespaceSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.before = configInt(config, "linesCountBeforeNamespace", r.before)
	r.after = configInt(config, "linesCountAfterNamespace", r.after)
	return r
}

func (r SlevomatNamespaceSpacing) withDefaults() SlevomatNamespaceSpacing {
	if !r.configured {
		r.before, r.after, r.configured = 1, 1, true
	}
	return r
}

var nsuLineComment = regexp.MustCompile(`^(?://|#)`)

func (r SlevomatNamespaceSpacing) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "namespace") }, func(ns int) int {
			r.checkBefore(f, ns)
			r.checkAfter(f, ns)
			return 0
		})
	})
}

func (r SlevomatNamespaceSpacing) checkBefore(f *slvFile, ns int) {
	before := f.prevNonWhitespace(ns - 1)
	if before < 0 {
		return
	}
	ws := ""
	isOpenTag := f.toks[before].kind == token.OpenTag
	isLineComment := nsuLineComment.MatchString(f.toks[before].content)
	switch {
	case isOpenTag:
		ws += nsuOpenTagRest(f.toks[before].content)
	case isLineComment:
		ws += f.eol
	}
	if before+1 != ns {
		ws += f.content(before+1, ns-1)
	}
	if strings.Count(ws, f.eol)-1 == r.before {
		return
	}
	f.beginChangeset()
	switch {
	case isOpenTag:
		f.replace(before, "<?php")
	case isLineComment:
		f.replace(before, strings.TrimRight(f.toks[before].content, f.eol))
	}
	f.removeBetweenIncluding(before+1, ns-1)
	for range r.before + 1 {
		f.add(before, f.eol)
	}
	f.endChangeset()
}

func (r SlevomatNamespaceSpacing) checkAfter(f *slvFile, ns int) {
	if f.nsuNamespaceOpener(ns) >= 0 {
		return
	}
	semicolon := f.nsuNextOf(ns+1, ";")
	if semicolon < 0 {
		return
	}
	after := f.nextNonWhitespace(semicolon+1, len(f.toks))
	if after < 0 {
		return
	}
	if strings.Count(f.content(semicolon+1, after-1), f.eol)-1 == r.after {
		return
	}
	f.beginChangeset()
	f.removeBetweenIncluding(semicolon+1, after-1)
	for range r.after + 1 {
		f.add(semicolon, f.eol)
	}
	f.endChangeset()
}

// nsuOpenTagRest is substr($openTag, strlen('<?php')): the whitespace PHP
// keeps in the open tag; blink lexes it as the next token instead.
func nsuOpenTagRest(v string) string {
	if len(v) <= 5 {
		return ""
	}
	return v[5:]
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/UseDoesNotStartWithBackslashSniff.php
// SlevomatUseDoesNotStartWithBackslash drops the leading "\" of an imported name.
type SlevomatUseDoesNotStartWithBackslash struct{}

func (SlevomatUseDoesNotStartWithBackslash) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\UseDoesNotStartWithBackslashSniff`
}

func (SlevomatUseDoesNotStartWithBackslash) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUseDoesNotStartWithBackslash{}.Name())
}

func (SlevomatUseDoesNotStartWithBackslash) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "use") }, func(use int) int {
			if !f.nsuIsImportUse(use) {
				return 0
			}
			next := f.nextEffective(use+1, len(f.toks))
			if next >= 0 && (f.toks[next].orig == "function" || f.toks[next].orig == "const") {
				next = f.nextEffective(next+1, len(f.toks))
			}
			if next < 0 || !f.isName(next) || !strings.HasPrefix(f.toks[next].orig, `\`) {
				return 0
			}
			f.beginChangeset()
			f.replace(next, strings.TrimLeft(f.toks[next].content, `\`))
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/UseFromSameNamespaceSniff.php
// SlevomatUseFromSameNamespace removes a use of a name from the current namespace.
type SlevomatUseFromSameNamespace struct{}

func (SlevomatUseFromSameNamespace) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\UseFromSameNamespaceSniff`
}

func (SlevomatUseFromSameNamespace) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUseFromSameNamespace{}.Name())
}

func (SlevomatUseFromSameNamespace) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "use") }, func(use int) int {
			if !f.nsuIsImportUse(use) {
				return 0
			}
			namespace := f.nsuCurrentNamespaceName(use)
			used := f.useName(use)
			if !strings.HasPrefix(used, namespace) {
				return 0
			}
			for j := use; j < len(f.toks); j++ {
				if f.isKeyword(j, "as") {
					return 0
				}
				if f.is(j, ";") {
					break
				}
			}
			rest := used[len(namespace):]
			if !strings.HasPrefix(rest, `\`) && namespace != "" {
				return 0
			}
			if strings.Contains(strings.TrimLeft(rest, `\`), `\`) {
				return 0
			}
			semicolon := f.nsuNextOf(use, ";")
			if semicolon < 0 {
				return 0
			}
			f.beginChangeset()
			f.removeBetweenIncluding(use, min(semicolon+1, len(f.toks)-1))
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/UseSpacingSniff.php
// SlevomatUseSpacing sets the blank lines around and between use statements.
type SlevomatUseSpacing struct {
	before, between, after int
	configured             bool
}

func (SlevomatUseSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\UseSpacingSniff`
}

func (SlevomatUseSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUseSpacing{}.Name())
}

func (r SlevomatUseSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.before = configInt(config, "linesCountBeforeFirstUse", r.before)
	r.between = configInt(config, "linesCountBetweenUseTypes", r.between)
	r.after = configInt(config, "linesCountAfterLastUse", r.after)
	return r
}

func (r SlevomatUseSpacing) withDefaults() SlevomatUseSpacing {
	if !r.configured {
		r.before, r.between, r.after, r.configured = 1, 0, 1, true
	}
	return r
}

func (r SlevomatUseSpacing) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		for _, g := range f.nsuFileUses() {
			r.checkBefore(f, g.uses[0])
			r.checkAfter(f, g.uses[len(g.uses)-1])
			r.checkBetween(f, g.uses, true)
			r.checkBetween(f, g.uses, false)
		}
	})
}

func (r SlevomatUseSpacing) checkBefore(f *slvFile, first *nsuUse) {
	before := f.prevNonWhitespace(first.pointer - 1)
	start := first.pointer
	if f.nsuIsComment(before) && f.toks[before].line+1 == f.toks[start].line {
		start = f.nsuCommentStart(before)
		before = f.prevNonWhitespace(start - 1)
	}
	if before < 0 || f.toks[start].line-f.toks[before].line-1 == r.before {
		return
	}
	f.beginChangeset()
	if f.toks[before].kind == token.OpenTag {
		f.replace(before, "<?php")
	}
	f.removeBetweenIncluding(before+1, start-1)
	for range r.before + 1 {
		f.add(before, f.eol)
	}
	f.endChangeset()
}

func (r SlevomatUseSpacing) checkAfter(f *slvFile, last *nsuUse) {
	end := f.nsuNextOf(last.pointer+1, ";")
	if end < 0 {
		return
	}
	after := f.nextNonWhitespace(end+1, len(f.toks))
	if after < 0 {
		return
	}
	if f.toks[after].kind == token.Comment && (f.toks[end].line == f.toks[after].line || f.toks[end].line+1 == f.toks[after].line) {
		end = f.nsuCommentEnd(after)
		after = f.nextNonWhitespace(end+1, len(f.toks))
		if after < 0 {
			return
		}
	}
	if f.toks[after].line-f.toks[end].line-1 == r.after {
		return
	}
	f.beginChangeset()
	f.removeBetweenIncluding(end+1, after-1)
	lines := r.after
	if nsuLineComment.MatchString(f.toks[end].content) {
		lines--
	}
	for range lines + 1 {
		f.add(end, f.eol)
	}
	f.endChangeset()
}

// checkBetween is checkLinesBetweenSameTypesOfUse (same) and
// checkLinesBetweenDifferentTypesOfUse.
func (r SlevomatUseSpacing) checkBetween(f *slvFile, uses []*nsuUse, same bool) {
	if len(uses) == 1 {
		return
	}
	want := 0
	if !same {
		want = r.between
	}
	var prev *nsuUse
	for _, use := range uses {
		if prev == nil {
			prev = use
			continue
		}
		if (use.kind == prev.kind) != same {
			if same {
				prev = nil
			} else {
				prev = use
			}
			continue
		}
		before := f.prevNonWhitespace(use.pointer - 1)
		start := use.pointer
		if f.nsuIsComment(before) && f.nsuFirstNonWhitespaceOnLine(before) == before && f.toks[before].line+1 == f.toks[start].line {
			start = f.nsuCommentStart(before)
		}
		if f.toks[start].line-f.toks[prev.pointer].line-1 == want {
			prev = use
			continue
		}
		semicolon := f.nsuNextOf(prev.pointer+1, ";")
		f.beginChangeset()
		f.removeBetweenIncluding(semicolon+1, start-1)
		for range want + 1 {
			f.add(semicolon, f.eol)
		}
		f.endChangeset()
		prev = use
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/UselessAliasSniff.php
// SlevomatUselessAlias drops an alias equal to the imported name.
type SlevomatUselessAlias struct{}

func (SlevomatUselessAlias) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\UselessAliasSniff`
}

func (SlevomatUselessAlias) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessAlias{}.Name())
}

func (SlevomatUselessAlias) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for _, g := range f.nsuFileUses() {
			for _, u := range g.uses {
				if !u.hasAlias || nsuLastPart(u.fqn) != u.alias {
					continue
				}
				as := u.pointer + 1
				for as < len(f.toks) && !f.isKeyword(as, "as") {
					as++
				}
				nameEnd := as - 1
				for nameEnd > 0 && !f.isName(nameEnd) {
					nameEnd--
				}
				semicolon := f.nsuNextOf(as+1, ";")
				if as >= len(f.toks) || semicolon < 0 {
					continue
				}
				f.beginChangeset()
				f.removeBetweenIncluding(nameEnd+1, semicolon-1)
				f.endChangeset()
			}
		}
	})
}

// nsuUse is slevomat's UseStatement.
type nsuUse struct {
	pointer  int
	name     string // as referenced in the file: the alias or the last name part
	fqn      string
	kind     string // "", "function" or "const"
	alias    string
	hasAlias bool
}

type nsuUseGroup struct {
	anchor int
	uses   []*nsuUse
}

// nsuFileUses is UseStatementHelper::getFileUseStatements: the import uses
// keyed by the open tag or namespace before them, in file order; a repeated
// name keeps its first place and takes the later statement.
func (f *slvFile) nsuFileUses() []nsuUseGroup {
	var anchors []int
	for i, t := range f.toks {
		if t.kind == token.OpenTag || f.isKeyword(i, "namespace") {
			anchors = append(anchors, i)
		}
	}
	if len(anchors) == 0 {
		return nil
	}
	var groups []nsuUseGroup
	index := map[int]map[string]int{}
	for i := anchors[0] + 1; i < len(f.toks); i++ {
		if f.isClassTypeKeyword(i) {
			if c := f.scopeCloser(i); c > i {
				i = c
			}
			continue
		}
		if !f.isKeyword(i, "use") || f.nsuIsGroupUse(i) || !f.nsuIsImportUse(i) {
			continue
		}
		anchor := anchors[0]
		if len(anchors) > 1 {
			for _, a := range slices.Backward(anchors) {
				if a < i {
					anchor = a
					break
				}
			}
		}
		u := &nsuUse{pointer: i, fqn: f.useName(i)}
		if n := f.nextEffective(i+1, len(f.toks)); n >= 0 && (f.toks[n].orig == "const" || f.toks[n].orig == "function") {
			u.kind = f.toks[n].orig
		}
		u.alias, u.hasAlias = f.nsuAlias(i)
		u.name = nsuLastPart(u.fqn)
		if u.hasAlias {
			u.name = u.alias
		}
		id := strings.ToLower(u.name)
		switch u.kind {
		case "const":
			id = "const " + u.name
		case "function":
			id = "function " + id
		}
		if index[anchor] == nil {
			groups = append(groups, nsuUseGroup{anchor: anchor})
			index[anchor] = map[string]int{}
		}
		g := &groups[len(groups)-1]
		for n := range groups {
			if groups[n].anchor == anchor {
				g = &groups[n]
			}
		}
		if n, ok := index[anchor][id]; ok {
			g.uses[n] = u
			continue
		}
		index[anchor][id] = len(g.uses)
		g.uses = append(g.uses, u)
	}
	return groups
}

// nsuAlias is UseStatementHelper::getAlias.
func (f *slvFile) nsuAlias(use int) (string, bool) {
	end := f.nsuNextOf(use+1, ";", ",")
	if end < 0 {
		end = len(f.toks)
	}
	for j := use + 1; j < end; j++ {
		if f.isKeyword(j, "as") {
			for k := j + 1; k < len(f.toks); k++ {
				if f.toks[k].kind == token.Ident {
					return f.toks[k].content, true
				}
			}
			return "", true
		}
	}
	return "", false
}

// nsuIsGroupUse is UseStatementHelper::isGroupUse.
func (f *slvFile) nsuIsGroupUse(use int) bool {
	for j := use + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return false
		}
		if f.nsuIsUseGroupOpen(j) {
			return true
		}
	}
	return false
}

// nsuIsUseGroupOpen tells T_OPEN_USE_GROUP, the "{" after "use A\".
func (f *slvFile) nsuIsUseGroupOpen(i int) bool {
	return f.is(i, "{") && f.is(f.prevEffective(i-1), `\`)
}

// nsuIsImportUse is UseStatementHelper::isImportUse.
func (f *slvFile) nsuIsImportUse(use int) bool {
	next := f.nextEffective(use+1, len(f.toks))
	if next < 0 || f.is(next, "(") {
		return false
	}
	if k := f.toks[next].kind; k == token.Keyword || k == token.Ident {
		if v := strings.ToLower(f.toks[next].orig); v == "function" || v == "const" {
			return true
		}
	}
	prev := -1
	for j := use; j >= 0; j-- {
		if f.toks[j].kind == token.OpenTag || f.isKeyword(j, "declare") || f.isKeyword(j, "namespace") ||
			(f.is(j, "{") || f.is(j, "}")) && !f.nsuIsUseGroupOpen(j) && !f.nsuIsUseGroupClose(j) {
			prev = j
			break
		}
	}
	if prev < 0 || !f.is(prev, "{") && !f.is(prev, "}") {
		return true
	}
	if f.is(prev, "}") && !f.closesScope(prev) || f.is(prev, "{") && f.toks[prev].closer >= 0 && !f.closesScope(f.toks[prev].closer) {
		return true
	}
	condition := f.scopeCondition(prev)
	if condition < 0 {
		return true
	}
	if f.is(prev, "{") && f.nsuIsClassType(condition) {
		return false
	}
	if f.isKeyword(condition, "use") {
		return false
	}
	if f.isKeyword(condition, "function") && !f.isClosure(condition) {
		return !f.nsuInClass(use)
	}
	return true
}

func (f *slvFile) nsuIsUseGroupClose(i int) bool {
	return f.is(i, "}") && f.toks[i].closer >= 0 && f.nsuIsUseGroupOpen(f.toks[i].closer)
}

// nsuIsClassType tells TokenHelper::CLASS_TYPE_WITH_ANONYMOUS_CLASS_TOKEN_CODES.
func (f *slvFile) nsuIsClassType(i int) bool {
	return f.isKeyword(i, "class") || f.isKeyword(i, "interface") || f.isKeyword(i, "trait") || f.isKeyword(i, "enum")
}

// nsuInClass is ClassHelper::getClassPointer() !== null.
func (f *slvFile) nsuInClass(i int) bool {
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		if f.isClassTypeKeyword(f.braceOwner(open)) {
			return true
		}
	}
	return false
}

// nsuNamespaceOpener returns the "{" of a bracketed namespace, or -1.
func (f *slvFile) nsuNamespaceOpener(ns int) int {
	n := f.nextEffective(ns+1, len(f.toks))
	if n >= 0 && f.isName(n) {
		n = f.nextEffective(n+1, len(f.toks))
	}
	if f.is(n, "{") && f.toks[n].closer >= 0 {
		return n
	}
	return -1
}

// nsuCurrentNamespaceName is NamespaceHelper::findCurrentNamespaceName, "" when
// there is none.
func (f *slvFile) nsuCurrentNamespaceName(i int) string {
	for j := i - 1; j >= 0; j-- {
		if !f.isKeyword(j, "namespace") {
			continue
		}
		start := f.nextEffective(j+1, len(f.toks))
		if start < 0 {
			return ""
		}
		end := start
		for f.isName(end + 1) {
			end++
		}
		return f.content(start, end)
	}
	return ""
}

// nsuNextOf returns the first token from i that is one of the punctuation
// marks, or -1.
func (f *slvFile) nsuNextOf(i int, marks ...string) int {
	for ; i >= 0 && i < len(f.toks); i++ {
		if f.toks[i].kind == token.Punct && slices.Contains(marks, f.toks[i].orig) {
			return i
		}
	}
	return -1
}

// nsuIsComment tells a Tokens::COMMENT_TOKENS token.
func (f *slvFile) nsuIsComment(i int) bool {
	return i >= 0 && (f.toks[i].kind == token.Comment || f.toks[i].kind == token.DocComment)
}

// nsuCommentStart is the docblock's comment_opener or
// CommentHelper::getMultilineCommentStartPointer.
func (f *slvFile) nsuCommentStart(end int) int {
	if f.toks[end].kind == token.DocComment {
		for j := end; j >= 0; j-- {
			if f.toks[j].doc == slvDocOpen {
				return j
			}
		}
		return end
	}
	start := end
	for {
		prev := -1
		for j := start - 1; j >= 0; j-- {
			if f.toks[j].kind == token.Comment {
				prev = j
				break
			}
		}
		if prev < 0 || f.toks[prev].line+1 != f.toks[start].line {
			return start
		}
		start = prev
	}
}

// nsuCommentEnd is CommentHelper::getMultilineCommentEndPointer.
func (f *slvFile) nsuCommentEnd(start int) int {
	end := start
	for {
		next := -1
		for j := end + 1; j < len(f.toks); j++ {
			if f.toks[j].kind == token.Comment {
				next = j
				break
			}
		}
		if next < 0 || f.toks[next].line-1 != f.toks[end].line {
			return end
		}
		end = next
	}
}

// nsuFirstNonWhitespaceOnLine is TokenHelper::findFirstNonWhitespaceOnLine,
// which also skips docblock whitespace.
func (f *slvFile) nsuFirstNonWhitespaceOnLine(i int) int {
	for j := f.firstOnLine(i); j < len(f.toks); j++ {
		if f.toks[j].kind != token.Whitespace && f.toks[j].doc != slvDocWhitespace {
			return j
		}
	}
	return -1
}

func nsuLastPart(name string) string {
	name = strings.TrimLeft(name, `\`)
	return name[strings.LastIndex(name, `\`)+1:]
}
