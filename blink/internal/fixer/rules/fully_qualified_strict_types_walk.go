package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// walk is one pass of applyFix's loop over a namespace scope: the
// symbol-discovery pass or the fixing pass, as c.discovering says.
func (c *fqCtx) walk(decl nsDecl) {
	sc := c.sc
	c.reserved = map[int]map[string]bool{}
	level := 0
	for i := decl.scopeStart; i < decl.scopeEnd && i < sc.n; i++ {
		t := sc.tok(i)
		switch {
		case t.Kind == token.Punct && t.Value == "{":
			level++
		case t.Kind == token.Punct && t.Value == "}":
			delete(c.reserved, level)
			level--
		case t.Kind == token.Variable:
			if sc.isTString(sc.prevM(i)) {
				c.fixPrevName(i)
			}
		case t.Kind == token.Punct && t.Value == "::":
			c.fixPrevName(i)
		case sc.isFunctionKeyword(i):
			c.fixFunction(i)
		case t.Kind == token.AttributeStart:
			c.fixAttribute(i)
		case sc.isKeyword(i, "catch"):
			c.fixCatch(i)
		case c.discovering && sc.isClassy(i):
			c.fixNextName(i)
		case sc.isKeyword(i, "extends", "implements"):
			c.fixExtendsImplements(i)
		case sc.isKeyword(i, "instanceof", "new") || sc.isTraitUse(i) || sc.isTypeColon(i):
			c.fixNextName(i)
		case c.discovering && t.Kind == token.Comment:
			if name, ok := fqCommentAttributeClass(t.Value); ok {
				c.determineShortType(name)
			}
		case t.Kind == token.DocComment:
			for _, id := range fqDocReservedIdentifiers(t.Value) {
				if c.reserved[level+1] == nil {
					c.reserved[level+1] = map[string]bool{}
				}
				c.reserved[level+1][id] = true
			}
			c.fixPhpDoc(i)
		}
	}
	c.reserved = nil
}

// tokOr returns token i, or a zero token past the end.
func (sc *symScan) tokOr(i int) token.Token {
	if i < 0 || i >= sc.n {
		return token.Token{}
	}
	return sc.tok(i)
}

