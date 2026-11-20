package rules

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/DocCommentSpacingSniff.php

// SlevomatDocCommentSpacing enforces the blank lines around the description
// and annotations of a doc comment and, when configured, the grouping and
// order of its annotations.
type SlevomatDocCommentSpacing struct {
	beforeFirst, betweenDescription, betweenTypes, betweenGroups, afterLast int
	groups                                                                  [][]string
	configured                                                              bool
}

func (SlevomatDocCommentSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\DocCommentSpacingSniff`
}

func (SlevomatDocCommentSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDocCommentSpacing{}.Name())
}

func (r SlevomatDocCommentSpacing) withDefaults() SlevomatDocCommentSpacing {
	r.betweenDescription, r.betweenGroups, r.configured = 1, 1, true
	return r
}

func (r SlevomatDocCommentSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = SlevomatDocCommentSpacing{}.withDefaults()
	r.beforeFirst = configInt(config, "linesCountBeforeFirstContent", r.beforeFirst)
	r.betweenDescription = configInt(config, "linesCountBetweenDescriptionAndAnnotations", r.betweenDescription)
	r.betweenTypes = configInt(config, "linesCountBetweenDifferentAnnotationsTypes", r.betweenTypes)
	r.betweenGroups = configInt(config, "linesCountBetweenAnnotationsGroups", r.betweenGroups)
	r.afterLast = configInt(config, "linesCountAfterLastContent", r.afterLast)
	list, _ := slvStringList(config["annotationsGroups"])
	for _, group := range list {
		names := []any{}
		for name := range strings.SplitSeq(group, ",") {
			names = append(names, name)
		}
		normalized, _ := slvStringList(names)
		var unique []string
		for _, name := range normalized {
			if !slices.Contains(unique, name) {
				unique = append(unique, name)
			}
		}
		r.groups = append(r.groups, unique)
	}
	return r
}

// cmAAnnotation is slevomat's Annotation with its start and end pointers.
type cmAAnnotation struct {
	name       string
	start, end int
}

func (r SlevomatDocCommentSpacing) Fix(s *tokens.Stream) bool {
	if !r.configured {
		r = r.withDefaults()
	}
	return slvRunDocs(s, func(d *slvDocs, open int) bool {
		r.process(d, open)
		return true
	})
}

func (r SlevomatDocCommentSpacing) process(d *slvDocs, open int) {
	f := d.f
	if d.isInline(open) {
		return
	}
	closer := f.docCloser(open)
	hasContent := false
	for i := open + 1; i < closer; i++ {
		if f.toks[i].doc != slvDocWhitespace && f.toks[i].doc != slvDocStar {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return
	}
	p := d.parse(open)
	if p == nil {
		return
	}
	children := p.root.fields[0].list
	if len(children) == 0 {
		return
	}
	firstStart, firstEnd := f.thAnnotationPointers(p, children[0])

	annotations := make([]*cmAAnnotation, 0, len(p.annotations))
	for _, a := range p.annotations {
		start, end := f.thAnnotationPointers(p, a.node)
		annotations = append(annotations, &cmAAnnotation{name: a.name, start: start, end: end})
	}
	sort.SliceStable(annotations, func(i, j int) bool { return annotations[i].start < annotations[j].start })

	firstAnnotation := -1
	lastContentEnd := firstEnd
	if len(annotations) > 0 {
		firstAnnotation = annotations[0].start
		lastContentEnd = annotations[len(annotations)-1].end
	}

	r.checkBeforeFirstContent(f, open, firstStart)
	r.checkBetweenDescriptionAndAnnotations(f, open, firstStart, firstEnd, firstAnnotation)
	if len(annotations) > 1 {
		if len(r.groups) == 0 {
			r.checkBetweenDifferentTypes(f, open, annotations)
		} else {
			r.checkGroups(f, open, annotations)
		}
	}
	r.checkAfterLastContent(f, open, closer, lastContentEnd)
}

// cmALines is max(substr_count(whitespace, eol) - 1, 0).
func cmALines(f *slvFile, whitespace string) int {
	return max(strings.Count(whitespace, f.eol)-1, 0)
}

func (f *slvFile) cmAAddNewline(i int) {
	f.replaceToken(i, f.tokenContent(i)+f.eol)
}

func (f *slvFile) cmAAddEmptyLines(i, count int, indentation string) {
	for range count {
		f.add(i, fmt.Sprintf("%s *%s", indentation, f.eol))
	}
}

func (r SlevomatDocCommentSpacing) checkBeforeFirstContent(f *slvFile, open, firstStart int) {
	opener := f.toks[open].content
	whitespace := opener[:min(len(opener), 3)] + f.content(open+1, firstStart-1)
	if cmALines(f, whitespace) == r.beforeFirst {
		return
	}
	indentation := f.indentation(open)
	f.beginChangeset()
	f.change(open, firstStart-1, "/**"+f.eol)
	f.cmAAddEmptyLines(open, r.beforeFirst, indentation)
	f.addBefore(firstStart, indentation+" * ")
	f.endChangeset()
}

var cmATrailingWhitespace = regexp.MustCompile(`\s+$`)

func (r SlevomatDocCommentSpacing) checkBetweenDescriptionAndAnnotations(f *slvFile, open, firstStart, firstEnd, firstAnnotation int) {
	if firstAnnotation < 0 || firstStart == firstAnnotation {
		return
	}
	whitespace := cmATrailingWhitespace.FindString(f.toks[firstEnd].content) + f.content(firstEnd+1, firstAnnotation-1)
	if cmALines(f, whitespace) == r.betweenDescription {
		return
	}
	indentation := f.indentation(open)
	f.beginChangeset()
	f.cmAAddNewline(firstEnd)
	f.slvlcRemoveBetween(firstEnd, firstAnnotation)
	f.cmAAddEmptyLines(firstEnd, r.betweenDescription, indentation)
	f.addBefore(firstAnnotation, indentation+" * ")
	f.endChangeset()
}

func (r SlevomatDocCommentSpacing) checkBetweenDifferentTypes(f *slvFile, open int, annotations []*cmAAnnotation) {
	indentation := f.indentation(open)
	previous := annotations[0]
	for _, a := range annotations[1:] {
		if a.name == previous.name {
			previous = a
			continue
		}
		if cmALines(f, f.content(previous.end+1, a.start-1)) == r.betweenTypes {
			previous = a
			continue
		}
		f.beginChangeset()
		f.slvlcRemoveBetween(previous.end, a.start)
		f.cmAAddNewline(previous.end)
		f.cmAAddEmptyLines(previous.end, r.betweenTypes, indentation)
		f.addBefore(a.start, indentation+" * ")
		f.endChangeset()
		previous = a
	}
}

func (r SlevomatDocCommentSpacing) checkGroups(f *slvFile, open int, annotations []*cmAAnnotation) {
	var groups [][]*cmAAnnotation
	var group []*cmAAnnotation
	var previous *cmAAnnotation
	for _, a := range annotations {
		if previous == nil || f.toks[previous.end].line+1 == f.toks[a.start].line {
			group = append(group, a)
			previous = a
			continue
		}
		groups = append(groups, group)
		group = []*cmAAnnotation{a}
		previous = a
	}
	if len(group) > 0 {
		groups = append(groups, group)
	}
	r.checkGroupsOrder(f, open, groups, annotations)
	r.checkBetweenGroups(f, open, groups)
}

func (r SlevomatDocCommentSpacing) checkBetweenGroups(f *slvFile, open int, groups [][]*cmAAnnotation) {
	for n := 1; n < len(groups); n++ {
		last := groups[n-1][len(groups[n-1])-1]
		first := groups[n][0]
		if f.toks[first.start].line-f.toks[last.end].line-1 == r.betweenGroups {
			continue
		}
		indentation := f.indentation(open)
		f.beginChangeset()
		f.cmAAddNewline(last.end)
		f.slvlcRemoveBetween(last.end, first.start)
		f.cmAAddEmptyLines(last.end, r.betweenGroups, indentation)
		f.addBefore(first.start, indentation+" * ")
		f.endChangeset()
	}
}

func cmASameAnnotations(a, b []*cmAAnnotation) bool {
	set := func(list []*cmAAnnotation) map[int]bool {
		out := map[int]bool{}
		for _, x := range list {
			out[x.start] = true
		}
		return out
	}
	return maps.Equal(set(a), set(b))
}

func (r SlevomatDocCommentSpacing) checkGroupsOrder(f *slvFile, open int, groups [][]*cmAAnnotation, annotations []*cmAAnnotation) {
	sorted := r.sortToGroups(annotations)
	incorrect := false
	positions := map[int]int{}
	fix := false
	var undefined []int
	for position, group := range groups {
		matched := false
		for sortedPosition, sortedGroup := range sorted {
			if cmASameAnnotations(group, sortedGroup) {
				positions[position] = sortedPosition
				matched = true
				break
			}
			if !r.isGroupDefined(group) {
				undefined = append(undefined, position)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		incorrect = true
		fix = true
	}

	if len(positions) == 0 && len(undefined) > 1 {
		incorrect = true
		fix = true
	}

	if !incorrect {
		for _, position := range undefined {
			next := 0
			for _, v := range positions {
				next = max(next, v)
			}
			positions[position] = next + 1
		}
		keys := slices.Sorted(maps.Keys(positions))
		order := make([]int, len(keys))
		for n := range order {
			order[n] = n
		}
		sort.SliceStable(order, func(i, j int) bool { return positions[keys[order[i]]] < positions[keys[order[j]]] })
		for n, o := range order {
			if n != o {
				fix = true
				break
			}
		}
	}

	for position, group := range groups {
		sortedPosition, ok := positions[position]
		if !ok || sortedPosition >= len(sorted) {
			continue
		}
		sortedGroup := sorted[sortedPosition]
		for n, a := range group {
			if n >= len(sortedGroup) || a != sortedGroup[n] {
				fix = true
				break
			}
		}
	}

	if !fix {
		return
	}

	first := groups[0][0]
	lastGroup := groups[len(groups)-1]
	last := lastGroup[len(lastGroup)-1]
	indentation := f.indentation(open)

	var b strings.Builder
	for n, sortedGroup := range sorted {
		if n > 0 {
			for range r.betweenGroups {
				fmt.Fprintf(&b, "%s *%s", indentation, f.eol)
			}
		}
		for _, a := range sortedGroup {
			fmt.Fprintf(&b, "%s * %s%s", indentation, strings.TrimSpace(f.content(a.start, a.end)), f.eol)
		}
	}
	fixed := b.String()

	closer := f.docCloser(open)
	eolBefore := -1
	for i := first.start - 1; i >= open; i-- {
		if f.toks[i].doc == slvDocWhitespace && f.toks[i].content == f.eol {
			eolBefore = i
			break
		}
	}
	contentEnd := last.end
	for i := last.end + 1; i < closer; i++ {
		if f.toks[i].doc == slvDocWhitespace && f.toks[i].content == f.eol {
			contentEnd = i
			break
		}
	}

	f.beginChangeset()
	if eolBefore < 0 {
		f.change(open, contentEnd, "/**"+f.eol+fixed)
	} else {
		f.change(eolBefore+1, contentEnd, fixed)
	}
	f.endChangeset()
}

func (r SlevomatDocCommentSpacing) isGroupDefined(group []*cmAAnnotation) bool {
	for _, a := range group {
		for _, names := range r.groups {
			for _, name := range names {
				if cmAIsMatched(a.name, name) {
					return true
				}
			}
		}
	}
	return false
}

func (r SlevomatDocCommentSpacing) sortToGroups(annotations []*cmAAnnotation) [][]*cmAAnnotation {
	byPosition := map[int][]*cmAAnnotation{}
	var notInAny []*cmAAnnotation
next:
	for _, a := range annotations {
		for position, names := range r.groups {
			for _, name := range names {
				if cmAIsMatched(a.name, name) {
					byPosition[position] = append(byPosition[position], a)
					continue next
				}
			}
		}
		notInAny = append(notInAny, a)
	}

	var out [][]*cmAAnnotation
	for _, position := range slices.Sorted(maps.Keys(byPosition)) {
		names := r.groups[position]
		expected := map[string]int{}
		for n, name := range names {
			expected[name] = n
		}
		order := func(name string) int {
			if o, ok := expected[name]; ok {
				return o
			}
			for n, expectedName := range names {
				if cmAInNamespace(expectedName, name) {
					return n
				}
			}
			return 0
		}
		group := byPosition[position]
		sort.SliceStable(group, func(i, j int) bool {
			if oi, oj := order(group[i].name), order(group[j].name); oi != oj {
				return oi < oj
			}
			return group[i].start < group[j].start
		})
		out = append(out, group)
	}
	if len(notInAny) > 0 {
		out = append(out, notInAny)
	}
	return out
}

func cmAInNamespace(namespace, name string) bool {
	if strings.HasSuffix(namespace, "*") && strings.HasPrefix(name, namespace[:len(namespace)-1]) {
		return true
	}
	if namespace == "" {
		return false
	}
	switch namespace[len(namespace)-1] {
	case '\\', '-', ':':
		return strings.HasPrefix(name, namespace)
	}
	return false
}

func cmAIsMatched(annotationName, name string) bool {
	return annotationName == name || cmAInNamespace(name, annotationName)
}

func (r SlevomatDocCommentSpacing) checkAfterLastContent(f *slvFile, open, closer, lastContentEnd int) {
	if cmALines(f, f.content(lastContentEnd+1, closer)) == r.afterLast {
		return
	}
	indentation := f.indentation(open)
	f.beginChangeset()
	f.slvlcRemoveBetween(lastContentEnd, closer)
	f.cmAAddNewline(lastContentEnd)
	f.cmAAddEmptyLines(lastContentEnd, r.afterLast, indentation)
	f.addBefore(closer, indentation+" ")
	f.endChangeset()
}
