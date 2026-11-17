package rules

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/AbstractPhpdocToTypeDeclarationFixer.php
//
// phpdocToType holds the options shared by the PhpdocTo*Type fixers. Its zero
// value is PHP-CS-Fixer's default on PHP >= 8.0: scalar and union types on.
type phpdocToType struct {
	NoScalarTypes bool
	NoUnionTypes  bool
	TypesMap      map[string]string
}

func (c phpdocToType) withConfig(config map[string]any) phpdocToType {
	if v, ok := config["scalar_types"].(bool); ok {
		c.NoScalarTypes = !v
	}
	if v, ok := config["union_types"].(bool); ok {
		c.NoUnionTypes = !v
	}
	if m, ok := config["types_map"].(map[string]any); ok {
		c.TypesMap = map[string]string{}
		for k, v := range m {
			if sv, ok := v.(string); ok {
				c.TypesMap[k] = sv
			}
		}
	}
	return c
}

var ptdScalarTypes = map[string]bool{"bool": true, "float": true, "int": true, "string": true}

// ptdVersionSpecificTypes all exist on the PHP >= 8.1 that blink assumes.
var ptdVersionSpecificTypes = map[string]bool{"void": true, "iterable": true, "object": true, "mixed": true, "never": true}

var ptdTypeAliases = map[string]string{
	"boolean": "bool", "callback": "callable", "double": "float", "false": "bool",
	"integer": "int", "list": "array", "real": "float", "true": "bool",
}

// ptdTypeInfo is _CommonTypeInfo.
type ptdTypeInfo struct {
	commonType string
	isNullable bool
}

// ptdTypeExpr is a TypeExpression with the namespace context its
// getCommonType resolves class names against.
type ptdTypeExpr struct {
	value     string
	namespace string
	uses      []ptdUse
}

type ptdUse struct{ short, full string }

func (e ptdTypeExpr) types() ([]string, string) { return docTypeMembers(e.value) }

func (e ptdTypeExpr) isUnionType() bool {
	members, glue := e.types()
	return len(members) > 1 && glue == "|"
}

func (e ptdTypeExpr) allowsNull() bool {
	members, _ := e.types()
	for _, t := range members {
		if t == "null" || t == "mixed" || strings.HasPrefix(t, "?") {
			return true
		}
	}
	return false
}

// commonType ports TypeExpression::getCommonType.
func (e ptdTypeExpr) commonType() (string, bool) {
	members, _ := e.types()
	mainType, hasMain := "", false
	for _, t := range members {
		if t == "null" {
			continue
		}
		t = strings.TrimPrefix(t, "?")
		if ptdEndsWithEmptyBrackets(t) {
			t = "array"
		} else if prefix, ok := ptdGenericPrefix(t); ok {
			t = prefix
		}
		if alias, ok := ptdTypeAliases[t]; ok {
			t = alias
		}
		if !hasMain || t == mainType {
			mainType, hasMain = t, true
			continue
		}
		parent, ok := e.parentType(t, mainType)
		if !ok {
			return "", false
		}
		mainType = parent
	}
	return mainType, hasMain
}

// ptdEndsWithEmptyBrackets is /\[\h*\]$/.
func ptdEndsWithEmptyBrackets(t string) bool {
	t = strings.TrimSuffix(t, "\n")
	if !strings.HasSuffix(t, "]") {
		return false
	}
	rest := t[:len(t)-1]
	for rest != "" {
		r, size := utf8.DecodeLastRuneInString(rest)
		if r == '[' {
			return true
		}
		if !pcreIsHSpace(r) {
			return false
		}
		rest = rest[:len(rest)-size]
	}
	return false
}

// ptdGenericPrefix is /^(.+?)\h*[<{(]/: the shortest non-empty prefix
// followed by optional horizontal space and an opening bracket.
func ptdGenericPrefix(t string) (string, bool) {
	if t == "" {
		return "", false
	}
	_, size := utf8.DecodeRuneInString(t)
	for k := size; k < len(t); {
		p := skipPHPHSpace(t, k)
		if p < len(t) && strings.IndexByte("<{(", t[p]) >= 0 {
			return t[:k], true
		}
		_, size = utf8.DecodeRuneInString(t[k:])
		k += size
	}
	return "", false
}

var ptdParentTypes = map[string]string{
	"array|Traversable":    "iterable",
	"array|iterable":       "iterable",
	"iterable|Traversable": "iterable",
	"self|static":          "self",
}

// parentType ports TypeExpression::getParentType.
func (e ptdTypeExpr) parentType(type1, type2 string) (string, bool) {
	types := []string{e.normalize(type1), e.normalize(type2)}
	if strings.ToLower(types[1]) < strings.ToLower(types[0]) {
		types[0], types[1] = types[1], types[0]
	}
	parent, ok := ptdParentTypes[types[0]+"|"+types[1]]
	return parent, ok
}