func (c *fqCtx) isName(i int) bool { return c.sc.isTString(i) || c.sc.isPunct(i, `\`) }

// fixPrevName shortens the name run before a variable or "::".
func (c *fqCtx) fixPrevName(index int) {
	start, end := -1, -1
	for {
		index = c.sc.prevM(index)
		if index < 0 || c.sc.isObjectOperator(index) {
			return
		}
		if c.isName(index) {
			start = index
			if end < 0 {
				end = index
			}
			continue
		}
		if end >= 0 {
			c.visit(start, end)
		}
		return
	}
}

// fixNextName shortens the name run after a token.
func (c *fqCtx) fixNextName(index int) {
	start, end := -1, -1
	for {
		index = c.sc.nextM(index)
		if index >= 0 && c.isName(index) {
			if start < 0 {
				start = index
			}
			end = index
			continue
		}
		if start >= 0 {
			c.visit(start, end)
		}
		return
	}
}

func (c *fqCtx) fixExtendsImplements(index int) {
	sc := c.sc
	isExtends := sc.isKeyword(index, "extends")
	index = sc.nextM(index)
	start, end := -1, -1
	for index >= 0 {
		if sc.isPunct(index, ",") || sc.isPunct(index, "{") || sc.isKeyword(index, "implements") {
			if start >= 0 {
				c.visit(start, end)
			}
			start = -1
			if sc.isPunct(index, "{") || (isExtends && sc.isKeyword(index, "implements")) {
				return
			}
		} else {
			if start < 0 {
				start = index
			}
			end = index
		}
		index = sc.nextM(index)
	}
}

func (c *fqCtx) fixCatch(index int) {
	sc := c.sc
	index = sc.nextM(sc.nextM(index)) // past "("
	start, end := -1, -1
	for index >= 0 {
		if sc.isPunct(index, ")") || sc.isKind(index, token.Variable) || sc.isTypeAlternation(index) {
			if start < 0 {
				return
			}
			c.visit(start, end)
			start = -1
			if sc.isPunct(index, ")") {
				return
			}
		} else {
			if start < 0 {
				start = index
			}
			end = index
		}
		index = sc.nextM(index)
	}
}

// fixAttribute shortens the class of each attribute of a #[...] group
// (AttributeAnalyzer::collectOne).
func (c *fqCtx) fixAttribute(open int) {
	sc := c.sc
	closing := sc.match[open]
	if closing < 0 {
		return
	}
	nextOf := func(from int, parenToo bool) int {
		for j := from + 1; j < sc.n; j++ {
			if sc.isPunct(j, ",") || sc.isKind(j, token.AttributeEnd) || (parenToo && sc.isPunct(j, "(")) {
				return j
			}
		}
		return -1
	}
	var starts []int
	index := open
	for {
		attrStart := index + 1
		index = nextOf(attrStart, true)
		if index < 0 {
			return
		}
		if sc.isPunct(index, "(") {
			if sc.match[index] < 0 {
				return
			}
			if index = nextOf(sc.match[index], false); index < 0 {
				return
			}
		}
		starts = append(starts, attrStart)
		next := index
		if next < closing {
			next = sc.nextM(index)
		}
		if next < 0 || next >= closing {
			break
		}
	}
	for _, index := range starts {
		for index >= 0 && c.isName(index) {
			index = sc.prevM(index)
		}
		if index >= 0 {
			c.fixNextName(index)
		}
	}
}

// fixFunction shortens the parameter and return types of a T_FUNCTION.
func (c *fqCtx) fixFunction(index int) {
	sc := c.sc
	open := sc.nextPunct(index, "(")
	if open < 0 || sc.match[open] < 0 {
		return
	}
	closing := sc.match[open]
	// FunctionsAnalyzer::getFunctionArguments keys the arguments by name
	var order []string
	byName := map[string]fqArg{}
	for _, r := range c.functionArguments(open, closing) {
		a := c.argumentInfo(r[0], r[1])
		if _, ok := byName[a.name]; !ok {
			order = append(order, a.name)
		}
		byName[a.name] = a
	}
	for _, name := range order {
		if a := byName[name]; a.typeStart >= 0 {
			c.replaceByShortType(a.typeStart, a.typeEnd)
		}
	}
	// getFunctionReturnType
	colon := sc.nextM(closing)
	if sc.isLambdaUse(colon) {
		lu := sc.nextPunct(colon, "(")
		if lu < 0 || sc.match[lu] < 0 {
			return
		}
		colon = sc.nextM(sc.match[lu])
	}
	if !sc.isTypeColon(colon) {
		return
	}
	typeStart := sc.nextM(colon)
	body := -1
	for j := colon + 1; j < sc.n; j++ {
		if sc.isPunct(j, "{") || sc.isPunct(j, ";") || sc.isPunct(j, "=>") {
			body = j
			break
		}
	}
	if typeStart < 0 || body < 0 {
		return
	}
	typeEnd := typeStart
	for j := typeStart; j < body; j++ {
		switch sc.tok(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		typeEnd = j
	}
	c.replaceByShortType(typeStart, typeEnd)
}

// functionArguments is ArgumentsAnalyzer::getArguments: the inclusive token
// ranges of the arguments between the parentheses.
func (c *fqCtx) functionArguments(open, closing int) [][2]int {
	sc := c.sc
	if first := sc.nextM(open); first < 0 || sc.isPunct(first, ")") {
		return nil
	}
	var out [][2]int
	argStart := open + 1
	j := open + 1
	for ; j < closing; j++ {
		t := sc.tok(j)
		if t.Kind == token.AttributeStart || (t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{")) {
			if sc.match[j] > j {
				j = sc.match[j]
			}
			continue
		}
		if sc.isPunct(j, ",") {
			if sc.nextM(j) == closing {
				break // trailing comma
			}
			out = append(out, [2]int{argStart, j - 1})
			argStart = j + 1
		}
	}
	return append(out, [2]int{argStart, j - 1})
}

// fqArg is an ArgumentAnalysis reduced to the name and the type range
// (typeStart -1 when there is no type or no name).
type fqArg struct {
	name               string
	typeStart, typeEnd int
}

// argumentInfo is ArgumentsAnalyzer::getArgumentInfo.
func (c *fqCtx) argumentInfo(start, end int) (a fqArg) {
	sc := c.sc
	a.typeStart, a.typeEnd = -1, -1
	sawName := false
	for j := start; j <= end && j < sc.n; j++ {
		t := sc.tok(j)
		switch t.Kind {
		case token.AttributeStart:
			if sc.match[j] > j {
				j = sc.match[j]
			}
			continue
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		case token.Variable:
			if !sawName {
				sawName = true
				a.name = t.Value
			}
			continue
		}
		if sc.isPunct(j, "...") || sc.isKeyword(j, "final", "public", "protected", "private", "readonly", "public(set)", "protected(set)", "private(set)") {
			continue
		}
		if sc.isPunct(j, "&") && !sc.isTypeIntersection(j) {
			continue
		}
		if sc.isPunct(j, "=") || sawName {
			continue
		}
		if a.typeStart < 0 {
			a.typeStart = j
		}
		a.typeEnd = j
	}
	if !sawName {
		a.typeStart, a.typeEnd = -1, -1
	}
	return a
}

// replaceByShortType shortens each class of a type declaration (getTypes).
func (c *fqCtx) replaceByShortType(start, end int) {
	sc := c.sc
	if sc.isPunct(start, "?") {
		start = sc.nextM(start)
	}
	atomStart, atomEnd := -1, -1
	for index := start; index >= 0 && index <= end; index = sc.nextM(index) {
		switch {
		case sc.isPunct(index, "("):
			atomStart = -1
		case sc.isPunct(index, "|") || sc.isPunct(index, "&") || sc.isPunct(index, ")"):
			if atomStart >= 0 {
				c.visit(atomStart, atomEnd)
			}
			atomStart = -1
		default:
			if atomStart < 0 {
				atomStart = index
			}
			atomEnd = index
		}
	}
	if atomStart >= 0 {
		c.visit(atomStart, atomEnd)
	}
}

// fqCommentAttributeClass matches /#\[\s*(REGEX_CLASS)/ in a comment.
func fqCommentAttributeClass(content string) (string, bool) {
	m := docTypeMatcher{src: content, utf: docUTF(content)}
	for i := 0; i+1 < len(content); i++ {
		if content[i] != '#' || content[i+1] != '[' {
			continue
		}
		p := docS.skip(content, i+2, m.utf)
		if e := fqRegexClassAt(m, p); e > p {
			return content[p:e], true
		}
	}
	return "", false
}

// fqRegexClassAt matches REGEX_CLASS, (?:\\?+identifier(\\identifier)*+), at p
// and returns its end, or -1.
func fqRegexClassAt(m docTypeMatcher, p int) int {
	if m.at(p) == '\\' {
		p++
	}
	p = m.identifier(p)
	if p < 0 {
		return -1
	}
	for m.at(p) == '\\' {
		q := m.identifier(p + 1)
		if q < 0 {
			break
		}
		p = q
	}
	return p
}

// fqCollectUses returns the class imports of a namespace
// (NamespaceUsesAnalyzer::getDeclarationsInNamespace with multi-uses) and the
// end of the statement of the last one, or -1.
func fqCollectUses(sc *symScan, decl nsDecl) (*fqUses, int) {
	u := fqNewUses()
	lastEnd := -1
	for i := decl.scopeStart; i <= decl.scopeEnd && i < sc.n; i++ {
		if !sc.isImportUse(i) {
			continue
		}
		end := -1
		for j := i + 1; j < sc.n; j++ {
			if sc.isPunct(j, ";") || sc.isKind(j, token.CloseTag) {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		isClass := !sc.isImportKindKeyword(sc.nextM(i))
		for _, imp := range fqParseUseDeclarations(sc, i, end) {
			if !isClass {
				continue
			}
			full := strings.TrimLeft(imp[0], `\`)
			if full == "" {
				continue
			}
			u.add(full, imp[1])
			lastEnd = end
		}
	}
	u.build()
	return u, lastEnd
}

// fqParseUseDeclarations is NamespaceUsesAnalyzer::parseDeclarations: the
// (full name, short name) of each import of the statement [start, end].
func fqParseUseDeclarations(sc *symScan, start, end int) [][2]string {
	var out [][2]string
	index := -1
	for j := start + 1; j < sc.n; j++ {
		if sc.isTString(j) || sc.isPunct(j, `\`) {
			index = j
			break
		}
	}
	isGroupClose := func(j int) bool {
		return sc.isPunct(j, "}") && sc.match[j] >= 0 && sc.isGroupImportOpen(sc.match[j])
	}
	nearest := func(index int) (full, short string, after int) {
		var b strings.Builder
		aliased := false
		for ; index >= 0; index = sc.nextM(index) {
			switch {
			case sc.isTString(index):
				short = sc.tok(index).Value
				if !aliased {
					b.WriteString(short)
				}
			case sc.isPunct(index, `\`):
				b.WriteString(`\`)
			case sc.isKeyword(index, "as"):
				aliased = true
			case sc.isPunct(index, ",") || sc.isPunct(index, ";") || sc.isGroupImportOpen(index) || isGroupClose(index) || sc.isKind(index, token.CloseTag):
				return b.String(), short, index
			}
		}
		return b.String(), short, -1
	}
	for index >= 0 && index <= end {
		full, short, after := nearest(index)
		if after < 0 {
			break
		}
		if sc.isGroupImportOpen(after) {
			groupEnd := sc.match[after]
			if groupEnd < 0 {
				break
			}
			gi := after
			for gi >= 0 && gi < groupEnd {
				chunk := sc.nextM(gi)
				if chunk < 0 || isGroupClose(chunk) {
					break
				}
				gFull, gShort, gAfter := nearest(chunk)
				if gFull != "" && gShort != "" {
					out = append(out, [2]string{full + gFull, gShort})
				}
				gi = gAfter
			}
			index = gi
		} else {
			if !isGroupClose(sc.prevM(after)) && full != "" && short != "" {
				out = append(out, [2]string{full, short})
			}
			index = after
		}
		if index < 0 {
			break
		}
		index = sc.nextM(index)
	}
	return out
}

// fqImportEdit is the insertion of applyFix's new imports
// (ImportProcessor::insertImports): after the last class import, else after
// the namespace declaration, else after the open tag or a leading declare.
func fqImportEdit(sc *symScan, decl nsDecl, lastUseEnd int, forImport *fqOrdered) (fqEdit, bool) {
	names := make([]string, 0, len(forImport.keys))
	for _, k := range forImport.keys {
		names = append(names, forImport.vals[k])
	}
	slices.Sort(names)
	var items []token.Token
	for _, name := range names {
		items = append(items,
			token.Token{Kind: token.Whitespace, Value: "\n"},
			token.Token{Kind: token.Keyword, Value: "use"},
			token.Token{Kind: token.Whitespace, Value: " "})
		items = append(items, fqStringToTokens(name)...)
		items = append(items, token.Token{Kind: token.Punct, Value: ";"})
	}
	// the token at the insertion point: ImportProcessor adds a newline unless
	// it is whitespace holding one
	finish := func(at int, next token.Token, hasNext bool) (fqEdit, bool) {
		if p := sc.prevM(at); p >= 0 && sc.tok(p).Kind == token.OpenTag && sc.tok(p).Value == "<?=" {
			return fqEdit{}, false
		}
		if !hasNext || next.Kind != token.Whitespace || !strings.Contains(next.Value, "\n") {
			items = append(items, token.Token{Kind: token.Whitespace, Value: "\n"})
		}
		return fqEdit{start: at, end: at - 1, toks: items}, true
	}
	// PHP throws on an insertion point past the last token
	finishAt := func(at int) (fqEdit, bool) {
		if at >= sc.n {
			return fqEdit{}, false
		}
		return finish(at, sc.tok(at), true)
	}
	implicitGlobal := decl.scopeStart == decl.declEnd
	switch {
	case lastUseEnd >= 0:
		return finishAt(lastUseEnd + 1)
	case !implicitGlobal:
		return finishAt(decl.declEnd + 1)
	}
	if decl.scopeStart == 0 {
		if first := sc.nextM(0); sc.isKeyword(first, "declare") {
			semi := sc.nextPunct(first, ";")
			if semi < 0 {
				return fqEdit{}, false
			}
			return finishAt(semi + 1)
		}
	}
	// right after the open tag, which in PHP holds the first whitespace
	// character after "<?php"
	open := decl.scopeStart
	if open < sc.n && sc.tok(open).Value == "<?=" {
		// insertImports puts "<?php\n?>" before the "<?=" and the imports
		// before its "?>"
		toks := []token.Token{{Kind: token.OpenTag, Value: "<?php"}, {Kind: token.Whitespace, Value: "\n"}}
		toks = append(toks, items...)
		toks = append(toks, token.Token{Kind: token.Whitespace, Value: "\n"}, token.Token{Kind: token.CloseTag, Value: "?>"})
		return fqEdit{start: open, end: open - 1, toks: toks}, true
	}
	if open >= sc.n || sc.tok(open).Kind != token.OpenTag || !strings.EqualFold(sc.tok(open).Value, "<?php") {
		return fqEdit{}, false
	}
	ws := sc.tokOr(open + 1)
	if open+1 >= sc.n || ws.Kind != token.Whitespace {
		return fqEdit{}, false
	}
	n := 1
	if strings.HasPrefix(ws.Value, "\r\n") {
		n = 2
	}
	rest := ws.Value[n:]
	e, ok := finish(open+1, token.Token{Kind: token.Whitespace, Value: rest}, rest != "")
	if !ok {
		return e, false
	}
	toks := append([]token.Token{{Kind: token.Whitespace, Value: ws.Value[:n]}}, e.toks...)
	if rest != "" {
		toks = append(toks, token.Token{Kind: token.Whitespace, Value: rest})
	}
	return fqEdit{start: open + 1, end: open + 1, toks: toks}, true
}
