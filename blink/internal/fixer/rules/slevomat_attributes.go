package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Attributes/AttributeAndTargetSpacingSniff.php
//
// SlevomatAttributeAndTargetSpacing sets the blank lines between an attribute
// and its target (linesCount, 0 by default; allowOnSameLine).
type SlevomatAttributeAndTargetSpacing struct {
	lines         int
	allowSameLine bool
}

func (SlevomatAttributeAndTargetSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Attributes\AttributeAndTargetSpacingSniff`
}

func (SlevomatAttributeAndTargetSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAttributeAndTargetSpacing{}.Name())
}

func (r SlevomatAttributeAndTargetSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.lines = configInt(config, "linesCount", r.lines)
	if v, ok := config["allowOnSameLine"].(bool); ok {
		r.allowSameLine = v
	}
	return r
}

func (r SlevomatAttributeAndTargetSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isAttributeOpen, func(open int) int {
			if !f.isValidAttribute(open) {
				return 0
			}
			closeIdx := f.toks[open].closer
			after := f.nextNonWhitespace(closeIdx+1, len(f.toks))
			for after >= 0 && f.toks[after].kind == token.Comment {
				after = f.nextNonWhitespace(after+1, len(f.toks))
			}
			if after < 0 || f.isAttributeOpen(after) {
				return 0
			}
			sameLine := f.toks[after].line == f.toks[closeIdx].line
			if sameLine {
				if r.allowSameLine {
					return 0
				}
			} else if f.toks[after].line-f.toks[closeIdx].line-1 == r.lines {
				return 0
			}
			eols := strings.Repeat(f.eol, r.lines+1)
			f.beginChangeset()
			if sameLine {
				indentation := f.indentation(f.firstNonWhitespaceOnLine(after))
				for j := closeIdx + 1; j < len(f.toks); j++ {
					v := f.tokenContent(j)
					if v == "" || strings.TrimSpace(v) != "" {
						break
					}
					f.replace(j, "")
				}
				f.addBefore(after, eols+indentation)
			} else {
				first := f.firstOnLine(after)
				f.removeBetweenIncluding(closeIdx+1, first-1)
				f.addBefore(first, eols)
			}
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Attributes/AttributesOrderSniff.php
//
// SlevomatAttributesOrder sorts the attribute groups of a target by a given
// order (with "Prefix\" and "Prefix*" entries) or alphabetically.
type SlevomatAttributesOrder struct {
	order        []string
	alphabetical bool
}

func (SlevomatAttributesOrder) Name() string {
	return `SlevomatCodingStandard\Sniffs\Attributes\AttributesOrderSniff`
}

func (SlevomatAttributesOrder) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAttributesOrder{}.Name())
}

func (r SlevomatAttributesOrder) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["order"].([]any); ok {
		r.order = nil
		for _, item := range v {
			if s, ok := item.(string); ok {
				r.order = append(r.order, strings.TrimLeft(strings.TrimSpace(s), `\`))
			}
		}
	}
	if v, ok := config["orderAlphabetically"].(bool); ok {
		r.alphabetical = v
	}
	return r
}

func (r SlevomatAttributesOrder) Fix(s *tokens.Stream) bool {
	// upstream throws on a missing or a double order, so nothing is fixed
	if (len(r.order) == 0) != r.alphabetical {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		var uses *slvUseStatements
		f.eachToken(f.isAttributeOpen, func(open int) int {
			if !f.isValidAttribute(open) {
				return 0
			}
			if before := f.prevNonWhitespace(open - 1); before >= 0 && f.toks[before].kind == token.AttributeEnd {
				return 0
			}
			groups := [][]slvAttribute{f.attributes(open)}
			last := f.toks[open].closer
			for {
				next := f.nextNonWhitespace(last+1, len(f.toks))
				if !f.isAttributeOpen(next) {
					break
				}
				groups = append(groups, f.attributes(next))
				last = f.toks[next].closer
			}
			for _, g := range groups {
				if len(g) == 0 {
					return 0
				}
			}
			expected := make([]int, len(groups))
			for i := range expected {
				expected[i] = i
			}
			if r.alphabetical {
				slices.SortStableFunc(expected, func(a, b int) int {
					return strnatcmp(groups[a][0].name, groups[b][0].name)
				})
			} else {
				if uses == nil {
					uses = f.useStatements()
				}
				positions := make([]int, len(groups))
				for i, g := range groups {
					positions[i] = r.position(strings.TrimLeft(uses.resolveClassName(f, g[0].name, g[0].opener), `\`))
				}
				slices.SortStableFunc(expected, func(a, b int) int { return positions[a] - positions[b] })
			}
			if slices.IsSorted(expected) {
				return 0
			}
			contents := make([]string, len(groups))
			for i, g := range groups {
				contents[i] = f.content(g[0].opener, f.toks[g[0].opener].closer)
			}
			sameLine := f.toks[open].line == f.toks[last].line
			indentation := f.indentation(open)
			f.beginChangeset()
			f.removeBetweenIncluding(open, last)
			for pos, g := range expected {
				if pos != 0 {
					if sameLine {
						f.add(open, " ")
					} else {
						f.add(open, indentation)
					}
				}
				f.add(open, contents[g])
				if !sameLine && pos != len(groups)-1 {
					f.replaceToken(open, f.tokenContent(open)+f.eol)
				}
			}
			f.endChangeset()
			return 0
		})
	})
}

// position is the index of the first order entry name matches, or 999.
func (r SlevomatAttributesOrder) position(name string) int {
	for i, entry := range r.order {
		switch {
		case name == entry,
			strings.HasSuffix(entry, `\`) && strings.HasPrefix(name, entry),
			strings.HasSuffix(entry, "*") && strings.HasPrefix(name, entry[:len(entry)-1]):
			return i
		}
	}
	return 999
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Attributes/DisallowAttributesJoiningSniff.php
//
// SlevomatDisallowAttributesJoining splits "#[A, B]" into "#[A]#[B]".
type SlevomatDisallowAttributesJoining struct{}

func (SlevomatDisallowAttributesJoining) Name() string {
	return `SlevomatCodingStandard\Sniffs\Attributes\DisallowAttributesJoiningSniff`
}

func (SlevomatDisallowAttributesJoining) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowAttributesJoining{}.Name())
}

func (SlevomatDisallowAttributesJoining) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isAttributeOpen, func(open int) int {
			if !f.isValidAttribute(open) {
				return 0
			}
			attrs := f.attributes(open)
			if len(attrs) <= 1 {
				return 0
			}
			f.beginChangeset()
			for i := 1; i < len(attrs); i++ {
				prev, attr := attrs[i-1], attrs[i]
				f.add(prev.end, "]")
				for j := prev.end + 1; j < attr.start; j++ {
					if f.tokenContent(j) == "," {
						f.replace(j, "")
					}
				}
				f.addBefore(attr.start, "#[")
			}
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Attributes/DisallowMultipleAttributesPerLineSniff.php
//
// SlevomatDisallowMultipleAttributesPerLine moves each attribute group of a
// target onto its own line.
type SlevomatDisallowMultipleAttributesPerLine struct{}

func (SlevomatDisallowMultipleAttributesPerLine) Name() string {
	return `SlevomatCodingStandard\Sniffs\Attributes\DisallowMultipleAttributesPerLineSniff`
}

func (SlevomatDisallowMultipleAttributesPerLine) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowMultipleAttributesPerLine{}.Name())
}

func (SlevomatDisallowMultipleAttributesPerLine) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isAttributeOpen, func(open int) int {
			if !f.isValidAttribute(open) {
				return 0
			}
			closeIdx := f.toks[open].closer
			next := -1
			for j := closeIdx + 1; j < len(f.toks); j++ {
				if f.isAttributeOpen(j) {
					next = j
					break
				}
			}
			if next < 0 || f.toks[closeIdx].line != f.toks[next].line ||
				f.attributeTarget(open) != f.attributeTarget(next) {
				return 0
			}
			before := f.prevNonWhitespace(next - 1)
			indentation := f.indentation(f.firstNonWhitespaceOnLine(open))
			f.beginChangeset()
			f.removeBetweenIncluding(before+1, next-1)
			f.addBefore(next, f.eol+indentation)
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Attributes/RequireAttributeAfterDocCommentSniff.php
//
// SlevomatRequireAttributeAfterDocComment moves a doc comment written after
// attributes above them.
type SlevomatRequireAttributeAfterDocComment struct{}

func (SlevomatRequireAttributeAfterDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Attributes\RequireAttributeAfterDocCommentSniff`
}

func (SlevomatRequireAttributeAfterDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireAttributeAfterDocComment{}.Name())
}

func (SlevomatRequireAttributeAfterDocComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isAttributeOpen, func(open int) int {
			if !f.isValidAttribute(open) {
				return 0
			}
			doc := f.toks[open].closer + 1
			for doc < len(f.toks) && f.toks[doc].kind == token.Whitespace {
				doc++
			}
			if doc >= len(f.toks) || f.toks[doc].kind != token.DocComment {
				return 0
			}
			docEnd := doc
			for docEnd+1 < len(f.toks) && f.toks[docEnd+1].kind == token.DocComment {
				docEnd++
			}
			docStart := f.firstOnLine(doc)
			docEnd = f.lastOnLine(docEnd)
			comment := f.content(docStart, docEnd)
			first := open
			for {
				before := f.prevNonWhitespace(first - 1)
				if before < 0 || f.toks[before].kind != token.AttributeEnd || f.toks[before].closer < 0 {
					break
				}
				first = f.toks[before].closer
			}
			f.beginChangeset()
			f.addBefore(f.firstOnLine(first), comment)
			f.removeBetweenIncluding(docStart, docEnd)
			f.endChangeset()
			return 0
		})
	})
}

func (f *slvFile) isAttributeOpen(i int) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.AttributeStart && f.toks[i].closer >= 0
}

