package rules

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// thTypeHint is slevomat's TypeHint: the hint without whitespace and its pointers.
type thTypeHint struct {
	hint       string
	nullable   bool
	start, end int
}

var (
	thWhitespace   = regexp.MustCompile(`\s+`)
	thNullInHint   = regexp.MustCompile(`(?i)(?:^|\|)null(?:\||$)`)
	thNullInRaw    = regexp.MustCompile(`(?i)(?:^|\|\s*)null(?:\s*\||$)`)
	thNullInParam  = regexp.MustCompile(`(?i)(?:^|(?:\|\s*))null(?:(?:\s*\|)|$)`)
	thTypeOperator = regexp.MustCompile(`\s*([|&])\s*`)
)

func thInt(config map[string]any, key string, def int) int {
	switch v := config[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
		return 0
	}
	return def
}

func thBool(config map[string]any, key string, def bool) bool {
	if v, ok := config[key].(bool); ok {
		return v
	}
	return def
}

func thString(config map[string]any, key string) string {
	v, _ := config[key].(string)
	return v
}

// isTypeHint is TokenHelper::TYPE_HINT_TOKEN_CODES.
func (n *slvNames) thIsTypeHint(i int) bool {
	return i >= 0 && i < len(n.f.toks) && n.isTypeHintToken(i)
}

// thIsOnlyTypeHint is TokenHelper::ONLY_TYPE_HINT_TOKEN_CODES.
func (n *slvNames) thIsOnlyTypeHint(i int) bool {
	return n.thIsTypeHint(i) && !n.typeOps[i]
}

// thTypeStart is TypeHintHelper::getStartPointer.
func (n *slvNames) thTypeStart(end int) int {
	f := n.f
	p := end - 1
	for p >= 0 && (f.toks[p].kind == token.Whitespace || n.thIsTypeHint(p)) {
		p--
	}
	return f.nextNonWhitespace(p+1, len(f.toks))
}

func (f *slvFile) thIsVariable(i int) bool {
	return f.toks[i].kind == token.Variable && f.toks[i].orig != "$"
}

func (f *slvFile) thIsNull(i int) bool {
	if i < 0 || f.toks[i].kind != token.Ident {
		return false
	}
	lc := strings.ToLower(f.toks[i].orig)
	return lc == "null" || lc == `\null`
}

// thReturnTypeHint is FunctionHelper::findReturnTypeHint.
func (n *slvNames) thReturnTypeHint(fn int) *thTypeHint {
	f := n.f
	open := f.functionParenOpen(fn)
	if open < 0 {
		return nil
	}
	colon := f.nextEffective(f.toks[open].closer+1, len(f.toks))
	if f.isKeyword(colon, "use") {
		o := f.nextEffective(colon+1, len(f.toks))
		if !f.is(o, "(") || f.toks[o].closer < 0 {
			return nil
		}
		colon = f.nextEffective(f.toks[o].closer+1, len(f.toks))
	}
	if !f.is(colon, ":") {
		return nil
	}
	start := f.nextEffective(colon+1, len(f.toks))
	if start < 0 {
		return nil
	}
	nullable := f.is(start, "?")
	after := -1
	if f.isKeyword(fn, "fn") {
		after = f.slvmArrowOf(fn)
	} else {
		for j := start + 1; j < len(f.toks); j++ {
			if f.is(j, "{") || f.is(j, ";") {
				after = j
				break
			}
		}
	}
	if after < 0 {
		return nil
	}
	end := f.prevEffective(after - 1)
	hint := thWhitespace.ReplaceAllString(f.content(start, end), "")
	if !nullable {
		nullable = thNullInHint.MatchString(hint)
	}
	return &thTypeHint{hint, nullable, start, end}
}

// thParameterTypeHints is FunctionHelper::getParametersTypeHints without the
// parameters lacking a type hint.
func (n *slvNames) thParameterTypeHints(fn int) []*thTypeHint {
	f := n.f
	open := f.functionParenOpen(fn)
	if open < 0 {
		return nil
	}
	var out []*thTypeHint
	for i := open + 1; i < f.toks[open].closer; i++ {
		if !f.thIsVariable(i) {
			continue
		}
		p := i - 1
		for p >= 0 && (f.isEmpty(p) || f.is(p, "&") || f.is(p, "...")) {
			p--
		}
		if !n.thIsTypeHint(p) {
			continue
		}
		end := p
		start := n.thTypeStart(end)
		nullable := false
		if before := f.prevEffective(start - 1); f.is(before, "?") {
			start, nullable = before, true
		}
		hint := thWhitespace.ReplaceAllString(f.content(start, end), "")
		if !nullable {
			nullable = thNullInHint.MatchString(hint)
		}
		out = append(out, &thTypeHint{hint, nullable, start, end})
	}
	return out
}

// thPropertyTypeHint is PropertyHelper::findTypeHint.
func (n *slvNames) thPropertyTypeHint(v int) *thTypeHint {
	f := n.f
	codeEnd := -1
	for j := v - 1; j >= 0; j-- {
		if f.is(j, ";") || f.is(j, "}") || f.is(j, "{") || f.is(j, ",") || f.is(j, "(") && !n.typeOps[j] {
			codeEnd = j
			break
		}
	}
	start := -1
	for j := v - 1; j >= max(codeEnd, 0); j-- {
		if !f.isEmpty(j) {
			start = j
			break
		}
	}
	if start < 0 {
		return nil
	}
	for j := start - 1; j >= max(codeEnd, 0); j-- {
		if f.isModifier(j) {
			start = j
		}
	}
	end := -1
	for j := v - 1; j >= start; j-- {
		if n.thIsTypeHint(j) {
			end = j
			break
		}
	}
	if end < 0 {
		return nil
	}
	typeStart := n.thTypeStart(end)
	nullable := false
	for j := typeStart - 1; j >= start; j-- {
		if !f.isEmpty(j) {
			if f.is(j, "?") {
				typeStart, nullable = j, true
			}
			break
		}
	}
	raw := f.content(typeStart, end)
	if !nullable {
		nullable = thNullInRaw.MatchString(raw)
	}
	return &thTypeHint{thWhitespace.ReplaceAllString(raw, ""), nullable, typeStart, end}
}