var ptdBuiltinTypes = map[string]bool{
	"array": true, "bool": true, "callable": true, "false": true, "float": true,
	"int": true, "iterable": true, "mixed": true, "never": true, "null": true,
	"object": true, "resource": true, "string": true, "true": true, "void": true,
}

// normalize ports TypeExpression::normalize.
func (e ptdTypeExpr) normalize(t string) string {
	if alias, ok := ptdTypeAliases[t]; ok {
		return alias
	}
	if ptdBuiltinTypes[t] {
		return t
	}
	if strings.HasSuffix(t, "[]") {
		return "array"
	}
	if i := strings.IndexByte(t, '<'); i > 0 {
		return t[:i]
	}
	if rest, ok := strings.CutPrefix(t, `\`); ok {
		return rest
	}
	for _, u := range e.uses {
		if u.short == t {
			return u.full
		}
	}
	if e.namespace == "" {
		return t
	}
	return e.namespace + `\` + t
}

// commonTypeInfo ports getCommonTypeInfo.
func (c phpdocToType) commonTypeInfo(e ptdTypeExpr, isReturnType bool, isSkipped func(string) bool) (ptdTypeInfo, bool) {
	commonType, ok := e.commonType()
	isNullable := e.allowsNull()
	if !ok {
		return ptdTypeInfo{}, false
	}
	if isNullable && commonType == "void" {
		return ptdTypeInfo{}, false
	}
	if commonType == "static" && !isReturnType {
		commonType = "self"
	}
	if isSkipped(commonType) {
		return ptdTypeInfo{}, false
	}
	if mapped, ok := c.TypesMap[commonType]; ok {
		commonType = mapped
	}
	if ptdScalarTypes[commonType] {
		if c.NoScalarTypes {
			return ptdTypeInfo{}, false
		}
	} else if !ptdIsClassName(commonType) {
		return ptdTypeInfo{}, false
	}
	return ptdTypeInfo{commonType: commonType, isNullable: isNullable}, true
}

// ptdIsClassName is /^(?:\\?+IDENT(\\IDENT)*+)$/.
func ptdIsClassName(t string) bool {
	i := 0
	if strings.HasPrefix(t, `\`) {
		i = 1
	}
	for {
		end, ok := parsePHPTypeIdentifier(t, i)
		if !ok {
			return false
		}
		if end == len(t) || (end == len(t)-1 && t[end] == '\n') {
			return true
		}
		if t[end] != '\\' {
			return false
		}
		i = end + 1
	}
}

// unionTypes ports getUnionTypes.
func (c phpdocToType) unionTypes(e ptdTypeExpr, isReturnType bool, isSkipped func(string) bool) (string, bool) {
	if !e.isUnionType() || c.NoUnionTypes {
		return "", false
	}
	members, glue := e.types()
	isNullable := e.allowsNull()
	var unionTypes []string
	containsOtherThanIterableType := false
	containsOtherThanEmptyType := false
	for _, t := range members {
		if t == "null" {
			continue
		}
		if isSkipped(t) {
			return "", false
		}
		info, ok := c.commonTypeInfo(ptdTypeExpr{value: t}, isReturnType, isSkipped)
		if !ok {
			return "", false
		}
		if !slices.Contains([]string{"array", "Traversable", "iterable"}, info.commonType) {
			containsOtherThanIterableType = true
		}
		if isReturnType && !slices.Contains([]string{"null", "void", "never"}, info.commonType) {
			containsOtherThanEmptyType = true
		}
		if info.isNullable {
			isNullable = true
		}
		unionTypes = append(unionTypes, info.commonType)
	}
	if !containsOtherThanIterableType || (isReturnType && !containsOtherThanEmptyType) {
		return "", false
	}
	if isNullable {
		unionTypes = append(unionTypes, "null")
	}
	var unique []string
	for _, t := range unionTypes {
		if !slices.Contains(unique, t) {
			unique = append(unique, t)
		}
	}
	return strings.Join(unique, glue), true
}

// resolveType runs getCommonTypeInfo, falling back to getUnionTypes, as each
// fixer's applyFix does.
func (c phpdocToType) resolveType(e ptdTypeExpr, isReturnType bool, isSkipped func(string) bool) (ptdTypeInfo, bool) {
	if info, ok := c.commonTypeInfo(e, isReturnType, isSkipped); ok {
		return info, true
	}
	if union, ok := c.unionTypes(e, isReturnType, isSkipped); ok {
		return ptdTypeInfo{commonType: union}, true
	}
	return ptdTypeInfo{}, false
}

// ptdFindFunctionDocComment ports findFunctionDocComment.
func ptdFindFunctionDocComment(s *tokens.Stream, index int) int {
	for {
		index = prevSignificantIndex(s, index)
		if index < 0 {
			return -1
		}
		t := s.At(index)
		if t.Kind == token.Comment {
			continue
		}
		if t.Kind == token.Keyword && slices.Contains([]string{"abstract", "final", "private", "protected", "public", "static"}, strings.ToLower(t.Value)) {
			continue
		}
		if t.Kind == token.DocComment {
			return index
		}
		return -1
	}
}

// ptdAnnotation is a DocBlock Annotation with the namespace context its type
// expression resolves against.
type ptdAnnotation struct {
	a         *phpDocAnnotation
	namespace string
	uses      []ptdUse
}

func (p ptdAnnotation) typeExpression() (ptdTypeExpr, bool) {
	value, ok := phpdocAnnotationTypesContent(p.a)
	if !ok {
		return ptdTypeExpr{}, false
	}
	return ptdTypeExpr{value: value, namespace: p.namespace, uses: p.uses}, true
}

// variableName ports Annotation::getVariableName:
// /@TAG\s+(TYPES\s*)?(&\s*)?(\.{3}\s*)?(?<variable>\$IDENT)(?:.*|$)/.
func (p ptdAnnotation) variableName() (string, bool) {
	content := p.a.content()
	types, _ := phpdocAnnotationTypesContent(p.a)
	at := "@" + p.a.tag
	afterTypes := func(i int) (string, bool) {
		if strings.HasPrefix(content[i:], "&") {
			i = pcreSkip(content, i+1, pcreIsSpace)
		}
		if strings.HasPrefix(content[i:], "...") {
			i = pcreSkip(content, i+3, pcreIsSpace)
		}
		if !strings.HasPrefix(content[i:], "$") {
			return "", false
		}
		end, ok := parsePHPTypeIdentifier(content, i+1)
		if !ok {
			return "", false
		}
		return content[i:end], true
	}
	for from := 0; ; {
		pos := strings.Index(content[from:], at)
		if pos < 0 {
			return "", false
		}
		pos += from
		from = pos + 1
		start := pos + len(at)
		i := pcreSkip(content, start, pcreIsSpace)
		if i == start {
			continue
		}
		if types != "" && strings.HasPrefix(content[i:], types) {
			if name, ok := afterTypes(pcreSkip(content, i+len(types), pcreIsSpace)); ok {
				return name, true
			}
		}
		if name, ok := afterTypes(i); ok {
			return name, true
		}
	}
}

// ptdAnnotationsOfType ports getAnnotationsFromDocComment.
func ptdAnnotationsOfType(s *tokens.Stream, tag string, docIndex int) []ptdAnnotation {
	namespace, uses := ptdNamespaceAt(s, docIndex)
	var out []ptdAnnotation
	for _, a := range newPHPDocBlock(s.At(docIndex).Value).annotationsOfType([]string{tag}) {
		out = append(out, ptdAnnotation{a: a, namespace: namespace, uses: uses})
	}
	return out
}

// ptdNamespaceAt returns the name of the namespace at index and its imports,
// as NamespacesAnalyzer::getNamespaceAt and getDeclarationsInNamespace.
func ptdNamespaceAt(s *tokens.Stream, index int) (string, []ptdUse) {
	for _, ns := range aliasNamespaceDeclarations(s) {
		if index < ns.scopeStart || index > ns.scopeEnd {
			continue
		}
		name := ""
		if !ns.global && aliasIsNamespaceDeclaration(s, ns.scopeStart) {
			var b strings.Builder
			for j := ns.scopeStart + 1; j < s.Len() && !isPunctVal(s, j, ";") && !isPunctVal(s, j, "{"); j++ {
				if k := s.At(j).Kind; k != token.Whitespace && k != token.Comment && k != token.DocComment {
					b.WriteString(s.At(j).Value)
				}
			}
			name = b.String()
		}
		var uses []ptdUse
		for _, use := range aliasImportUseIndexes(s) {
			if use >= ns.scopeStart && use <= ns.scopeEnd {
				uses = append(uses, ptdParseUse(s, use)...)
			}
		}
		return name, uses
	}
	return "", nil
}

// ptdParseUse reads the imports of the "use" statement at kw, group imports
// included.
func ptdParseUse(s *tokens.Stream, kw int) []ptdUse {
	var out []ptdUse
	var prefix, name strings.Builder
	alias := ""
	inAlias, inGroup := false, false
	flush := func() {
		full := strings.TrimPrefix(prefix.String()+name.String(), `\`)
		if name.Len() > 0 {
			short := alias
			if short == "" {
				short = full[strings.LastIndexByte(full, '\\')+1:]
			}
			out = append(out, ptdUse{short: short, full: full})
		}
		name.Reset()
		alias, inAlias = "", false
	}
	for j := kw + 1; j < s.Len(); j++ {
		t := s.At(j)
		switch {
		case t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment:
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "as"):
			inAlias = true
		case t.Kind == token.Keyword && (strings.EqualFold(t.Value, "function") || strings.EqualFold(t.Value, "const")):
		case t.Kind == token.Punct && t.Value == "{":
			prefix.WriteString(name.String())
			name.Reset()
			inGroup = true
		case t.Kind == token.Punct && t.Value == ",":
			flush()
		case t.Kind == token.Punct && t.Value == "}":
			flush()
			inGroup = false
			prefix.Reset()
		case t.Kind == token.Punct && t.Value == ";", t.Kind == token.CloseTag:
			flush()
			return out
		case inAlias:
			alias = t.Value
		default:
			name.WriteString(t.Value)
		}
	}
	if !inGroup {
		flush()
	}
	return out
}

// ptdTypesToExclude ports getTypesToExclude: the phpstan/psalm type aliases a
// docblock defines or imports.
func ptdTypesToExclude(content string) []string {
	var out []string
	d := newPHPDocBlock(content)
	for _, a := range d.annotationsOfType([]string{"phpstan-type", "psalm-type"}) {
		if v, ok := phpdocAnnotationTypesContent(a); ok {
			out = append(out, v)
		}
	}
	for _, a := range d.annotationsOfType([]string{"phpstan-import-type", "psalm-import-type"}) {
		if alias, ok := ptdImportAlias(strings.TrimSpace(a.content())); ok {
			out = append(out, alias)
			continue
		}
		if v, ok := phpdocAnnotationTypesContent(a); ok {
			out = append(out, v)
		}
	}
	return out
}

// ptdImportAlias is /\bas\s+(IDENT)$/.
func ptdImportAlias(content string) (string, bool) {
	for from := 0; ; {
		pos := strings.Index(content[from:], "as")
		if pos < 0 {
			return "", false
		}
		pos += from
		from = pos + 1
		if pos > 0 && isDocIdentChar(content[pos-1]) {
			continue
		}
		start := pcreSkip(content, pos+2, pcreIsSpace)
		if start == pos+2 {
			continue
		}
		end, ok := parsePHPTypeIdentifier(content, start)
		if ok && (end == len(content) || (end == len(content)-1 && content[end] == '\n')) {
			return content[start:end], true
		}
	}
}

// ptdTypeTokens lexes typ out of template (one %s) as Tokens::fromCode does
// for createTokensFromRawType, and reports whether it is valid syntax there,
// as isValidSyntax: a reserved word is no type name unless it is part of a
// namespaced name.
func ptdTypeTokens(template, typ string) ([]token.Token, bool) {
	prefix, _, _ := strings.Cut(template, "%s")
	start, end := len(prefix), len(prefix)+len(typ)
	var out []token.Token
	for _, t := range lexer.Lex(strings.Replace(template, "%s", typ, 1)) {
		if t.Pos >= start && t.Pos < end {
			t.Pos = 0
			out = append(out, t)
		}
	}
	for i, t := range out {
		lv := strings.ToLower(t.Value)
		if t.Kind != token.Keyword && !ptdMagicConstants[lv] {
			continue
		}
		namespaced := (i > 0 && out[i-1].Value == `\`) || (i+1 < len(out) && out[i+1].Value == `\`)
		if namespaced {
			out[i].Kind = token.Ident
			continue
		}
		if lv != "array" && lv != "callable" && lv != "static" {
			return nil, false
		}
	}
	return out, true
}

// ptdMagicConstants lex as names in blink, but PHP gives each its own token.
var ptdMagicConstants = map[string]bool{
	"__class__": true, "__dir__": true, "__file__": true, "__function__": true, "__line__": true,
	"__method__": true, "__namespace__": true, "__property__": true, "__trait__": true,
}

// ptdTypeDeclarationTokens ports createTypeDeclarationTokens.
func ptdTypeDeclarationTokens(typeTokens []token.Token, typ string, isNullable bool) []token.Token {
	var out []token.Token
	if isNullable && typ != "mixed" {
		out = append(out, token.Token{Kind: token.Punct, Value: "?"})
	}
	for i, t := range typeTokens {
		// scalars, void, iterable and object must be unqualified
		if t.Value == `\` && i+1 < len(typeTokens) {
			next := typeTokens[i+1]
			if next.Kind == token.Ident && (ptdScalarTypes[next.Value] || ptdVersionSpecificTypes[next.Value]) {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

// ptdInsertSlices applies insertions keyed by original index, as insertSlices.
func ptdInsertSlices(s *tokens.Stream, inserts map[int][]token.Token) bool {
	indexes := make([]int, 0, len(inserts))
	for i := range inserts {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	for _, i := range slices.Backward(indexes) {
		s.InsertSliceAt(i, inserts[i])
	}
	return len(indexes) > 0
}
