package rules

import (
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clBModifiers is TokenHelper::PROPERTY_MODIFIERS_TOKEN_CODES.
var clBModifiers = map[string]bool{
	"final": true, "abstract": true, "var": true, "public": true, "public(set)": true,
	"protected": true, "protected(set)": true, "private": true, "private(set)": true,
	"readonly": true, "static": true,
}

var clBDefaultModifiersOrder = []string{
	"final, abstract",
	"var, public, public(set), protected, protected(set), private, private(set)",
	"static, readonly",
}

func (f *slvFile) clBIsModifier(i int) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Keyword && clBModifiers[strings.ToLower(f.toks[i].orig)]
}

// clBIsScopeModifier is Tokens::SCOPE_MODIFIERS plus T_READONLY.
func (f *slvFile) clBIsScopeModifier(i int) bool {
	if !f.clBIsModifier(i) {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "final", "abstract", "var", "static":
		return false
	}
	return true
}

// clBIsFunction tells a T_FUNCTION: the "function" of a named function or method.
func (f *slvFile) clBIsFunction(i int) bool {
	return f.isKeyword(i, "function") && !f.isClosure(i) && !f.isKeyword(f.prevEffective(i-1), "use")
}

func (f *slvFile) clBIsClass(i int) bool {
	return f.isKeyword(i, "class") && !f.is(f.prevEffective(i-1), "::")
}

// clBIsOpenParenthesis tells a T_OPEN_PARENTHESIS from a DNF type parenthesis.
func (n *slvNames) clBIsOpenParenthesis(i int) bool {
	return n.f.is(i, "(") && !n.typeOps[i]
}

// clBConditions are PHP_CodeSniffer's conditions of i, innermost first.
func (f *slvFile) clBConditions(i int) []int {
	var out []int
	for o := f.enclosing(i); o >= 0; o = f.enclosing(o) {
		if !f.is(o, "{") {
			continue
		}
		if c := f.scopeCondition(o); c >= 0 {
			out = append(out, c)
		}
	}
	return out
}

func (f *slvFile) clBIsOOScope(i int) bool {
	return f.isOOScopeOwner(i)
}

