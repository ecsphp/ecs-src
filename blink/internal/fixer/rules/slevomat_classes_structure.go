package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ClassStructureSniff.php
//
// SlevomatClassStructure moves class members into the configured order of
// groups (uses, constants, properties, constructor, methods, ...).
type SlevomatClassStructure struct {
	groups       []string
	methodGroups [][2]string
}

const (
	clAUses                         = "uses"
	clAPublicConstants              = "public constants"
	clAProtectedConstants           = "protected constants"
	clAPrivateConstants             = "private constants"
	clAPublicProperties             = "public properties"
	clAPublicStaticProperties       = "public static properties"
	clAProtectedProperties          = "protected properties"
	clAProtectedStaticProperties    = "protected static properties"
	clAPrivateProperties            = "private properties"
	clAPrivateStaticProperties      = "private static properties"
	clAConstructor                  = "constructor"
	clAStaticConstructors           = "static constructors"
	clADestructor                   = "destructor"
	clAInvokeMethod                 = "invoke method"
	clAMagicMethods                 = "magic methods"
	clAPublicMethods                = "public methods"
	clAPublicAbstractMethods        = "public abstract methods"
	clAPublicFinalMethods           = "public final methods"
	clAPublicStaticMethods          = "public static methods"
	clAPublicStaticAbstractMethods  = "public static abstract methods"
	clAPublicStaticFinalMethods     = "public static final methods"
	clAProtectedMethods             = "protected methods"
	clAProtectedAbstractMethods     = "protected abstract methods"
	clAProtectedFinalMethods        = "protected final methods"
	clAProtectedStaticMethods       = "protected static methods"
	clAProtectedStaticAbstractMeths = "protected static abstract methods"
	clAProtectedStaticFinalMethods  = "protected static final methods"
	clAPrivateMethods               = "private methods"
	clAPrivateStaticMethods         = "private static methods"
	clAEnumCases                    = "enum cases"

	clAShortcutFinalMethods = "final methods"
)

var clASupportedGroups = []string{
	clAUses, clAEnumCases, clAPublicConstants, clAProtectedConstants, clAPrivateConstants,
	clAPublicProperties, clAPublicStaticProperties, clAProtectedProperties, clAProtectedStaticProperties,
	clAPrivateProperties, clAPrivateStaticProperties, clAPublicStaticFinalMethods, clAPublicStaticAbstractMethods,
	clAProtectedStaticFinalMethods, clAProtectedStaticAbstractMeths, clAPublicFinalMethods, clAPublicAbstractMethods,
	clAProtectedFinalMethods, clAProtectedAbstractMethods, clAConstructor, clAStaticConstructors, clADestructor,
	clAPublicMethods, clAPublicStaticMethods, clAProtectedMethods, clAProtectedStaticMethods, clAPrivateMethods,
	clAPrivateStaticMethods, clAMagicMethods,
}

var clAShortcuts = map[string][]string{
	"constants":         {clAPublicConstants, clAProtectedConstants, clAPrivateConstants},
	"static properties": {clAPublicStaticProperties, clAProtectedStaticProperties, clAPrivateStaticProperties},
	"properties":        {"static properties", clAPublicProperties, clAProtectedProperties, clAPrivateProperties},
	"all public methods": {clAPublicFinalMethods, clAPublicStaticFinalMethods, clAPublicAbstractMethods,
		clAPublicStaticAbstractMethods, clAPublicStaticMethods, clAPublicMethods},
	"all protected methods": {clAProtectedFinalMethods, clAProtectedStaticFinalMethods, clAProtectedAbstractMethods,
		clAProtectedStaticAbstractMeths, clAProtectedStaticMethods, clAProtectedMethods},
	"all private methods":   {clAPrivateStaticMethods, clAPrivateMethods},
	clAShortcutFinalMethods: {clAPublicFinalMethods, clAProtectedFinalMethods, clAPublicStaticFinalMethods, clAProtectedStaticFinalMethods},
	"abstract methods": {clAPublicAbstractMethods, clAProtectedAbstractMethods, clAPublicStaticAbstractMethods,
		clAProtectedStaticAbstractMeths},
	"static methods": {clAStaticConstructors, clAPublicStaticFinalMethods, clAProtectedStaticFinalMethods,
		clAPublicStaticAbstractMethods, clAProtectedStaticAbstractMeths, clAPublicStaticMethods,
		clAProtectedStaticMethods, clAPrivateStaticMethods},
	"methods": {clAShortcutFinalMethods, "abstract methods", "static methods", clAConstructor, clADestructor,
		clAPublicMethods, clAProtectedMethods, clAPrivateMethods, clAMagicMethods},
}