// thFixTypeHint is the sniffs' fixTypeHint.
func (f *slvFile) thFixTypeHint(t *thTypeHint, fixed string) {
	f.beginChangeset()
	f.change(t.start, t.end, fixed)
	f.endChangeset()
}

// thWithoutNull is getTypeHintContentWithoutNull.
func (n *slvNames) thWithoutNull(t *thTypeHint) string {
	f := n.f
	if strings.EqualFold(f.toks[t.end].content, "null") {
		prev := -1
		for j := t.end - 1; j >= 0; j-- {
			if n.thIsOnlyTypeHint(j) {
				prev = j
				break
			}
		}
		return f.content(t.start, prev)
	}
	var b strings.Builder
	for i := t.start; i <= t.end; i++ {
		if strings.EqualFold(f.toks[i].content, "null") {
			for i++; i < len(f.toks) && !n.thIsOnlyTypeHint(i); i++ {
			}
			if i >= len(f.toks) {
				break
			}
		}
		b.WriteString(f.toks[i].content)
	}
	return b.String()
}

// thFormat holds the options UnionTypeHintFormat and DNFTypeHintFormat share.
type thFormat struct {
	shortNullable, nullPosition string
}

// thNullability is the short nullable and null position part of checkTypeHint.
func (n *slvNames) thNullability(t *thTypeHint, o thFormat, count int) {
	f := n.f
	if !t.nullable {
		return
	}
	short := strings.HasPrefix(t.hint, "?")
	if o.shortNullable == "yes" && count == 2 && !short {
		f.thFixTypeHint(t, "?"+n.thWithoutNull(t))
	} else if o.shortNullable == "no" && short {
		f.thFixTypeHint(t, t.hint[1:]+"|null")
	}
	if short || o.shortNullable == "yes" && count == 2 {
		return
	}
	if o.nullPosition == "first" && !strings.EqualFold(f.toks[t.start].content, "null") {
		f.thFixTypeHint(t, "null|"+n.thWithoutNull(t))
	} else if o.nullPosition == "last" && !strings.EqualFold(f.toks[t.end].content, "null") {
		f.thFixTypeHint(t, n.thWithoutNull(t)+"|null")
	}
}