// clBIsProperty is PropertyHelper::isProperty.
func (n *slvNames) clBIsProperty(v int, promoted bool) bool {
	f := n.f
	p := v - 1
	for p >= 0 && (f.isEmpty(p) || n.isTypeHintToken(p) || n.nullable[p]) {
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
	if f.clBIsScopeModifier(p) {
		nested := false
		for o := f.enclosing(p); o >= 0; o = f.enclosing(o) {
			if n.clBIsOpenParenthesis(o) {
				nested = true
				break
			}
		}
		if !nested {
			return true
		}
		ctor := -1
		for j := p - 1; j >= 0; j-- {
			if f.clBIsFunction(j) {
				ctor = j
				break
			}
		}
		if ctor < 0 {
			return true
		}
		open := f.functionParenOpen(ctor)
		return open >= 0 && f.toks[open].closer < p || promoted
	}
	conditions := f.clBConditions(v)
	if len(conditions) == 0 || !f.clBIsOOScope(conditions[0]) {
		return false
	}
	for j := v - 1; j >= 0; j-- {
		if f.isKeyword(j, "function") || f.isKeyword(j, "fn") {
			return false
		}
		if f.is(j, ";") || f.is(j, "}") || f.is(j, "{") {
			break
		}
	}
	for j := v - 1; j >= 0; j-- {
		if n.clBIsOpenParenthesis(j) {
			if f.toks[j].closer > v {
				if pp := f.prevEffective(j - 1); pp >= 0 && (f.toks[pp].content == "get" || f.toks[pp].content == "set") {
					return false
				}
			}
			break
		}
	}
	for j := v - 1; j >= 0; j-- {
		if f.is(j, "{") {
			if f.toks[j].closer > v && f.scopeCondition(j) < 0 {
				return false
			}
			break
		}
	}
	return true
}

// clBStartPointer is PropertyHelper::getStartPointer.
func (n *slvNames) clBStartPointer(v int) int {
	f := n.f
	codeEnd := -1
	for j := v - 1; j >= 0; j-- {
		if f.is(j, ";") || f.is(j, "}") || f.is(j, "{") || f.is(j, ",") || n.clBIsOpenParenthesis(j) {
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
	for {
		possible := -1
		for j := start - 1; j >= max(codeEnd, 0); j-- {
			if f.clBIsModifier(j) {
				possible = j
				break
			}
		}
		if possible < 0 {
			return start
		}
		start = possible
	}
}

// clBEndPointer is PropertyHelper::getEndPointer.
func (f *slvFile) clBEndPointer(v int) int {
	for j := v + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return j
		}
		if f.is(j, "{") {
			return f.toks[j].closer
		}
	}
	return -1
}

// clBPropertyTypeHint is PropertyHelper::findTypeHint.
func (n *slvNames) clBPropertyTypeHint(v int) *thTypeHint {
	f := n.f
	start := n.clBStartPointer(v)
	end := -1
	for j := v - 1; j >= max(start, 0); j-- {
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
	for j := typeStart - 1; j >= max(start, 0); j-- {
		if !f.isEmpty(j) {
			if n.nullable[j] {
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

// clBParameterTypeHints is FunctionHelper::getParametersTypeHints.
func (n *slvNames) clBParameterTypeHints(fn int) map[string]*thTypeHint {
	f := n.f
	out := map[string]*thTypeHint{}
	open := f.functionParenOpen(fn)
	if open < 0 {
		return out
	}
	for i := open + 1; i < f.toks[open].closer; i++ {
		if !f.isVariable(i) {
			continue
		}
		name := f.toks[i].content
		p := i - 1
		for p >= 0 && (f.isEmpty(p) || f.is(p, "&") && !n.typeOps[p] || f.is(p, "...")) {
			p--
		}
		if p < 0 || !n.thIsTypeHint(p) {
			out[name] = nil
			continue
		}
		end := p
		start := n.thTypeStart(end)
		nullable := false
		if before := f.prevEffective(start - 1); before >= 0 && n.nullable[before] {
			start, nullable = before, true
		}
		hint := thWhitespace.ReplaceAllString(f.content(start, end), "")
		if !nullable {
			nullable = thNullInHint.MatchString(hint)
		}
		out[name] = &thTypeHint{hint, nullable, start, end}
	}
	return out
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/PropertyDeclarationSniff.php
//
// SlevomatPropertyDeclaration orders the modifiers of a property and fixes the
// spacing around its type hint.
type SlevomatPropertyDeclaration struct {
	modifiersOrder []string
	checkPromoted  bool
	multipleSpaces bool
}

func (SlevomatPropertyDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\PropertyDeclarationSniff`
}

func (SlevomatPropertyDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatPropertyDeclaration{}.Name())
}

func (r SlevomatPropertyDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	if order, ok := slvStringList(config["modifiersOrder"]); ok {
		r.modifiersOrder = order
	}
	r.checkPromoted = thBool(config, "checkPromoted", false)
	r.multipleSpaces = thBool(config, "enableMultipleSpacesBetweenModifiersCheck", false)
	return r
}

// clBModifierGroups is getNormalizedModifiersOrder; ok is false for an unknown
// modifier, which makes upstream throw.
func (r SlevomatPropertyDeclaration) clBModifierGroups() ([][]string, bool) {
	groups := r.modifiersOrder
	if len(groups) == 0 {
		groups = clBDefaultModifiersOrder
	}
	var out [][]string
	for _, group := range groups {
		var modifiers []string
		for m := range strings.SplitSeq(strings.ToLower(group), ",") {
			m = strings.TrimSpace(m)
			if !clBModifiers[m] {
				return nil, false
			}
			modifiers = append(modifiers, m)
		}
		out = append(out, modifiers)
	}
	return out, true
}

func (r SlevomatPropertyDeclaration) Fix(s *tokens.Stream) bool {
	groups, valid := r.clBModifierGroups()
	original := s.Render()
	aborted := false
	changed := slvRun(s, func(f *slvFile) {
		if aborted {
			return
		}
		n := newSlvNames(f)
		f.eachToken(f.clBIsModifier, func(i int) int {
			if !aborted {
				r.process(n, i, groups, valid, &aborted)
			}
			return 0
		})
	})
	if aborted {
		if changed {
			s.ReplaceRange(0, s.Len()-1, lexer.Lex(original))
		}
		return false
	}
	return changed
}

func (r SlevomatPropertyDeclaration) process(n *slvNames, modifier int, groups [][]string, valid bool, aborted *bool) {
	f := n.f
	if f.isKeyword(f.prevEffective(modifier-1), "as") {
		return
	}
	next := f.nextEffective(modifier+1, len(f.toks))
	if next < 0 || f.clBIsModifier(next) {
		return
	}
	if f.isKeyword(modifier, "static") {
		if f.is(next, "::") || f.is(next, "(") && !n.typeOps[next] || f.is(next, "{") || f.is(next, ";") || f.is(next, "|") && n.typeOps[next] {
			return
		}
	}
	v := -1
	for j := modifier + 1; j < len(f.toks); j++ {
		if f.clBIsClass(j) || f.clBIsFunction(j) || f.isKeyword(j, "const") {
			return
		}
		if f.isVariable(j) {
			v = j
			break
		}
	}
	if v < 0 || !n.clBIsProperty(v, r.checkPromoted) {
		return
	}
	first := n.clBStartPointer(v)
	if !r.checkModifiersOrder(f, v, first, modifier, groups, valid) {
		*aborted = true
		return
	}
	r.checkSpacesBetweenModifiers(f, first, modifier)
	r.checkTypeHintSpacing(n, v, modifier)
}

func (f *slvFile) clBModifierPointers(first, last int) []int {
	var out []int
	for j := max(first, 0); j <= last; j++ {
		if f.clBIsModifier(j) {
			out = append(out, j)
		}
	}
	return out
}

func (r SlevomatPropertyDeclaration) checkModifiersOrder(f *slvFile, v, first, last int, groups [][]string, valid bool) bool {
	modifiers := f.clBModifierPointers(first, last)
	if len(modifiers) < 2 {
		return true
	}
	if !valid {
		return false
	}
	expected := map[int]int{}
	for _, m := range modifiers {
		name := strings.ToLower(f.toks[m].orig)
		position := 0
		found := false
		for _, group := range groups {
			if k := slices.Index(group, name); k >= 0 {
				expected[m] = position + k
				found = true
				break
			}
			position += len(group)
		}
		if !found {
			expected[m] = position
		}
	}
	wrong := false
	for i := 1; i < len(modifiers) && !wrong; i++ {
		for j := range i {
			if expected[modifiers[i]] < expected[modifiers[j]] {
				wrong = true
				break
			}
		}
	}
	if !wrong {
		return true
	}
	sorted := slices.Clone(modifiers)
	sort.SliceStable(sorted, func(a, b int) bool { return expected[sorted[a]] < expected[sorted[b]] })
	contents := make([]string, len(sorted))
	for k, m := range sorted {
		contents[k] = f.toks[m].content
	}
	f.beginChangeset()
	f.change(first, last, strings.Join(contents, " "))
	f.endChangeset()
	return true
}

func (r SlevomatPropertyDeclaration) checkSpacesBetweenModifiers(f *slvFile, first, last int) {
	if !r.multipleSpaces {
		return
	}
	modifiers := f.clBModifierPointers(first, last)
	if len(modifiers) < 2 {
		return
	}
	wrong := false
	for i := 0; i < len(modifiers)-1; i++ {
		if f.content(modifiers[i]+1, modifiers[i+1]-1) != " " {
			wrong = true
			break
		}
	}
	if !wrong {
		return
	}
	contents := make([]string, len(modifiers))
	for k, m := range modifiers {
		contents[k] = f.toks[m].content
	}
	f.beginChangeset()
	f.change(first, last, strings.Join(contents, " "))
	f.endChangeset()
}

func (r SlevomatPropertyDeclaration) checkTypeHintSpacing(n *slvNames, v, last int) {
	f := n.f
	end := -1
	for j := v - 1; j >= last; j-- {
		if n.thIsTypeHint(j) {
			end = j
			break
		}
	}
	if end < 0 {
		return
	}
	start := n.thTypeStart(end)
	nullability := -1
	for j := start - 1; j >= last; j-- {
		if !f.isEmpty(j) {
			if n.nullable[j] {
				nullability = j
			}
			break
		}
	}
	if f.toks[last+1].kind != token.Whitespace {
		f.beginChangeset()
		f.add(last, " ")
		f.endChangeset()
	} else if f.toks[last+1].content != " " {
		f.beginChangeset()
		f.replace(last+1, " ")
		f.endChangeset()
	}
	if f.toks[end+1].kind != token.Whitespace {
		f.beginChangeset()
		f.add(end, " ")
		f.endChangeset()
	} else if f.toks[end+1].content != " " {
		f.beginChangeset()
		f.replace(end+1, " ")
		f.endChangeset()
	}
	if nullability < 0 || nullability+1 == start {
		return
	}
	f.beginChangeset()
	f.replace(nullability+1, "")
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/RequireConstructorPropertyPromotionSniff.php
//
// SlevomatRequireConstructorPropertyPromotion promotes a property its
// constructor only assigns from a parameter.
type SlevomatRequireConstructorPropertyPromotion struct {
	disabled bool
}

func (SlevomatRequireConstructorPropertyPromotion) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\RequireConstructorPropertyPromotionSniff`
}

func (SlevomatRequireConstructorPropertyPromotion) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireConstructorPropertyPromotion{}.Name())
}

func (r SlevomatRequireConstructorPropertyPromotion) WithConfig(config map[string]any) fixer.Fixer {
	r.disabled = !thBool(config, "enable", true)
	return r
}

func (r SlevomatRequireConstructorPropertyPromotion) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		c := &clBPromotion{f: f, d: newSlvDocs(f), n: newSlvNames(f)}
		f.eachToken(f.clBIsFunction, func(i int) int {
			c.process(i)
			return 0
		})
	})
}

type clBPromotion struct {
	f *slvFile
	d *slvDocs
	n *slvNames
}

func (c *clBPromotion) process(fn int) {
	f, n := c.f, c.n
	name := f.nextEffective(fn+1, len(f.toks))
	if name < 0 || strings.ToLower(f.toks[name].content) != "__construct" {
		return
	}
	closeIdx := f.scopeCloser(fn)
	if closeIdx < 0 {
		return
	}
	open := f.functionParenOpen(fn)
	if open < 0 {
		return
	}
	var parameters []int
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.isVariable(i) {
			parameters = append(parameters, i)
		}
	}
	var withoutPromotion []int
	for _, p := range parameters {
		before := -1
		for j := p - 1; j >= 0; j-- {
			if f.is(j, ",") || n.clBIsOpenParenthesis(j) {
				before = j
				break
			}
		}
		if f.clBIsModifier(f.nextEffective(before+1, len(f.toks))) {
			continue
		}
		before = f.prevEffective(p - 1)
		if f.is(before, "...") {
			continue
		}
		if f.is(before, "&") && !n.typeOps[before] {
			before = f.prevEffective(before - 1)
		}
		if f.isKeyword(before, "callable") {
			continue
		}
		withoutPromotion = append(withoutPromotion, p)
	}
	if len(withoutPromotion) == 0 {
		return
	}
	class := -1
	for _, cond := range f.clBConditions(fn) {
		if f.clBIsOOScope(cond) {
			class = cond
			break
		}
	}
	if class < 0 {
		return
	}
	classClose := f.scopeCloser(class)
	if classClose < 0 {
		return
	}
	var properties []int
	for j := f.toks[classClose].closer + 1; j < classClose; j++ {
		if f.isVariable(j) && n.clBIsProperty(j, false) {
			properties = append(properties, j)
		}
	}
	if len(properties) == 0 {
		return
	}
	paramTypeHints := n.clBParameterTypeHints(fn)
	for _, p := range withoutPromotion {
		parameterName := f.toks[p].content
		for _, prop := range properties {
			if f.toks[prop].content != parameterName {
				continue
			}
			c.promote(fn, closeIdx, p, prop, paramTypeHints[parameterName])
		}
	}
}

func (c *clBPromotion) promote(fn, closeIdx, p, prop int, parameterTypeHint *thTypeHint) {
	f, n := c.f, c.n
	parameterName := f.toks[p].content
	propertyEnd := f.clBEndPointer(prop)
	if propertyEnd < 0 || f.is(propertyEnd, "}") {
		return
	}
	if c.isDocCommentUseful(prop) || c.hasAttribute(prop) {
		return
	}
	propertyTypeHint := n.clBPropertyTypeHint(prop)
	if (parameterTypeHint == nil) != (propertyTypeHint == nil) {
		return
	}
	if parameterTypeHint != nil && parameterTypeHint.hint != propertyTypeHint.hint {
		return
	}
	assignment := c.assignment(f.toks[closeIdx].closer, closeIdx, parameterName)
	if assignment < 0 || c.isModifiedBefore(f.toks[closeIdx].closer, parameterName, assignment) {
		return
	}

	docOpen := f.docOpenBefore(prop)
	beforeProperty := prop
	if docOpen >= 0 {
		beforeProperty = docOpen
	}
	beforeProperty = f.firstOnLine(beforeProperty)
	propertyStart := n.clBStartPointer(prop)
	modifiers := f.clBModifierPointers(propertyStart, prop-1)
	if len(modifiers) == 0 {
		return
	}
	modifiersContent := f.content(modifiers[0], modifiers[len(modifiers)-1])

	hasDefault := false
	defaultValue := ""
	for j := prop + 1; j < propertyEnd; j++ {
		if f.is(j, "=") {
			hasDefault = true
			defaultValue = strings.TrimSpace(f.content(j+1, propertyEnd-1))
			break
		}
	}
	semicolon := -1
	for j := prop + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			semicolon = j
			break
		}
	}
	nextCode := f.nextNonWhitespace(semicolon+1, len(f.toks))
	if semicolon < 0 || nextCode < 0 {
		return
	}
	afterProperty := f.firstOnLine(nextCode)

	beforeParameter := -1
	for j := p - 1; j >= 0; j-- {
		if f.is(j, ",") || n.clBIsOpenParenthesis(j) || f.toks[j].kind == token.AttributeEnd {
			beforeParameter = j
			break
		}
	}
	parameterStart := f.nextEffective(beforeParameter+1, len(f.toks))
	parameterHasDefault := f.is(f.nextEffective(p+1, len(f.toks)), "=")

	beforeAssignment := f.firstOnLine(assignment - 1)
	afterAssignment := f.lastOnLine(assignment)

	f.beginChangeset()
	f.removeBetweenIncluding(beforeProperty, afterProperty-1)
	f.addBefore(parameterStart, modifiersContent+" ")
	if !parameterHasDefault && hasDefault {
		f.add(p, " = "+defaultValue)
	}
	f.removeBetweenIncluding(beforeAssignment, afterAssignment)
	f.endChangeset()
}

