package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/UnusedUsesSniff.php

// SlevomatUnusedUses removes the import uses a file does not reference.
type SlevomatUnusedUses struct {
	searchAnnotations      bool
	ignoredAnnotationNames []string
	ignoredAnnotations     []string
}

func (SlevomatUnusedUses) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\UnusedUsesSniff`
}

func (SlevomatUnusedUses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUnusedUses{}.Name())
}

func (r SlevomatUnusedUses) WithConfig(config map[string]any) fixer.Fixer {
	r.searchAnnotations, _ = config["searchAnnotations"].(bool)
	r.ignoredAnnotationNames, _ = slvStringList(config["ignoredAnnotationNames"])
	r.ignoredAnnotations, _ = slvStringList(config["ignoredAnnotations"])
	return r
}

func (r SlevomatUnusedUses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		if openTag := f.slvFirstOpenTag(); openTag >= 0 {
			r.process(f, openTag)
		}
	})
}

func (r SlevomatUnusedUses) process(f *slvFile, start int) {
	groups := f.nsuFileUses()
	byAnchor := map[int]map[string]*nsuUse{}
	for _, g := range groups {
		byAnchor[g.anchor] = map[string]*nsuUse{}
		for _, u := range g.uses {
			byAnchor[g.anchor][uuUseID(u)] = u
		}
	}
	namespaces := slices.Clone(f.refNamespacePointers())
	slices.Reverse(namespaces)
	before := func(i int) int {
		for _, ns := range namespaces {
			if ns < i {
				return ns
			}
		}
		return start
	}

	used := map[int]map[string]bool{}
	markUsed := func(anchor int, id string) {
		if used[anchor] == nil {
			used[anchor] = map[string]bool{}
		}
		used[anchor][id] = true
	}
	names := newSlvNames(f)
	for _, refs := range [][]slvReferencedName{names.referencedNames(start), uuReferencedNamesInAttributes(names, start)} {
		for _, ref := range refs {
			anchor := before(ref.start)
			parts := strings.Split(strings.TrimLeft(ref.name, `\`), `\`)
			id := slvUseID(slvRefClass, parts[0])
			if len(parts) == 1 {
				id = slvUseID(ref.typ, parts[0])
			}
			if strings.HasPrefix(ref.name, `\`) || byAnchor[anchor] == nil || byAnchor[anchor][id] == nil {
				continue
			}
			markUsed(anchor, id)
		}
	}

	if r.searchAnnotations {
		d := newSlvDocs(f)
		for open := start + 1; open < len(f.toks); open++ {
			if !f.isDocOpen(open) {
				continue
			}
			closer := f.docCloser(open)
			annotations := d.annotations(open, "")
			anchor := before(open - 1)
			if len(annotations) > 0 && byAnchor[anchor] != nil {
				var contents []string
				for _, g := range groups {
					if g.anchor != anchor {
						continue
					}
					for _, u := range g.uses {
						if byAnchor[anchor][uuUseID(u)] != u || u.kind != "" {
							continue
						}
						if contents == nil {
							contents = r.annotationContents(d.parse(open), annotations)
						}
						for _, c := range contents {
							if uuMentions(c, u.name) {
								markUsed(anchor, uuUseID(u))
							}
						}
					}
				}
			}
			open = closer
		}
	}

	for _, g := range groups {
		for _, u := range g.uses {
			id := uuUseID(u)
			if byAnchor[g.anchor][id] != u || used[g.anchor][id] {
				continue
			}
			end := u.pointer
			for end < len(f.toks) && !f.is(end, ";") {
				end++
			}
			f.beginChangeset()
			f.removeBetweenIncluding(u.pointer, min(end+1, len(f.toks)-1))
			f.endChangeset()
		}
	}
}

// annotationContents is UnusedUsesSniff::getAnnotationsContentsToCheck.
func (r SlevomatUnusedUses) annotationContents(doc *slvParsedDoc, annotations []slvAnnotation) []string {
	ignoredNames := append(slices.Clone(r.ignoredAnnotationNames), "@param", "@throws", "@property", "@method")
	contents := []string{}
	for _, a := range annotations {
		if slices.Contains(r.ignoredAnnotations, a.name) || a.isInvalid() {
			continue
		}
		value := a.node.value()
		if value.kind == "GenericTagValueNode" {
			contents = append(contents, a.name, uuDocText(doc, value))
			continue
		}
		for _, n := range pdNodesByType(a.node, "IdentifierTypeNode") {
			if slvTypeHintIsSimple(n.name) || slvTypeHintIsSimpleUnofficial(n.name) || !slvTypeHintIsTypeName(n.name) {
				continue
			}
			contents = append(contents, n.name)
		}
		for _, n := range pdNodesByType(a.node, "DoctrineAnnotation") {
			if !slices.Contains(ignoredNames, n.name) {
				contents = append(contents, n.name)
			}
		}
		for _, n := range pdNodesByType(a.node, "ConstFetchNode") {
			contents = append(contents, n.className)
		}
	}
	return contents
}

// uuDocText is the text of a doc comment node with each line break as "\n",
// which is all GenericTagValueNode::$value differs in for uuMentions.
func uuDocText(doc *slvParsedDoc, n *pdNode) string {
	var b strings.Builder
	for i := max(n.start, 0); i <= n.end && i < len(doc.toks); i++ {
		if doc.toks[i].t == pdPhpdocEOL {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(doc.toks[i].v)
	}
	return b.String()
}

// uuMentions is preg_match('~(?<=^|[^a-z\\])(name)(?=\s|::|\\|\||\[|$)~im', content).
func uuMentions(content, name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i+len(name) <= len(content); i++ {
		if !uuASCIIEqualFold(content[i:i+len(name)], name) {
			continue
		}
		if i > 0 {
			if c := content[i-1]; pdIsLetter(c) || c == '\\' {
				continue
			}
		}
		rest := content[i+len(name):]
		if rest == "" || strings.HasPrefix(rest, "::") || strings.ContainsRune(" \t\n\v\f\r\\|[", rune(rest[0])) {
			return true
		}
	}
	return false
}

// uuASCIIEqualFold folds only ASCII letters, as PCRE's /i without /u does.
func uuASCIIEqualFold(a, b string) bool {
	for i := range len(a) {
		x, y := a[i], b[i]
		if pdIsLetter(x) && pdIsLetter(y) {
			x, y = x|0x20, y|0x20
		}
		if x != y {
			return false
		}
	}
	return true
}

func uuUseID(u *nsuUse) string {
	switch u.kind {
	case "function":
		return slvUseID(slvRefFunction, u.name)
	case "const":
		return slvUseID(slvRefConstant, u.name)
	}
	return slvUseID(slvRefClass, u.name)
}

// uuReferencedNamesInAttributes is ReferencedNameHelper::getAllReferencedNamesInAttributes.
func uuReferencedNamesInAttributes(n *slvNames, openTag int) []slvReferencedName {
	f := n.f
	var out []slvReferencedName
	for a := openTag + 1; a < len(f.toks); a++ {
		if f.toks[a].kind != token.AttributeStart || f.toks[a].closer < 0 {
			continue
		}
		closer := f.toks[a].closer
		level := 0
		for p := a + 1; p < closer; p++ {
			switch {
			case f.is(p, "("):
				level++
				continue
			case f.is(p, ")"):
				level--
				continue
			case !n.isNameToken(p):
				continue
			}
			typ := slvRefClass
			prev := f.prevEffective(p - 1)
			switch {
			case prev >= 0 && (f.toks[prev].kind == token.OpenTag || f.toks[prev].kind == token.AttributeStart):
			case f.is(prev, ",") && level == 0:
			case n.isReferencedName(p):
				typ = n.referenceType(p, p)
			default:
				continue
			}
			out = append(out, slvReferencedName{f.toks[p].content, a, closer, typ})
		}
	}
	return out
}
