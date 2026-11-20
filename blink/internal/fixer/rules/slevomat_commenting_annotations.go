package rules

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// cmcAnnotationPointers are Annotation::getStartPointer and getEndPointer.
func (f *slvFile) cmcAnnotationPointers(p *slvParsedDoc, a slvAnnotation) (int, int) {
	return f.thAnnotationPointers(p, a.node)
}

var (
	cmcStandardAnnotations = []string{
		"api", "author", "category", "copyright", "deprecated", "example", "filesource", "global", "ignore",
		"inheritDoc", "internal", "license", "link", "method", "package", "param", "property", "property-read",
		"property-write", "return", "see", "since", "source", "subpackage", "throws", "todo", "uses", "used-by",
		"var", "version",
	}
	cmcStaticAnalysisAnnotations = []string{
		"api", "allow-private-mutation", "assert", "assert-if-true", "assert-if-false", "consistent-constructor",
		"consistent-templates", "extends", "external-mutation-free", "implements", "mixin", "ignore-falsable-return",
		"ignore-nullable-return", "ignore-var", "ignore-variable-method", "ignore-variable-property", "immutable",
		"import-type", "internal", "method", "mutation-free", "no-named-arguments", "param", "param-out", "property",
		"property-read", "property-write", "psalm-check-type", "psalm-check-type-exact", "psalm-suppress",
		"psalm-trace", "pure", "readonly", "readonly-allow-private-mutation", "require-extends",
		"require-implements", "return", "seal-properties", "self-out", "template", "template-covariant",
		"template-extends", "template-implements", "template-use", "this-out", "type", "var", "yield",
	}
	cmcPhpunitAnnotations = []string{
		"author", "after", "afterClass", "backupGlobals", "backupStaticAttributes", "before", "beforeClass",
		"codeCoverageIgnore", "codeCoverageIgnoreStart", "codeCoverageIgnoreEnd", "covers", "coversDefaultClass",
		"coversNothing", "dataProvider", "depends", "doesNotPerformAssertions", "group", "large", "medium",
		"preserveGlobalState", "requires", "runTestsInSeparateProcesses", "runInSeparateProcess", "small", "test",
		"testdox", "testWith", "ticket", "uses",
	}
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/AnnotationNameSniff.php
//
// SlevomatAnnotationName fixes the letter case of known annotation names.
type SlevomatAnnotationName struct {
	annotations []string
	configured  bool
}

func (SlevomatAnnotationName) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\AnnotationNameSniff`
}

func (SlevomatAnnotationName) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAnnotationName{}.Name())
}

func (r SlevomatAnnotationName) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := slvStringList(config["annotations"]); ok {
		r.annotations, r.configured = list, true
	}
	return r
}

// names is getNormalizedAnnotationNames: lowercased name to name, in order.
func (r SlevomatAnnotationName) names() ([]string, map[string]string) {
	var names []string
	if r.configured {
		for _, a := range r.annotations {
			if a = strings.TrimSpace(a); a != "" {
				names = append(names, strings.TrimLeft(a, "@"))
			}
		}
	} else {
		names = slices.Concat(cmcStandardAnnotations, cmcPhpunitAnnotations, cmcStaticAnalysisAnnotations)
		for _, a := range cmcStaticAnalysisAnnotations {
			if strings.HasPrefix(a, "psalm") {
				continue
			}
			for _, prefix := range []string{"psalm", "phpstan"} {
				names = append(names, prefix+"-"+a)
			}
		}
	}
	var ordered []string
	byLower := map[string]string{}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		lc := strings.ToLower("@" + n)
		if _, ok := byLower[lc]; !ok {
			ordered = append(ordered, lc)
		}
		byLower[lc] = "@" + n
	}
	return ordered, byLower
}

func (r SlevomatAnnotationName) Fix(s *tokens.Stream) bool {
	ordered, correct := r.names()
	values := make([]string, len(ordered))
	for n, lc := range ordered {
		values[n] = regexp.QuoteMeta(correct[lc])
	}
	inline := regexp.MustCompile(`(?i)\{(` + strings.Join(values, "|") + `)\}`)
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(f.isDocOpen, func(open int) int {
			p := d.parse(open)
			if p != nil {
				for _, a := range p.annotations {
					want, ok := correct[strings.ToLower(a.name)]
					if !ok || want == a.name {
						continue
					}
					start, _ := f.cmcAnnotationPointers(p, a)
					bare := strings.TrimLeft(a.name, "@")
					if strings.TrimLeft(d.useStatements().resolveClassName(f, bare, start), `\`) != bare {
						continue
					}
					f.beginChangeset()
					f.replace(start, want)
					f.endChangeset()
				}
			}
			closeIdx := f.docCloser(open)
			content := f.content(open, closeIdx)
			for _, m := range inline.FindAllStringSubmatchIndex(content, -1) {
				found := content[m[2]:m[3]]
				want := correct[strings.ToLower(found)]
				if want == found {
					continue
				}
				f.beginChangeset()
				f.change(open, closeIdx, content[:m[2]]+want+content[m[3]:])
				f.endChangeset()
			}
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/ThrowsAnnotationsOrderSniff.php
//
// SlevomatThrowsAnnotationsOrder sorts the @throws annotations of a function.
type SlevomatThrowsAnnotationsOrder struct {
	caseSensitive bool
}

func (SlevomatThrowsAnnotationsOrder) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\ThrowsAnnotationsOrderSniff`
}

