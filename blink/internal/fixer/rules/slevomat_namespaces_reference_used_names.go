package rules

import (
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/ReferenceUsedNamesOnlySniff.php
//
// SlevomatReferenceUsedNamesOnly replaces fully qualified names with use
// statements.
type SlevomatReferenceUsedNamesOnly struct {
	searchAnnotations                          bool
	allowFullyQualifiedExceptions              bool
	allowFullyQualifiedGlobalClasses           bool
	allowFullyQualifiedGlobalFunctions         bool
	allowFallbackGlobalFunctions               bool
	allowFullyQualifiedGlobalConstants         bool
	allowFallbackGlobalConstants               bool
	allowWhenNoNamespace                       bool
	allowPartialUses                           bool
	allowFullyQualifiedNameForCollidingClasses bool
	allowFullyQualifiedNameForCollidingFuncs   bool
	allowFullyQualifiedNameForCollidingConsts  bool
	specialExceptionNames                      []string
	ignoredNames                               []string
	namespacesRequiredToUse                    []string
	anyNamespaceRequiredToUse                  bool
	namespacesAllowedToUsePartially            []rnPartial
	namespacesRequiredToUsePartially           []rnPartial
	configured                                 bool
}

type rnPartial struct{ namespace, alias string }

func (SlevomatReferenceUsedNamesOnly) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\ReferenceUsedNamesOnlySniff`
}

func (SlevomatReferenceUsedNamesOnly) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReferenceUsedNamesOnly{}.Name())
}

func (r SlevomatReferenceUsedNamesOnly) withDefaults() SlevomatReferenceUsedNamesOnly {
	if !r.configured {
		r.allowFallbackGlobalFunctions = true
		r.allowFallbackGlobalConstants = true
		r.allowWhenNoNamespace = true
		r.allowPartialUses = true
		r.configured = true
	}
	return r
}

func (r SlevomatReferenceUsedNamesOnly) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	for key, field := range map[string]*bool{
		"searchAnnotations":                            &r.searchAnnotations,
		"allowFullyQualifiedExceptions":                &r.allowFullyQualifiedExceptions,
		"allowFullyQualifiedGlobalClasses":             &r.allowFullyQualifiedGlobalClasses,
		"allowFullyQualifiedGlobalFunctions":           &r.allowFullyQualifiedGlobalFunctions,
		"allowFallbackGlobalFunctions":                 &r.allowFallbackGlobalFunctions,
		"allowFullyQualifiedGlobalConstants":           &r.allowFullyQualifiedGlobalConstants,
		"allowFallbackGlobalConstants":                 &r.allowFallbackGlobalConstants,
		"allowWhenNoNamespace":                         &r.allowWhenNoNamespace,
		"allowPartialUses":                             &r.allowPartialUses,
		"allowFullyQualifiedNameForCollidingClasses":   &r.allowFullyQualifiedNameForCollidingClasses,
		"allowFullyQualifiedNameForCollidingFunctions": &r.allowFullyQualifiedNameForCollidingFuncs,
		"allowFullyQualifiedNameForCollidingConstants": &r.allowFullyQualifiedNameForCollidingConsts,
	} {
		if v, ok := config[key].(bool); ok {
			*field = v
		}
	}
	if v, ok := slvStringList(config["specialExceptionNames"]); ok {
		r.specialExceptionNames = v
	}
	if v, ok := slvStringList(config["ignoredNames"]); ok {
		r.ignoredNames = v
	}
	if list, ok := config["namespacesRequiredToUse"].([]any); ok {
		r.namespacesRequiredToUse, _ = slvStringList(list)
		r.anyNamespaceRequiredToUse = len(list) > 0
	}
	if v, ok := slvStringList(config["namespacesAllowedToUsePartially"]); ok {
		r.namespacesAllowedToUsePartially = rnPartials(v)
	}
	if v, ok := slvStringList(config["namespacesRequiredToUsePartially"]); ok {
		r.namespacesRequiredToUsePartially = rnPartials(v)
	}
	return r
}

var rnNamespaceWithAlias = regexp.MustCompile(`(?is)^(.+?)\s+as\s+(.+)$`)

// rnPartials is normalizePartialUseNamespaces: an ordered map of canonical
// namespaces to aliases, a repeated namespace keeping its first place.
func rnPartials(settings []string) []rnPartial {
	var out []rnPartial
	for _, s := range settings {
		ns, alias := strings.TrimSpace(s), ""
		if m := rnNamespaceWithAlias.FindStringSubmatch(ns); m != nil {
			ns, alias = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		}
		ns = strings.TrimLeft(ns, `\`)
		if i := slices.IndexFunc(out, func(p rnPartial) bool { return p.namespace == ns }); i >= 0 {
			out[i].alias = alias
			continue
		}
		out = append(out, rnPartial{ns, alias})
	}
	return out
}

const (
	rnSourceCode = iota
	rnSourceAnnotation
	rnSourceAnnotationConstantFetch
	rnSourceAttribute
)

type rnReference struct {
	ref    slvReferencedName
	source int
	doc    int     // the doc comment opener of an annotation
	node   *pdNode // IdentifierTypeNode or ConstFetchNode
}

type rnError struct {
	rnReference
	canonical string
	fallback  bool
}

// rnUses are UseStatementHelper's file use statements of one pass.
type rnUses struct {
	groups []nsuUseGroup
}

func (u rnUses) forPointer(p int) []*nsuUse {
	if len(u.groups) == 1 {
		return u.groups[0].uses
	}
	for _, g := range slices.Backward(u.groups) {
		if g.anchor < p {
			return g.uses
		}
	}
	return nil
}

func rnUseKind(t slvRefType) string {
	switch t {
	case slvRefFunction:
		return "function"
	case slvRefConstant:
		return "const"
	}
	return ""
}

// rnUseByID looks a use statement up by UseStatement::getUniqueId.
func rnUseByID(uses []*nsuUse, t slvRefType, name string) *nsuUse {
	kind := rnUseKind(t)
	for _, u := range uses {
		if u.kind == kind && rnNormalized(t, u.name) == rnNormalized(t, name) {
			return u
		}
	}
	return nil
}

// rnNormalized is UseStatement::normalizedNameAsReferencedInFile.
func rnNormalized(t slvRefType, name string) string {
	if t == slvRefConstant {
		return name
	}
	return strings.ToLower(name)
}

func rnUnqualified(name string) string {
	name = strings.TrimLeft(name, `\`)
	return name[strings.LastIndex(name, `\`)+1:]
}

// rnInNamespace is NamespaceHelper::isTypeInNamespace.
func rnInNamespace(name, ns string) bool {
	return strings.HasPrefix(strings.TrimLeft(name, `\`)+`\`, ns+`\`)
}

// rnResolve is NamespaceHelper::resolveName of a class or function.
func rnResolve(f *slvFile, uses rnUses, name string, t slvRefType, pointer int) string {
	if strings.HasPrefix(name, `\`) {
		return name
	}
	forPointer := uses.forPointer(pointer)
	if u := rnUseByID(forPointer, t, strings.TrimLeft(name, `\`)); u != nil {
		return `\` + u.fqn
	}
	parts := strings.Split(strings.TrimLeft(name, `\`), `\`)
	if u := rnUseByID(forPointer, t, parts[0]); u != nil && len(parts) > 1 {
		return `\` + u.fqn + `\` + strings.Join(parts[1:], `\`)
	}
	resolved := `\` + name
	if t == slvRefConstant && slvIsDefinedConstant(name) {
		return resolved
	}
	if ns, ok := f.refCurrentNamespace(pointer); ok {
		resolved = `\` + ns + resolved
	}
	return resolved
}

func (r SlevomatReferenceUsedNamesOnly) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	aborted := false
	src := s.Render()
	changed := slvRun(s, func(f *slvFile) {
		if !aborted && !r.process(f) {
			aborted = true
		}
	})
	if aborted && changed {
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(src))
		return false
	}
	return changed
}

// process runs the sniff on the first open tag; it is false where upstream throws.
func (r SlevomatReferenceUsedNamesOnly) process(f *slvFile) bool {
	openTag := f.slvFirstOpenTag()
	if openTag < 0 {
		return true
	}
	namespaces := f.refNamespacePointers()
	if len(namespaces) == 0 && !r.allowWhenNoNamespace {
		return true
	}
	names := newSlvNames(f)
	docs := newSlvDocs(f)
	uses := rnUses{groups: f.nsuFileUses()}
	references := r.references(f, names, docs, openTag)

	definedClasses := map[string]string{}
	for i := range f.toks {
		if !f.isClassTypeKeyword(i) {
			continue
		}
		if name := rnDeclaredName(f, i); name != "" {
			definedClasses[strings.ToLower(name)] = rnResolve(f, uses, name, slvRefClass, i)
		}
	}
	definedFunctions := map[string]bool{}
	for i := range f.toks {
		if !names.isFunctionKeyword(i) {
			continue
		}
		if o := f.enclosingScope(i); o >= 0 && f.isOOScopeOwner(f.braceOwner(o)) {
			continue
		}
		definedFunctions[strings.ToLower(rnDeclaredName(f, i))] = true
	}
	definedConstants := map[string]bool{}
	for i := range f.toks {
		if !f.isKeyword(i, "const") || !names.isConstKeyword(i) {
			continue
		}
		if o := f.enclosingScope(i); o >= 0 && !f.isKeyword(f.braceOwner(o), "namespace") {
			continue
		}
		for j := i + 1; j < len(f.toks); j++ {
			if f.toks[j].kind == token.Ident {
				definedConstants[f.toks[j].content] = true
				break
			}
		}
	}
	classReferences := map[string]string{}
	for _, ref := range references {
		if ref.source != rnSourceCode || ref.ref.typ != slvRefClass {
			continue
		}
		classReferences[strings.ToLower(ref.ref.name)] = rnResolve(f, uses, ref.ref.name, ref.ref.typ, ref.ref.start)
	}

	var errors []rnError
	for _, ref := range references {
		forPointer := uses.forPointer(ref.ref.start)
		name := ref.ref.name
		typ := ref.ref.typ
		canonical := strings.TrimLeft(name, `\`)
		unqualified := rnUnqualified(name)
		switch strings.ToLower(unqualified) {
		case "true", "false", "null":
			continue
		}
		colliding := rnUseByID(forPointer, typ, unqualified)
		partial := rnPartialUse(name, forPointer)
		isFullyQualified := strings.HasPrefix(name, `\`) || len(namespaces) == 0 && slvHasNamespace(name) && partial == nil
		isGlobalFallback := !isFullyQualified && !slvHasNamespace(name) && len(namespaces) > 0 && rnUseByID(forPointer, typ, name) == nil
		isFunctionFallback := typ == slvRefFunction && isGlobalFallback &&
			!definedFunctions[strings.ToLower(name)] && slvFunctionExists(name)
		isConstantFallback := typ == slvRefConstant && isGlobalFallback &&
			!definedConstants[name] && slvIsDefinedConstant(name)

		if isFullyQualified && !slices.ContainsFunc(forPointer, func(u *nsuUse) bool {
			return u.kind == rnUseKind(typ) && u.fqn == canonical
		}) {
			collidingOther := colliding != nil && canonical != strings.TrimLeft(colliding.fqn, `\`)
			switch {
			case typ == slvRefClass && r.allowFullyQualifiedNameForCollidingClasses:
				lower := strings.ToLower(unqualified)
				if d, ok := definedClasses[lower]; ok && canonical != strings.TrimLeft(d, `\`) {
					continue
				}
				if c, ok := classReferences[lower]; ok && name != c {
					continue
				}
				if collidingOther {
					continue
				}
			case typ == slvRefFunction && r.allowFullyQualifiedNameForCollidingFuncs:
				if definedFunctions[strings.ToLower(unqualified)] || collidingOther {
					continue
				}
			case typ == slvRefConstant && r.allowFullyQualifiedNameForCollidingConsts:
				if definedConstants[unqualified] || collidingOther {
					continue
				}
			}
		}

		if !isFullyQualified && !isFunctionFallback && !isConstantFallback {
			continue
		}
		if isFullyQualified && !r.isRequiredToBeUsed(name) {
			continue
		}
		isException := strings.HasSuffix(name, "Exception") || name == `\Throwable` ||
			strings.HasSuffix(name, "Error") && !slvHasNamespace(name) || slices.Contains(r.specialExceptionNames, canonical)
		if isException && !slices.Contains(r.ignoredNames, canonical) && r.allowFullyQualifiedExceptions {
			continue
		}
		if isFullyQualified && !slvHasNamespace(name) && len(namespaces) == 0 {
			if !r.fixWithoutNamespace(f, docs, ref) {
				return false
			}
			continue
		}
		shouldBeUsed := slvHasNamespace(name)
		if !shouldBeUsed {
			switch typ {
			case slvRefFunction:
				shouldBeUsed = !r.allowFallbackGlobalFunctions
				if isFullyQualified {
					shouldBeUsed = !r.allowFullyQualifiedGlobalFunctions
				}
			case slvRefConstant:
				shouldBeUsed = !r.allowFallbackGlobalConstants
				if isFullyQualified {
					shouldBeUsed = !r.allowFullyQualifiedGlobalConstants
				}
			default:
				shouldBeUsed = !r.allowFullyQualifiedGlobalClasses
			}
		}
		if shouldBeUsed {
			errors = append(errors, rnError{ref, canonical, isFunctionFallback || isConstantFallback})
		}
	}
	if len(errors) == 0 {
		return true
	}
	return r.fixErrors(f, docs, uses, openTag, errors, definedClasses, definedFunctions, definedConstants, classReferences)
}

// rnDeclaredName is the first T_STRING after a class or function keyword.
func rnDeclaredName(f *slvFile, i int) string {
	for j := i + 1; j < len(f.toks); j++ {
		if f.is(j, "(") || f.is(j, "{") {
			return ""
		}
		if f.toks[j].kind == token.Ident {
			return f.toks[j].content
		}
	}
	return ""
}

func (r SlevomatReferenceUsedNamesOnly) fixWithoutNamespace(f *slvFile, docs *slvDocs, ref rnReference) bool {
	f.beginChangeset()
	switch ref.source {
	case rnSourceAnnotation:
		if !rnFixAnnotation(f, docs, ref.doc, ref.node, ref.ref.name[1:]) {
			return false
		}
	case rnSourceAnnotationConstantFetch:
		if !rnFixAnnotation(f, docs, ref.doc, ref.node, ref.ref.name[1:]+"::"+ref.node.name) {
			return false
		}
	default:
		f.replace(ref.ref.start, f.toks[ref.ref.start].content[1:])
	}
	f.endChangeset()
	return true
}

// rnFixAnnotation is AnnotationHelper::fixAnnotation plus FixerHelper::change,
// on the doc comment as parsed at the start of the pass.
func rnFixAnnotation(f *slvFile, docs *slvDocs, open int, node *pdNode, replacement string) bool {
	p := docs.parse(open)
	out, ok := pdFix(p.root, p.toks, node, replacement)
	if !ok {
		return false
	}
	f.change(p.open, p.close, out)
	return true
}

func (r SlevomatReferenceUsedNamesOnly) fixErrors(f *slvFile, docs *slvDocs, uses rnUses, openTag int, errors []rnError,
	definedClasses map[string]string, definedFunctions, definedConstants map[string]bool, classReferences map[string]string,
) bool {
	added := map[slvRefType][]string{}
	f.beginChangeset()
	for _, e := range errors {
		typ := e.ref.typ
		required, hasRequired := r.requiredPartialUse(e.canonical)
		canonicalToUse := e.canonical
		nameToReference := rnUnqualified(canonicalToUse)
		suffix := ""
		if hasRequired {
			canonicalToUse = required.namespace
			nameToReference = required.alias
			if nameToReference == "" {
				nameToReference = rnUnqualified(required.namespace)
			}
			if e.canonical != required.namespace {
				suffix = `\` + e.canonical[len(required.namespace)+1:]
			}
		}
		nameToReplace := nameToReference + suffix
		canonicalToReference := rnNormalized(typ, nameToReplace)
		forPointer := uses.forPointer(e.ref.start)

		canBeFixed := !slices.ContainsFunc(added[typ], func(use string) bool {
			return strings.EqualFold(rnUnqualified(use), rnUnqualified(canonicalToUse))
		})
		hasExisting, hasCollision := false, false
		for _, u := range forPointer {
			if u.kind != rnUseKind(typ) {
				continue
			}
			if u.fqn == canonicalToUse {
				hasExisting = true
				break
			}
			if rnNormalized(typ, u.name) == canonicalToReference {
				hasCollision = true
			}
		}
		if !hasExisting {
			d, definedClass := definedClasses[canonicalToReference]
			c, referencedClass := classReferences[canonicalToReference]
			if typ == slvRefClass && definedClass && canonicalToUse != strings.TrimLeft(d, `\`) ||
				typ == slvRefClass && referencedClass && canonicalToUse != strings.TrimLeft(c, `\`) ||
				typ == slvRefFunction && definedFunctions[canonicalToReference] ||
				typ == slvRefConstant && definedConstants[canonicalToReference] ||
				hasCollision {
				canBeFixed = false
			}
		}
		if !canBeFixed {
			continue
		}

		addUse := !slices.Contains(added[typ], canonicalToUse)
		if _, ok := definedClasses[canonicalToReference]; typ == slvRefClass && ok {
			addUse = false
		}
		for _, u := range forPointer {
			if u.kind != rnUseKind(typ) || u.fqn != canonicalToUse {
				continue
			}
			nameToReplace = u.name + suffix
			addUse = false
			f.replace(u.pointer, f.tokenContent(u.pointer))
			break
		}
		if addUse {
			place := rnUsePlace(f, openTag, forPointer)
			kind := rnUseKind(typ)
			if kind != "" {
				kind += " "
			}
			alias := ""
			if hasRequired && required.alias != "" {
				alias = " as " + required.alias
			}
			f.replaceToken(place, f.tokenContent(place)+f.eol)
			f.add(place, "use "+kind+canonicalToUse+alias+";")
			added[typ] = append(added[typ], canonicalToUse)
		}

		switch e.source {
		case rnSourceAnnotation:
			if !rnFixAnnotation(f, docs, e.doc, e.node, nameToReplace) {
				return false
			}
		case rnSourceAnnotationConstantFetch:
			if !rnFixAnnotation(f, docs, e.doc, e.node, nameToReplace+"::"+e.node.name) {
				return false
			}
		case rnSourceAttribute:
			content := rnReplaceWord(f.content(e.ref.start, e.ref.end), e.ref.name, nameToReplace)
			f.change(e.ref.start, e.ref.end, content)
		default:
			f.change(e.ref.start, e.ref.end, nameToReplace)
		}
	}
	f.endChangeset()
	return true
}

// rnReplaceWord is preg_replace('~(?<=\W)name(?=\W)~', replacement, s).
func rnReplaceWord(s, name, replacement string) string {
	isWord := func(c byte) bool { return c == '_' || pdIsLetter(c) || pdIsDigit(c) }
	var b strings.Builder
	last := 0
	for i := 1; i+len(name) < len(s); {
		if s[i:i+len(name)] == name && !isWord(s[i-1]) && !isWord(s[i+len(name)]) {
			b.WriteString(s[last:i])
			b.WriteString(replacement)
			i += len(name)
			last = i
			continue
		}
		i++
	}
	b.WriteString(s[last:])
	return b.String()
}

// rnUsePlace is getUseStatementPlacePointer.
func rnUsePlace(f *slvFile, openTag int, uses []*nsuUse) int {
	if len(uses) > 0 {
		return f.nsuNextOf(uses[len(uses)-1].pointer+1, ";")
	}
	for i := openTag + 1; i < len(f.toks); i++ {
		if f.isKeyword(i, "namespace") {
			return f.nsuNextOf(i+1, ";", "{")
		}
	}
	place := openTag
	if !strings.HasSuffix(f.toks[openTag].content, f.eol) && openTag+1 < len(f.toks) && f.toks[openTag+1].content == f.eol {
		place++
	}
	if c := f.nextNonWhitespace(openTag+1, len(f.toks)); c >= 0 && f.nsuIsComment(c) {
		if end := rnCommentEnd(f, c); end >= 0 {
			if strings.HasSuffix(f.toks[end].content, f.eol) {
				place = end
			} else if nl := end + 1; nl < len(f.toks) && f.toks[nl].content == f.eol {
				after := f.nextNonWhitespace(nl+1, len(f.toks))
				if after < 0 {
					after = len(f.toks)
				}
				for j := nl + 1; j < after; j++ {
					if f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol {
						place = end
						break
					}
				}
			}
		}
	}
	if n := f.nextEffective(place+1, len(f.toks)); n >= 0 && f.isKeyword(n, "declare") {
		return f.nsuNextOf(n+1, ";")
	}
	return place
}

// rnCommentEnd is CommentHelper::getCommentEndPointer, or -1.
func rnCommentEnd(f *slvFile, start int) int {
	if f.isDocOpen(start) {
		return f.docCloser(start)
	}
	c := f.toks[start].content
	if strings.HasPrefix(c, "//") || strings.HasPrefix(c, "#") {
		return start
	}
	if !strings.HasPrefix(c, "/*") {
		return -1
	}
	end := start
	for j := start + 1; j < len(f.toks) && f.toks[j].kind == token.Comment && f.toks[j].doc == slvDocNone; j++ {
		end = j
	}
	return end
}

func (r SlevomatReferenceUsedNamesOnly) isRequiredToBeUsed(name string) bool {
	for _, ns := range r.namespacesRequiredToUse {
		if rnInNamespace(name, ns) {
			return true
		}
	}
	canonical := strings.TrimLeft(name, `\`)
	for _, p := range r.namespacesRequiredToUsePartially {
		if canonical == p.namespace {
			return true
		}
	}
	return !r.anyNamespaceRequiredToUse
}

func (r SlevomatReferenceUsedNamesOnly) requiredPartialUse(name string) (rnPartial, bool) {
	for _, p := range r.namespacesRequiredToUsePartially {
		if rnInNamespace(name, p.namespace) {
			return p, true
		}
	}
	return rnPartial{}, false
}

// rnPartialUse is getPartialUse: the use statement a qualified name starts with.
func rnPartialUse(name string, uses []*nsuUse) *nsuUse {
	for _, u := range uses {
		if strings.HasPrefix(name, u.name+`\`) {
			return u
		}
	}
	return nil
}

// references is getReferences: code, attribute and, when searched, annotation references.
func (r SlevomatReferenceUsedNamesOnly) references(f *slvFile, names *slvNames, docs *slvDocs, openTag int) []rnReference {
	var out []rnReference
	for _, ref := range names.referencedNames(openTag) {
		out = append(out, rnReference{ref: ref, source: rnSourceCode})
	}
	for _, ref := range names.referencedNamesInAttributes(openTag) {
		out = append(out, rnReference{ref: ref, source: rnSourceAttribute})
	}
	if !r.searchAnnotations {
		return out
	}
	for open := openTag + 1; open < len(f.toks); open++ {
		if !f.isDocOpen(open) {
			continue
		}
		if docs.parse(open) != nil {
			for _, a := range docs.annotations(open, "") {
				for _, n := range pdNodesByType(a.node, "IdentifierTypeNode") {
					lower := strings.ToLower(n.name)
					if slvTypeHintIsSimple(lower) || slvTypeHintIsSimpleUnofficial(lower) || !slvTypeHintIsTypeName(n.name) {
						continue
					}
					out = append(out, rnReference{slvReferencedName{n.name, open, open, slvRefClass}, rnSourceAnnotation, open, n})
				}
				for _, n := range pdNodesByType(a.node, "ConstFetchNode") {
					out = append(out, rnReference{slvReferencedName{n.className, open, open, slvRefClass}, rnSourceAnnotationConstantFetch, open, n})
				}
			}
		}
		open = f.docCloser(open)
	}
	return out
}

var slvFunctions = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	for name := range strings.FieldsSeq(slvFunctionNames) {
		m[name] = true
	}
	return m
})

// slvFunctionExists is function_exists() in the PHP that runs ECS, for the
// functions of its core and bundled extensions and the global ones its
// dependencies define.
func slvFunctionExists(name string) bool {
	return slvFunctions()[strings.ToLower(strings.TrimPrefix(name, `\`))]
}

const slvFunctionNames = "" +
	"abs acos acosh addcslashes addslashes array_all array_any array_change_key_case array_chunk array_column " +
	"array_combine array_count_values array_diff array_diff_assoc array_diff_key array_diff_uassoc array_diff_ukey " +
	"array_fill array_fill_keys array_filter array_find array_find_key array_flip array_intersect " +
	"array_intersect_assoc array_intersect_key array_intersect_uassoc array_intersect_ukey array_is_list " +
	"array_key_exists array_key_first array_key_last array_keys array_map array_merge array_merge_recursive " +
	"array_multisort array_pad array_pop array_product array_push array_rand array_reduce array_replace " +
	"array_replace_recursive array_reverse array_search array_shift array_slice array_splice array_sum array_udiff " +
	"array_udiff_assoc array_udiff_uassoc array_uintersect array_uintersect_assoc array_uintersect_uassoc " +
	"array_unique array_unshift array_values array_walk array_walk_recursive arsort asin asinh asort assert " +
	"assert_options atan atan2 atanh base64_decode base64_encode base_convert basename bdump bin2hex bindec boolval " +
	"call_user_func call_user_func_array ceil chdir checkdate checkdnsrr chgrp chmod chop chown chr chroot " +
	"chunk_split class_alias class_exists class_implements class_parents class_uses clearstatcache " +
	"cli_get_process_title cli_set_process_title closedir closelog compact connection_aborted connection_status " +
	"constant convert_uudecode convert_uuencode copy cos cosh count count_chars crc32 crypt ctype_alnum ctype_alpha " +
	"ctype_cntrl ctype_digit ctype_graph ctype_lower ctype_print ctype_punct ctype_space ctype_upper ctype_xdigit " +
	"current date date_add date_create date_create_from_format date_create_immutable " +
	"date_create_immutable_from_format date_date_set date_default_timezone_get date_default_timezone_set date_diff " +
	"date_format date_get_last_errors date_interval_create_from_date_string date_interval_format date_isodate_set " +
	"date_modify date_offset_get date_parse date_parse_from_format date_sub date_sun_info date_sunrise date_sunset " +
	"date_time_set date_timestamp_get date_timestamp_set date_timezone_get date_timezone_set debug_backtrace " +
	"debug_print_backtrace debug_zval_dump decbin dechex decoct define defined deg2rad die dir dirname " +
	"disk_free_space disk_total_space diskfreespace dl dns_check_record dns_get_mx dns_get_record doubleval dump " +
	"dumpe end enum_exists error_clear_last error_get_last error_log error_reporting escapeshellarg escapeshellcmd " +
	"exec exit exp explode expm1 extension_loaded extract fast_has_parent_constructor fast_node_named fclose " +
	"fdatasync fdiv feof fflush fgetc fgetcsv fgets file file_exists file_get_contents file_put_contents fileatime " +
	"filectime filegroup fileinode filemtime fileowner fileperms filesize filetype filter_has_var filter_id " +
	"filter_input filter_input_array filter_list filter_var filter_var_array floatval flock floor flush fmod " +
	"fnmatch fopen forward_static_call forward_static_call_array fpassthru fpow fprintf fputcsv fputs fread fscanf " +
	"fseek fsockopen fstat fsync ftell ftok ftruncate func_get_arg func_get_args func_num_args function_exists " +
	"fwrite gc_collect_cycles gc_disable gc_enable gc_enabled gc_mem_caches gc_status get_browser get_called_class " +
	"get_cfg_var get_class get_class_methods get_class_vars get_current_user get_debug_type get_declared_classes " +
	"get_declared_interfaces get_declared_traits get_defined_constants get_defined_functions get_defined_vars " +
	"get_extension_funcs get_headers get_html_translation_table get_include_path get_included_files " +
	"get_loaded_extensions get_mangled_object_vars get_meta_tags get_object_vars get_parent_class " +
	"get_required_files get_resource_id get_resource_type get_resources getcwd getdate getenv gethostbyaddr " +
	"gethostbyname gethostbynamel gethostname getimagesize getimagesizefromstring getlastmod getmxrr getmygid " +
	"getmyinode getmypid getmyuid getopt getprotobyname getprotobynumber getrandmax getrusage getservbyname " +
	"getservbyport gettimeofday gettype glob gmdate gmmktime gmstrftime hash hash_algos hash_copy hash_equals " +
	"hash_file hash_final hash_hkdf hash_hmac hash_hmac_algos hash_hmac_file hash_init hash_pbkdf2 hash_update " +
	"hash_update_file hash_update_stream header header_register_callback header_remove headers_list headers_sent " +
	"hebrev hex2bin hexdec highlight_file highlight_string hrtime html_entity_decode htmlentities htmlspecialchars " +
	"htmlspecialchars_decode http_build_query http_clear_last_response_headers http_get_last_response_headers " +
	"http_response_code hypot iconv iconv_get_encoding iconv_mime_decode iconv_mime_decode_headers " +
	"iconv_mime_encode iconv_set_encoding iconv_strlen iconv_strpos iconv_strrpos iconv_substr idate " +
	"ignore_user_abort image_type_to_extension image_type_to_mime_type implode in_array inet_ntop inet_pton " +
	"ini_alter ini_get ini_get_all ini_parse_quantity ini_restore ini_set intdiv interface_exists intval ip2long " +
	"iptcembed iptcparse is_a is_array is_bool is_callable is_countable is_dir is_double is_executable is_file " +
	"is_finite is_float is_infinite is_int is_integer is_iterable is_link is_long is_nan is_null is_numeric " +
	"is_object is_readable is_resource is_scalar is_string is_subclass_of is_uploaded_file is_writable is_writeable " +
	"iterator_apply iterator_count iterator_to_array join json_decode json_encode json_last_error " +
	"json_last_error_msg json_validate key key_exists krsort ksort lcfirst lcg_value lchgrp lchown levenshtein " +
	"libxml_clear_errors libxml_disable_entity_loader libxml_get_errors libxml_get_external_entity_loader " +
	"libxml_get_last_error libxml_set_external_entity_loader libxml_set_streams_context libxml_use_internal_errors " +
	"link linkinfo localeconv localtime log log10 log1p long2ip lstat ltrim mail max mb_check_encoding mb_chr " +
	"mb_convert_case mb_convert_encoding mb_convert_kana mb_convert_variables mb_decode_mimeheader " +
	"mb_decode_numericentity mb_detect_encoding mb_detect_order mb_encode_mimeheader mb_encode_numericentity " +
	"mb_encoding_aliases mb_ereg mb_ereg_match mb_ereg_replace mb_ereg_replace_callback mb_ereg_search " +
	"mb_ereg_search_getpos mb_ereg_search_getregs mb_ereg_search_init mb_ereg_search_pos mb_ereg_search_regs " +
	"mb_ereg_search_setpos mb_eregi mb_eregi_replace mb_get_info mb_http_input mb_http_output mb_internal_encoding " +
	"mb_language mb_lcfirst mb_list_encodings mb_ltrim mb_ord mb_output_handler mb_parse_str mb_preferred_mime_name " +
	"mb_regex_encoding mb_regex_set_options mb_rtrim mb_scrub mb_send_mail mb_split mb_str_pad mb_str_split " +
	"mb_strcut mb_strimwidth mb_stripos mb_stristr mb_strlen mb_strpos mb_strrchr mb_strrichr mb_strripos " +
	"mb_strrpos mb_strstr mb_strtolower mb_strtoupper mb_strwidth mb_substitute_character mb_substr mb_substr_count " +
	"mb_trim mb_ucfirst md5 md5_file memory_get_peak_usage memory_get_usage memory_reset_peak_usage metaphone " +
	"method_exists mhash mhash_count mhash_get_block_size mhash_get_hash_name mhash_keygen_s2k microtime min mkdir " +
	"mktime move_uploaded_file mt_getrandmax mt_rand mt_srand natcasesort natsort net_get_interfaces next nl2br " +
	"nl_langinfo number_format ob_clean ob_end_clean ob_end_flush ob_flush ob_get_clean ob_get_contents " +
	"ob_get_flush ob_get_length ob_get_level ob_get_status ob_implicit_flush ob_list_handlers ob_start octdec " +
	"opendir openlog ord output_add_rewrite_var output_reset_rewrite_vars pack parse_ini_file parse_ini_string " +
	"parse_str parse_url passthru password_algos password_get_info password_hash password_needs_rehash " +
	"password_verify pathinfo pclose pfsockopen php_ini_loaded_file php_ini_scanned_files php_sapi_name " +
	"php_strip_whitespace php_uname phpcredits phpinfo phpversion pi popen pos pow preg_filter preg_grep " +
	"preg_last_error preg_last_error_msg preg_match preg_match_all preg_quote preg_replace preg_replace_callback " +
	"preg_replace_callback_array preg_split prev print_r printf proc_close proc_get_status proc_nice proc_open " +
	"proc_terminate property_exists putenv quoted_printable_decode quoted_printable_encode quotemeta rad2deg rand " +
	"random_bytes random_int range rawurldecode rawurlencode readdir readfile readlink realpath realpath_cache_get " +
	"realpath_cache_size register_shutdown_function register_tick_function rename request_parse_body reset " +
	"restore_error_handler restore_exception_handler rewind rewinddir rmdir round rsort rtrim scandir serialize " +
	"set_error_handler set_exception_handler set_file_buffer set_include_path set_time_limit setcookie setlocale " +
	"setrawcookie settype sha1 sha1_file shell_exec show_source shuffle similar_text sin sinh sizeof sleep " +
	"socket_get_status socket_set_blocking socket_set_timeout sort soundex spl_autoload spl_autoload_call " +
	"spl_autoload_extensions spl_autoload_functions spl_autoload_register spl_autoload_unregister spl_classes " +
	"spl_object_hash spl_object_id sprintf sqrt srand sscanf stat str_contains str_decrement str_ends_with " +
	"str_getcsv str_increment str_ireplace str_pad str_repeat str_replace str_rot13 str_shuffle str_split " +
	"str_starts_with str_word_count strcasecmp strchr strcmp strcoll strcspn stream_bucket_append " +
	"stream_bucket_make_writeable stream_bucket_new stream_bucket_prepend stream_context_create " +
	"stream_context_get_default stream_context_get_options stream_context_get_params stream_context_set_default " +
	"stream_context_set_option stream_context_set_options stream_context_set_params stream_copy_to_stream " +
	"stream_filter_append stream_filter_prepend stream_filter_register stream_filter_remove stream_get_contents " +
	"stream_get_filters stream_get_line stream_get_meta_data stream_get_transports stream_get_wrappers " +
	"stream_is_local stream_isatty stream_register_wrapper stream_resolve_include_path stream_select " +
	"stream_set_blocking stream_set_chunk_size stream_set_read_buffer stream_set_timeout stream_set_write_buffer " +
	"stream_socket_accept stream_socket_client stream_socket_enable_crypto stream_socket_get_name " +
	"stream_socket_pair stream_socket_recvfrom stream_socket_sendto stream_socket_server stream_socket_shutdown " +
	"stream_supports_lock stream_wrapper_register stream_wrapper_restore stream_wrapper_unregister strftime " +
	"strip_tags stripcslashes stripos stripslashes stristr strlen strnatcasecmp strnatcmp strncasecmp strncmp " +
	"strpbrk strpos strptime strrchr strrev strripos strrpos strspn strstr strtok strtolower strtotime strtoupper " +
	"strtr strval substr substr_compare substr_count substr_replace symlink sys_get_temp_dir sys_getloadavg syslog " +
	"system tan tanh tempnam time time_nanosleep time_sleep_until timezone_abbreviations_list " +
	"timezone_identifiers_list timezone_location_get timezone_name_from_abbr timezone_name_get timezone_offset_get " +
	"timezone_open timezone_transitions_get timezone_version_get tmpfile token_get_all token_name touch " +
	"trait_exists trigger_deprecation trigger_error trim uasort ucfirst ucwords uksort umask uniqid unlink unpack " +
	"unregister_tick_function unserialize urldecode urlencode user_error usleep usort utf8_decode utf8_encode " +
	"var_dump var_export version_compare vfprintf vprintf vsprintf wordwrap zend_version"