// thEachTypeHint calls check for the type hints of the properties and
// functions the format sniffs register.
func (f *slvFile) thEachTypeHint(n *slvNames, i int, check func(*thTypeHint)) {
	if f.thIsVariable(i) {
		if !f.isProperty(i) {
			return
		}
		if t := n.thPropertyTypeHint(i); t != nil {
			check(t)
		}
		return
	}
	if t := n.thReturnTypeHint(i); t != nil {
		check(t)
	}
	for _, t := range n.thParameterTypeHints(i) {
		check(t)
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/ClassConstantTypeHintSniff.php
//
// SlevomatClassConstantTypeHint adds the native type of a class constant's
// literal value and drops the useless @var annotations of class constants.
type SlevomatClassConstantTypeHint struct {
	disableNative bool
	fixable       string
}

func (SlevomatClassConstantTypeHint) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\ClassConstantTypeHintSniff`
}

func (SlevomatClassConstantTypeHint) SourceURL() string {
	return fixer.SourceURLFor(SlevomatClassConstantTypeHint{}.Name())
}

func (r SlevomatClassConstantTypeHint) WithConfig(config map[string]any) fixer.Fixer {
	r.disableNative = !thBool(config, "enableNativeTypeHint", true)
	if v := thString(config, "fixableNativeTypeHint"); v != "" {
		r.fixable = v
	}
	return r
}

func (r SlevomatClassConstantTypeHint) Fix(s *tokens.Stream) bool {
	fixable := r.fixable
	if fixable == "" {
		fixable = "yes"
	}
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(func(i int) bool { return f.isKeyword(i, "const") }, func(c int) int {
			if f.classPointer(c) < 0 {
				return 0
			}
			if !r.disableNative {
				f.thConstantNativeTypeHint(c, fixable)
			}
			f.thUselessConstantAnnotation(d, c, true)
			return 0
		})
	})
}

func (f *slvFile) thConstantName(c int) int {
	for j := c + 1; j < len(f.toks); j++ {
		if f.is(j, "=") {
			return f.prevEffective(j - 1)
		}
	}
	return -1
}

func (f *slvFile) thConstantNativeTypeHint(c int, fixable string) {
	name := f.thConstantName(c)
	if name < 0 || f.prevEffective(name-1) != c {
		return
	}
	eq := f.nextEffective(name+1, len(f.toks))
	value := f.nextEffective(eq+1, len(f.toks))
	if f.is(value, "-") {
		value = f.nextEffective(value+1, len(f.toks))
	}
	typeHint := f.thLiteralType(value)
	if typeHint == "" || fixable == "no" || fixable == "private" && !f.thIsConstantPrivate(c) {
		return
	}
	f.beginChangeset()
	f.add(c, " "+typeHint)
	f.endChangeset()
}

// thLiteralType is ClassConstantTypeHintSniff::$tokenToTypeHintMapping.
func (f *slvFile) thLiteralType(i int) string {
	if i < 0 {
		return ""
	}
	t := f.toks[i]
	switch t.kind {
	case token.Ident:
		switch strings.TrimPrefix(strings.ToLower(t.orig), `\`) {
		case "true":
			return "true"
		case "false":
			return "false"
		case "null":
			return "null"
		}
	case token.Number:
		if thIsFloat(t.orig) {
			return "float"
		}
		return "int"
	case token.String:
		if t.heredoc || !t.interp {
			return "string"
		}
	case token.Punct:
		if f.isShortArrayOpen(i) {
			return "array"
		}
	}
	return ""
}

// thIsFloat tells a T_DNUMBER: a float literal or an integer too big for int.
func thIsFloat(v string) bool {
	v = strings.ReplaceAll(v, "_", "")
	lv := strings.ToLower(v)
	if strings.HasPrefix(lv, "0x") || strings.HasPrefix(lv, "0b") {
		_, err := strconv.ParseUint(lv[2:], map[byte]int{'x': 16, 'b': 2}[lv[1]], 64)
		return err != nil || !thFitsInt(lv)
	}
	if strings.ContainsAny(lv, ".e") {
		return true
	}
	_, err := strconv.ParseInt(lv, 0, 64)
	return err != nil
}

func thFitsInt(lv string) bool {
	_, err := strconv.ParseInt(lv, 0, 64)
	return err == nil
}

func (f *slvFile) thIsConstantPrivate(c int) bool {
	for j := c - 1; j >= 0; j-- {
		switch {
		case f.isKeyword(j, "private"):
			return true
		case f.is(j, "{"), f.is(j, "}"), f.is(j, ";"):
			return false
		}
	}
	return false
}

// thUselessConstantAnnotation is the doc comment check of ClassConstantTypeHint
// and UselessConstantTypeHint; typed tells the former, which keeps an
// annotation more specific than the constant's native type.
func (f *slvFile) thUselessConstantAnnotation(d *slvDocs, c int, typed bool) {
	open := f.docOpenBefore(c)
	if open < 0 {
		return
	}
	annotations := d.annotations(open, "@var")
	if len(annotations) == 0 {
		return
	}
	a := annotations[0]
	if typed {
		name := f.thConstantName(c)
		if name < 0 {
			return
		}
		if typeHint := f.prevEffective(name - 1); typeHint != c && !f.thIsAnnotationUseless(d, open, c, f.toks[typeHint].content, a) {
			return
		}
	}
	p := d.parse(open)
	var start, end int
	if !f.thHasDescription(open) && len(annotations) == 1 {
		start = open
		for start >= 0 && f.toks[start].line == f.toks[open].line {
			start--
		}
		end = p.close
	} else {
		annStart, annEnd := f.thAnnotationPointers(p, a.node)
		start = -1
		for j := annStart - 1; j >= 0; j-- {
			if f.toks[j].doc == slvDocWhitespace && f.toks[j].content == f.eol {
				start = j
				break
			}
		}
		if start < 0 {
			return
		}
		end = annEnd
	}
	f.beginChangeset()
	f.removeBetweenIncluding(start, end)
	f.endChangeset()
}

// thHasDescription is DocCommentHelper::hasDocCommentDescription of a doc comment.
func (f *slvFile) thHasDescription(open int) bool {
	closeIdx := f.docCloser(open)
	for j := open + 1; j < closeIdx; j++ {
		switch f.toks[j].doc {
		case slvDocWhitespace, slvDocStar:
			continue
		case slvDocString:
			return true
		}
		return false
	}
	return false
}

// thAnnotationPointers are ParsedDocComment::getNodeStartPointer and
// getNodeEndPointer of a tag.
func (f *slvFile) thAnnotationPointers(p *slvParsedDoc, node *pdNode) (int, int) {
	line := f.toks[p.open].line + p.toks[node.start].line - 1
	search := p.open + 1
	for i := p.open + 1; i < p.close; i++ {
		if f.toks[i].line == line {
			search = i
			break
		}
	}
	isAnnotation := func(i int) bool { return f.toks[i].doc == slvDocTag || f.toks[i].doc == slvDocString }
	start := search
	for start < len(f.toks) && !isAnnotation(start) {
		start++
	}
	var b strings.Builder
	for i := node.start; i <= node.end; i++ {
		b.WriteString(p.toks[i].v)
	}
	length := len(strings.TrimSpace(b.String()))
	end := start
	content := 0
	for i := start; i < len(f.toks); i++ {
		content += len(f.toks[i].content)
		if content >= length {
			end = i
			break
		}
	}
	for end > 0 && !isAnnotation(end) {
		end--
	}
	return start, end
}

// thIsAnnotationUseless is AnnotationHelper::isAnnotationUseless of a typed
// class constant's @var.
func (f *slvFile) thIsAnnotationUseless(d *slvDocs, open, c int, typeHint string, a slvAnnotation) bool {
	if a.isInvalid() {
		return false
	}
	p := d.parse(open)
	value := a.node.value()
	t := value.fields[0].node
	if thHasDescriptionAfter(p.toks, t.end+1, value.end) {
		return false
	}
	if typeHint == "array" || typeHint == "iterable" {
		if lc := strings.ToLower(t.name); t.kind != "IdentifierTypeNode" || lc != "array" && lc != "iterable" {
			return false
		}
	}
	if thContainsStaticOrThis(t) {
		return false
	}
	if thContainsJustTwoTypes(t) {
		return f.thTypeHintEqualsAnnotation(d, c, typeHint, thPrint(p.toks, t))
	}
	switch t.kind {
	case "ObjectShapeNode", "ConstTypeNode", "GenericTypeNode", "CallableTypeNode",
		"ConditionalTypeNode", "ConditionalTypeForParameterNode":
		return false
	case "IdentifierTypeNode":
		lc := strings.ToLower(t.name)
		if lc == "true" || lc == "false" || lc == "null" {
			return false
		}
		if slvTypeHintIsSimpleUnofficial(lc) && t.name != "object" && t.name != "mixed" {
			return false
		}
	}
	return f.thTypeHintEqualsAnnotation(d, c, typeHint, thTypeHintFromOneType(p.toks, t))
}

// thHasDescriptionAfter tells a tag value with text after its type and variable.
func thHasDescriptionAfter(toks []pdToken, from, to int) bool {
	variable := false
	for i := from; i <= to && i < len(toks); i++ {
		switch toks[i].t {
		case pdHorizontalWS, pdPhpdocEOL:
			continue
		case pdVariable, pdThisVariable:
			if !variable {
				variable = true
				continue
			}
		}
		return true
	}
	return false
}

func thContainsStaticOrThis(t *pdNode) bool {
	switch t.kind {
	case "ThisTypeNode":
		return true
	case "IdentifierTypeNode":
		return strings.ToLower(t.name) == "static"
	case "UnionTypeNode", "IntersectionTypeNode":
		return slices.ContainsFunc(t.fields[0].list, thContainsStaticOrThis)
	}
	return false
}

func thContainsJustTwoTypes(t *pdNode) bool {
	switch t.kind {
	case "NullableTypeNode":
		return thContainsOneType(t.fields[0].node)
	case "UnionTypeNode", "IntersectionTypeNode":
		return len(t.fields[0].list) == 2
	}
	return false
}

func thContainsOneType(t *pdNode) bool {
	switch t.kind {
	case "IdentifierTypeNode", "ThisTypeNode", "GenericTypeNode", "CallableTypeNode",
		"ObjectShapeNode", "ArrayShapeNode", "ArrayTypeNode":
		return true
	case "ConstTypeNode":
		switch t.fields[0].node.kind {
		case "ConstExprIntegerNode", "ConstExprFloatNode", "ConstExprStringNode":
			return true
		}
	}
	return false
}

// thRaw is the trimmed source text of a phpdoc node.
func thRaw(toks []pdToken, t *pdNode) string {
	var b strings.Builder
	for i := t.start; i <= t.end && i < len(toks); i++ {
		b.WriteString(toks[i].v)
	}
	return strings.TrimSpace(b.String())
}

// thPrint is AnnotationTypeHelper::print.
func thPrint(toks []pdToken, t *pdNode) string {
	if out, ok := pdPrint(t); ok {
		return out
	}
	return thRaw(toks, t)
}

// thTypeHintFromOneType is AnnotationTypeHelper::getTypeHintFromOneType.
func thTypeHintFromOneType(toks []pdToken, t *pdNode) string {
	switch t.kind {
	case "GenericTypeNode":
		name := thPrint(toks, t.fields[0].node)
		switch strings.ToLower(name) {
		case "non-empty-array", "list", "non-empty-list":
			return "array"
		}
		return name
	case "IdentifierTypeNode":
		lc := strings.ToLower(t.name)
		switch lc {
		case "true", "false":
			return "bool"
		case "positive-int", "non-positive-int", "negative-int", "non-negative-int", "literal-int", "int-mask":
			return "int"
		case "callable-array", "callable-string":
			return "callable"
		case "non-empty-array", "list", "non-empty-list":
			return "array"
		}
		if strings.HasSuffix(lc, "-string") {
			return "string"
		}
		return t.name
	case "ArrayTypeNode", "ArrayShapeNode":
		return "array"
	case "ObjectShapeNode":
		return "object"
	case "ConstTypeNode":
		switch t.fields[0].node.kind {
		case "ConstExprIntegerNode":
			return "int"
		case "ConstExprFloatNode":
			return "float"
		case "ConstExprStringNode":
			return "string"
		}
	}
	return thPrint(toks, t)
}

// thTypeHintEqualsAnnotation is TypeHintHelper::typeHintEqualsAnnotation.
func (f *slvFile) thTypeHintEqualsAnnotation(d *slvDocs, pointer int, typeHint, annotation string) bool {
	a := thSplitTypes(thNormalize(typeHint))
	b := thSplitTypes(thNormalize(annotation))
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == "|" || a[i] == "&") && a[i] != b[i] {
			return false
		}
		if f.thFullyQualified(d, pointer, a[i]) != f.thFullyQualified(d, pointer, b[i]) {
			return false
		}
	}
	return true
}

var thTypeSplit = regexp.MustCompile(`[&|]`)

func thSplitTypes(v string) []string {
	var out []string
	last := 0
	for _, m := range thTypeSplit.FindAllStringIndex(v, -1) {
		out = append(out, v[last:m[0]], v[m[0]:m[1]])
		last = m[1]
	}
	return append(out, v[last:])
}

func (f *slvFile) thFullyQualified(d *slvDocs, pointer int, typeHint string) string {
	if slvTypeHintIsSimple(typeHint) {
		switch typeHint {
		case "integer":
			return "int"
		case "boolean":
			return "bool"
		}
		return typeHint
	}
	return d.useStatements().resolveClassName(f, typeHint, pointer)
}

// thNormalize is TypeHintHelper::normalize.
func thNormalize(typeHint string) string {
	if strings.HasPrefix(typeHint, "?") {
		typeHint = typeHint[1:] + "|null"
	}
	switch typeHint {
	case "never", "never-return", "never-returns", "no-return":
		return "never"
	}
	delimiter := "|"
	var hints []string
	for _, part := range thSplitTypes(typeHint) {
		if part == "|" || part == "&" {
			delimiter = part
			continue
		}
		hints = append(hints, part)
	}
	if slices.Contains(hints, "mixed") {
		return "mixed"
	}
	var converted []string
	for _, h := range hints {
		switch {
		case h == "scalar" && delimiter != "&":
			converted = append(converted, "string", "int", "float", "bool")
		case h == "numeric" && delimiter != "&":
			converted = append(converted, "int", "float", "string")
		case h == "array-key" && delimiter != "&":
			converted = append(converted, "int", "string")
		default:
			converted = append(converted, h)
		}
	}
	var unique []string
	for _, h := range converted {
		if !slices.Contains(unique, h) {
			unique = append(unique, h)
		}
	}
	if len(unique) > 1 {
		for i, h := range unique {
			if h == "void" {
				unique[i] = "null"
			}
		}
	}
	sort.Strings(unique)
	return strings.Join(unique, delimiter)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/DNFTypeHintFormatSniff.php
//
// SlevomatDNFTypeHintFormat formats the spaces, the short nullable and the null
// position of native union, intersection and DNF type hints, and optionally of
// the types in doc comments.
type SlevomatDNFTypeHintFormat struct {
	disable                        bool
	docComments                    bool
	aroundOperators, inParentheses string
	format                         thFormat
}

func (SlevomatDNFTypeHintFormat) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\DNFTypeHintFormatSniff`
}

func (SlevomatDNFTypeHintFormat) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDNFTypeHintFormat{}.Name())
}

