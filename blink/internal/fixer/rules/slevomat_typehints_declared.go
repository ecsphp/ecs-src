package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// thdOptions are the enable* options and traversableTypeHints the declared
// type hint sniffs share.
type thdOptions struct {
	object, static, mixed, union, intersection, never, standalone, native bool
	traversable                                                           []string
}

func thdParseOptions(config map[string]any) thdOptions {
	o := thdOptions{
		object:       thBool(config, "enableObjectTypeHint", true),
		static:       thBool(config, "enableStaticTypeHint", true),
		mixed:        thBool(config, "enableMixedTypeHint", true),
		union:        thBool(config, "enableUnionTypeHint", true),
		intersection: thBool(config, "enableIntersectionTypeHint", true),
		never:        thBool(config, "enableNeverTypeHint", true),
		standalone:   thBool(config, "enableStandaloneNullTrueFalseTypeHints", true),
		native:       thBool(config, "enableNativeTypeHint", true),
	}
	list, _ := slvStringList(config["traversableTypeHints"])
	for _, t := range list {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, `\`) {
			t = `\` + t
		}
		o.traversable = append(o.traversable, t)
	}
	return o
}

func thdDefaultOptions() thdOptions { return thdParseOptions(nil) }

// thdCtx is one pass of a declared type hint sniff over a file.
type thdCtx struct {
	f     *slvFile
	d     *slvDocs
	n     *slvNames
	sc    *phpcsScope
	o     thdOptions
	sniff string
}

func newThdCtx(f *slvFile, o thdOptions, sniff string) *thdCtx {
	return &thdCtx{f: f, d: newSlvDocs(f), n: newSlvNames(f), o: o, sniff: sniff}
}

func (c *thdCtx) name(code string) string { return c.sniff + "." + code }

func (c *thdCtx) isTraversable(t string) bool {
	return t == "array" || t == "iterable" || slices.Contains(c.o.traversable, t)
}

func (c *thdCtx) fq(pointer int, t string) string { return slvFullyQualifiedTypeHint(c.d, pointer, t) }

func (c *thdCtx) isTypeDefined(pointer int, t string) bool {
	open := c.f.docOpenBefore(pointer)
	return open >= 0 && c.d.isTypeDefinedInAnnotation(open, t)
}

// annotations is AnnotationHelper::getAnnotations of a non-docblock pointer.
func (c *thdCtx) annotations(pointer int, name string) (*slvParsedDoc, []slvAnnotation) {
	open := c.f.docOpenBefore(pointer)
	if open < 0 {
		return nil, nil
	}
	return c.d.parse(open), c.d.annotations(open, name)
}

func (c *thdCtx) isSuppressed(pointer int, name string) bool {
	return c.f.thIsSuppressed(c.d, pointer, name)
}

// reportUselessSuppress is the sniffs' reportUselessSuppress with
// SuppressHelper::removeSuppressAnnotation.
func (c *thdCtx) reportUselessSuppress(pointer int, suppressed bool, name string) {
	if !suppressed {
		return
	}
	f := c.f
	p, annotations := c.annotations(pointer, "@phpcsSuppress")
	var found *pdNode
	for _, a := range annotations {
		if v := a.node.value(); v.start >= 0 && thRaw(p.toks, v) == name {
			found = a.node
			break
		}
	}
	if found == nil {
		return
	}
	start, end := f.thAnnotationPointers(p, found)
	changeStart := start
	for j := start - 1; j >= 0; j-- {
		if f.toks[j].doc == slvDocOpen || f.toks[j].doc == slvDocStar {
			if f.toks[j].doc == slvDocStar {
				changeStart = j
			}
			break
		}
	}
	changeEnd := thdNextStarOrClose(f, end+1) - 1
	f.beginChangeset()
	f.removeBetweenIncluding(changeStart, changeEnd)
	f.endChangeset()
}

// thdNextStarOrClose is the next T_DOC_COMMENT_STAR or T_DOC_COMMENT_CLOSE_TAG.
func thdNextStarOrClose(f *slvFile, from int) int {
	for j := from; j < len(f.toks); j++ {
		if f.toks[j].doc == slvDocStar || f.toks[j].doc == slvDocClose {
			return j
		}
	}
	return len(f.toks)
}

// removeAnnotationLine removes an annotation from the T_DOC_COMMENT_STAR
// before it (searched down to bound) to the token before the next star or
// closer at or after from.
func (c *thdCtx) removeAnnotationLine(start, bound, from int) {
	f := c.f
	changeStart := start
	for j := start - 1; j >= bound && j >= 0; j-- {
		if f.toks[j].doc == slvDocStar {
			changeStart = j
			break
		}
	}
	changeEnd := thdNextStarOrClose(f, from) - 1
	f.beginChangeset()
	f.removeBetweenIncluding(changeStart, changeEnd)
	f.endChangeset()
}

// hasOverride is AttributeHelper::hasAttribute(pointer, '\Override'), which
// only reads the first of the attribute groups before pointer.
func (c *thdCtx) hasOverride(pointer int) bool {
	f := c.f
	opener := -1
	from := pointer - 1
	for {
		found := -1
		for j := from; j >= 0; j-- {
			if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
				break
			}
			if f.toks[j].kind == token.AttributeEnd && f.isAttributeOpen(f.toks[j].closer) {
				found = j
				break
			}
		}
		if found < 0 {
			break
		}
		opener = f.toks[found].closer
		from = opener
	}
	if opener < 0 {
		return false
	}
	for _, a := range f.attributes(opener) {
		if c.d.useStatements().resolveClassName(f, a.name, opener) == `\Override` {
			return true
		}
	}
	return false
}

// hasInheritdoc is DocCommentHelper::hasInheritdocAnnotation.
func (c *thdCtx) hasInheritdoc(open int) bool {
	if open < 0 {
		return false
	}
	p := c.d.parse(open)
	if p == nil {
		return false
	}
	for _, child := range p.root.fields[0].list {
		switch child.kind {
		case "PhpDocTagNode":
			if strings.ToLower(child.name) == "@inheritdoc" {
				return true
			}
			if v := child.value(); v.start >= 0 && strings.Contains(strings.ToLower(uuDocText(p, v)), "{@inheritdoc}") {
				return true
			}
		case "PhpDocTextNode":
			if strings.Contains(strings.ToLower(uuDocText(p, child)), "{@inheritdoc}") {
				return true
			}
		}
	}
	return false
}

// thdHasDescription tells a @param, @var or @return value with a description.
func thdHasDescription(toks []pdToken, v *pdNode) bool {
	if v.kind == "TypelessParamTagValueNode" {
		for i := max(v.start, 0); i <= v.end && i < len(toks); i++ {
			switch toks[i].t {
			case pdHorizontalWS, pdPhpdocEOL, pdReference, pdVariadic:
				continue
			case pdVariable:
				return thdNonBlank(toks, i+1, v.end)
			}
			return true
		}
		return false
	}
	t := v.fields[0].node
	i := t.end + 1
	skipWS := func() {
		for i <= v.end && i < len(toks) && (toks[i].t == pdHorizontalWS || toks[i].t == pdPhpdocEOL) {
			i++
		}
	}
	switch v.kind {
	case "ParamTagValueNode":
		for i <= v.end && i < len(toks) {
			skipWS()
			if i <= v.end && i < len(toks) && (toks[i].t == pdReference || toks[i].t == pdVariadic) {
				i++
				continue
			}
			break
		}
		if i <= v.end && i < len(toks) && toks[i].t == pdVariable {
			i++
		}
	case "VarTagValueNode":
		skipWS()
		if i <= v.end && i < len(toks) && (toks[i].t == pdVariable || toks[i].t == pdThisVariable) {
			i++
		}
	}
	return thdNonBlank(toks, i, v.end)
}

func thdNonBlank(toks []pdToken, from, to int) bool {
	for i := from; i <= to && i < len(toks); i++ {
		if toks[i].t != pdHorizontalWS && toks[i].t != pdPhpdocEOL {
			return true
		}
	}
	return false
}

// isAnnotationUseless is AnnotationHelper::isAnnotationUseless.
func (c *thdCtx) isAnnotationUseless(pointer int, th *thTypeHint, p *slvParsedDoc, a slvAnnotation) bool {
	if a.isInvalid() || th == nil {
		return false
	}
	v := a.node.value()
	if thdHasDescription(p.toks, v) {
		return false
	}
	if v.kind == "TypelessParamTagValueNode" {
		return true
	}
	t := v.fields[0].node
	without := strings.TrimPrefix(th.hint, "?")
	if c.isTraversable(c.fq(pointer, without)) {
		if lc := strings.ToLower(t.name); t.kind != "IdentifierTypeNode" || lc != "array" && lc != "iterable" {
			return false
		}
	}
	if thContainsStaticOrThis(t) {
		return false
	}
	if thContainsJustTwoTypes(t) ||
		c.o.union && (t.kind == "UnionTypeNode" || t.kind == "IdentifierTypeNode" && thdIsUnofficialUnion(t.name)) ||
		c.o.intersection && t.kind == "IntersectionTypeNode" {
		return c.f.thTypeHintEqualsAnnotation(c.d, pointer, th.hint, thPrint(p.toks, t))
	}
	switch t.kind {
	case "ObjectShapeNode", "ConstTypeNode", "GenericTypeNode", "CallableTypeNode",
		"ConditionalTypeNode", "ConditionalTypeForParameterNode":
		return false
	case "IdentifierTypeNode":
		lc := strings.ToLower(t.name)
		if lc == "true" || lc == "false" || lc == "null" {
			return th.hint == lc && c.o.standalone
		}
		if slvTypeHintIsSimpleUnofficial(lc) && t.name != "object" && t.name != "mixed" {
			return false
		}
	}
	return c.f.thTypeHintEqualsAnnotation(c.d, pointer, without, thdFromOneType(p.toks, t, false, false))
}

func thdIsUnofficialUnion(t string) bool {
	return t == "scalar" || t == "numeric" || t == "array-key"
}

func thdUnofficialUnion(t string) []string {
	switch t {
	case "scalar":
		return []string{"string", "int", "float", "bool"}
	case "numeric":
		return []string{"int", "float", "string"}
	}
	return []string{"int", "string"}
}

var thdStringSuffix = regexp.MustCompile(`(?i)-string$`)

// fromOneType is AnnotationTypeHelper::getTypeHintFromOneType.
func thdFromOneType(toks []pdToken, t *pdNode, union, standalone bool) string {
	switch t.kind {
	case "IdentifierTypeNode":
		switch lc := strings.ToLower(t.name); lc {
		case "true":
			if standalone {
				return "true"
			}
			return "bool"
		case "false":
			if union || standalone {
				return "false"
			}
			return "bool"
		case "positive-int", "non-positive-int", "negative-int", "non-negative-int", "literal-int", "int-mask":
			return "int"
		case "callable-array", "callable-string":
			return "callable"
		}
		if thdStringSuffix.MatchString(t.name) {
			return "string"
		}
		switch strings.ToLower(t.name) {
		case "non-empty-array", "list", "non-empty-list":
			return "array"
		}
		return t.name
	case "CallableTypeNode":
		return thPrint(toks, t.fields[0].node)
	case "UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode":
		return thdString(toks, t)
	}
	return thTypeHintFromOneType(toks, t)
}

// thdString is the __toString of a union, intersection or nullable type,
// which, unlike the Printer, wraps unions and intersections in parentheses.
func thdString(toks []pdToken, t *pdNode) string {
	switch t.kind {
	case "UnionTypeNode", "IntersectionTypeNode":
		sep := " | "
		if t.kind == "IntersectionTypeNode" {
			sep = " & "
		}
		parts := make([]string, len(t.fields[0].list))
		for i, inner := range t.fields[0].list {
			parts[i] = thdString(toks, inner)
		}
		return "(" + strings.Join(parts, sep) + ")"
	case "NullableTypeNode":
		return "?" + thdString(toks, t.fields[0].node)
	}
	return thPrint(toks, t)
}

// isValidTypeHint is TypeHintHelper::isValidTypeHint.
func thdIsValidTypeHint(t string, object, static, mixed, standalone bool) bool {
	if slvTypeHintIsSimple(t) {
		return true
	}
	switch t {
	case "object":
		return object
	case "static":
		return static
	case "mixed":
		return mixed
	case "null", "true", "false":
		return standalone
	}
	return !slvTypeHintIsSimpleUnofficial(t)
}

type thdKind uint8

const (
	thdParameter thdKind = iota
	thdReturn
	thdProperty
)

// nativeTypeHint is the type hint derivation the three sniffs share before
// their fix; false where upstream gives up.
func (c *thdCtx) nativeTypeHint(kind thdKind, pointer int, toks []pdToken, original *pdNode, promoted bool) (hint string, nullable, ok bool) {
	t := original
	if t.kind == "NullableTypeNode" {
		t = t.fields[0].node
	}
	canTry := c.o.union && t.kind == "UnionTypeNode"
	var hints, traversable []string
	if thContainsOneType(t) {
		hints = append(hints, thdFromOneType(toks, t, false, c.o.standalone))
	} else if t.kind == "UnionTypeNode" || t.kind == "IntersectionTypeNode" {
		for _, inner := range t.fields[0].list {
			if !thContainsOneType(inner) {
				return "", false, false
			}
			h := thdFromOneType(toks, inner, canTry, false)
			if strings.ToLower(h) == "null" {
				nullable = true
				continue
			}
			if inner.kind != "ArrayTypeNode" && inner.kind != "ArrayShapeNode" && c.isTraversable(c.fq(pointer, h)) {
				traversable = thdAppendUnique(traversable, h)
			}
			hints = append(hints, h)
		}
		if len(traversable) > 1 && !canTry {
			return "", false, false
		}
	}
	hints = thdUnique(hints)
	if len(traversable) > 0 && slices.ContainsFunc(t.fields[0].list, func(n *pdNode) bool { return n.kind == "ArrayTypeNode" }) {
		union := canTry
		if kind == thdProperty {
			union = c.o.union
		}
		hints = c.traversableTypeHints(pointer, toks, t, union)
	}
	if len(hints) == 0 {
		return "", false, false
	}
	var converted []string
	for _, h := range hints {
		if c.o.union && thdIsUnofficialUnion(h) {
			canTry = true
			converted = append(converted, thdUnofficialUnion(h)...)
		} else {
			converted = append(converted, h)
		}
	}
	converted = thdUnique(converted)
	if len(converted) > 1 && (t.kind == "UnionTypeNode" && !canTry || t.kind == "IntersectionTypeNode" && !c.o.intersection) {
		return "", false, false
	}
	for i, h := range converted {
		if kind == thdProperty && h == "callable" {
			return "", false, false
		}
		if canTry && h == "false" {
			continue
		}
		if kind == thdParameter && promoted && h == "callable" {
			return "", false, false
		}
		var valid bool
		switch kind {
		case thdParameter:
			valid = thdIsValidTypeHint(h, c.o.object, false, c.o.mixed, c.o.standalone)
		case thdReturn:
			valid = thdIsValidTypeHint(h, c.o.object, c.o.static, c.o.mixed, c.o.standalone)
		default:
			valid = thdIsValidTypeHint(h, true, false, c.o.mixed, c.o.standalone)
		}
		if !valid || c.isTypeDefined(pointer, h) {
			return "", false, false
		}
		if kind == thdReturn {
			switch h {
			case "void", "never", "never-return", "never-returns", "no-return":
				return "", false, false
			}
		}
		switch h {
		case "integer":
			converted[i] = "int"
		case "boolean":
			converted[i] = "bool"
		}
	}
	if original.kind == "NullableTypeNode" {
		nullable = true
	}
	if slices.Contains(converted, "mixed") {
		return "mixed", nullable, true
	}
	if original.kind == "IntersectionTypeNode" {
		return strings.Join(converted, "&"), nullable, true
	}
	hint = strings.Join(converted, "|")
	if nullable {
		if len(converted) > 1 {
			hint += "|null"
		} else {
			hint = "?" + hint
		}
	}
	return hint, nullable, true
}

func thdAppendUnique(list []string, v string) []string {
	if slices.Contains(list, v) {
		return list
	}
	return append(list, v)
}

func thdUnique(list []string) []string {
	var out []string
	for _, v := range list {
		out = thdAppendUnique(out, v)
	}
	return out
}

// traversableTypeHints is AnnotationTypeHelper::getTraversableTypeHintsFromType.
func (c *thdCtx) traversableTypeHints(pointer int, toks []pdToken, t *pdNode, union bool) []string {
	var hints []string
	for _, inner := range t.fields[0].list {
		switch inner.kind {
		case "GenericTypeNode", "ThisTypeNode", "IdentifierTypeNode":
			hints = append(hints, thdFromOneType(toks, inner, false, false))
		}
	}
	if !union && len(hints) > 1 {
		return nil
	}
	for _, h := range hints {
		if !c.isTraversable(c.fq(pointer, h)) {
			return nil
		}
	}
	return hints
}

// containsTraversable is AnnotationTypeHelper::containsTraversableType.
func (c *thdCtx) containsTraversable(pointer int, t *pdNode) bool {
	switch t.kind {
	case "GenericTypeNode", "ArrayShapeNode", "ArrayTypeNode":
		return true
	case "ObjectShapeNode":
		return false
	case "IdentifierTypeNode":
		return c.isTraversable(c.fq(pointer, t.name))
	case "UnionTypeNode", "IntersectionTypeNode":
		for _, inner := range t.fields[0].list {
			if c.containsTraversable(pointer, inner) {
				return true
			}
		}
		return false
	case "ConditionalTypeNode":
		return c.containsTraversable(pointer, t.fields[2].node) || c.containsTraversable(pointer, t.fields[3].node)
	case "ConditionalTypeForParameterNode":
		return c.containsTraversable(pointer, t.fields[1].node) || c.containsTraversable(pointer, t.fields[2].node)
	}
	return false
}

// containsItemsSpecification is
// AnnotationTypeHelper::containsItemsSpecificationForTraversable.
func (c *thdCtx) containsItemsSpecification(pointer int, t *pdNode, in bool) bool {
	switch t.kind {
	case "GenericTypeNode":
		for _, g := range t.fields[1].list {
			if !c.containsItemsSpecification(pointer, g, true) {
				return false
			}
		}
		return true
	case "ArrayShapeNode", "ObjectShapeNode":
		for _, item := range t.fields[0].list {
			if !c.containsItemsSpecification(pointer, item.fields[1].node, true) {
				return false
			}
		}
		return true
	case "NullableTypeNode", "ArrayTypeNode":
		return c.containsItemsSpecification(pointer, t.fields[0].node, true)
	case "IdentifierTypeNode":
		if c.isTypeDefined(pointer, t.name) {
			return true
		}
		if !in {
			return false
		}
		return !c.isTraversable(c.fq(pointer, t.name))
	case "ThisTypeNode", "ConstTypeNode", "CallableTypeNode":
		return in
	case "UnionTypeNode", "IntersectionTypeNode":
		for _, inner := range t.fields[0].list {
			if !in && inner.kind == "IdentifierTypeNode" && strings.ToLower(inner.name) == "null" {
				continue
			}
			if c.containsItemsSpecification(pointer, inner, in) {
				return true
			}
		}
		return false
	case "ConditionalTypeNode":
		return c.containsItemsSpecification(pointer, t.fields[2].node, in) || c.containsItemsSpecification(pointer, t.fields[3].node, in)
	case "ConditionalTypeForParameterNode":
		return c.containsItemsSpecification(pointer, t.fields[1].node, in) || c.containsItemsSpecification(pointer, t.fields[2].node, in)
	}
	return false
}

// thdFunctionName is FunctionHelper::getName.
func (f *slvFile) thdFunctionName(fn int) string {
	open := f.functionParenOpen(fn)
	for j := fn + 1; j < open; j++ {
		if k := f.toks[j].kind; k == token.Ident || k == token.Keyword {
			return f.toks[j].content
		}
	}
	return ""
}

// thdIsMethod is FunctionHelper::isMethod.
func (f *slvFile) thdIsMethod(fn int) bool {
	o := f.enclosingScope(fn)
	if o < 0 {
		return false
	}
	owner := f.braceOwner(o)
	return f.isOOScopeOwner(owner) || f.isKeyword(owner, "class")
}

// thdReturnsValue is FunctionHelper::returnsValue.
func (c *thdCtx) returnsValue(fn int) bool {
	f := c.f
	closer := f.scopeCloser(fn)
	if closer < 0 {
		return false
	}
	first := f.toks[closer].closer + 1
	if c.sc == nil {
		c.sc = newPhpcsScope(f)
	}
	for i := first; i < closer; i++ {
		if f.toks[i].kind == token.Keyword && strings.HasPrefix(strings.ToLower(f.toks[i].orig), "yield") && slvlcSameScope(c.sc, i, first) {
			return true
		}
	}
	for i := first; i < closer; i++ {
		if f.isKeyword(i, "return") && slvlcSameScope(c.sc, i, first) {
			return !f.is(f.nextEffective(i+1, len(f.toks)), ";")
		}
	}
	return false
}

type thdParam struct {
	name string
	hint *thTypeHint
}

// parameters is FunctionHelper::getParametersTypeHints.
func (c *thdCtx) parameters(fn int) []thdParam {
	f, n := c.f, c.n
	open := f.functionParenOpen(fn)
	if open < 0 {
		return nil
	}
	var out []thdParam
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.toks[i].kind != token.Variable {
			continue
		}
		name := f.toks[i].content
		var hint *thTypeHint
		before := i - 1
		for before >= 0 && (f.isEmpty(before) || f.is(before, "&") || f.is(before, "...")) {
			before--
		}
		if slvIsTypeHintToken(n, before) {
			start := before - 1
			for start >= 0 && (f.toks[start].kind == token.Whitespace || slvIsTypeHintToken(n, start)) {
				start--
			}
			start = f.nextNonWhitespace(start+1, len(f.toks))
			nullable := false
			if p := f.prevEffective(start - 1); p >= 0 && f.is(p, "?") && n.nullable[p] {
				start, nullable = p, true
			}
			h := thWhitespace.ReplaceAllString(f.content(start, before), "")
			if !nullable {
				nullable = thNullInHint.MatchString(h)
			}
			hint = &thTypeHint{h, nullable, start, before}
		}
		idx := slices.IndexFunc(out, func(p thdParam) bool { return p.name == name })
		if idx >= 0 {
			out[idx].hint = hint
		} else {
			out = append(out, thdParam{name, hint})
		}
	}
	return out
}

func (f *slvFile) thdParameterPointer(fn int, name string) int {
	open := f.functionParenOpen(fn)
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.toks[i].kind == token.Variable && f.toks[i].content == name {
			return i
		}
	}
	return -1
}

type thdParamAnnotation struct {
	a slvAnnotation
	p *slvParsedDoc
}

// parameterAnnotations is FunctionHelper::getValidParametersAnnotations, or
// getValidPrefixedParametersAnnotations with the prefixes.
func (c *thdCtx) parameterAnnotations(fn int, prefixes ...string) map[string]thdParamAnnotation {
	f := c.f
	out := map[string]thdParamAnnotation{}
	tags := []string{""}
	if len(prefixes) > 0 {
		tags = prefixes
	}
	for _, prefix := range tags {
		if f.thdFunctionName(fn) == "__construct" {
			open := f.functionParenOpen(fn)
			for i := open + 1; i < f.toks[open].closer; i++ {
				if f.toks[i].kind != token.Variable {
					continue
				}
				p, vars := c.annotations(i, "@"+prefix+"var")
				if len(vars) == 0 {
					continue
				}
				out[f.toks[i].content] = thdParamAnnotation{vars[0], p}
			}
		}
		p, params := c.annotations(fn, "@"+prefix+"param")
		for _, a := range params {
			if a.isInvalid() {
				continue
			}
			v := a.node.value()
			name := v.text
			if v.kind == "TypelessParamTagValueNode" {
				name = thdTypelessName(p.toks, v)
			}
			out[name] = thdParamAnnotation{a, p}
		}
	}
	return out
}

func thdTypelessName(toks []pdToken, v *pdNode) string {
	for i := max(v.start, 0); i <= v.end && i < len(toks); i++ {
		if toks[i].t == pdVariable {
			return toks[i].v
		}
	}
	return ""
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/ParameterTypeHintSniff.php
//
// SlevomatParameterTypeHint adds the native parameter type hints the @param
// annotations allow and removes the useless @param annotations.
type SlevomatParameterTypeHint struct {
	options *thdOptions
}

func (SlevomatParameterTypeHint) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\ParameterTypeHintSniff`
}

func (SlevomatParameterTypeHint) SourceURL() string {
	return fixer.SourceURLFor(SlevomatParameterTypeHint{}.Name())
}

func (r SlevomatParameterTypeHint) WithConfig(config map[string]any) fixer.Fixer {
	o := thdParseOptions(config)
	r.options = &o
	return r
}

func (r SlevomatParameterTypeHint) opts() thdOptions {
	if r.options != nil {
		return *r.options
	}
	return thdDefaultOptions()
}

func (r SlevomatParameterTypeHint) Fix(s *tokens.Stream) bool {
	o := r.opts()
	return slvRun(s, func(f *slvFile) {
		c := newThdCtx(f, o, "SlevomatCodingStandard.TypeHints.ParameterTypeHint")
		f.eachToken(f.isNamedFunction, func(fn int) int {
			if f.functionParenOpen(fn) < 0 || c.isSuppressed(fn, c.sniff) {
				return 0
			}
			params := c.parameters(fn)
			annotations := c.parameterAnnotations(fn)
			prefixed := c.parameterAnnotations(fn, "psalm-", "phpstan-")
			c.parameterTypeHints(fn, params, annotations, prefixed)
			c.parameterTraversable(fn, params, annotations, prefixed)
			c.parameterUseless(fn, params, annotations)
			return 0
		})
	})
}

func (c *thdCtx) isInherited(fn int) bool {
	return c.hasOverride(fn) || c.hasInheritdoc(c.f.docOpenBefore(fn))
}

func (c *thdCtx) parameterTypeHints(fn int, params []thdParam, annotations, prefixed map[string]thdParamAnnotation) {
	f := c.f
	inherited := c.isInherited(fn)
	anyName, nativeName := c.name("MissingAnyTypeHint"), c.name("MissingNativeTypeHint")
	anySuppressed, nativeSuppressed := c.isSuppressed(fn, anyName), c.isSuppressed(fn, nativeName)
	suppressedErrors := 0
	constructor := f.thdIsMethod(fn) && strings.ToLower(f.thdFunctionName(fn)) == "__construct"
	open := f.functionParenOpen(fn)
	for _, param := range params {
		if param.hint != nil {
			continue
		}
		promoted := false
		if constructor {
			pointer := f.thdParameterPointer(fn, param.name)
			before := -1
			for j := pointer - 1; j >= 0; j-- {
				if f.is(j, ",") || f.is(j, "(") && !c.n.typeOps[j] {
					before = j
					break
				}
			}
			v := f.nextEffective(before+1, len(f.toks))
			promoted = f.isKeyword(v, "public") || f.isKeyword(v, "protected") || f.isKeyword(v, "private")
		}
		pa, ok := annotations[param.name]
		if !ok || pa.a.node.value().kind == "TypelessParamTagValueNode" {
			if _, ok := prefixed[param.name]; ok {
				continue
			}
			if anySuppressed {
				suppressedErrors++
			}
			continue
		}
		v := pa.a.node.value()
		if v.kind == "InvalidTagValueNode" {
			continue
		}
		t := v.fields[0].node
		if t.kind == "IdentifierTypeNode" && strings.ToLower(t.name) == "null" && !c.o.standalone {
			continue
		}
		hint, _, ok := c.nativeTypeHint(thdParameter, fn, pa.p.toks, t, promoted)
		if !ok {
			continue
		}
		if nativeSuppressed {
			suppressedErrors++
			continue
		}
		if inherited {
			continue
		}
		pointer := f.thdParameterPointer(fn, param.name)
		before := pointer
		for {
			prev := f.prevEffective(before - 1)
			if prev <= open || !f.is(prev, "&") && !f.is(prev, "...") {
				break
			}
			before = prev
		}
		f.beginChangeset()
		f.addBefore(before, hint+" ")
		f.endChangeset()
	}
	if suppressedErrors > 0 {
		return
	}
	c.reportUselessSuppress(fn, anySuppressed, anyName)
	c.reportUselessSuppress(fn, nativeSuppressed, nativeName)
}

func (c *thdCtx) parameterTraversable(fn int, params []thdParam, annotations, prefixed map[string]thdParamAnnotation) {
	name := c.name("MissingTraversableTypeHintSpecification")
	suppressed := c.isSuppressed(fn, name)
	useless := true
	for _, param := range params {
		if _, ok := prefixed[param.name]; ok {
			continue
		}
		pa, annotated := annotations[param.name]
		typed := annotated && pa.a.node.value().kind != "TypelessParamTagValueNode" && pa.a.node.value().kind != "InvalidTagValueNode"
		has := false
		if param.hint != nil && c.isTraversable(c.fq(fn, param.hint.hint)) {
			has = true
		} else if typed && c.containsTraversable(fn, pa.a.node.value().fields[0].node) {
			has = true
		}
		if has && !annotated {
			useless = false
			continue
		}
		if !typed {
			continue
		}
		t := pa.a.node.value().fields[0].node
		if !has && !c.containsTraversable(fn, t) || c.containsItemsSpecification(fn, t, false) {
			continue
		}
		useless = false
	}
	if useless {
		c.reportUselessSuppress(fn, suppressed, name)
	}
}

func (c *thdCtx) parameterUseless(fn int, params []thdParam, annotations map[string]thdParamAnnotation) {
	f := c.f
	name := c.name("UselessAnnotation")
	suppressed := c.isSuppressed(fn, name)
	useless := true
	for _, param := range params {
		pa, ok := annotations[param.name]
		if !ok || pa.a.node.value().kind == "TypelessParamTagValueNode" {
			continue
		}
		if !c.isAnnotationUseless(fn, param.hint, pa.p, pa.a) {
			continue
		}
		useless = false
		if suppressed {
			continue
		}
		start, end := f.thAnnotationPointers(pa.p, pa.a.node)
		c.removeAnnotationLine(start, pa.p.open, end)
	}
	if useless {
		c.reportUselessSuppress(fn, suppressed, name)
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/ReturnTypeHintSniff.php
//
// SlevomatReturnTypeHint adds the native return type hints the @return
// annotations allow, void to functions and closures returning nothing, and
// removes the useless @return annotations.
type SlevomatReturnTypeHint struct {
	options *thdOptions
}

func (SlevomatReturnTypeHint) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\ReturnTypeHintSniff`
}

func (SlevomatReturnTypeHint) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReturnTypeHint{}.Name())
}

func (r SlevomatReturnTypeHint) WithConfig(config map[string]any) fixer.Fixer {
	o := thdParseOptions(config)
	r.options = &o
	return r
}

func (r SlevomatReturnTypeHint) Fix(s *tokens.Stream) bool {
	o := thdDefaultOptions()
	if r.options != nil {
		o = *r.options
	}
	return slvRun(s, func(f *slvFile) {
		c := newThdCtx(f, o, "SlevomatCodingStandard.TypeHints.ReturnTypeHint")
		f.eachToken(func(i int) bool { return f.isKeyword(i, "function") }, func(fn int) int {
			if f.functionParenOpen(fn) < 0 || c.isSuppressed(fn, c.sniff) {
				return 0
			}
			if f.isClosure(fn) {
				c.closureReturnTypeHint(fn)
				return 0
			}
			th := c.n.thReturnTypeHint(fn)
			var annotation *slvAnnotation
			p, returns := c.annotations(fn, "@return")
			if len(returns) > 0 {
				annotation = &returns[0]
			}
			prefixed := 0
			_, all := c.annotations(fn, "")
			for _, a := range all {
				if !a.isInvalid() && (a.name == "@psalm-return" || a.name == "@phpstan-return") {
					prefixed++
				}
			}
			c.returnTypeHint(fn, th, p, annotation, prefixed)
			c.returnTraversable(fn, th, annotation, prefixed)
			c.returnUseless(fn, th, p, annotation)
			return 0
		})
	})
}

func thdReturnType(a *slvAnnotation) *pdNode {
	if a == nil || a.isInvalid() {
		return nil
	}
	return a.node.value().fields[0].node
}

func thdIsNever(t string) bool {
	switch t {
	case "never", "never-return", "never-returns", "no-return":
		return true
	}
	return false
}

func (c *thdCtx) returnTypeHint(fn int, th *thTypeHint, p *slvParsedDoc, annotation *slvAnnotation, prefixed int) {
	f := c.f
	inherited := c.isInherited(fn)
	anyName, nativeName := c.name("MissingAnyTypeHint"), c.name("MissingNativeTypeHint")
	anySuppressed, nativeSuppressed := c.isSuppressed(fn, anyName), c.isSuppressed(fn, nativeName)
	t := thdReturnType(annotation)
	hasAnnotation := t != nil
	isNever := t != nil && t.kind == "IdentifierTypeNode" && thdIsNever(strings.ToLower(t.name))
	if th != nil {
		c.reportUselessSuppress(fn, anySuppressed, anyName)
		c.reportUselessSuppress(fn, nativeSuppressed, nativeName)
		if c.o.never && th.hint == "void" && isNever {
			f.beginChangeset()
			f.replace(th.start, "never")
			f.endChangeset()
		}
		if c.o.standalone && th.hint == "bool" && t != nil && t.kind == "IdentifierTypeNode" {
			if lc := strings.ToLower(t.name); lc == "true" || lc == "false" {
				f.beginChangeset()
				f.replace(th.start, lc)
				f.endChangeset()
			}
		}
		return
	}
	switch f.thdFunctionName(fn) {
	case "__construct", "__destruct", "__clone":
		c.reportUselessSuppress(fn, anySuppressed, anyName)
		c.reportUselessSuppress(fn, nativeSuppressed, nativeName)
		return
	}
	voidOrNever := t != nil && t.kind == "IdentifierTypeNode" && (strings.ToLower(t.name) == "void" || isNever)
	abstract := f.scopeCloser(fn) < 0
	var returnsValue bool
	if abstract {
		returnsValue = hasAnnotation && !voidOrNever
	} else {
		returnsValue = c.returnsValue(fn)
	}
	if (returnsValue || abstract) && !hasAnnotation {
		if prefixed != 0 {
			c.reportUselessSuppress(fn, anySuppressed, anyName)
		}
		return
	}
	if !returnsValue && (!hasAnnotation || voidOrNever) {
		if !nativeSuppressed && !inherited {
			fixed := "void"
			if c.o.never && isNever {
				fixed = "never"
			}
			f.beginChangeset()
			f.add(f.toks[f.functionParenOpen(fn)].closer, ": "+fixed)
			f.endChangeset()
		}
		return
	}
	if returnsValue && voidOrNever {
		return
	}
	hint, _, ok := c.nativeTypeHint(thdReturn, fn, p.toks, t, false)
	if !ok {
		c.reportUselessSuppress(fn, nativeSuppressed, nativeName)
		return
	}
	if nativeSuppressed || inherited {
		return
	}
	f.beginChangeset()
	f.add(f.toks[f.functionParenOpen(fn)].closer, ": "+hint)
	f.endChangeset()
}

func (c *thdCtx) returnTraversable(fn int, th *thTypeHint, annotation *slvAnnotation, prefixed int) {
	name := c.name("MissingTraversableTypeHintSpecification")
	suppressed := c.isSuppressed(fn, name)
	t := thdReturnType(annotation)
	has := th != nil && c.isTraversable(c.fq(fn, strings.TrimPrefix(th.hint, "?"))) ||
		t != nil && c.containsTraversable(fn, t)
	if t == nil {
		if has && prefixed != 0 {
			c.reportUselessSuppress(fn, suppressed, name)
		}
		return
	}
	if !has && !c.containsTraversable(fn, t) || c.containsItemsSpecification(fn, t, false) {
		c.reportUselessSuppress(fn, suppressed, name)
	}
}

func (c *thdCtx) returnUseless(fn int, th *thTypeHint, p *slvParsedDoc, annotation *slvAnnotation) {
	if annotation == nil {
		return
	}
	f := c.f
	name := c.name("UselessAnnotation")
	suppressed := c.isSuppressed(fn, name)
	if !c.isAnnotationUseless(fn, th, p, *annotation) {
		c.reportUselessSuppress(fn, suppressed, name)
		return
	}
	if suppressed {
		return
	}
	start, end := f.thAnnotationPointers(p, annotation.node)
	c.removeAnnotationLine(start, p.open, end+1)
}

func (c *thdCtx) closureReturnTypeHint(fn int) {
	f := c.f
	if c.n.thReturnTypeHint(fn) != nil || c.returnsValue(fn) {
		return
	}
	closer := f.scopeCloser(fn)
	if closer < 0 {
		return
	}
	position := f.prevEffective(f.toks[closer].closer - 1)
	if position < fn {
		return
	}
	f.beginChangeset()
	f.add(position, ": void")
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/PropertyTypeHintSniff.php
//
// SlevomatPropertyTypeHint adds the native property type hints the @var
// annotations allow and removes the useless @var annotations.
type SlevomatPropertyTypeHint struct {
	options *thdOptions
}

func (SlevomatPropertyTypeHint) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\PropertyTypeHintSniff`
}

func (SlevomatPropertyTypeHint) SourceURL() string {
	return fixer.SourceURLFor(SlevomatPropertyTypeHint{}.Name())
}

func (r SlevomatPropertyTypeHint) WithConfig(config map[string]any) fixer.Fixer {
	o := thdParseOptions(config)
	r.options = &o
	return r
}

func (r SlevomatPropertyTypeHint) Fix(s *tokens.Stream) bool {
	o := thdDefaultOptions()
	if r.options != nil {
		o = *r.options
	}
	if !o.native {
		o.mixed, o.union, o.intersection, o.standalone = false, false, false, false
	}
	return slvRun(s, func(f *slvFile) {
		c := newThdCtx(f, o, "SlevomatCodingStandard.TypeHints.PropertyTypeHint")
		f.eachToken(f.isModifier, func(i int) int {
			c.property(i)
			return 0
		})
	})
}

func (c *thdCtx) property(modifier int) {
	f := c.f
	if f.isKeyword(f.prevEffective(modifier-1), "as") {
		return
	}
	if next := f.nextEffective(modifier+1, len(f.toks)); next >= 0 && f.isModifier(next) {
		return
	}
	v := -1
	for j := modifier + 1; j < len(f.toks); j++ {
		if f.toks[j].kind == token.Variable {
			v = j
			break
		}
		if f.isKeyword(j, "function") || f.isKeyword(j, "const") || f.isClassTypeKeyword(j) && !f.is(f.prevEffective(j-1), "::") {
			return
		}
	}
	if v < 0 || !f.isProperty(v) || c.isSuppressed(v, c.sniff) {
		return
	}
	var annotation *slvAnnotation
	var p *slvParsedDoc
	prefixed := 0
	open := f.docOpenBefore(v)
	if open >= 0 {
		if c.hasInheritdoc(open) {
			return
		}
		p = c.d.parse(open)
		if vars := c.d.annotations(open, "@var"); len(vars) > 0 {
			annotation = &vars[0]
		}
		for _, a := range c.d.annotations(open, "") {
			if !a.isInvalid() && (a.name == "@psalm-var" || a.name == "@phpstan-var") {
				prefixed++
			}
		}
	}
	th := c.n.thPropertyTypeHint(v)
	c.propertyTypeHint(v, th, p, annotation, prefixed)
	c.propertyTraversable(v, th, annotation, prefixed)
	c.propertyUseless(v, open, th, p, annotation)
}

func thdVarType(a *slvAnnotation) *pdNode {
	if a == nil || a.node.value().kind != "VarTagValueNode" {
		return nil
	}
	return a.node.value().fields[0].node
}

func (c *thdCtx) propertyTypeHint(v int, th *thTypeHint, p *slvParsedDoc, annotation *slvAnnotation, prefixed int) {
	f := c.f
	inherited := c.hasOverride(v)
	anyName, nativeName := c.name("MissingAnyTypeHint"), c.name("MissingNativeTypeHint")
	anySuppressed, nativeSuppressed := c.isSuppressed(v, anyName), c.isSuppressed(v, nativeName)
	if th != nil {
		c.reportUselessSuppress(v, anySuppressed, anyName)
		c.reportUselessSuppress(v, nativeSuppressed, nativeName)
		return
	}
	t := thdVarType(annotation)
	if t == nil {
		if prefixed != 0 {
			c.reportUselessSuppress(v, anySuppressed, anyName)
		}
		return
	}
	if !c.o.native {
		return
	}
	hint, nullable, ok := c.nativeTypeHint(thdProperty, v, p.toks, t, false)
	if !ok {
		c.reportUselessSuppress(v, nativeSuppressed, nativeName)
		return
	}
	if nativeSuppressed || inherited {
		return
	}
	f.beginChangeset()
	f.addBefore(v, hint+" ")
	if nullable {
		if next := f.nextEffective(v+1, len(f.toks)); f.is(next, ";") || f.is(next, ",") {
			f.add(v, " = null")
		}
	}
	f.endChangeset()
}

func (c *thdCtx) propertyTraversable(v int, th *thTypeHint, annotation *slvAnnotation, prefixed int) {
	name := c.name("MissingTraversableTypeHintSpecification")
	suppressed := c.isSuppressed(v, name)
	t := thdVarType(annotation)
	has := th != nil && c.isTraversable(c.fq(v, strings.TrimPrefix(th.hint, "?"))) ||
		t != nil && c.containsTraversable(v, t)
	if t == nil {
		if has && prefixed != 0 {
			c.reportUselessSuppress(v, suppressed, name)
		}
		return
	}
	if !has && !c.containsTraversable(v, t) || c.containsItemsSpecification(v, t, false) {
		c.reportUselessSuppress(v, suppressed, name)
	}
}

func (c *thdCtx) propertyUseless(v, open int, th *thTypeHint, p *slvParsedDoc, annotation *slvAnnotation) {
	if annotation == nil {
		return
	}
	f := c.f
	name := c.name("UselessAnnotation")
	suppressed := c.isSuppressed(v, name)
	if !c.isAnnotationUseless(v, th, p, *annotation) {
		c.reportUselessSuppress(v, suppressed, name)
		return
	}
	if suppressed {
		return
	}
	onlyVars := !f.thHasDescription(open)
	for _, a := range c.d.annotations(open, "") {
		if a.name != "@var" {
			onlyVars = false
		}
	}
	if onlyVars {
		end := f.nextEffective(p.close+1, len(f.toks)) - 1
		f.beginChangeset()
		f.removeBetweenIncluding(open, end)
		f.endChangeset()
		return
	}
	start, end := f.thAnnotationPointers(p, annotation.node)
	star := -1
	for j := start - 1; j >= 0; j-- {
		if f.toks[j].doc == slvDocStar {
			star = j
			break
		}
	}
	if star < 0 {
		return
	}
	c.removeAnnotationLine(star+1, star, end+1)
}
