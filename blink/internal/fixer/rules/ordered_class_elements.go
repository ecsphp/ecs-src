package rules

import (
	"sort"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedClassElementsFixer.php
//
// OrderedClassElements is a port of OrderedClassElementsFixer. The zero value
// uses PHP-CS-Fixer's default order.
type OrderedClassElements struct {
	order         []string
	sortAlpha     bool
	caseSensitive bool
}

func (OrderedClassElements) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\OrderedClassElementsFixer`
}

func (OrderedClassElements) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/OrderedClassElementsFixer.php"
}

var oceDefaultOrder = []string{
	"use_trait", "case", "constant_public", "constant_protected", "constant_private",
	"property_public", "property_protected", "property_private", "construct", "destruct",
	"magic", "phpunit", "method_public", "method_protected", "method_private",
}

// oceTypeHierarchy is TYPE_HIERARCHY: a type not in the order takes the
// position of its first listed parent.
var oceTypeHierarchy = []struct {
	name    string
	parents []string
}{
	{"use_trait", nil}, {"public", nil}, {"protected", nil}, {"private", nil},
	{"case", []string{"public"}},
	{"constant", nil},
	{"constant_public", []string{"constant", "public"}},
	{"constant_protected", []string{"constant", "protected"}},
	{"constant_private", []string{"constant", "private"}},
	{"property", nil},
	{"property_static", []string{"property"}},
	{"property_public", []string{"property", "public"}},
	{"property_protected", []string{"property", "protected"}},
	{"property_private", []string{"property", "private"}},
	{"property_public_abstract", []string{"property_abstract", "property_public"}},
	{"property_public_readonly", []string{"property_readonly", "property_public"}},
	{"property_protected_abstract", []string{"property_abstract", "property_protected"}},
	{"property_protected_readonly", []string{"property_readonly", "property_protected"}},
	{"property_private_readonly", []string{"property_readonly", "property_private"}},
	{"property_public_static", []string{"property_static", "property_public"}},
	{"property_protected_static", []string{"property_static", "property_protected"}},
	{"property_private_static", []string{"property_static", "property_private"}},
	{"method", nil},
	{"method_abstract", []string{"method"}},
	{"method_static", []string{"method"}},
	{"method_public", []string{"method", "public"}},
	{"method_protected", []string{"method", "protected"}},
	{"method_private", []string{"method", "private"}},
	{"method_public_abstract", []string{"method_abstract", "method_public"}},
	{"method_protected_abstract", []string{"method_abstract", "method_protected"}},
	{"method_private_abstract", []string{"method_abstract", "method_private"}},
	{"method_public_abstract_static", []string{"method_abstract", "method_static", "method_public"}},
	{"method_protected_abstract_static", []string{"method_abstract", "method_static", "method_protected"}},
	{"method_private_abstract_static", []string{"method_abstract", "method_static", "method_private"}},
	{"method_public_static", []string{"method_static", "method_public"}},
	{"method_protected_static", []string{"method_static", "method_protected"}},
	{"method_private_static", []string{"method_static", "method_private"}},
}

var oceSpecialTypes = map[string]bool{"construct": true, "destruct": true, "magic": true, "phpunit": true}

var ocePhpunitMethods = map[string]int{
	"setupbeforeclass": 1, "dosetupbeforeclass": 2, "teardownafterclass": 3, "doteardownafterclass": 4,
	"setup": 5, "dosetup": 6, "assertpreconditions": 7, "assertpostconditions": 8, "teardown": 9, "doteardown": 10,
}

func (f OrderedClassElements) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := singleSpaceStringList(config["order"]); ok {
		f.order = append([]string(nil), list...)
	}
	if v, ok := config["sort_algorithm"].(string); ok {
		f.sortAlpha = v == "alpha"
	}
	if v, ok := config["case_sensitive"].(bool); ok {
		f.caseSensitive = v
	}
	return f
}

// typePositions is configurePostNormalisation.
func (f OrderedClassElements) typePositions() map[string]int {
	order := f.order
	if order == nil {
		order = oceDefaultOrder
	}
	pos := map[string]int{}
	for i, t := range order {
		pos[t] = i
	}
	unset := map[string]bool{}
	for _, h := range oceTypeHierarchy {
		if _, ok := pos[h.name]; ok {
			continue
		}
		found := false
		for _, parent := range h.parents {
			if p, ok := pos[parent]; ok && !unset[parent] {
				pos[h.name] = p
				found = true
				break
			}
		}
		if !found {
			pos[h.name] = 0
			unset[h.name] = true
		}
	}
	last := len(order)
	for k, p := range pos {
		if unset[k] {
			p = last
		}
		pos[k] = p * 10 // the last digit orders the phpunit methods
	}
	return pos
}

type oceElement struct {
	start, end                 int
	visibility                 string
	abstract, static, readonly bool
	typ, name                  string
}

func (f OrderedClassElements) Fix(s *tokens.Stream) bool {
	changed := false
	positions := f.typePositions()
	for i := 1; i < s.Len(); i++ {
		if !cdIsClassy(s, i) {
			continue
		}
		open := nextPunctOfKind(s, i, "{")
		if open < 0 {
			break
		}
		i = open
		elements := oceElements(s, open)
		if len(elements) == 0 {
			continue
		}
		end := elements[len(elements)-1].end
		sorted := f.sortElements(elements, positions)
		same := true
		for k := range sorted {
			if sorted[k].start != elements[k].start {
				same = false
				break
			}
		}
		if !same {
			var repl []token.Token
			for _, e := range sorted {
				repl = append(repl, s.Tokens()[e.start:e.end+1]...)
			}
			s.ReplaceRange(open+1, end, repl)
			changed = true
		}
		i = end
	}
	return changed
}

// oceElements is getElements for the body opened at open.
func oceElements(s *tokens.Stream, open int) []oceElement {
	var elements []oceElement
	start := open + 1
	for {
		e := oceElement{start: start, visibility: "public"}
		i := start
		for ; ; i++ {
			if i >= s.Len() {
				return elements
			}
			t := s.At(i)
			if t.Kind == token.AttributeStart {
				if end := s.AttributeEnd(i); end > i {
					i = end
				}
				continue
			}
			if isPunctVal(s, i, "}") {
				return elements
			}
			switch {
			case kwIs(t, "abstract"):
				e.abstract = true
				continue
			case kwIs(t, "static"):
				e.static = true
				continue
			case kwIs(t, "readonly"):
				e.readonly = true
			case kwIs(t, "protected"), kwIs(t, "private"):
				// "private(set)" is a token of its own: not a visibility here
				e.visibility = strings.ToLower(t.Value)
				continue
			}
			if !kwIs(t, "use") && !kwIs(t, "case") && !kwIs(t, "const") && t.Kind != token.Variable && !kwIs(t, "function") {
				continue
			}
			break
		}
		e.typ = oceElementType(s, i)
		switch e.typ {
		case "phpunit":
			e.name = strings.ToLower(s.At(sigNext(s, i)).Value)
		case "property":
			e.name = s.At(i).Value
		case "constant":
			if j := nextPunctOfKind(s, i, "="); j >= 0 {
				e.name = s.At(sigPrev(s, j)).Value
			}
		default:
			if n := sigNext(s, i); n >= 0 {
				e.name = s.At(n).Value
			}
		}
		e.end = oceElementEnd(s, i)
		if e.end < 0 {
			return elements
		}
		elements = append(elements, e)
		start = e.end + 1
	}
}

// oceElementType is detectElementType for the use/case/const/variable/function
// token at i.
func oceElementType(s *tokens.Stream, i int) string {
	t := s.At(i)
	switch {
	case kwIs(t, "use"):
		return "use_trait"
	case kwIs(t, "case"):
		return "case"
	case kwIs(t, "const"):
		return "constant"
	case t.Kind == token.Variable:
		return "property"
	}
	name := ""
	if n := sigNext(s, i); n >= 0 {
		if isPunctVal(s, n, "&") {
			// a by-reference method: PHP-CS-Fixer reads "&" as the name
			return "method"
		}
		name = s.At(n).Value
	}
	switch strings.ToLower(name) {
	case "__construct":
		return "construct"
	case "__destruct":
		return "destruct"
	}
	if _, ok := ocePhpunitMethods[strings.ToLower(name)]; ok {
		return "phpunit"
	}
	if strings.HasPrefix(name, "__") {
		return "magic"
	}
	return "method"
}

// oceElementEnd is findElementEnd.
func oceElementEnd(s *tokens.Stream, i int) int {
	index := nextPunctOfKind(s, i, "(", "{", ";")
	if index < 0 {
		return -1
	}
	if isPunctVal(s, index, "(") {
		if index = s.MatchForward(index); index < 0 {
			return -1
		}
		if index = nextPunctOfKind(s, index, "{", ";"); index < 0 {
			return -1
		}
	}
	if isPunctVal(s, index, "{") {
		if index = s.MatchForward(index); index < 0 {
			return -1
		}
	}
	for index++; index < s.Len(); index++ {
		t := s.At(index)
		if (t.Kind == token.Whitespace && !strings.ContainsAny(t.Value, "\n\r")) || isComment(t) {
			continue
		}
		break
	}
	index--
	if s.At(index).Kind == token.Whitespace {
		return index - 1
	}
	return index
}

// sortElements is sortElements/getTypePosition/sortGroupElements.
func (f OrderedClassElements) sortElements(elements []oceElement, positions map[string]int) []oceElement {
	sorted := append([]oceElement(nil), elements...)
	sort.SliceStable(sorted, func(a, b int) bool {
		ea, eb := sorted[a], sorted[b]
		if pa, pb := f.typePosition(ea, positions), f.typePosition(eb, positions); pa != pb {
			return pa < pb
		}
		if f.sortAlpha {
			if f.caseSensitive {
				return ea.name < eb.name
			}
			return strings.ToLower(ea.name) < strings.ToLower(eb.name)
		}
		return ea.start < eb.start
	})
	return sorted
}

func (f OrderedClassElements) typePosition(e oceElement, positions map[string]int) int {
	typ := e.typ
	if typ == "method" || typ == "magic" || typ == "phpunit" {
		if p, ok := positions["method:"+e.name]; ok {
			return p
		}
	}
	if oceSpecialTypes[typ] {
		if p, ok := positions[typ]; ok {
			if typ == "phpunit" {
				p += ocePhpunitMethods[e.name]
			}
			return p
		}
		typ = "method"
	}
	if typ == "constant" || typ == "property" || typ == "method" {
		typ += "_" + e.visibility
		if e.abstract {
			typ += "_abstract"
		}
		if e.static {
			typ += "_static"
		}
		if e.readonly {
			typ += "_readonly"
		}
	}
	return positions[typ]
}

// memberSpanEnd returns the last token index of the member starting at m: the
// terminating ";" for properties/consts/trait-use, or the matching "}" of a
// method body. Nested (), [] and {} groups are skipped. Returns -1 on trouble.
func memberSpanEnd(s *tokens.Stream, m int) int {
	for k := m; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			mm := s.MatchForward(k)
			if mm < 0 {
				return -1
			}
			k = mm
		case "{":
			return s.MatchForward(k)
		case ";":
			return k
		case "}":
			return -1
		}
	}
	return -1
}