func (r SlevomatDNFTypeHintFormat) WithConfig(config map[string]any) fixer.Fixer {
	r.disable = !thBool(config, "enable", true)
	r.docComments = thBool(config, "enableForDocComments", false)
	r.aroundOperators = thString(config, "withSpacesAroundOperators")
	r.inParentheses = thString(config, "withSpacesInsideParentheses")
	r.format = thFormat{thString(config, "shortNullable"), thString(config, "nullPosition")}
	return r
}

func (r SlevomatDNFTypeHintFormat) Fix(s *tokens.Stream) bool {
	if r.disable {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		d := newSlvDocs(f)
		f.eachToken(func(i int) bool {
			return f.thIsVariable(i) || f.isFunctionToken(i) || f.isDocOpen(i)
		}, func(i int) int {
			if f.isDocOpen(i) {
				if r.docComments {
					r.processDocComment(f, d, i)
				}
				return 0
			}
			f.thEachTypeHint(n, i, func(t *thTypeHint) { r.check(n, t) })
			return 0
		})
	})
}

func (r SlevomatDNFTypeHintFormat) check(n *slvNames, t *thTypeHint) {
	f := n.f
	count := strings.Count(t.hint, "|") + strings.Count(t.hint, "&") + 1
	if count > 1 {
		original := f.content(t.start, t.end)
		isOperator := func(i int) bool { return n.typeOps[i] && (f.is(i, "|") || f.is(i, "&")) }
		switch r.aroundOperators {
		case "no":
			for i := t.start; i < t.end; i++ {
				if f.toks[i].kind == token.Whitespace && (isOperator(i-1) || isOperator(i+1)) {
					f.thFixTypeHint(t, thTypeOperator.ReplaceAllString(original, "$1"))
					break
				}
			}
		case "yes":
			for i := t.start; i < t.end; i++ {
				if isOperator(i) && (f.toks[i-1].content != " " || f.toks[i+1].content != " ") {
					f.thFixTypeHint(t, thTypeOperator.ReplaceAllString(original, " $1 "))
					break
				}
			}
		}
		isOpen := func(i int) bool { return n.typeOps[i] && f.is(i, "(") }
		isClose := func(i int) bool { return n.typeOps[i] && f.is(i, ")") }
		switch r.inParentheses {
		case "no":
			for i := t.start; i < t.end; i++ {
				if f.toks[i].kind == token.Whitespace && (isOpen(i-1) || isClose(i+1)) {
					fixed := regexp.MustCompile(`\(\s+`).ReplaceAllString(original, "(")
					f.thFixTypeHint(t, regexp.MustCompile(`\s+\)`).ReplaceAllString(fixed, ")"))
					break
				}
			}
		case "yes":
			for i := t.start; i <= t.end; i++ {
				if isOpen(i) && f.toks[i+1].content != " " || isClose(i) && f.toks[i-1].content != " " {
					fixed := regexp.MustCompile(`\(\s*`).ReplaceAllString(original, "( ")
					f.thFixTypeHint(t, regexp.MustCompile(`\s*\)`).ReplaceAllString(fixed, " )"))
					break
				}
			}
		}
	}
	if strings.Contains(t.hint, "&") {
		return
	}
	n.thNullability(t, r.format, count)
}