var clASpecialMethods = map[string]string{
	"__construct": clAConstructor, "__destruct": clADestructor, "__call": clAMagicMethods,
	"__callstatic": clAMagicMethods, "__get": clAMagicMethods, "__set": clAMagicMethods,
	"__isset": clAMagicMethods, "__unset": clAMagicMethods, "__sleep": clAMagicMethods,
	"__wakeup": clAMagicMethods, "__serialize": clAMagicMethods, "__unserialize": clAMagicMethods,
	"__tostring": clAMagicMethods, "__invoke": clAInvokeMethod, "__set_state": clAMagicMethods,
	"__clone": clAMagicMethods, "__debuginfo": clAMagicMethods,
}

func (SlevomatClassStructure) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ClassStructureSniff`
}

func (SlevomatClassStructure) SourceURL() string {
	return fixer.SourceURLFor(SlevomatClassStructure{}.Name())
}

// WithConfig takes methodGroups in key order where a map lost it; the ECS
// loader calls WithRawConfig, which keeps the order.
func (r SlevomatClassStructure) WithConfig(config map[string]any) fixer.Fixer {
	var keys []string
	if m, ok := config["methodGroups"].(map[string]any); ok {
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	return r.configure(config, keys)
}

func (r SlevomatClassStructure) WithRawConfig(raw []byte) fixer.Fixer {
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return r
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return r
	}
	keys, err := jsonObjectKeys(fields["methodGroups"])
	if err != nil {
		return r
	}
	return r.configure(config, keys)
}

func (r SlevomatClassStructure) configure(config map[string]any, keys []string) fixer.Fixer {
	if v, ok := config["groups"].([]any); ok {
		r.groups = nil
		for _, item := range v {
			if s, ok := item.(string); ok {
				r.groups = append(r.groups, s)
			}
		}
	}
	switch v := config["methodGroups"].(type) {
	case map[string]any:
		r.methodGroups = nil
		for _, k := range keys {
			r.methodGroups = clAAppendMethodGroup(r.methodGroups, k, v[k])
		}
	case []any:
		r.methodGroups = nil
		for i, value := range v {
			r.methodGroups = clAAppendMethodGroup(r.methodGroups, fmt.Sprint(i), value)
		}
	}
	return r
}

// clAAppendMethodGroup is one entry of SniffSettingsHelper::normalizeAssociativeArray.
func clAAppendMethodGroup(list [][2]string, key string, value any) [][2]string {
	key = strings.TrimSpace(key)
	var v string
	switch value := value.(type) {
	case string:
		v = strings.TrimSpace(value)
	case float64:
		v = fmt.Sprint(value)
	default:
		return list
	}
	if key == "" || v == "" {
		return list
	}
	for i := range list {
		if list[i][0] == key {
			list[i][1] = v
			return list
		}
	}
	return append(list, [2]string{key, v})
}

type clAMethodRequirement struct {
	name        string
	hasName     bool
	attributes  []string
	annotations []string
}

type clAMethodGroup struct {
	group        string
	requirements []clAMethodRequirement
}

// clASplitList is preg_split('~\s*,\s*~', $v, -1, PREG_SPLIT_NO_EMPTY) of a
// trimmed string.
func clASplitList(v string) []string {
	var out []string
	for part := range strings.SplitSeq(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// methodGroupsNormalized is getNormalizedMethodGroups.
func (r SlevomatClassStructure) methodGroupsNormalized() []clAMethodGroup {
	var out []clAMethodGroup
	for _, entry := range r.methodGroups {
		group := strings.ToLower(entry[0])
		idx := slices.IndexFunc(out, func(g clAMethodGroup) bool { return g.group == group })
		if idx < 0 {
			out = append(out, clAMethodGroup{group: group})
			idx = len(out) - 1
		}
		out[idx].requirements = nil
		for _, definition := range clASplitList(entry[1]) {
			// preg_split('~(?=[#@])~')
			var parts []string
			last := 0
			for i := 0; i < len(definition); i++ {
				if definition[i] == '#' || definition[i] == '@' {
					parts = append(parts, definition[last:i])
					last = i
				}
			}
			parts = append(parts, definition[last:])
			req := clAMethodRequirement{name: parts[0], hasName: parts[0] != ""}
			for _, p := range parts[1:] {
				if p[0] == '#' {
					req.attributes = append(req.attributes, p[1:])
				} else {
					req.annotations = append(req.annotations, p[1:])
				}
			}
			out[idx].requirements = append(out[idx].requirements, req)
		}
	}
	return out
}

// groupsNormalized is getNormalizedGroups; it is false where upstream throws
// an UnsupportedClassGroupException or a MissingClassGroupsException.
func (r SlevomatClassStructure) groupsNormalized(methodGroups []clAMethodGroup) (map[string]int, bool) {
	isMethodGroup := func(g string) bool {
		return slices.ContainsFunc(methodGroups, func(m clAMethodGroup) bool { return m.group == g })
	}
	var keys []string
	withShortcuts := map[string]int{}
	order := 1
	for _, raw := range r.groups {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, g := range clASplitList(strings.ToLower(raw)) {
			g = strings.Join(strings.Fields(g), " ")
			_, shortcut := clAShortcuts[g]
			if !slices.Contains(clASupportedGroups, g) && !shortcut && g != clAInvokeMethod && !isMethodGroup(g) {
				return nil, false
			}
			if _, ok := withShortcuts[g]; !ok {
				keys = append(keys, g)
			}
			withShortcuts[g] = order
		}
		order++
	}
	normalized := map[string]int{}
	for _, g := range keys {
		groupOrder := withShortcuts[g]
		if slices.Contains(clASupportedGroups, g) || g == clAInvokeMethod || isMethodGroup(g) {
			normalized[g] = groupOrder
			continue
		}
		for _, unpacked := range clAUnpackShortcut(g) {
			if _, ok := withShortcuts[unpacked]; ok {
				continue
			}
			if _, ok := normalized[unpacked]; ok {
				continue
			}
			normalized[unpacked] = groupOrder
		}
	}
	if len(normalized) == 0 && len(methodGroups) == 0 {
		for i, g := range clASupportedGroups {
			normalized[g] = i
		}
		return normalized, true
	}
	for _, g := range clASupportedGroups {
		if _, ok := normalized[g]; !ok {
			return nil, false
		}
	}
	for _, m := range methodGroups {
		if _, ok := normalized[m.group]; !ok {
			return nil, false
		}
	}
	return normalized, true
}

func clAUnpackShortcut(shortcut string) []string {
	var groups []string
	for _, item := range clAShortcuts[shortcut] {
		_, isShortcut := clAShortcuts[item]
		switch {
		case slices.Contains(clASupportedGroups, item):
			groups = append(groups, item)
		case !isShortcut && slices.Contains(clAShortcuts[clAShortcutFinalMethods], item):
		default:
			groups = append(groups, clAUnpackShortcut(item)...)
		}
	}
	return groups
}

func (r SlevomatClassStructure) Fix(s *tokens.Stream) bool {
	methodGroups := r.methodGroupsNormalized()
	order, ok := r.groupsNormalized(methodGroups)
	if !ok {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		if !slices.ContainsFunc(f.toks, func(t slvTok) bool {
			return t.kind == token.Keyword && clAIsClassKeyword(t.orig)
		}) {
			return
		}
		c := &clACtx{f: f, sc: newPhpcsScope(f), order: order, methodGroups: methodGroups}
		_, c.levels = c.sc.levelMap()
		f.eachToken(c.isOOScope, func(i int) int {
			c.process(i)
			return 0
		})
	})
}

func clAIsClassKeyword(v string) bool {
	switch strings.ToLower(v) {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

type clACtx struct {
	f            *slvFile
	sc           *phpcsScope
	levels       []int
	order        map[string]int
	methodGroups []clAMethodGroup
	docs         *slvDocs
	uses         *slvUseStatements
	classes      []int
}

// isOOScope tells a Tokens::OO_SCOPE_TOKENS token with a scope.
func (c *clACtx) isOOScope(i int) bool {
	switch c.sc.code[i] {
	case "class", "interface", "trait", "enum":
		return c.scopeCloser(i) >= 0
	}
	return false
}

// scopeCloser is the scope_closer of a scope condition, or -1.
func (c *clACtx) scopeCloser(i int) int {
	if c.sc.cond[i] != i || c.sc.opener[i] < 0 {
		return -1
	}
	return c.sc.closer[i]
}

type clAFirstMember struct {
	group   string
	pointer int
}

func (c *clACtx) process(classPointer int) {
	rootLevel := c.levels[classPointer]
	closer := c.sc.closer[classPointer]
	lastMember := c.sc.opener[classPointer]
	expected := ""
	var firstMembers []clAFirstMember
	for {
		first, last, group, ok := c.findNextGroup(lastMember, closer, rootLevel)
		if !ok {
			return
		}
		lastMember = last
		if _, ok := c.order[group]; group == clAInvokeMethod && !ok {
			group = clAMagicMethods
		}
		expectedOrder := 0
		if expected != "" {
			expectedOrder = c.order[expected]
		}
		if c.order[group] >= expectedOrder {
			if k := slices.IndexFunc(firstMembers, func(m clAFirstMember) bool { return m.group == group }); k >= 0 {
				firstMembers[k].pointer = first
			} else {
				firstMembers = append(firstMembers, clAFirstMember{group, first})
			}
			expected = group
			continue
		}
		for _, m := range firstMembers {
			if c.order[m.group] <= c.order[group] {
				continue
			}
			c.fixIncorrectGroupOrder(first, last, m.pointer)
			return
		}
	}
}

func (c *clACtx) isMember(i int) bool {
	switch c.sc.code[i] {
	case "use", "enum_case", "const":
		return true
	case "variable":
		return c.f.isVariable(i)
	case "function":
		return !c.f.isClosure(i)
	}
	return false
}

// findNextGroup is findNextGroup: the first and last member and the group of
// the next run of members of one group after pointer.
func (c *clACtx) findNextGroup(pointer, closer, rootLevel int) (first, last int, group string, ok bool) {
	f := c.f
	current := pointer
	for {
		next := -1
		for i := current + 1; i < closer; i++ {
			if c.isMember(i) {
				next = i
				break
			}
		}
		if next < 0 {
			break
		}
		current = next
		if c.sc.code[current] == "variable" && !f.isProperty(current) {
			continue
		}
		if c.levels[current]-rootLevel != 1 {
			continue
		}
		g := c.groupFor(current)
		if !ok {
			group, first, ok = g, current, true
		}
		if g != group {
			break
		}
		last = current
		if c.sc.code[current] == "variable" {
			current = c.propertyEnd(current)
		} else if sc := c.scopeCloser(current); sc >= 0 {
			current = sc
		}
	}
	return first, last, group, ok
}

// propertyEnd is PropertyHelper::getEndPointer.
func (c *clACtx) propertyEnd(v int) int {
	f := c.f
	for i := v + 1; i < len(f.toks); i++ {
		if f.is(i, ";") {
			return i
		}
		if f.is(i, "{") {
			return f.toks[i].closer
		}
	}
	return len(f.toks) - 1
}

func (c *clACtx) groupFor(p int) string {
	f := c.f
	switch c.sc.code[p] {
	case "use":
		return clAUses
	case "enum_case":
		return clAEnumCases
	case "const":
		switch c.visibility(p) {
		case "public":
			return clAPublicConstants
		case "protected":
			return clAProtectedConstants
		}
		return clAPrivateConstants
	case "function":
		name := strings.ToLower(f.thdFunctionName(p))
		if g, ok := clASpecialMethods[name]; ok {
			return g
		}
		if g, ok := c.resolveMethodGroup(p, name); ok {
			return g
		}
		visibility := c.visibility(p)
		static := c.previousIs(p, "static")
		final := c.previousIs(p, "final")
		if c.previousIs(p, "abstract") {
			if visibility == "public" {
				return clAPick(static, clAPublicStaticAbstractMethods, clAPublicAbstractMethods)
			}
			return clAPick(static, clAProtectedStaticAbstractMeths, clAProtectedAbstractMethods)
		}
		if static && visibility == "public" && c.isStaticConstructor(p) {
			return clAStaticConstructors
		}
		switch visibility {
		case "public":
			if final {
				return clAPick(static, clAPublicStaticFinalMethods, clAPublicFinalMethods)
			}
			return clAPick(static, clAPublicStaticMethods, clAPublicMethods)
		case "protected":
			if final {
				return clAPick(static, clAProtectedStaticFinalMethods, clAProtectedFinalMethods)
			}
			return clAPick(static, clAProtectedStaticMethods, clAProtectedMethods)
		}
		return clAPick(static, clAPrivateStaticMethods, clAPrivateMethods)
	}
	static := c.previousIs(p, "static")
	switch c.visibility(p) {
	case "public", "public(set)":
		return clAPick(static, clAPublicStaticProperties, clAPublicProperties)
	case "protected":
		return clAPick(static, clAProtectedStaticProperties, clAProtectedProperties)
	}
	return clAPick(static, clAPrivateStaticProperties, clAPrivateProperties)
}

func clAPick(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

func (c *clACtx) isMemberEnd(i int) bool {
	return c.f.is(i, "{") || c.f.is(i, "}") || c.f.is(i, ";")
}

// visibility is getVisibilityForToken: the lowercased scope modifier before
// p, skipping protected(set) and private(set), or "public".
func (c *clACtx) visibility(p int) string {
	f := c.f
	prev := p - 1
	for {
		found := -1
		for i := prev - 1; i >= 0; i-- {
			if c.isMemberEnd(i) || clAIsScopeModifier(f, i) {
				found = i
				break
			}
		}
		if found < 0 || c.isMemberEnd(found) {
			return "public"
		}
		v := strings.ToLower(f.toks[found].orig)
		if v == "protected(set)" || v == "private(set)" {
			prev = found
			continue
		}
		return v
	}
}

func clAIsScopeModifier(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "public", "protected", "private", "public(set)", "protected(set)", "private(set)":
		return true
	}
	return false
}

// previousIs tells whether the keyword comes before p in its declaration
// (isMemberStatic, isMethodFinal, isMethodAbstract).
func (c *clACtx) previousIs(p int, keyword string) bool {
	for i := p - 1; i >= 0; i-- {
		if c.isMemberEnd(i) {
			return false
		}
		if c.f.isKeyword(i, keyword) {
			return true
		}
	}
	return false
}

func (c *clACtx) resolveMethodGroup(p int, method string) (string, bool) {
	for _, g := range c.methodGroups {
		for _, req := range g.requirements {
			if req.hasName {
				required := strings.ToLower(req.name)
				switch {
				case strings.HasSuffix(required, "*"):
					prefix := required[:len(required)-1]
					if method == prefix || !strings.HasPrefix(method, prefix) {
						continue
					}
				case strings.HasPrefix(required, "*"):
					suffix := required[1:]
					if method == suffix || !strings.HasSuffix(method, suffix) {
						continue
					}
				case method != required:
					continue
				}
			}
			if c.hasAnnotations(p, req.annotations) && c.hasAttributes(p, req.attributes) {
				return g.group, true
			}
		}
	}
	return "", false
}

func (c *clACtx) docsOf() *slvDocs {
	if c.docs == nil {
		c.docs = newSlvDocs(c.f)
	}
	return c.docs
}

func (c *clACtx) hasAnnotations(p int, required []string) bool {
	if len(required) == 0 {
		return true
	}
	names := map[string]bool{}
	if open := c.f.docOpenBefore(p); open >= 0 {
		for _, a := range c.docsOf().annotations(open, "") {
			names[a.name] = true
		}
	}
	for _, r := range required {
		if !names["@"+r] {
			return false
		}
	}
	return true
}

func (c *clACtx) hasAttributes(p int, required []string) bool {
	if len(required) == 0 {
		return true
	}
	names := map[string]bool{}
	for _, name := range c.attributeNames(p) {
		names[strings.ToLower(strings.TrimLeft(name, `\`))] = true
	}
	for _, r := range required {
		if !names[strings.ToLower(r)] {
			return false
		}
	}
	return true
}

// attributeNames are the fully qualified names of AttributeHelper::getAttributes
// of a member: the attributes of its first attribute group.
func (c *clACtx) attributeNames(p int) []string {
	f := c.f
	opener := -1
	start := p - 1
	for {
		candidate := -1
		for i := start; i >= 0; i-- {
			if f.toks[i].kind == token.AttributeEnd || c.isMemberEnd(i) {
				candidate = i
				break
			}
		}
		if candidate < 0 || f.toks[candidate].kind != token.AttributeEnd || f.toks[candidate].closer < 0 {
			break
		}
		opener = f.toks[candidate].closer
		start = opener
	}
	if opener < 0 {
		return nil
	}
	if c.uses == nil {
		c.uses = f.useStatements()
	}
	var out []string
	for _, a := range f.attributes(opener) {
		out = append(out, c.uses.resolveClassName(f, a.name, opener))
	}
	return out
}

// isStaticConstructor tells a public static method returning self, static
// or the name of the class token last before it.
func (c *clACtx) isStaticConstructor(fn int) bool {
	parent := c.parentClassName(fn)
	if hint, ok := c.returnTypeHint(fn); ok {
		hint = strings.TrimPrefix(hint, "?")
		return hint == "self" || hint == parent
	}
	open := c.f.docOpenBefore(fn)
	if open < 0 {
		return false
	}
	annotations := c.docsOf().annotations(open, "@return")
	if len(annotations) == 0 {
		return false
	}
	t := thdReturnType(&annotations[0])
	if t == nil || t.kind != "IdentifierTypeNode" {
		return false
	}
	return t.name == "static" || t.name == "self" || t.name == parent
}

// returnTypeHint is FunctionHelper::findReturnTypeHint of a method.
func (c *clACtx) returnTypeHint(fn int) (string, bool) {
	f := c.f
	open := f.functionParenOpen(fn)
	if open < 0 || f.toks[open].closer < 0 {
		return "", false
	}
	colon := f.nextEffective(f.toks[open].closer+1, len(f.toks))
	if !f.is(colon, ":") {
		return "", false
	}
	start := f.nextEffective(colon+1, len(f.toks))
	if start < 0 {
		return "", false
	}
	after := c.sc.opener[fn]
	if c.scopeCloser(fn) < 0 {
		after = -1
		for i := start + 1; i < len(f.toks); i++ {
			if f.is(i, ";") {
				after = i
				break
			}
		}
	}
	if after < 0 {
		return "", false
	}
	end := f.prevEffective(after - 1)
	return thWhitespace.ReplaceAllString(f.content(start, end), ""), true
}

// parentClassName is getParentClassName: the name of the last class type
// token before p.
func (c *clACtx) parentClassName(p int) string {
	f := c.f
	if c.classes == nil {
		c.classes = []int{}
		for i, t := range f.toks {
			if t.kind == token.Keyword && clAIsClassKeyword(t.orig) {
				c.classes = append(c.classes, i)
			}
		}
	}
	classPointer := -1
	for _, cp := range c.classes {
		if cp >= p {
			break
		}
		classPointer = cp
	}
	if classPointer < 0 {
		return ""
	}
	prev := f.prevEffective(classPointer - 1)
	if f.isKeyword(prev, "readonly") {
		prev = f.prevEffective(prev - 1)
	}
	if f.isKeyword(prev, "new") {
		return "class@anonymous"
	}
	end := c.sc.opener[classPointer]
	if end < 0 {
		end = len(f.toks)
	}
	for i := classPointer + 1; i < end; i++ {
		if f.toks[i].kind == token.Ident && !strings.Contains(f.toks[i].orig, `\`) {
			return f.toks[i].content
		}
	}
	return ""
}

func (c *clACtx) fixIncorrectGroupOrder(groupFirst, groupLast, nextGroupMember int) {
	f := c.f
	previousMemberEnd := c.previousMemberEnd(groupFirst)
	groupStart := c.groupStart(groupFirst, previousMemberEnd)
	groupEnd := c.groupEnd(groupLast)
	groupContent := f.content(groupStart, groupEnd)
	nextGroupMemberStart := c.groupStart(nextGroupMember, -1)

	f.beginChangeset()
	f.removeBetweenIncluding(groupStart, groupEnd)
	lines := 0
	for i := previousMemberEnd; i < groupStart; i++ {
		if f.toks[i].kind == token.Whitespace {
			lines++
			f.replace(i, "")
		}
	}
	f.addBefore(nextGroupMemberStart, groupContent+strings.Repeat(f.eol, lines))
	f.endChangeset()
}

func (c *clACtx) previousMemberEnd(member int) int {
	for i := member - 1; i >= 0; i-- {
		if c.isMemberEnd(i) {
			return i
		}
	}
	return -1
}

// groupStart is findGroupStartPointer: the first token on the line of the
// member's doc comment, or of its first effective token.
func (c *clACtx) groupStart(member, previousMemberEnd int) int {
	f := c.f
	start := f.docOpenBefore(member - 1)
	if member-1 >= 0 && f.isDocOpen(member-1) {
		start = member - 1
	}
	if start < 0 {
		if previousMemberEnd < 0 {
			previousMemberEnd = c.previousMemberEnd(member)
		}
		start = f.nextEffective(previousMemberEnd+1, len(f.toks))
	}
	// File::findFirstOnLine([T_OPEN_CURLY_BRACKET, T_CLOSE_CURLY_BRACKET, T_SEMICOLON], $start, true)
	found := 0
	line := f.toks[start].line
	for i := start; i >= 0 && f.toks[i].line >= line; i-- {
		if !c.isMemberEnd(i) {
			found = i
		}
	}
	return found
}

// groupEnd is findGroupEndPointer.
func (c *clACtx) groupEnd(member int) int {
	f := c.f
	switch c.sc.code[member] {
	case "function", "use":
		if closer := c.scopeCloser(member); closer >= 0 {
			return closer
		}
	}
	for i := member + 1; i < len(f.toks); i++ {
		if f.is(i, ";") {
			return i
		}
		if f.is(i, "{") {
			return f.toks[i].closer
		}
	}
	return len(f.toks) - 1
}