func (c *clBPromotion) isDocCommentUseful(prop int) bool {
	f := c.f
	open := f.docOpenBefore(prop)
	if open < 0 {
		return false
	}
	if f.thHasDescription(open) {
		return true
	}
	p := c.d.parse(open)
	for _, a := range c.d.annotations(open, "") {
		value := a.node.value()
		if value.kind != "VarTagValueNode" {
			return true
		}
		if thdHasDescription(p.toks, value) {
			return true
		}
	}
	return false
}

func (c *clBPromotion) hasAttribute(prop int) bool {
	f := c.f
	for j := prop - 1; j >= 0; j-- {
		if f.toks[j].kind == token.AttributeEnd {
			return true
		}
		if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			return false
		}
	}
	return false
}

// assignment is getAssignment: the "$this" of "$this->name = $name;", or -1.
func (c *clBPromotion) assignment(open, closeIdx int, parameterName string) int {
	f := c.f
	name := strings.TrimPrefix(parameterName, "$")
	for i := open + 1; i < closeIdx; i++ {
		if f.toks[i].content != "$this" {
			continue
		}
		operator := f.nextEffective(i+1, len(f.toks))
		if !f.is(operator, "->") {
			continue
		}
		namePointer := f.nextEffective(operator+1, len(f.toks))
		if namePointer < 0 || f.toks[namePointer].content != name {
			continue
		}
		equal := f.nextEffective(namePointer+1, len(f.toks))
		if !f.is(equal, "=") {
			continue
		}
		variable := f.nextEffective(equal+1, len(f.toks))
		if variable < 0 || f.toks[variable].content != parameterName {
			continue
		}
		semicolon := f.nextEffective(variable+1, len(f.toks))
		if !f.is(semicolon, ";") {
			continue
		}
		for _, cond := range f.clBConditions(semicolon) {
			for _, k := range []string{"if", "elseif", "else", "switch"} {
				if f.isKeyword(cond, k) {
					return -1
				}
			}
		}
		return i
	}
	return -1
}

var clBAssignmentOperators = map[string]bool{
	"=": true, "&=": true, "|=": true, ".=": true, "/=": true, "-=": true, "**=": true, "%=": true,
	"*=": true, "+=": true, "^=": true, "<<=": true, ">>=": true, "??=": true,
}

func (c *clBPromotion) isModifiedBefore(open int, parameterName string, assignment int) bool {
	f := c.f
	for i := assignment - 1; i > open; i-- {
		if !f.isVariable(i) || f.toks[i].content != parameterName {
			continue
		}
		next := f.nextEffective(i+1, len(f.toks))
		if next < 0 {
			continue
		}
		if f.toks[next].kind == token.Punct && clBAssignmentOperators[f.toks[next].orig] {
			return true
		}
		if f.is(next, "=>") && f.isDoubleArrow(next) {
			return true
		}
		if f.is(next, "++") {
			return true
		}
		// upstream searches forward from the token before the variable
		if f.is(f.nextEffective(i-1, len(f.toks)), "--") {
			return true
		}
	}
	return false
}