func (r SlevomatDNFTypeHintFormat) processDocComment(f *slvFile, d *slvDocs, open int) {
	p := d.parse(open)
	if p == nil {
		return
	}
	for _, a := range p.annotations {
		if a.isInvalid() {
			continue
		}
		if r.aroundOperators != "" {
			for _, u := range pdNodesByType(a.node, "UnionTypeNode") {
				r.docOperatorSpacing(f, p, u, "|")
			}
			for _, u := range pdNodesByType(a.node, "IntersectionTypeNode") {
				r.docOperatorSpacing(f, p, u, "&")
			}
		}
		if r.inParentheses != "" {
			for _, u := range pdNodesByType(a.node, "UnionTypeNode") {
				for _, t := range u.fields[0].list {
					if t.kind == "IntersectionTypeNode" {
						r.docParenthesesSpacing(f, p, t)
					}
				}
			}
		}
		if r.format.nullPosition != "" {
			for _, u := range pdNodesByType(a.node, "UnionTypeNode") {
				r.docNullPosition(d, p, u)
			}
		}
		switch r.format.shortNullable {
		case "no":
			for _, t := range pdNodesByType(a.node, "NullableTypeNode") {
				raw := thRaw(p.toks, t)
				inner := thPrint(p.toks, t.fields[0].node)
				sep := "|"
				if r.aroundOperators == "yes" {
					sep = " | "
				}
				fixed := inner + sep + "null"
				if r.format.nullPosition == "first" {
					fixed = "null" + sep + inner
				}
				f.thFixDocText(p, raw, fixed)
			}
		case "yes":
			for _, u := range pdNodesByType(a.node, "UnionTypeNode") {
				types := u.fields[0].list
				if len(types) != 2 {
					continue
				}
				var null, other *pdNode
				for _, t := range types {
					if t.kind == "IdentifierTypeNode" && strings.ToLower(t.name) == "null" {
						null = t
					} else {
						other = t
					}
				}
				if null == nil || other == nil || other.kind == "IntersectionTypeNode" {
					continue
				}
				f.thFixDocText(p, thRaw(p.toks, u), "?"+thPrint(p.toks, other))
			}
		}
	}
}

