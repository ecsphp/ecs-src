package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnusedImportsFixer.php
//
// NoUnusedImports removes unused imports (a whole "use" statement, or one name of
// a multi or group use) and imports of the current namespace. It ports
// NoUnusedImportsFixer and its analyzers on clearableTokens.
type NoUnusedImports struct{}

func (NoUnusedImports) Name() string {
	return `PhpCsFixer\Fixer\Import\NoUnusedImportsFixer`
}

func (NoUnusedImports) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnusedImportsFixer.php"
}

// importDecl mirrors NamespaceUseAnalysis: one imported name.
type importDecl struct {
	kind             string // "class", "function" or "const"
	full, short      string
	aliased, inMulti bool
	start, end       int // the whole statement: "use" .. ";" or "?>"
	chunkStart       int // in a multi or group use: this name's first token ...
	chunkEnd         int // ... and last one
}

// namespaceDecl mirrors NamespaceAnalysis.
type namespaceDecl struct {
	name                 string
	scopeStart, scopeEnd int
}

func (NoUnusedImports) Fix(s *tokens.Stream) bool {
	c := newClearableTokens(s)
	decls := c.useDeclarations()
	if len(decls) == 0 {
		return false
	}
	changed := false
	for _, ns := range c.namespaceDeclarations() {
		var current []importDecl
		ignored := map[int]int{}
		for _, d := range decls {
			if d.start >= ns.scopeStart && d.end <= ns.scopeEnd {
				current = append(current, d)
				ignored[d.start] = d.end
			}
		}
		for _, d := range current {
			if !c.isImportUsed(ns, d, ignored) {
				c.removeUseDeclaration(d, false)
				changed = true
			}
		}
		// removeUsesInSameNamespace
		prefix := ns.name + `\`
		for _, d := range current {
			if d.aliased {
				continue
			}
			full := strings.TrimLeft(d.full, `\`)
			// a case-sensitive prefix match (strncmp), as PHP-CS-Fixer
			if strings.HasPrefix(full, prefix) && !strings.Contains(full[len(prefix):], `\`) {
				c.removeUseDeclaration(d, false)
				changed = true
			}
		}
	}
	if changed {
		c.writeBack(s)
	}
	return changed
}

// useDeclarations is NamespaceUsesAnalyzer::getDeclarationsFromTokens($tokens, true).
func (c *clearableTokens) useDeclarations() []importDecl {
	var out []importDecl
	for i := range c.toks {
		if !c.isImportUse(i) {
			continue
		}
		end := c.nextTokenOf(i, func(j int) bool { return c.punct(j, ";") || c.kind(j, token.CloseTag) })
		if end < 0 {
			continue
		}
		out = append(out, c.parseDeclarations(i, end)...)
	}
	return out
}

// parseDeclarations is NamespaceUsesAnalyzer::parseDeclarations().
func (c *clearableTokens) parseDeclarations(start, end int) []importDecl {
	kind := "class"
	if t := c.nextMeaningful(start); c.keyword(t, "function") {
		kind = "function"
	} else if c.keyword(t, "const") {
		kind = "const"
	}
	multiAt := c.nextTokenOf(start, func(j int) bool { return c.punct(j, ",") || c.groupOpen(j) })
	multi := multiAt >= 0 && multiAt < end
	var out []importDecl
	add := func(full, short string, aliased, inMulti bool, chunkStart, chunkEnd int) {
		if full == "" || short == "" {
			return
		}
		out = append(out, importDecl{
			kind: kind, full: full, short: short, aliased: aliased, inMulti: inMulti,
			start: start, end: end, chunkStart: chunkStart, chunkEnd: chunkEnd,
		})
	}
	index := c.nextTokenOf(start, func(j int) bool { return c.tString(j) || c.punct(j, `\`) })
	for index >= 0 && index <= end {
		q := c.nearestQualifiedName(index)
		switch {
		case c.groupOpen(q.after):
			groupIndex, groupEnd := q.after, c.match[q.after]
			for groupIndex < groupEnd {
				chunkStart := c.nextMeaningful(groupIndex)
				if chunkStart < 0 || c.groupClose(chunkStart) {
					break // a trailing comma
				}
				g := c.nearestQualifiedName(chunkStart)
				add(q.full+g.full, g.short, g.aliased, true, chunkStart, c.prevMeaningful(g.after))
				if g.after < 0 {
					break
				}
				groupIndex = g.after
			}
			index = groupIndex
		case c.punct(q.after, ",", ";") || c.kind(q.after, token.CloseTag):
			prev := c.prevMeaningful(q.after)
			if !c.groupClose(prev) {
				chunkStart, chunkEnd := -1, -1
				if multi {
					chunkStart, chunkEnd = index, prev
				}
				add(q.full, q.short, q.aliased, multi, chunkStart, chunkEnd)
			}
			index = q.after
		}
		if index < 0 {
			break
		}
		index = c.nextMeaningful(index)
	}
	return out
}

type qualifiedName struct {
	full, short string
	aliased     bool
	after       int
}

// nearestQualifiedName is NamespaceUsesAnalyzer::getNearestQualifiedName().
func (c *clearableTokens) nearestQualifiedName(index int) qualifiedName {
	var q qualifiedName
	for index >= 0 {
		switch {
		case c.tString(index):
			q.short = c.toks[index].Value
			if !q.aliased {
				q.full += q.short
			}
		case c.punct(index, `\`):
			q.full += `\`
		case c.keyword(index, "as"):
			q.aliased = true
		case c.punct(index, ",", ";") || c.groupOpen(index) || c.groupClose(index) || c.kind(index, token.CloseTag):
			q.after = index
			return q
		}
		index = c.nextMeaningful(index)
	}
	q.after = -1
	return q
}

// namespaceDeclarations is NamespacesAnalyzer::getDeclarations().
func (c *clearableTokens) namespaceDeclarations() []namespaceDecl {
	var out []namespaceDecl
	n := len(c.toks)
	for i := 1; i < n; i++ {
		if !c.namespaceKeyword(i) {
			continue
		}
		declEnd := c.nextTokenOf(i, func(j int) bool { return c.punct(j, ";", "{") })
		if declEnd < 0 {
			break
		}
		var name strings.Builder
		for j := i + 1; j < declEnd; j++ {
			name.WriteString(c.value(j))
		}
		scopeEnd := n - 1
		if c.punct(declEnd, "{") {
			if m := c.match[declEnd]; m >= 0 {
				scopeEnd = m
			}
		} else if next := c.nextTokenOf(declEnd, c.namespaceKeyword); next >= 0 {
			scopeEnd = next - 1
		}
		out = append(out, namespaceDecl{name: strings.Trim(name.String(), " \t\n\r\x00\v"), scopeStart: i, scopeEnd: scopeEnd})
		i = scopeEnd
	}
	if len(out) == 0 && slices.ContainsFunc(c.toks, func(t token.Token) bool {
		return t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=")
	}) {
		start := 0
		if n > 0 && c.toks[0].Kind == token.InlineHTML {
			start = 1
		}
		out = append(out, namespaceDecl{scopeStart: start, scopeEnd: n - 1})
	}
	return out
}

// isImportUsed is NoUnusedImportsFixer::isImportUsed(). ignored maps the start
// of each use statement of the namespace to its end.
func (c *clearableTokens) isImportUsed(ns namespaceDecl, d importDecl, ignored map[int]int) bool {
	inAttribute := false
	shortLower := strings.ToLower(d.short)
	for i := ns.scopeStart; i <= ns.scopeEnd; i++ {
		if c.kind(i, token.AttributeStart) {
			inAttribute = true
			continue
		}
		if c.kind(i, token.AttributeEnd) {
			inAttribute = false
			continue
		}
		if end, ok := ignored[i]; ok {
			i = end
			continue
		}
		// compare the name first: telling a T_STRING apart reads its neighbours
		if c.kind(i, token.Ident) && strings.EqualFold(c.toks[i].Value, d.short) && c.tString(i) {
			prevIndex := c.prevMeaningful(i)
			if c.namespaceKeyword(prevIndex) {
				i = c.nextTokenOf(i, func(j int) bool { return c.punct(j, ";", "{") || c.kind(j, token.CloseTag) })
				if i < 0 {
					return false
				}
				continue
			}
			nextIndex := c.nextMeaningful(i)
			if c.punct(prevIndex, `\`, "::") || c.keyword(prevIndex, "function") || c.objectOperator(prevIndex) {
				continue
			}
			if c.keyword(prevIndex, "const") && c.punct(nextIndex, "=") {
				continue
			}
			if inAttribute {
				return true
			}
			if c.belongsToGotoLabel(nextIndex) {
				continue
			}
			kind := "class"
			switch {
			case c.isConstantInvocation(i):
				kind = "const"
			case c.punct(nextIndex, "(") && !c.keyword(prevIndex, "new") && !c.kind(prevIndex, token.AttributeStart):
				kind = "function"
			}
			if kind == d.kind {
				return true
			}
			continue
		}
		// a class named like a PHP keyword (enum, list, ...) that blink's lexer
		// reads as the keyword: count it in a class-reference position
		if d.kind == "class" && c.kind(i, token.Keyword) && strings.EqualFold(c.toks[i].Value, d.short) && c.keywordIsClassRef(i) {
			return true
		}
		if c.isComment(i) && commentReferences(c.toks[i].Value, shortLower) {
			return true
		}
	}
	return false
}

func (c *clearableTokens) keywordIsClassRef(i int) bool {
	prev := c.prevTokenOf(i, c.nonWhitespace)
	if c.keyword(prev, "extends", "implements", "new", "instanceof") || c.punct(prev, "(", ",", ":", "|", "&", "?") {
		return true
	}
	next := c.nextTokenOf(i, c.nonWhitespace)
	return c.punct(next, "::") || c.kind(next, token.Variable)
}

// removeUseDeclaration is NoUnusedImportsFixer::removeUseDeclaration().
func (c *clearableTokens) removeUseDeclaration(d importDecl, forceCompleteRemoval bool) {
	start, end := d.start, d.end
	if d.inMulti && !forceCompleteRemoval {
		start, end = d.chunkStart, d.chunkEnd
	}
	loopStart := end - 1
	if d.inMulti || forceCompleteRemoval {
		loopStart = end
	}
	for i := loopStart; i >= start; i-- {
		if c.isComment(i) {
			continue
		}
		if !c.isWhitespace(i) || !strings.Contains(c.value(i), "\n") {
			c.clear(i)
			continue
		}
		// a multi-line whitespace keeps its line feed (and indent) after a comment
		if c.isComment(c.prevTokenOf(i, c.nonWhitespace)) {
			v := c.value(i)
			c.setWhitespace(i, v[strings.LastIndex(v, "\n"):])
		} else {
			c.clearAndMergeSurroundingWhitespace(i)
		}
	}
	if !forceCompleteRemoval && d.inMulti {
		c.cleanUpAfterImportChunkRemoval(d)
		return
	}
	if c.punct(d.end, ";") {
		c.clear(d.end) // a "?>" stays
	}
	c.cleanUpSurroundingNewLines(d)
}

// cleanUpAfterImportChunkRemoval is NoUnusedImportsFixer::cleanUpAfterImportChunkRemoval().
func (c *clearableTokens) cleanUpAfterImportChunkRemoval(d importDecl) {
	before := c.prevMeaningful(d.chunkStart)
	after := c.nextMeaningful(d.chunkEnd)
	_, hasBefore := c.scanUntilNewLine(after, -1)
	_, hasAfter := c.scanUntilNewLine(after, 1)
	// merging the surrounding whitespace is safe only when a token other than
	// whitespace stays on the line (no "\n    \n    ")
	safe := hasBefore || hasAfter
	clearToken := func(i int) {
		if safe {
			c.clearAndMergeSurroundingWhitespace(i)
		} else {
			c.clear(i)
		}
	}
	if c.punct(after, ",") {
		clearToken(after)
	} else if c.punct(before, ",") {
		clearToken(before)
	}
	// a single space where one belongs, none before a comma or a closing brace
	for i := before; i >= 0 && i <= after; i++ {
		if !c.hasID(i) || !c.kind(i, token.Whitespace) || strings.Trim(c.toks[i].Value, " ") != "" {
			continue
		}
		if n := c.nextMeaningful(i); c.punct(n, ",", ";") || c.groupClose(n) {
			c.clear(i)
		} else {
			c.setWhitespace(i, " ")
		}
		if c.groupOpen(c.prevMeaningful(i)) {
			c.clear(i)
		}
	}
	c.removeLineIfEmpty(d)
	c.removeImportStatementIfEmpty(d)
}

// cleanUpSurroundingNewLines is NoUnusedImportsFixer::cleanUpSurroundingNewLines().
func (c *clearableTokens) cleanUpSurroundingNewLines(d importDecl) {
	prev := d.start - 1
	if c.isWhitespace(prev) {
		c.setWhitespace(prev, strings.TrimRight(c.value(prev), " \t"))
	}
	next := c.nonEmptySibling(d.end, 1)
	if next < 0 {
		return
	}
	if c.isWhitespace(next) {
		v := strings.TrimLeft(c.value(next), " \t")
		if strings.HasPrefix(v, "\r\n") {
			v = v[2:]
		} else if strings.HasPrefix(v, "\n") {
			v = v[1:]
		}
		c.setWhitespace(next, v)
	}
	if c.isWhitespace(prev) && c.isWhitespace(next) {
		c.setWhitespace(next, c.value(prev)+c.value(next))
		c.clear(prev)
	}
}

// removeImportStatementIfEmpty is NoUnusedImportsFixer::removeImportStatementIfEmpty().
func (c *clearableTokens) removeImportStatementIfEmpty(d importDecl) {
	// a group whose names were all removed: "use Foo\{};"
	if brace := c.prevTokenOf(d.end, c.groupClose); brace > d.start && c.groupOpen(c.prevMeaningful(brace)) {
		c.removeUseDeclaration(d, true)
	}
	// a multi-use whose names were all removed: "use;"
	if b := c.prevMeaningful(d.end); c.isImportUse(b) || c.value(b) == "function" || c.value(b) == "const" {
		c.removeUseDeclaration(d, true)
	}
}

// removeLineIfEmpty is NoUnusedImportsFixer::removeLineIfEmpty().
func (c *clearableTokens) removeLineIfEmpty(d importDecl) {
	if !d.inMulti {
		return
	}
	beforeNL, hasBefore := c.scanUntilNewLine(d.chunkStart, -1)
	afterNL, hasAfter := c.scanUntilNewLine(d.chunkEnd, 1)
	if beforeNL >= 0 && !hasBefore && afterNL >= 0 && !hasAfter {
		for i := beforeNL; i < afterNL; i++ {
			c.clear(i)
		}
	}
}

// scanUntilNewLine is NoUnusedImportsFixer::scanForNonEmptyTokensUntilNewLineFound():
// the index of the first whitespace starting with "\n" in direction dir (or
// -1), and whether a token other than whitespace came first.
func (c *clearableTokens) scanUntilNewLine(i, dir int) (int, bool) {
	has := false
	for i >= 0 {
		i = c.nonEmptySibling(i, dir)
		if i < 0 || !c.hasID(i) {
			continue
		}
		if !c.isWhitespace(i) {
			has = true
		} else if strings.HasPrefix(c.value(i), "\n") {
			return i, has
		}
	}
	return -1, has
}

// commentReferences reports whether short appears in v as a whole word, not
// preceded by an identifier char, "$" or "\".
func commentReferences(v, short string) bool {
	lv := strings.ToLower(v)
	from := 0
	for {
		idx := strings.Index(lv[from:], short)
		if idx < 0 {
			return false
		}
		p := from + idx
		before := byte(' ')
		if p > 0 {
			before = lv[p-1]
		}
		after := byte(' ')
		if p+len(short) < len(lv) {
			after = lv[p+len(short)]
		}
		if !isIdentByte(before) && before != '$' && before != '\\' && !isIdentByte(after) {
			return true
		}
		from = p + 1
	}
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
