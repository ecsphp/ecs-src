package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Shared logic for the three "phpdoc to native type" fixers, ported from
// PHP-CS-Fixer's AbstractPhpdocToTypeDeclarationFixer. They read a `@param`,
// `@return` or `@var` type and, when it maps cleanly onto a native type hint,
// add it to the signature. They are risky/experimental and only fix cases that
// are byte-exact with PHP-CS-Fixer 3.95 under PHP 8; a genuinely ambiguous type
// is left untouched.

// phpdocTypeOptions holds the three shared options with inverted booleans so the
// zero value matches PHP-CS-Fixer's defaults (scalar_types and union_types on).
type phpdocTypeOptions struct {
	ScalarTypesOff bool
	UnionTypesOff  bool
	TypesMap       map[string]string
}

type phpdocTypeConfig struct {
	scalarTypes bool
	unionTypes  bool
	typesMap    map[string]string
}

func (o phpdocTypeOptions) config() phpdocTypeConfig {
	return phpdocTypeConfig{scalarTypes: !o.ScalarTypesOff, unionTypes: !o.UnionTypesOff, typesMap: o.TypesMap}
}

func (o phpdocTypeOptions) withConfig(config map[string]any) phpdocTypeOptions {
	if v, ok := config["scalar_types"].(bool); ok {
		o.ScalarTypesOff = !v
	}
	if v, ok := config["union_types"].(bool); ok {
		o.UnionTypesOff = !v
	}
	if v, ok := config["types_map"].(map[string]any); ok {
		m := make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				m[k] = s
			}
		}
		o.TypesMap = m
	}
	return o
}

var phpdocScalarTypes = map[string]bool{"bool": true, "float": true, "int": true, "string": true}

// phpdocVersionSpecificTypes are available on the PHP 8 the fixers target.
var phpdocVersionSpecificTypes = map[string]bool{"void": true, "iterable": true, "object": true, "mixed": true, "never": true}

var phpdocTypeAliases = map[string]string{
	"boolean": "bool", "callback": "callable", "double": "float", "false": "bool",
	"integer": "int", "list": "array", "real": "float", "true": "bool",
}

var phpdocNormalizeBuiltins = map[string]bool{
	"array": true, "bool": true, "callable": true, "false": true, "float": true,
	"int": true, "iterable": true, "mixed": true, "never": true, "null": true,
	"object": true, "resource": true, "string": true, "true": true, "void": true,
}

type phpdocTypeInfo struct {
	commonType string
	isNullable bool
}

// phpdocAnnotationInfo is one annotation of a docblock: its types content (as
// Annotation::getTypesContent) and variable name (Annotation::getVariableName).
type phpdocAnnotationInfo struct {
	typesContent string
	hasTypes     bool
	variableName string
}

// phpdocFindFunctionDocComment mirrors findFunctionDocComment: walking back over
// comments and modifiers (but not attributes) to the docblock, or -1.
func phpdocFindFunctionDocComment(s *tokens.Stream, index int) int {
	for {
		index = getPrevNonWhitespace(s, index)
		if index < 0 {
			return -1
		}
		t := s.At(index)
		if t.Kind == token.Comment {
			continue
		}
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "abstract", "final", "private", "protected", "public", "static":
				continue
			}
		}
		break
	}
	if s.At(index).Kind == token.DocComment {
		return index
	}
	return -1
}

// phpdocAnnotationsOfType mirrors getAnnotationsFromDocComment: the annotations
// of the docblock at docIndex whose tag is tag, in document order.
func phpdocAnnotationsOfType(s *tokens.Stream, docIndex int, tag string) []phpdocAnnotationInfo {
	content := s.At(docIndex).Value
	lines := splitDocLines(content)
	var out []phpdocAnnotationInfo
	for _, a := range docAnnotations(lines) {
		if docTagName(lines[a.start]) != tag {
			continue
		}
		annotation := strings.Join(lines[a.start:a.end+1], "")
		info := phpdocAnnotationInfo{}
		if start, end, ok := docTypesRange(annotation, tag); ok {
			info.typesContent = annotation[start:end]
			info.hasTypes = true
			info.variableName = phpdocVariableName(annotation, end)
		}
		out = append(out, info)
	}
	return out
}

// phpdocVariableName mirrors Annotation::getVariableName starting at the end of
// the types content: the first "$name" past an optional "&" and "...".
func phpdocVariableName(annotation string, p int) string {
	skipWS := func(i int) int {
		for i < len(annotation) && isDocS(annotation[i]) {
			i++
		}
		return i
	}
	p = skipWS(p)
	if p < len(annotation) && annotation[p] == '&' {
		p = skipWS(p + 1)
	}
	if strings.HasPrefix(annotation[p:], "...") {
		p = skipWS(p + 3)
	}
	if p >= len(annotation) || annotation[p] != '$' {
		return ""
	}
	q := p + 1
	for q < len(annotation) && isDocIdentChar(annotation[q]) {
		q++
	}
	if q == p+1 {
		return ""
	}
	return annotation[p:q]
}