func (r SlevomatDNFTypeHintFormat) docOperatorSpacing(f *slvFile, p *slvParsedDoc, t *pdNode, operator string) {
	raw := thRaw(p.toks, t)
	op := regexp.QuoteMeta(operator)
	switch r.aroundOperators {
	case "no":
		if !regexp.MustCompile(`\s` + op + `|` + op + `\s`).MatchString(raw) {
			return
		}
		f.thFixDocText(p, raw, regexp.MustCompile(`\s*`+op+`\s*`).ReplaceAllLiteralString(raw, operator))
	case "yes":
		if !thMissingSpaceAround(raw, operator) {
			return
		}
		f.thFixDocText(p, raw, regexp.MustCompile(`\s*`+op+`\s*`).ReplaceAllLiteralString(raw, " "+operator+" "))
	}
}

// thMissingSpaceAround is preg_match('~(?<! )OP|OP(?! )~').
func thMissingSpaceAround(raw, operator string) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != operator[0] {
			continue
		}
		if i == 0 || raw[i-1] != ' ' || i+1 >= len(raw) || raw[i+1] != ' ' {
			return true
		}
	}
	return false
}

func (r SlevomatDNFTypeHintFormat) docParenthesesSpacing(f *slvFile, p *slvParsedDoc, t *pdNode) {
	raw := thRaw(p.toks, t)
	switch r.inParentheses {
	case "no":
		if !regexp.MustCompile(`^\(\s|\s\)$`).MatchString(raw) {
			return
		}
		fixed := regexp.MustCompile(`\(\s+`).ReplaceAllString(raw, "(")
		f.thFixDocText(p, raw, regexp.MustCompile(`\s+\)`).ReplaceAllString(fixed, ")"))
	case "yes":
		if !regexp.MustCompile(`^\([^ ]|[^ ]\)$`).MatchString(raw) {
			return
		}
		fixed := regexp.MustCompile(`^\(\s*`).ReplaceAllString(raw, "( ")
		f.thFixDocText(p, raw, regexp.MustCompile(`\s*\)$`).ReplaceAllString(fixed, " )"))
	}
}

func (r SlevomatDNFTypeHintFormat) docNullPosition(d *slvDocs, p *slvParsedDoc, u *pdNode) {
	types := u.fields[0].list
	at := -1
	for i, t := range types {
		if t.kind == "IdentifierTypeNode" && strings.ToLower(t.name) == "null" {
			at = i
			break
		}
	}
	if at < 0 || r.format.nullPosition == "last" && at == len(types)-1 || r.format.nullPosition == "first" && at == 0 {
		return
	}
	reordered := make([]*pdNode, 0, len(types))
	for i, t := range types {
		if i != at {
			reordered = append(reordered, t)
		}
	}
	if r.format.nullPosition == "first" {
		reordered = append([]*pdNode{types[at]}, reordered...)
	} else {
		reordered = append(reordered, types[at])
	}
	fixed := pdClone(u)
	fixed.fields[0] = pdL(reordered)
	d.fixNode(p.open, u, fixed)
}