func (SlevomatThrowsAnnotationsOrder) SourceURL() string {
	return fixer.SourceURLFor(SlevomatThrowsAnnotationsOrder{}.Name())
}

func (r SlevomatThrowsAnnotationsOrder) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["caseSensitive"].(bool); ok {
		r.caseSensitive = v
	}
	return r
}

func (r SlevomatThrowsAnnotationsOrder) Fix(s *tokens.Stream) bool {
	compare := SlevomatCatchExceptionsOrder(r).compare
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(func(i int) bool { return f.isKeyword(i, "function") }, func(fn int) int {
			open := f.docOpenBefore(fn)
			if open < 0 {
				return 0
			}
			annotations := d.annotations(open, "@throws")
			if len(annotations) <= 1 {
				return 0
			}
			p := d.parse(open)
			actual := make([]string, len(annotations))
			for n, a := range annotations {
				actual[n] = cmcExceptionName(p, a)
			}
			order := make([]int, len(annotations))
			for n := range order {
				order[n] = n
			}
			sort.SliceStable(order, func(a, b int) bool { return compare(actual[order[a]], actual[order[b]]) < 0 })
			if slices.IsSorted(order) {
				return 0
			}
			contents := make([]string, len(annotations))
			pointers := make([][2]int, len(annotations))
			for n, a := range annotations {
				start, end := f.cmcAnnotationPointers(p, a)
				pointers[n] = [2]int{start, end}
				contents[n] = strings.Trim(f.content(start, end), " \t\n\r\x00\x0B")
			}
			first, last := pointers[0][0], pointers[len(pointers)-1][1]
			indentation := f.indentation(first)
			var b strings.Builder
			for n, key := range order {
				if n > 0 {
					b.WriteString(f.eol + indentation)
				}
				b.WriteString(contents[key])
			}
			f.beginChangeset()
			f.change(first, last, b.String())
			f.endChangeset()
			return 0
		})
	})
}