func (f *slvFile) removeBetweenIncluding(start, end int) {
	for i := start; i <= end; i++ {
		f.replaceToken(i, "")
	}
}

// slvAttribute is slevomat's Attribute: one attribute of a "#[...]" group.
type slvAttribute struct {
	opener     int
	name       string
	start, end int
}

// attributes is AttributeHelper::getAttributes for the group opened at open.
func (f *slvFile) attributes(open int) []slvAttribute {
	closeIdx := f.toks[open].closer
	var out []slvAttribute
	actual := open
	for actual >= 0 {
		start := f.nextEffective(actual+1, closeIdx)
		if start < 0 {
			break
		}
		end := start
		for f.isName(end + 1) {
			end++
		}
		attr := slvAttribute{opener: open, name: f.content(start, end), start: start, end: end}
		after := f.nextEffective(end+1, closeIdx)
		switch {
		case after < 0:
			return append(out, attr)
		case f.is(after, ","):
			out = append(out, attr)
			actual = after
		case f.is(after, "(") && f.toks[after].closer >= 0:
			attr.end = f.toks[after].closer
			out = append(out, attr)
			actual = f.nextEffective(attr.end+1, closeIdx)
		default:
			// upstream loops forever on such an attribute
			return out
		}
	}
	return out
}

func (f *slvFile) isValidAttribute(open int) bool {
	return f.attributeTarget(open) >= 0
}

// attributeTarget is AttributeHelper::getAttributeTarget, or -1.
func (f *slvFile) attributeTarget(open int) int {
	for i := open; i < len(f.toks); i++ {
		t := f.toks[i]
		if f.isVariable(i) {
			if f.isProperty(i) || f.closureIsParameter(i) {
				return i
			}
			return -1
		}
		if t.kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.orig) {
		case "class", "function", "fn", "const", "enum", "interface", "trait":
			return i
		case "case":
			if o := f.enclosing(i); o >= 0 && f.is(o, "{") && f.isKeyword(f.braceOwner(o), "enum") {
				return i
			}
		}
	}
	return -1
}