// thFixDocText is fixDocCommentTypeText.
func (f *slvFile) thFixDocText(p *slvParsedDoc, original, fixed string) {
	content := strings.ReplaceAll(f.content(p.open, p.close), original, fixed)
	f.beginChangeset()
	f.change(p.open, p.close, content)
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/DeclareStrictTypesSniff.php
//
// SlevomatDeclareStrictTypes requires declare(strict_types=1) after the open
// tag, with the configured format and lines around it.
type SlevomatDeclareStrictTypes struct {
	onFirstLine         bool
	before, after, eqSp *int
}

func (SlevomatDeclareStrictTypes) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\DeclareStrictTypesSniff`
}

func (SlevomatDeclareStrictTypes) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDeclareStrictTypes{}.Name())
}

func (r SlevomatDeclareStrictTypes) WithConfig(config map[string]any) fixer.Fixer {
	r.onFirstLine = thBool(config, "declareOnFirstLine", false)
	for key, target := range map[string]**int{"linesCountBeforeDeclare": &r.before, "linesCountAfterDeclare": &r.after, "spacesCountAroundEqualsSign": &r.eqSp} {
		if _, ok := config[key]; ok {
			v := thInt(config, key, 1)
			*target = &v
		}
	}
	return r
}

func thIntOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func (r SlevomatDeclareStrictTypes) Fix(s *tokens.Stream) bool {
	before, after, eqSp := thIntOr(r.before, 1), thIntOr(r.after, 1), thIntOr(r.eqSp, 1)
	declaration := "strict_types" + strings.Repeat(" ", eqSp) + "=" + strings.Repeat(" ", eqSp) + "1"
	isOpenTag := func(f *slvFile, i int) bool {
		return f.toks[i].kind == token.OpenTag && !strings.HasPrefix(f.toks[i].orig, "<?=")
	}
	return slvRun(s, func(f *slvFile) {
		first := -1
		for i := range f.toks {
			if isOpenTag(f, i) {
				first = i
				break
			}
		}
		if first < 0 {
			return
		}
		r.process(f, first, declaration, before, after)
	})
}

func (r SlevomatDeclareStrictTypes) process(f *slvFile, open int, declaration string, linesBefore, linesAfter int) {
	declare := f.nextEffective(open+1, len(f.toks))
	if declare < 0 || !f.isKeyword(declare, "declare") {
		end := open + 1
		if strings.HasSuffix(f.toks[open].content, f.eol) || end >= len(f.toks) {
			end = open
		}
		f.beginChangeset()
		f.change(open, end, "<?php declare("+declaration+");"+f.eol)
		f.endChangeset()
		return
	}
	parenOpen := f.nextEffective(declare+1, len(f.toks))
	if !f.is(parenOpen, "(") || f.toks[parenOpen].closer < 0 {
		return
	}
	parenClose := f.toks[parenOpen].closer
	strict := -1
	for i := parenOpen + 1; i < parenClose; i++ {
		if f.toks[i].kind == token.Ident && f.toks[i].content == "strict_types" {
			strict = i
			break
		}
	}
	if strict < 0 {
		f.beginChangeset()
		f.addBefore(parenClose, ", "+declaration)
		f.endChangeset()
		return
	}
	number := -1
	for i := strict + 1; i < len(f.toks); i++ {
		if f.toks[i].kind == token.Number && !thIsFloat(f.toks[i].orig) {
			number = i
			break
		}
	}
	if number < 0 {
		return
	}
	if f.toks[number].content != "1" {
		f.beginChangeset()
		f.replace(number, "1")
		f.endChangeset()
		return
	}
	if f.content(strict, number) != declaration {
		f.beginChangeset()
		f.change(strict, number, declaration)
		f.endChangeset()
	}
	before := f.prevNonWhitespace(declare - 1)
	whitespace := ""
	if before == open && len(f.toks[open].content) > 5 {
		whitespace += f.toks[open].content[5:]
	}
	if before+1 != declare {
		whitespace += f.content(before+1, declare-1)
	}
	if r.onFirstLine {
		if whitespace != " " {
			f.beginChangeset()
			f.change(open, declare-1, "<?php ")
			f.endChangeset()
		}
	} else {
		onFirstLine := f.toks[declare].line == f.toks[open].line
		lines := linesBefore
		count := 0
		if !onFirstLine {
			count = strings.Count(whitespace, f.eol)
			if f.toks[before].kind == token.Comment && thIsLineComment(f.toks[before].content) {
				lines--
			} else {
				count--
			}
		}
		if onFirstLine || count != linesBefore {
			f.beginChangeset()
			if before == open {
				f.replace(open, "<?php")
			}
			f.removeBetweenIncluding(before+1, declare-1)
			for range lines + 1 {
				f.replaceToken(before, f.tokenContent(before)+f.eol)
			}
			f.endChangeset()
		}
	}
	semicolon := f.nextEffective(parenClose+1, len(f.toks))
	if semicolon < 0 {
		return
	}
	next := f.nextNonWhitespace(semicolon+1, len(f.toks))
	if next < 0 {
		return
	}
	newLines := strings.Count(f.content(semicolon+1, next-1), f.eol)
	count := 0
	if newLines > 0 {
		count = newLines - 1
	}
	if count == linesAfter {
		return
	}
	f.beginChangeset()
	f.removeBetweenIncluding(semicolon+1, next-1)
	for range linesAfter + 1 {
		f.replaceToken(semicolon, f.tokenContent(semicolon)+f.eol)
	}
	f.endChangeset()
}

func thIsLineComment(v string) bool {
	return strings.HasPrefix(v, "//") || strings.HasPrefix(v, "#")
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/NullableTypeForNullDefaultValueSniff.php
//
// SlevomatNullableTypeForNullDefaultValue makes the type of a parameter with a
// null default value nullable.
type SlevomatNullableTypeForNullDefaultValue struct{}

func (SlevomatNullableTypeForNullDefaultValue) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\NullableTypeForNullDefaultValueSniff`
}

func (SlevomatNullableTypeForNullDefaultValue) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNullableTypeForNullDefaultValue{}.Name())
}

func (SlevomatNullableTypeForNullDefaultValue) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		d := newSlvDocs(f)
		f.eachToken(f.isFunctionToken, func(fn int) int {
			if f.thIsSuppressed(d, fn, "SlevomatCodingStandard.TypeHints.NullableTypeForNullDefaultValue") {
				return 0
			}
			open := f.functionParenOpen(fn)
			for i := open + 1; i < f.toks[open].closer; i++ {
				if !f.thIsVariable(i) {
					continue
				}
				eq := f.nextEffective(i+1, len(f.toks))
				if !f.is(eq, "=") || !f.thIsNull(f.nextEffective(eq+1, len(f.toks))) {
					continue
				}
				end := i - 1
				for end > open && (f.isEmpty(end) || f.is(end, "&") || f.is(end, "...")) {
					end--
				}
				if end <= open || !n.thIsOnlyTypeHint(end) {
					continue
				}
				start := n.thTypeStart(end)
				typeHint := f.content(start, end)
				if strings.EqualFold(typeHint, "mixed") {
					continue
				}
				if p := f.prevEffective(start - 1); p >= open && f.is(p, "?") {
					continue
				}
				if thNullInParam.MatchString(typeHint) {
					continue
				}
				f.beginChangeset()
				if strings.Contains(typeHint, "|") {
					f.add(end, "|null")
				} else {
					f.addBefore(start, "?")
				}
				f.endChangeset()
			}
			return 0
		})
	})
}

