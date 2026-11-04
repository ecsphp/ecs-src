package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php
//
// ClassDefinition is a port of ClassDefinitionFixer. Its fields are the fixer's
// options; their zero values are PHP-CS-Fixer's defaults, except
// inlineConstructorArguments, which the registry sets.
type ClassDefinition struct {
	singleLine                     bool
	singleItemSingleLine           bool
	multiLineExtendsEachSingleLine bool
	spaceBeforeParenthesis         bool
	inlineConstructorArguments     bool
}

func (f ClassDefinition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["single_line"].(bool); ok {
		f.singleLine = v
	}
	if v, ok := config["single_item_single_line"].(bool); ok {
		f.singleItemSingleLine = v
	}
	if v, ok := config["multi_line_extends_each_single_line"].(bool); ok {
		f.multiLineExtendsEachSingleLine = v
	}
	if v, ok := config["space_before_parenthesis"].(bool); ok {
		f.spaceBeforeParenthesis = v
	}
	if v, ok := config["inline_constructor_arguments"].(bool); ok {
		f.inlineConstructorArguments = v
	}
	return f
}

func (ClassDefinition) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ClassDefinitionFixer`
}

func (ClassDefinition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php"
}

// Fix follows applyFix: classy tokens right to left. Tokens are cleared (left
// empty) as PHP-CS-Fixer does, so the indices it computes stay valid, and the
// empty tokens are dropped at the end.
func (f ClassDefinition) Fix(s *tokens.Stream) bool {
	version, before := s.Version(), s.Render()
	for index := s.Len() - 4; index > 0; index-- {
		if cdIsClassy(s, index) {
			f.fixClassyDefinition(s, index)
		}
	}
	s.ClearEmptyTokens()
	return s.Version() != version && s.Render() != before
}

// cdIsClassy reports whether the token at i is a T_CLASS, T_INTERFACE, T_TRAIT
// or T_ENUM as PHP-CS-Fixer sees them: not "::class" nor "->class", and "enum"
// only when it declares one (PHP lexes it as a name otherwise).
func cdIsClassy(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "class", "interface", "trait":
	case "enum":
		if n := nextSignificantIndex(s, i); n < 0 || s.At(n).Kind != token.Ident {
			return false
		}
	default:
		return false
	}
	return !memberPrev(s, i) && !isNamedArgumentName(s, i)
}

// isNamedArgumentName reports whether the name at i (a reserved word
// included, "f(class: $c)") is the name of a named argument.
func isNamedArgumentName(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	if n < 0 || !isPunctVal(s, n, ":") {
		return false
	}
	p := sigPrev(s, i)
	return p >= 0 && (isPunctVal(s, p, "(") || isPunctVal(s, p, ","))
}

// anonClassNewIndex returns the index of the "new" keyword governing the
// anonymous class whose keyword is at classIdx, or -1 for a named declaration.
// prevMeaningfulIndex steps over a leading attribute ("new #[Attr] class").
func anonClassNewIndex(s *tokens.Stream, classIdx int) int {
	p := prevMeaningfulSkippingAttributes(s, classIdx)
	if p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "new") {
		return p
	}
	return -1
}

// cdIsAnonymousClass is TokensAnalyzer::isAnonymousClass.
func cdIsAnonymousClass(s *tokens.Stream, classy int) bool {
	if !strings.EqualFold(s.At(classy).Value, "class") {
		return false
	}
	p := prevMeaningfulSkippingAttributes(s, classy)
	if p >= 0 && kwIs(s.At(p), "readonly") {
		p = prevMeaningfulSkippingAttributes(s, p)
	}
	return p >= 0 && kwIs(s.At(p), "new")
}

type cdInheritance struct {
	start, count int
	multiLine    bool
}

type cdInfo struct {
	classy, open, start         int
	extends, implements         *cdInheritance
	anonymous                   bool
	final, abstract, readonlyAt int // -1 when absent
}

func (f ClassDefinition) fixClassyDefinition(s *tokens.Stream, classy int) {
	info, ok := cdDefinitionInfo(s, classy)
	if !ok {
		return
	}
	if info.implements != nil {
		f.fixInheritance(s, info.open, info.implements, true)
	}
	if info.extends != nil {
		open := info.open
		if info.implements != nil {
			open = info.implements.start
		}
		f.fixInheritance(s, open, info.extends, false)
	}
	info.open = f.fixOpenSpacing(s, info)
	var end int
	switch {
	case info.implements != nil:
		end = info.implements.start
	case info.extends != nil:
		end = info.extends.start
	default:
		end = prevSignificantIndex(s, info.open)
	}
	if info.anonymous && !f.inlineConstructorArguments {
		for (s.At(end).Kind != token.Punct || s.At(end).Value != ")") && !kwIs(s.At(end), "class") {
			// anonymous class with extends and/or implements
			start := sigPrev(s, end)
			f.makeSingleLine(s, start, end)
			end = start
		}
		if s.At(end).Kind == token.Punct && s.At(end).Value == ")" {
			// skip the constructor arguments of the anonymous class
			end = s.MatchBackward(end)
		}
	}
	f.makeSingleLine(s, info.start, end)
	cdSortModifiers(s, info)
}

func cdDefinitionInfo(s *tokens.Stream, classy int) (cdInfo, bool) {
	open := nextPunctOfKind(s, classy, "{")
	if open < 0 {
		return cdInfo{}, false
	}
	info := cdInfo{classy: classy, open: open, final: -1, abstract: -1, readonlyAt: -1}
	kw := strings.ToLower(s.At(classy).Value)
	if kw != "trait" {
		if e := cdFindKeyword(s, "extends", classy, open); e >= 0 {
			info.extends = cdInheritanceInfo(s, e)
		}
		if kw != "interface" {
			if im := cdFindKeyword(s, "implements", classy, open); im >= 0 {
				info.implements = cdInheritanceInfo(s, im)
			}
			info.anonymous = cdIsAnonymousClass(s, classy)
		}
	}
	if info.anonymous {
		info.start = classy
		for j := classy - 1; j >= 0; j-- {
			if kwIs(s.At(j), "new") {
				info.start = j
				break
			}
		}
		return info, true
	}
	info.start = classy
	for j := sigPrev(s, classy); j >= 0; j = sigPrev(s, j) {
		switch {
		case kwIs(s.At(j), "final"):
			info.final = j
		case kwIs(s.At(j), "abstract"):
			info.abstract = j
		case kwIs(s.At(j), "readonly"):
			info.readonlyAt = j
		default:
			return info, true
		}
		info.start = j
	}
	return info, true
}

func cdFindKeyword(s *tokens.Stream, kw string, from, to int) int {
	for j := from; j <= to; j++ {
		if kwIs(s.At(j), kw) {
			return j
		}
	}
	return -1
}

// cdInheritanceInfo is getClassyInheritanceInfo for the extends/implements
// keyword at start.
func cdInheritanceInfo(s *tokens.Stream, start int) *cdInheritance {
	info := &cdInheritance{start: start, count: 1}
	end := start + 1
	for ; end < s.Len(); end++ {
		t := s.At(end)
		if (t.Kind == token.Punct && t.Value == "{") || kwIs(t, "implements") || kwIs(t, "extends") {
			break
		}
	}
	if end < s.Len() && s.At(end).Value == "{" {
		end = prevSignificantIndex(s, end)
	}
	for i := start + 1; i < end; i++ {
		if t := s.At(i); t.Kind == token.Punct && t.Value == "," {
			info.count++
			continue
		}
		if !info.multiLine && strings.Contains(s.At(i).Value, "\n") {
			info.multiLine = true
		}
	}
	return info
}

// fixInheritance is fixClassyDefinitionImplements (implements) or
// fixClassyDefinitionExtends.
func (f ClassDefinition) fixInheritance(s *tokens.Stream, open int, info *cdInheritance, implements bool) {
	end := prevSignificantIndex(s, open)
	switch {
	case f.singleLine || !info.multiLine:
		f.makeSingleLine(s, info.start, end)
		info.multiLine = false
	case f.singleItemSingleLine && info.count == 1:
		f.makeSingleLine(s, info.start, end)
		info.multiLine = false
	case implements || f.multiLineExtendsEachSingleLine:
		f.makeInheritanceMultiLine(s, info.start, end)
		info.multiLine = true
	}
}

// fixOpenSpacing is fixClassyDefinitionOpenSpacing; it returns the index of
// the opening brace.
func (f ClassDefinition) fixOpenSpacing(s *tokens.Stream, info cdInfo) int {
	spacing := "\n"
	if info.anonymous {
		spacing = " "
		if info.implements != nil {
			if info.implements.multiLine {
				spacing = "\n"
			}
		} else if info.extends != nil && info.extends.multiLine {
			spacing = "\n"
		}
	}
	open := nextPunctOfKind(s, info.classy, "{")
	if spacing != " " && strings.Contains(s.At(open-1).Value, "\n") {
		return open
	}
	if s.At(open-1).Kind == token.Whitespace {
		if spacing != " " || !isComment(s.At(prevSignificantIndex(s, open-1))) {
			s.Set(open-1, token.Token{Kind: token.Whitespace, Value: spacing})
		}
		return open
	}
	s.InsertAt(open, token.Token{Kind: token.Whitespace, Value: spacing})
	return open + 1
}

func cdIsLineComment(t token.Token) bool {
	return strings.HasPrefix(t.Value, "//") || strings.HasPrefix(t.Value, "#")
}

// makeSingleLine is makeClassyDefinitionSingleLine over [start, end].
func (f ClassDefinition) makeSingleLine(s *tokens.Stream, start, end int) {
	space := token.Token{Kind: token.Whitespace, Value: " "}
	for i := end; i >= start; i-- {
		t := s.At(i)
		if t.Kind == token.Whitespace {
			prev, next := s.At(i-1), s.At(i+1)
			if strings.Contains(t.Value, "\n") {
				if prev.Kind == token.AttributeEnd || next.Kind == token.AttributeStart {
					continue
				}
				if (isComment(prev) && strings.HasSuffix(prev.Value, "]")) || (isComment(next) && strings.HasPrefix(next.Value, "#[")) {
					continue
				}
				if prev.Kind == token.DocComment || next.Kind == token.DocComment {
					continue
				}
			}
			if isComment(prev) {
				if !cdIsLineComment(prev) {
					s.Set(i, space)
				}
				continue
			}
			if isComment(next) {
				if !strings.HasPrefix(next.Value, "//") {
					s.Set(i, space)
				}
				continue
			}
			if kwIs(prev, "class") && next.Kind == token.Punct && next.Value == "(" {
				if f.spaceBeforeParenthesis {
					s.Set(i, space)
				} else {
					clearTokenAt(s, i)
				}
				continue
			}
			if ((prev.Kind != token.Punct || prev.Value != ",") && next.Kind == token.Punct && (next.Value == "," || next.Value == ")")) ||
				(prev.Kind == token.Punct && prev.Value == "(") {
				clearTokenAt(s, i)
				continue
			}
			s.Set(i, space)
			continue
		}
		if t.Kind == token.Punct && t.Value == "," && s.At(i+1).Kind != token.Whitespace {
			s.InsertAt(i+1, space)
			continue
		}
		if f.spaceBeforeParenthesis && kwIs(t, "class") && s.At(i+1).Kind != token.Whitespace {
			s.InsertAt(i+1, space)
			continue
		}
		if !isComment(t) {
			continue
		}
		if n := s.At(i + 1); n.Kind != token.Whitespace && !isComment(n) && !strings.Contains(t.Value, "\n") {
			s.InsertAt(i+1, space)
		}
		if p := s.At(i - 1); p.Kind != token.Whitespace && !isComment(p) {
			s.InsertAt(i, space)
		}
	}
}

// makeInheritanceMultiLine is makeClassyInheritancePartMultiLine over
// [start, end]: each item of the list on a line of its own.
func (f ClassDefinition) makeInheritanceMultiLine(s *tokens.Stream, start, end int) {
	for i := end; i > start; i-- {
		prev := -1
		for j := i - 1; j >= 0; j-- {
			if t := s.At(j); (t.Kind == token.Punct && t.Value == ",") || kwIs(t, "implements") || kwIs(t, "extends") {
				prev = j
				break
			}
		}
		if prev < 0 {
			return
		}
		breakAt := sigNext(s, prev)
		f.makeSingleLine(s, breakAt, i)
		onOwnLine := false
		for j := breakAt; j > prev; j-- {
			if strings.Contains(s.At(j).Value, "\n") {
				onOwnLine = true
				break
			}
		}
		if !onOwnLine {
			ws := token.Token{Kind: token.Whitespace, Value: "\n    "}
			if s.At(breakAt-1).Kind == token.Whitespace {
				s.Set(breakAt-1, ws)
			} else {
				s.InsertAt(breakAt, ws)
			}
		}
		i = prev + 1
	}
}

// cdSortModifiers is sortClassModifiers: "readonly" goes after "final" or
// "abstract".
func cdSortModifiers(s *tokens.Stream, info cdInfo) {
	if info.readonlyAt < 0 {
		return
	}
	for _, m := range []int{info.final, info.abstract} {
		if m < info.readonlyAt {
			continue
		}
		ro, acc := s.At(info.readonlyAt), s.At(m)
		s.Set(info.readonlyAt, acc)
		s.Set(m, ro)
		return
	}
}