// phpdocTypesToExclude mirrors getTypesToExclude: template types declared by
// phpstan/psalm type and import-type annotations, which must not be used.
func phpdocTypesToExclude(content string) []string {
	lines := splitDocLines(content)
	var out []string
	for _, a := range docAnnotations(lines) {
		name := docTagName(lines[a.start])
		annotation := strings.Join(lines[a.start:a.end+1], "")
		switch name {
		case "phpstan-type", "psalm-type":
			if start, end, ok := docTypesRange(annotation, name); ok {
				out = append(out, annotation[start:end])
			}
		case "phpstan-import-type", "psalm-import-type":
			if alias, ok := phpdocImportAsAlias(strings.TrimSpace(annotation)); ok {
				out = append(out, alias)
				continue
			}
			if start, end, ok := docTypesRange(annotation, name); ok {
				out = append(out, annotation[start:end])
			}
		}
	}
	return out
}

// phpdocImportAsAlias matches a trailing "as <identifier>" of an import-type.
func phpdocImportAsAlias(content string) (string, bool) {
	idx := strings.LastIndex(content, "as")
	for idx >= 0 {
		before := idx == 0 || isDocS(content[idx-1])
		after := idx+2 < len(content) && isDocS(content[idx+2])
		if before && after {
			q := idx + 2
			for q < len(content) && isDocS(content[q]) {
				q++
			}
			start := q
			for q < len(content) && isDocIdentChar(content[q]) {
				q++
			}
			if q > start && q == len(content) {
				return content[start:q], true
			}
		}
		idx = strings.LastIndex(content[:idx], "as")
	}
	return "", false
}

// phpdocCommonType mirrors TypeExpression::getCommonType for the union/member
// list; ok is false when PHP returns null.
func phpdocCommonType(members []string, hasNamespace bool) (string, bool) {
	var mainType string
	have := false
	for _, t := range members {
		if t == "null" {
			continue
		}
		t = strings.TrimPrefix(t, "?")
		t = phpdocReduceType(t)
		if a, ok := phpdocTypeAliases[t]; ok {
			t = a
		}
		if !have || t == mainType {
			mainType = t
			have = true
			continue
		}
		parent, ok := phpdocParentType(t, mainType, hasNamespace)
		if !ok {
			return "", false
		}
		mainType = parent
	}
	if !have {
		return "", false
	}
	return mainType, true
}

// phpdocReduceType applies getCommonType's per-type reduction: an array suffix
// becomes "array" and a generic/shape/callable head keeps its leading name.
func phpdocReduceType(t string) string {
	if phpdocHasArraySuffix(t) {
		return "array"
	}
	if head, ok := phpdocGenericHead(t); ok {
		return head
	}
	return t
}

func isHWS(c byte) bool { return c == ' ' || c == '\t' || c == 0xa0 }

// phpdocHasArraySuffix matches /\[\h*\]$/.
func phpdocHasArraySuffix(t string) bool {
	r := len(t)
	if r == 0 || t[r-1] != ']' {
		return false
	}
	r--
	for r > 0 && isHWS(t[r-1]) {
		r--
	}
	return r > 0 && t[r-1] == '['
}

// phpdocGenericHead matches /^(.+?)\h*[<{(]/ and returns group 1.
func phpdocGenericHead(t string) (string, bool) {
	i := strings.IndexAny(t, "<{(")
	if i <= 0 {
		return "", false
	}
	end := i
	for end > 0 && isHWS(t[end-1]) {
		end--
	}
	if end == 0 {
		return "", false
	}
	return t[:end], true
}

// phpdocParentType mirrors TypeExpression::getParentType for the four pairs PHP
// reduces, using a namespace-safe normalize.
func phpdocParentType(t1, t2 string, hasNamespace bool) (string, bool) {
	n1, ok1 := phpdocNormalize(t1, hasNamespace)
	n2, ok2 := phpdocNormalize(t2, hasNamespace)
	if !ok1 || !ok2 {
		return "", false
	}
	if phpdocNatCaseLess(n2, n1) {
		n1, n2 = n2, n1
	}
	parents := map[string]string{
		"array|iterable": "iterable", "array|Traversable": "iterable",
		"iterable|Traversable": "iterable", "self|static": "self",
	}
	v, ok := parents[n1+"|"+n2]
	return v, ok
}