// thIsSuppressed is SuppressHelper::isSniffSuppressed.
func (f *slvFile) thIsSuppressed(d *slvDocs, pointer int, name string) bool {
	open := f.docOpenBefore(pointer)
	if open < 0 {
		return false
	}
	p := d.parse(open)
	for _, a := range d.annotations(open, "@phpcsSuppress") {
		v := a.node.value()
		suppress := ""
		if v.start >= 0 {
			suppress = strings.SplitN(thRaw(p.toks, v), " ", 2)[0]
		}
		if suppress == name || strings.HasPrefix(name, suppress+".") {
			return true
		}
	}
	return false
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/ParameterTypeHintSpacingSniff.php
//
// SlevomatParameterTypeHintSpacing puts one space between a parameter's type
// hint and the parameter and none after its nullability symbol.
type SlevomatParameterTypeHintSpacing struct{}

func (SlevomatParameterTypeHintSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\ParameterTypeHintSpacingSniff`
}

func (SlevomatParameterTypeHintSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatParameterTypeHintSpacing{}.Name())
}

func (SlevomatParameterTypeHintSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		f.eachToken(f.isFunctionToken, func(fn int) int {
			open := f.functionParenOpen(fn)
			first, last := open+1, f.toks[open].closer-1
			for i := first; i <= last; i++ {
				if !f.thIsVariable(i) {
					continue
				}
				paramStart := first
				for j := i - 1; j >= first; j-- {
					if f.is(j, ",") {
						paramStart = j
						break
					}
				}
				bound := paramStart
				for j := i - 1; j >= paramStart; j-- {
					if f.toks[j].kind == token.AttributeEnd {
						bound = j
						break
					}
				}
				end := -1
				for j := i - 1; j >= bound; j-- {
					if n.thIsTypeHint(j) {
						end = j
						break
					}
				}
				if end < 0 {
					continue
				}
				start := n.thTypeStart(end)
				if f.toks[end+1].kind != token.Whitespace {
					f.beginChangeset()
					f.add(end, " ")
					f.endChangeset()
				} else if f.toks[end+1].content != " " {
					f.beginChangeset()
					f.replace(end+1, " ")
					f.endChangeset()
				}
				nullable := -1
				for j := start - 1; j >= paramStart; j-- {
					if !f.isEmpty(j) {
						if f.is(j, "?") {
							nullable = j
						}
						break
					}
				}
				if nullable < 0 || nullable+1 == start {
					continue
				}
				f.beginChangeset()
				f.replace(nullable+1, "")
				f.endChangeset()
			}
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/ReturnTypeHintSpacingSniff.php
//
// SlevomatReturnTypeHintSpacing puts one space after the return type colon,
// none after its nullability symbol and the configured spaces before the colon.
type SlevomatReturnTypeHintSpacing struct {
	before int
}

func (SlevomatReturnTypeHintSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\ReturnTypeHintSpacingSniff`
}

func (SlevomatReturnTypeHintSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReturnTypeHintSpacing{}.Name())
}

func (r SlevomatReturnTypeHintSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.before = thInt(config, "spacesCountBeforeColon", r.before)
	return r
}

func (r SlevomatReturnTypeHintSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		f.eachToken(f.isFunctionToken, func(fn int) int {
			t := n.thReturnTypeHint(fn)
			if t == nil {
				return 0
			}
			colon := f.prevEffective(t.start - 1)
			if f.toks[colon+1].kind != token.Whitespace {
				f.beginChangeset()
				f.add(colon, " ")
				f.endChangeset()
			} else if f.toks[colon+1].content != " " {
				f.beginChangeset()
				f.replace(colon+1, " ")
				f.endChangeset()
			}
			if f.is(t.start, "?") && f.toks[t.start+1].kind == token.Whitespace {
				f.beginChangeset()
				f.replace(t.start+1, "")
				f.endChangeset()
			}
			expected := strings.Repeat(" ", r.before)
			if !f.is(colon-1, ")") && f.toks[colon-1].content != expected {
				f.beginChangeset()
				f.replace(colon-1, expected)
				f.endChangeset()
			} else if f.is(colon-1, ")") && r.before != 0 {
				f.beginChangeset()
				f.add(colon-1, expected)
				f.endChangeset()
			}
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/UnionTypeHintFormatSniff.php
//
// SlevomatUnionTypeHintFormat formats the spaces, the short nullable and the
// null position of native union type hints.
type SlevomatUnionTypeHintFormat struct {
	disable    bool
	withSpaces string
	format     thFormat
}

func (SlevomatUnionTypeHintFormat) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\UnionTypeHintFormatSniff`
}

func (SlevomatUnionTypeHintFormat) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUnionTypeHintFormat{}.Name())
}

func (r SlevomatUnionTypeHintFormat) WithConfig(config map[string]any) fixer.Fixer {
	r.disable = !thBool(config, "enable", true)
	r.withSpaces = thString(config, "withSpaces")
	r.format = thFormat{thString(config, "shortNullable"), thString(config, "nullPosition")}
	return r
}

func (r SlevomatUnionTypeHintFormat) Fix(s *tokens.Stream) bool {
	if r.disable {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		f.eachToken(func(i int) bool { return f.thIsVariable(i) || f.isFunctionToken(i) }, func(i int) int {
			f.thEachTypeHint(n, i, func(t *thTypeHint) { r.check(n, t) })
			return 0
		})
	})
}

func (r SlevomatUnionTypeHintFormat) check(n *slvNames, t *thTypeHint) {
	f := n.f
	count := strings.Count(t.hint, "|") + 1
	if count > 1 {
		switch r.withSpaces {
		case "no":
			for i := t.start + 1; i < t.end; i++ {
				if f.toks[i].kind == token.Whitespace {
					f.thFixTypeHint(t, t.hint)
					break
				}
			}
		case "yes":
			for i := t.start; i < t.end; i++ {
				if n.typeOps[i] && f.is(i, "|") && (f.toks[i-1].content != " " || f.toks[i+1].content != " ") {
					f.thFixTypeHint(t, strings.Join(strings.Split(t.hint, "|"), " | "))
					break
				}
			}
		}
	}
	n.thNullability(t, r.format, count)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/UselessConstantTypeHintSniff.php
//
// SlevomatUselessConstantTypeHint removes the @var annotations of constants.
type SlevomatUselessConstantTypeHint struct{}

func (SlevomatUselessConstantTypeHint) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\UselessConstantTypeHintSniff`
}

func (SlevomatUselessConstantTypeHint) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessConstantTypeHint{}.Name())
}

func (SlevomatUselessConstantTypeHint) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(func(i int) bool { return f.isKeyword(i, "const") }, func(c int) int {
			f.thUselessConstantAnnotation(d, c, false)
			return 0
		})
	})
}