// isProperty is PropertyHelper::isProperty for a variable after an attribute.
func (f *slvFile) isProperty(v int) bool {
	p := v - 1
	for p >= 0 && (f.isEmpty(p) || f.isTypeHintToken(p)) {
		p--
	}
	if p < 0 {
		return false
	}
	if f.isKeyword(p, "final") || f.isKeyword(p, "abstract") {
		return true
	}
	if f.isKeyword(p, "static") {
		p = f.prevEffective(p - 1)
	}
	if f.isKeyword(p, "public") || f.isKeyword(p, "protected") || f.isKeyword(p, "private") || f.isKeyword(p, "readonly") {
		for o := f.enclosing(p); o >= 0; o = f.enclosing(o) {
			if f.is(o, "(") {
				fn := -1
				for j := p - 1; j >= 0; j-- {
					if f.isKeyword(j, "function") {
						fn = j
						break
					}
				}
				if fn < 0 {
					return true
				}
				open := f.functionParenOpen(fn)
				return open >= 0 && f.toks[open].closer < p
			}
		}
		return true
	}
	o := f.enclosing(v)
	for o >= 0 && !f.is(o, "{") {
		o = f.enclosing(o)
	}
	if o < 0 || !f.isOOScopeOwner(f.braceOwner(o)) {
		return false
	}
	for j := v - 1; j > o; j-- {
		switch {
		case f.isKeyword(j, "function"), f.isKeyword(j, "fn"):
			return false
		case f.is(j, ";"), f.is(j, "{"), f.is(j, "}"):
			return true
		}
	}
	return true
}

func (f *slvFile) isOOScopeOwner(i int) bool {
	for _, k := range []string{"class", "interface", "trait", "enum"} {
		if f.isKeyword(i, k) {
			return true
		}
	}
	return false
}

// isTypeHintToken tells a token of a property's type: a name, a type keyword,
// "?", "|", "&" or a DNF parenthesis.
func (f *slvFile) isTypeHintToken(i int) bool {
	t := f.toks[i]
	switch t.kind {
	case token.Ident:
		return true
	case token.Keyword:
		switch strings.ToLower(t.orig) {
		case "array", "callable", "static", "self", "parent":
			return !f.isKeyword(i, "static")
		}
	case token.Punct:
		switch t.orig {
		case "?", "|", "&", "(", ")":
			return true
		}
	}
	return false
}

// slvUseStatements are a file's import "use" statements, grouped by the open
// tag or namespace before them (UseStatementHelper::getFileUseStatements).
type slvUseStatements struct {
	groups []int
	byKey  map[int]map[string]string
}