// phpdocNormalize mirrors TypeExpression::normalize. Namespace-use resolution is
// not ported: ok is false for a bare class name in a non-global namespace,
// where PHP would qualify it, so the parent-pair reduction is skipped rather
// than risk a wrong result.
func phpdocNormalize(t string, hasNamespace bool) (string, bool) {
	if a, ok := phpdocTypeAliases[t]; ok {
		return a, true
	}
	if phpdocNormalizeBuiltins[t] {
		return t, true
	}
	if strings.HasSuffix(t, "[]") {
		return "array", true
	}
	if i := strings.IndexByte(t, '<'); i > 0 {
		return t[:i], true
	}
	if rest, ok := strings.CutPrefix(t, "\\"); ok {
		return rest, true
	}
	if !hasNamespace {
		return t, true
	}
	return "", false
}

// phpdocNatCaseLess reports whether a sorts before b case-insensitively, the
// order natcasesort gives the short type names used here.
func phpdocNatCaseLess(a, b string) bool {
	return strings.ToLower(a) < strings.ToLower(b)
}

func phpdocAllowsNull(members []string) bool {
	for _, t := range members {
		if t == "null" || t == "mixed" || strings.HasPrefix(t, "?") {
			return true
		}
	}
	return false
}

// phpdocCommonTypeInfo mirrors getCommonTypeInfo; ok is false when PHP returns
// null.
func phpdocCommonTypeInfo(members []string, isReturnType bool, isSkipped func(string) bool, cfg phpdocTypeConfig, hasNamespace bool) (phpdocTypeInfo, bool) {
	commonType, ok := phpdocCommonType(members, hasNamespace)
	if !ok {
		return phpdocTypeInfo{}, false
	}
	isNullable := phpdocAllowsNull(members)
	if isNullable && commonType == "void" {
		return phpdocTypeInfo{}, false
	}
	if commonType == "static" && !isReturnType {
		commonType = "self"
	}
	if isSkipped(commonType) {
		return phpdocTypeInfo{}, false
	}
	if mapped, ok := cfg.typesMap[commonType]; ok {
		commonType = mapped
	}
	if phpdocScalarTypes[commonType] {
		if !cfg.scalarTypes {
			return phpdocTypeInfo{}, false
		}
	} else if !phpdocMatchRegexClass(commonType) {
		return phpdocTypeInfo{}, false
	}
	return phpdocTypeInfo{commonType: commonType, isNullable: isNullable}, true
}

