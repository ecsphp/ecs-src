package rules

import (
	"regexp"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Shared port of DataProviderAnalyzer, used by the four PhpUnitDataProvider
// fixers. It finds every PHPUnit data provider method referenced from a
// `@dataProvider` docblock tag or a `#[DataProvider('name')]` attribute.

// puProviderUsage is one reference to a data provider: the token that carries it
// (a doc comment, or the string literal inside a #[DataProvider] attribute) and,
// for a doc comment, the byte offset of the match within that comment.
type puProviderUsage struct {
	index  int
	offset int
}

// puProviderAnalysis mirrors DataProviderAnalysis: the provider method's name
// (as written), its name-token index and the places that reference it.
type puProviderAnalysis struct {
	name      string
	nameIndex int
	usages    []puProviderUsage
}

// puDataProviderDocRe matches `@dataProvider <name>`, where the name may be a
// `Class::method` reference (those are dropped later, as they name no local
// method).
var puDataProviderDocRe = regexp.MustCompile(`@dataProvider[ \t]+((?:\\?[A-Za-z_\x80-\xff][A-Za-z0-9_\x80-\xff]*(?:\\[A-Za-z_\x80-\xff][A-Za-z0-9_\x80-\xff]*)*::)?[A-Za-z_\x80-\xff][A-Za-z0-9_\x80-\xff]*)`)

// puDataProviders mirrors DataProviderAnalyzer::getDataProviders for [start, end).
func puDataProviders(s *tokens.Stream, start, end int) []puProviderAnalysis {
	methods := map[string]int{}
	var methodOrder []int
	for i := start; i < end && i < s.Len(); i++ {
		if !kwIs(s.At(i), "function") {
			continue
		}
		n := puNextNonWhitespace(s, i)
		if n < 0 || s.At(n).Kind != token.Ident {
			continue
		}
		methods[strings.ToLower(s.At(n).Value)] = n
		methodOrder = append(methodOrder, n)
	}

	usagesByName := map[string][]puProviderUsage{}
	var nameOrder []string
	add := func(name string, u puProviderUsage) {
		if _, ok := usagesByName[name]; !ok {
			nameOrder = append(nameOrder, name)
		}
		usagesByName[name] = append(usagesByName[name], u)
	}

	for _, methodNameIdx := range methodOrder {
		attrs, docIdx := puDataProviderContext(s, methodNameIdx)
		for _, a := range attrs {
			add(a.name, puProviderUsage{index: a.index, offset: 0})
		}
		if docIdx >= 0 {
			content := s.At(docIdx).Value
			for _, m := range puDataProviderDocRe.FindAllStringSubmatchIndex(content, -1) {
				add(content[m[2]:m[3]], puProviderUsage{index: docIdx, offset: m[0]})
			}
		}
	}

	byMethod := map[int]puProviderAnalysis{}
	for _, name := range nameOrder {
		nameIdx, ok := methods[strings.ToLower(name)]
		if !ok {
			continue
		}
		byMethod[nameIdx] = puProviderAnalysis{
			name:      s.At(nameIdx).Value,
			nameIndex: nameIdx,
			usages:    usagesByName[name],
		}
	}

	analyses := make([]puProviderAnalysis, 0, len(byMethod))
	for _, a := range byMethod {
		analyses = append(analyses, a)
	}
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].nameIndex < analyses[j].nameIndex
	})
	return analyses
}

// puAttrProvider is one #[DataProvider('name')] reference: the string-literal
// token index and the provider name it holds.
type puAttrProvider struct {
	index int
	name  string
}

// puDataProviderContext mirrors getAttributeIndexAndDocCommentIndices: walk back
// from the method name to the previous ; { } or open tag, collecting every
// #[DataProvider] usage and the nearest-to-the-boundary doc comment.
func puDataProviderContext(s *tokens.Stream, methodNameIdx int) ([]puAttrProvider, int) {
	docIdx := -1
	var attrs []puAttrProvider
	for i := methodNameIdx - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{" || t.Value == "}") {
			break
		}
		if t.Kind == token.OpenTag {
			break
		}
		switch t.Kind {
		case token.DocComment:
			docIdx = i
		case token.AttributeEnd:
			if attrStart := s.AttributeStartOf(i); attrStart >= 0 {
				attrs = append(attrs, puDataProviderAttrs(s, attrStart, i)...)
				i = attrStart
			}
		}
	}
	sort.Slice(attrs, func(a, b int) bool { return attrs[a].index < attrs[b].index })
	return attrs, docIdx
}

// puDataProviderAttrs finds the #[DataProvider('name')] usages in the attribute
// [start, end]. The attribute name is matched on its short part, like the other
// PHPUnit attribute ports.
func puDataProviderAttrs(s *tokens.Stream, start, end int) []puAttrProvider {
	var out []puAttrProvider
	for i := start + 1; i < end; i++ {
		if s.At(i).Kind != token.Ident || s.At(i).Value != "DataProvider" {
			continue
		}
		open := sigNext(s, i)
		if !isPunctVal(s, open, "(") {
			continue
		}
		arg := sigNext(s, open)
		if arg < 0 || !s.At(arg).IsLiteral() {
			continue
		}
		if !isPunctVal(s, sigNext(s, arg), ")") {
			continue
		}
		out = append(out, puAttrProvider{index: arg, name: puStripQuotes(s.At(arg).Value)})
	}
	return out
}

// puStripQuotes drops the surrounding quotes of a string literal, mirroring
// substr($content, 1, -1).
func puStripQuotes(v string) string {
	if len(v) >= 2 {
		return v[1 : len(v)-1]
	}
	return v
}

// puNextFunctionKeyword returns the index of the first `function` keyword after
// from (bounded by end), or -1.
func puNextFunctionKeyword(s *tokens.Stream, from, end int) int {
	for i := from + 1; i < end && i < s.Len(); i++ {
		if kwIs(s.At(i), "function") {
			return i
		}
	}
	return -1
}

// puNextMethodNameIndex mirrors findSequence([[T_FUNCTION],[T_STRING]]): the name
// token of the first method at or after from, or -1.
func puNextMethodNameIndex(s *tokens.Stream, from, end int) int {
	for i := from; i < end && i < s.Len(); i++ {
		if !kwIs(s.At(i), "function") {
			continue
		}
		if n := sigNext(s, i); n >= 0 && s.At(n).Kind == token.Ident {
			return n
		}
	}
	return -1
}

// puFindFunctionNamed returns the `function` keyword index of a method named
// name in [start, end), or -1.
func puFindFunctionNamed(s *tokens.Stream, start, end int, name string) int {
	for i := start; i < end && i < s.Len(); i++ {
		if !kwIs(s.At(i), "function") {
			continue
		}
		if n := sigNext(s, i); n >= 0 && s.At(n).Kind == token.Ident && s.At(n).Value == name {
			return i
		}
	}
	return -1
}
