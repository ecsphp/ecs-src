package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModifierKeywordsFixer.php
//
// ModifierKeywords gives class constants, properties and methods an explicit
// visibility and orders the modifier keywords canonically:
// inheritance (abstract/final), visibility, set-visibility, scope (static),
// mutation (readonly), then the type declaration and name.
//
// The `elements` option limits which member kinds (const, property, method) are
// fixed; nil fixes all three, matching the PHP-CS-Fixer default.
type ModifierKeywords struct {
	// elements is the set of enabled member kinds; nil means all three.
	elements map[string]bool
}

func (f ModifierKeywords) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := arrayNotationStringList(config["elements"]); ok {
		set := make(map[string]bool, len(list))
		for _, item := range list {
			set[item] = true
		}
		f.elements = set
	}
	return f
}

func (f ModifierKeywords) enabled(kind string) bool {
	if f.elements == nil {
		return true
	}
	return f.elements[kind]
}

func (ModifierKeywords) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ModifierKeywordsFixer`
}

func (ModifierKeywords) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModifierKeywordsFixer.php"
}

func (f ModifierKeywords) Fix(s *tokens.Stream) bool {
	elements := classyElements(s)
	changed := false
	for _, e := range slices.Backward(elements) {
		kind := e.kind
		if kind == "promoted_property" {
			kind = "property"
		}
		if (kind != "const" && kind != "method" && kind != "property") || !f.enabled(kind) {
			continue
		}
		if f.fixElement(s, e) {
			changed = true
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// classyElement is one entry of PHP-CS-Fixer's TokensAnalyzer::getClassyElements:
// the index of a member's defining token (the variable of a property, the
// "function" of a method, the "const", "use" or "case" keyword) and its kind.
type classyElement struct {
	index int
	kind  string // const, method, property, promoted_property, trait_import, case
}

// classyElements lists the members of every classy body in source order, like
// getClassyElements, plus the promoted properties of each constructor.
func classyElements(s *tokens.Stream) []classyElement {
	var out []classyElement
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Keyword || !classLikeKeywords[strings.ToLower(t.Value)] {
			continue
		}
		open := classyBodyOpen(s, i)
		if open < 0 {
			continue
		}
		out = append(out, classBodyElements(s, open)...)
	}
	slices.SortFunc(out, func(a, b classyElement) int { return a.index - b.index })
	return out
}

// classyBodyOpen returns the "{" opening the body of the classy declared by the
// keyword at kw, skipping the arguments of an anonymous class, or -1.
func classyBodyOpen(s *tokens.Stream, kw int) int {
	for j := kw + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(":
			if m := s.MatchForward(j); m > j {
				j = m
			}
		case "{":
			return j
		case ";", ")", "}":
			return -1
		}
	}
	return -1
}

func classBodyElements(s *tokens.Stream, open int) []classyElement {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var out []classyElement
	curly, parens := 1, 0
	for k := open + 1; k < closeIdx; k++ {
		t := s.At(k)
		switch t.Kind {
		case token.Punct:
			switch t.Value {
			case "(":
				parens++
			case ")":
				parens--
			case "{":
				curly++
			case "}":
				curly--
			}
			continue
		case token.AttributeStart:
			if curly == 1 {
				if end := s.AttributeEnd(k); end > k {
					k = end
				}
			}
			continue
		}
		if curly != 1 {
			continue
		}
		if t.Kind == token.Variable && parens == 0 {
			out = append(out, classyElement{k, "property"})
			continue
		}
		if t.Kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.Value) {
		case "function":
			name := nextMeaningfulIndex(s, k)
			if name < 0 || isPunctVal(s, name, "(") {
				continue
			}
			out = append(out, classyElement{k, "method"})
			if isPunctVal(s, name, "&") {
				name = nextMeaningfulIndex(s, name)
			}
			if name >= 0 && strings.EqualFold(s.At(name).Value, "__construct") {
				out = append(out, promotedProperties(s, nextMeaningfulIndex(s, name))...)
			}
		case "const":
			out = append(out, classyElement{k, "const"})
		case "use":
			out = append(out, classyElement{k, "trait_import"})
		case "case":
			out = append(out, classyElement{k, "case"})
		}
	}
	return out
}

// promotedProperties returns the variable of each constructor parameter, in
// the list opened at open, that has a promotion modifier.
func promotedProperties(s *tokens.Stream, open int) []classyElement {
	if !isPunctVal(s, open, "(") {
		return nil
	}
	closeIdx := s.MatchForward(open)
	var out []classyElement
	seen := map[int]bool{}
	for k := open + 1; k < closeIdx; k++ {
		t := s.At(k)
		if t.Kind != token.Keyword {
			continue
		}
		switch lw := strings.ToLower(t.Value); {
		case visibilityModifiers[lw], isSetVisibility(lw), lw == "readonly", lw == "final":
		default:
			continue
		}
		for v := k + 1; v < s.Len(); v++ {
			if s.At(v).Kind == token.Variable {
				if !seen[v] {
					seen[v] = true
					out = append(out, classyElement{v, "promoted_property"})
				}
				break
			}
		}
	}
	return out
}

// isSetVisibility reports whether the lowercased keyword is one of PHP 8.4's
// set-visibility modifiers (T_PUBLIC_SET, T_PROTECTED_SET, T_PRIVATE_SET).
func isSetVisibility(lw string) bool {
	return lw == "public(set)" || lw == "protected(set)" || lw == "private(set)"
}

// modifierRole classifies the token at j while walking back from a member, as
// ModifierKeywordsFixer's EXPECTED_KINDS do: "absfinal", "static", "readonly",
// "set", "visibility", "type" (properties only), "ref" (a by-reference "&"),
// or "" when the modifier run ends.
func modifierRole(s *tokens.Stream, j int, property bool) string {
	t := s.At(j)
	switch t.Kind {
	case token.Keyword:
		switch lw := strings.ToLower(t.Value); {
		case lw == "abstract" || lw == "final":
			return "absfinal"
		case lw == "static":
			return "static"
		case lw == "readonly":
			return "readonly"
		case isSetVisibility(lw):
			return "set"
		case visibilityModifiers[lw] || lw == "var":
			return "visibility"
		case lw == "array" && property:
			return "type"
		}
	case token.Ident:
		if property {
			return "type"
		}
	case token.Punct:
		switch t.Value {
		case "&":
			if n := nextMeaningfulIndex(s, j); property && n >= 0 && s.At(n).Kind != token.Variable && !isPunctVal(s, n, "...") {
				return "type" // an intersection type
			}
			return "ref"
		case `\`, "?", "|":
			if property {
				return "type"
			}
		case "(", ")":
			if property && isDNFTypeParen(s, j) {
				return "type"
			}
		}
	}
	return ""
}

// isDNFTypeParen reports whether the "(" or ")" at j delimits the conjunction
// of a DNF type, "(A&B)|C" or "C|(A&B)".
func isDNFTypeParen(s *tokens.Stream, j int) bool {
	open, closeIdx := j, j
	if s.At(j).Value == "(" {
		closeIdx = s.MatchForward(j)
	} else {
		open = s.MatchBackward(j)
	}
	if open < 0 || closeIdx < 0 {
		return false
	}
	for k := open + 1; k < closeIdx; k++ {
		switch t := s.At(k); t.Kind {
		case token.Ident, token.Whitespace:
		case token.Punct:
			if t.Value != "&" && t.Value != `\` {
				return false
			}
		default:
			return false
		}
	}
	return isPunctVal(s, prevMeaningfulIndex(s, open), "|") || isPunctVal(s, nextMeaningfulIndex(s, closeIdx), "|")
}

// fixElement is ModifierKeywordsFixer::applyFix for one element.
func (ModifierKeywords) fixElement(s *tokens.Stream, e classyElement) bool {
	index := e.index
	property := e.kind == "property" || e.kind == "promoted_property"
	absFinal, visibility, visibilitySet, static, typeIdx, readOnly := -1, -1, -1, -1, -1, -1
	prev := prevMeaningfulIndex(s, index)
	for prev >= 0 {
		role := modifierRole(s, prev, property)
		if role == "" {
			break
		}
		switch role {
		case "absfinal":
			absFinal = prev
		case "static":
			static = prev
		case "readonly":
			readOnly = prev
		case "set":
			visibilitySet = prev
		case "type":
			typeIdx = prev
		case "visibility":
			visibility = prev
		}
		prev = prevMeaningfulIndex(s, prev)
	}
	if typeIdx >= 0 {
		index = typeIdx
	}
	if e.kind == "property" && isPunctVal(s, prev, ",") {
		return false
	}
	changed := false
	place := func(kw int) {
		if modifierPlacedProperly(s, kw, index) {
			index = kw
			return
		}
		moveModifierBefore(s, kw, index)
		changed = true
	}
	swap := static
	if swap < 0 {
		swap = readOnly
	}
	if swap >= 0 {
		place(swap)
	}
	if visibilitySet >= 0 {
		place(visibilitySet)
	}
	if visibility < 0 {
		s.InsertSliceAt(index, []token.Token{{Kind: token.Keyword, Value: "public"}, {Kind: token.Whitespace, Value: " "}})
		changed = true
	} else {
		if strings.EqualFold(s.At(visibility).Value, "var") {
			s.Set(visibility, token.Token{Kind: token.Keyword, Value: "public"})
			changed = true
		}
		place(visibility)
	}
	if absFinal >= 0 && !modifierPlacedProperly(s, absFinal, index) {
		moveModifierBefore(s, absFinal, index)
		changed = true
	}
	return changed
}

// modifierPlacedProperly is isKeywordPlacedProperly: the keyword at kw is
// followed by exactly one space and then the token at compared (or a "&" and
// then it).
func modifierPlacedProperly(s *tokens.Stream, kw, compared int) bool {
	if kw+1 >= s.Len() || s.At(kw+1).Value != " " {
		return false
	}
	return kw+2 == compared || (kw+3 == compared && isPunctVal(s, kw+2, "&"))
}

// moveModifierBefore is moveTokenAndEnsureSingleSpaceFollows: it inserts the
// keyword at from and a space before to, then clears the keyword and the
// whitespace after it (left as empty tokens until the fixer ends, as
// PHP-CS-Fixer's Tokens::clearAt does).
func moveModifierBefore(s *tokens.Stream, from, to int) {
	s.InsertSliceAt(to, []token.Token{s.At(from), {Kind: token.Whitespace, Value: " "}})
	s.Set(from, token.Token{Kind: token.Whitespace})
	if s.At(from+1).Kind == token.Whitespace {
		s.Set(from+1, token.Token{Kind: token.Whitespace})
	}
}