func (f *slvFile) useStatements() *slvUseStatements {
	u := &slvUseStatements{byKey: map[int]map[string]string{}}
	var anchors []int
	for i, t := range f.toks {
		if t.kind == token.OpenTag || f.isKeyword(i, "namespace") {
			anchors = append(anchors, i)
		}
	}
	if len(anchors) == 0 {
		return u
	}
	for i := anchors[0] + 1; i < len(f.toks); i++ {
		if f.isClassTypeKeyword(i) {
			if c := f.scopeCloser(i); c > i {
				i = c
			}
			continue
		}
		if !f.isKeyword(i, "use") || f.isGroupUse(i) || !f.isImportUse(i) {
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
		kind := ""
		if n := f.nextEffective(i+1, len(f.toks)); n >= 0 {
			switch strings.ToLower(f.toks[n].orig) {
			case "const":
				kind = "const "
			case "function":
				kind = "function "
			}
		}
		fqn := f.useName(i)
		name := fqn[strings.LastIndex(fqn, `\`)+1:]
		if alias := f.useAlias(i); alias != "" {
			name = alias
		}
		id := kind + strings.ToLower(name)
		if kind == "const " {
			id = kind + name
		}
		if u.byKey[anchor] == nil {
			u.groups = append(u.groups, anchor)
			u.byKey[anchor] = map[string]string{}
		}
		u.byKey[anchor][id] = fqn
	}
	return u
}

// forPointer is UseStatementHelper::getUseStatementsForPointer.
func (u *slvUseStatements) forPointer(i int) map[string]string {
	if len(u.groups) == 1 {
		return u.byKey[u.groups[0]]
	}
	for _, g := range slices.Backward(u.groups) {
		if g < i {
			return u.byKey[g]
		}
	}
	return nil
}

// resolveClassName is NamespaceHelper::resolveClassName.
func (u *slvUseStatements) resolveClassName(f *slvFile, name string, pointer int) string {
	if strings.HasPrefix(name, `\`) {
		return name
	}
	uses := u.forPointer(pointer)
	canonical := strings.TrimLeft(name, `\`)
	if fqn, ok := uses[strings.ToLower(canonical)]; ok {
		return `\` + fqn
	}
	parts := strings.Split(canonical, `\`)
	if fqn, ok := uses[strings.ToLower(parts[0])]; ok && len(parts) > 1 {
		return `\` + fqn + `\` + strings.Join(parts[1:], `\`)
	}
	name = `\` + name
	ns := -1
	for i := pointer - 1; i >= 0; i-- {
		if f.isKeyword(i, "namespace") {
			ns = i
			break
		}
	}
	if ns >= 0 {
		if n := f.nextEffective(ns+1, len(f.toks)); n >= 0 {
			name = `\` + f.content(n, n) + name
		}
	}
	return name
}

func (f *slvFile) isClassTypeKeyword(i int) bool {
	if !f.isKeyword(i, "class") && !f.isKeyword(i, "interface") && !f.isKeyword(i, "trait") && !f.isKeyword(i, "enum") {
		return false
	}
	return !f.isKeyword(f.prevEffective(i-1), "new")
}

// isGroupUse tells "use A\{B, C};".
func (f *slvFile) isGroupUse(use int) bool {
	for j := use + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return false
		}
		if f.is(j, "{") {
			return true
		}
	}
	return false
}

// isImportUse is UseStatementHelper::isImportUse outside class bodies.
func (f *slvFile) isImportUse(use int) bool {
	n := f.nextEffective(use+1, len(f.toks))
	if f.is(n, "(") {
		return false
	}
	for j := use - 1; j >= 0; j-- {
		if f.toks[j].kind == token.OpenTag || f.isKeyword(j, "declare") || f.isKeyword(j, "namespace") {
			return true
		}
		if f.is(j, "{") || f.is(j, "}") {
			open := j
			if f.is(j, "}") {
				open = f.toks[j].closer
			}
			if open < 0 {
				return true
			}
			owner := f.braceOwner(open)
			if f.is(j, "{") && (f.isOOScopeOwner(owner)) {
				return false
			}
			return true
		}
	}
	return true
}

// useName is UseStatementHelper::getFullyQualifiedTypeNameFromUse.
func (f *slvFile) useName(use int) string {
	end := use + 1
	for end < len(f.toks) && !f.is(end, ";") && !f.is(end, ",") && !f.isKeyword(end, "as") {
		end++
	}
	end--
	if end > use && f.isEmpty(end) {
		end = f.prevEffective(end)
	}
	start := end
	for start-1 > use && f.isName(start-1) {
		start--
	}
	return strings.TrimLeft(f.content(start, end), `\`)
}

// useAlias is UseStatementHelper::getAlias, or "".
func (f *slvFile) useAlias(use int) string {
	for j := use + 1; j < len(f.toks) && !f.is(j, ";") && !f.is(j, ","); j++ {
		if f.isKeyword(j, "as") {
			for k := j + 1; k < len(f.toks); k++ {
				if f.toks[k].kind == token.Ident {
					return f.toks[k].content
				}
			}
		}
	}
	return ""
}