// cmcExceptionName is ThrowsAnnotationsOrderSniff::getExceptionName.
func cmcExceptionName(p *slvParsedDoc, a slvAnnotation) string {
	if a.isInvalid() {
		return ""
	}
	t := a.node.value().fields[0].node
	if t.kind == "IdentifierTypeNode" {
		return t.name
	}
	if t.kind == "UnionTypeNode" && len(t.fields[0].list) > 0 && t.fields[0].list[0].kind == "IdentifierTypeNode" {
		return t.fields[0].list[0].name
	}
	return thPrint(p.toks, t)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/UselessFunctionDocCommentSniff.php
//
// SlevomatUselessFunctionDocComment removes doc comments of functions that
// only repeat the native type hints.
type SlevomatUselessFunctionDocComment struct {
	traversable []string
}

func (SlevomatUselessFunctionDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\UselessFunctionDocCommentSniff`
}

func (SlevomatUselessFunctionDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessFunctionDocComment{}.Name())
}

func (r SlevomatUselessFunctionDocComment) WithConfig(config map[string]any) fixer.Fixer {
	r.traversable = thdParseOptions(map[string]any{"traversableTypeHints": config["traversableTypeHints"]}).traversable
	return r
}

func (r SlevomatUselessFunctionDocComment) Fix(s *tokens.Stream) bool {
	o := thdOptions{traversable: r.traversable}
	return slvRun(s, func(f *slvFile) {
		c := newThdCtx(f, o, "SlevomatCodingStandard.Commenting.UselessFunctionDocComment")
		f.eachToken(func(i int) bool { return f.isKeyword(i, "function") && !f.isClosure(i) }, func(fn int) int {
			open := f.docOpenBefore(fn)
			if open < 0 || c.hasInheritdoc(open) || f.thHasDescription(open) {
				return 0
			}
			p := c.d.parse(open)
			returnHint := c.n.thReturnTypeHint(fn)
			if returns := c.d.annotations(open, "@return"); len(returns) > 0 && !c.isAnnotationUseless(fn, returnHint, p, returns[0]) {
				return 0
			}
			params := c.parameters(fn)
			for name, pa := range c.parameterAnnotations(fn) {
				idx := slices.IndexFunc(params, func(p thdParam) bool { return p.name == name })
				if idx < 0 || !c.isAnnotationUseless(fn, params[idx].hint, pa.p, pa.a) {
					return 0
				}
			}
			for _, a := range c.d.annotations(open, "") {
				if a.name != "@param" && a.name != "@return" {
					return 0
				}
			}
			end := f.nextEffective(f.docCloser(open)+1, len(f.toks))
			if end < 0 {
				end = len(f.toks)
			}
			f.beginChangeset()
			f.removeBetweenIncluding(open, end-1)
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/UselessInheritDocCommentSniff.php
//
// SlevomatUselessInheritDocComment removes doc comments holding only
// @inheritDoc where the native type hints say everything.
type SlevomatUselessInheritDocComment struct{}

func (SlevomatUselessInheritDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\UselessInheritDocCommentSniff`
}

func (SlevomatUselessInheritDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessInheritDocComment{}.Name())
}

var cmcInheritDocOnly = regexp.MustCompile(`(?i)^(?:\{@inheritDoc\}|@inheritDoc)$`)

func cmcIsSimpleIterable(t string) bool {
	lc := strings.ToLower(t)
	return lc == "array" || lc == "iterable"
}

func (SlevomatUselessInheritDocComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := newSlvNames(f)
		f.eachToken(f.isDocOpen, func(open int) int {
			closeIdx := f.docCloser(open)
			var b strings.Builder
			for i := open + 1; i < closeIdx; i++ {
				if f.toks[i].doc == slvDocWhitespace || f.toks[i].doc == slvDocStar {
					continue
				}
				b.WriteString(f.toks[i].content)
			}
			if !cmcInheritDocOnly.MatchString(b.String()) {
				return 0
			}
			owner := -1
			for i := closeIdx + 1; i < len(f.toks); i++ {
				if f.isAttributeOpen(i) {
					i = f.toks[i].closer
					continue
				}
				if f.isKeyword(i, "function") || f.isKeyword(i, "fn") || slvIsTypeHintToken(n, i) {
					owner = i
					break
				}
			}
			if owner < 0 {
				return 0
			}
			if f.isKeyword(owner, "function") || f.isKeyword(owner, "fn") {
				returnHint := n.thReturnTypeHint(owner)
				if returnHint == nil || cmcIsSimpleIterable(strings.TrimPrefix(returnHint.hint, "?")) {
					return 0
				}
				c := &thdCtx{f: f, n: n}
				for _, p := range c.parameters(owner) {
					if p.hint == nil || cmcIsSimpleIterable(p.hint.hint) {
						return 0
					}
				}
			}
			start := open - 1
			for start > 0 && f.toks[start].line == f.toks[open].line {
				start--
			}
			f.beginChangeset()
			f.removeBetweenIncluding(start, closeIdx)
			f.endChangeset()
			return 0
		})
	})
}