// phpdocUnionTypes mirrors getUnionTypes under PHP 8; "" and false when PHP
// returns null.
func phpdocUnionTypes(members []string, glue string, isReturnType bool, isSkipped func(string) bool, cfg phpdocTypeConfig, hasNamespace bool) (string, bool) {
	if glue != "|" {
		return "", false
	}
	if !cfg.unionTypes {
		return "", false
	}
	isNullable := phpdocAllowsNull(members)
	var unionTypes []string
	containsOtherThanIterable := false
	containsOtherThanEmpty := false
	for _, t := range members {
		if t == "null" {
			continue
		}
		if isSkipped(t) {
			return "", false
		}
		info, ok := phpdocCommonTypeInfo([]string{t}, isReturnType, isSkipped, cfg, hasNamespace)
		if !ok {
			return "", false
		}
		ct := info.commonType
		if !containsOtherThanIterable && ct != "array" && ct != "Traversable" && ct != "iterable" {
			containsOtherThanIterable = true
		}
		if isReturnType && !containsOtherThanEmpty && ct != "null" && ct != "void" && ct != "never" {
			containsOtherThanEmpty = true
		}
		if !isNullable && info.isNullable {
			isNullable = true
		}
		unionTypes = append(unionTypes, ct)
	}
	if !containsOtherThanIterable {
		return "", false
	}
	if isReturnType && !containsOtherThanEmpty {
		return "", false
	}
	if isNullable {
		unionTypes = append(unionTypes, "null")
	}
	seen := map[string]bool{}
	var unique []string
	for _, t := range unionTypes {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	return strings.Join(unique, glue), true
}

// phpdocMatchRegexClass mirrors AbstractPhpdocToTypeDeclarationFixer::REGEX_CLASS.
func phpdocMatchRegexClass(s string) bool {
	i := 0
	if i < len(s) && s[i] == '\\' {
		i++
	}
	seg := func() bool {
		if i >= len(s) || !isDocIdentChar(s[i]) {
			return false
		}
		if isDigitByte(s[i]) && (i == 0 || s[i-1] != '*') {
			return false
		}
		for i < len(s) && isDocIdentChar(s[i]) {
			i++
		}
		return true
	}
	if !seg() {
		return false
	}
	for i < len(s) && s[i] == '\\' {
		i++
		if !seg() {
			return false
		}
	}
	return i == len(s)
}

// phpdocTypeTokens builds the native type-hint tokens, mirroring
// createTypeDeclarationTokens (nullable mark, then the type, with a leading "\"
// dropped before an unqualified scalar or version-specific type).
func phpdocTypeTokens(typeStr string, isNullable bool) []token.Token {
	var toks []token.Token
	if isNullable && typeStr != "mixed" {
		toks = append(toks, token.Token{Kind: token.Punct, Value: "?"})
	}
	return append(toks, phpdocTokenizeType(typeStr)...)
}

func phpdocTokenizeType(typeStr string) []token.Token {
	var toks []token.Token
	for i := 0; i < len(typeStr); {
		if c := typeStr[i]; c == '\\' || c == '|' || c == '&' {
			toks = append(toks, token.Token{Kind: token.Punct, Value: string(c)})
			i++
			continue
		}
		j := i
		for j < len(typeStr) && typeStr[j] != '\\' && typeStr[j] != '|' && typeStr[j] != '&' {
			j++
		}
		toks = append(toks, token.Token{Kind: token.Ident, Value: typeStr[i:j]})
		i = j
	}
	for idx := 0; idx < len(toks); idx++ {
		t := toks[idx]
		if t.Kind == token.Ident && (phpdocScalarTypes[t.Value] || phpdocVersionSpecificTypes[t.Value]) {
			if idx > 0 && toks[idx-1].Kind == token.Punct && toks[idx-1].Value == "\\" {
				toks = slices.Delete(toks, idx-1, idx)
				idx--
			}
		}
	}
	return toks
}

// phpdocValidTypeSyntax conservatively mirrors isValidSyntax: whether the type
// is syntactically usable as a native hint. A bare reserved word, or a
// namespaced builtin, is rejected (left as a no-op) rather than risk a wrong
// result.
func phpdocValidTypeSyntax(t string) bool {
	members := strings.FieldsFunc(t, func(r rune) bool { return r == '|' || r == '&' })
	if len(members) == 0 {
		return false
	}
	for _, m := range members {
		if !phpdocValidSingleType(strings.TrimPrefix(m, "?")) {
			return false
		}
	}
	return true
}

func phpdocValidSingleType(m string) bool {
	if m == "" {
		return false
	}
	if rest, ok := strings.CutPrefix(m, "\\"); ok {
		if !strings.Contains(rest, "\\") && phpdocTypeKeyword[strings.ToLower(rest)] {
			return false
		}
		return true
	}
	if strings.Contains(m, "\\") {
		return true
	}
	low := strings.ToLower(m)
	if phpdocTypeKeyword[low] {
		return true
	}
	return !phpdocReservedInvalid[low]
}

var phpdocTypeKeyword = map[string]bool{
	"array": true, "callable": true, "bool": true, "float": true, "int": true,
	"string": true, "void": true, "iterable": true, "object": true, "mixed": true,
	"never": true, "self": true, "parent": true, "static": true, "false": true,
	"true": true, "null": true,
}

// phpdocReservedInvalid are reserved words that are a parse error as a bare type.
var phpdocReservedInvalid = map[string]bool{
	"and": true, "as": true, "break": true, "case": true, "catch": true, "class": true,
	"clone": true, "const": true, "continue": true, "declare": true, "default": true,
	"do": true, "echo": true, "else": true, "elseif": true, "empty": true,
	"enddeclare": true, "endfor": true, "endforeach": true, "endif": true,
	"endswitch": true, "endwhile": true, "enum": true, "eval": true, "exit": true,
	"extends": true, "final": true, "finally": true, "fn": true, "for": true,
	"foreach": true, "function": true, "global": true, "goto": true, "if": true,
	"implements": true, "include": true, "include_once": true, "instanceof": true,
	"insteadof": true, "interface": true, "isset": true, "list": true, "match": true,
	"namespace": true, "new": true, "or": true, "print": true, "private": true,
	"protected": true, "public": true, "readonly": true, "require": true,
	"require_once": true, "return": true, "switch": true, "throw": true, "trait": true,
	"try": true, "unset": true, "use": true, "var": true, "while": true, "xor": true,
	"yield": true,
}

// phpdocFileHasNamespace reports whether the file declares a named namespace.
func phpdocFileHasNamespace(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		if kwIs(s.At(i), "namespace") {
			if n := sigNext(s, i); n >= 0 && s.At(n).Kind == token.Ident {
				return true
			}
		}
	}
	return false
}

// phpdocApplyInserts splices the recorded token slices in, highest index first
// so earlier indices stay valid, mirroring Tokens::insertSlices.
func phpdocApplyInserts(s *tokens.Stream, inserts map[int][]token.Token) bool {
	if len(inserts) == 0 {
		return false
	}
	keys := make([]int, 0, len(inserts))
	for k := range inserts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for i := len(keys) - 1; i >= 0; i-- {
		s.InsertSliceAt(keys[i], inserts[keys[i]])
	}
	return true
}

func phpdocParamSkipped(t string) bool  { return t == "resource" || t == "static" || t == "void" }
func phpdocReturnSkipped(t string) bool { return t == "resource" || t == "null" }
