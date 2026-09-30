package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// class member kinds for the separation rule, mirroring PHP-CS-Fixer's element
// types (const|method|property|trait_import|case).
const (
	sepUnknown = iota
	sepConst
	sepMethod
	sepProperty
	sepTraitImport
	sepCase
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassAttributesSeparationFixer.php
//
// ClassAttributesSeparation normalizes the blank lines between two consecutive
// top-level class members per the fixer's DEFAULT config: one blank line between
// methods, properties and consts; none between trait imports or enum cases. It
// is conservative - see the caveats on Fix.
//
// elements mirrors the "elements" option (element -> one|none|only_if_meta); a
// nil value (the zero value) means the default map below. A member type absent
// from the map is left untouched, matching PHP-CS-Fixer.
type ClassAttributesSeparation struct {
	elements map[string]string
}

var classSepDefaultSpacing = map[string]string{
	"const": "one", "method": "one", "property": "one",
	"trait_import": "none", "case": "none",
}

func (f ClassAttributesSeparation) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["elements"].(map[string]any); ok {
		m := make(map[string]string, len(raw))
		for k, v := range raw {
			if sv, ok := v.(string); ok {
				m[k] = sv
			}
		}
		f.elements = m
	}
	return f
}

// classSepTypeName maps a sep* member kind to its option element name.
func classSepTypeName(sepType int) string {
	switch sepType {
	case sepConst:
		return "const"
	case sepMethod:
		return "method"
	case sepProperty:
		return "property"
	case sepTraitImport:
		return "trait_import"
	case sepCase:
		return "case"
	}
	return ""
}

// classSepSpacingFor returns the configured spacing for a member kind and whether
// that kind is targeted at all.
func (f ClassAttributesSeparation) classSepSpacingFor(sepType int) (string, bool) {
	name := classSepTypeName(sepType)
	if name == "" {
		return "", false
	}
	if f.elements == nil {
		return classSepDefaultSpacing[name], true
	}
	sp, ok := f.elements[name]
	return sp, ok
}

func (ClassAttributesSeparation) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ClassAttributesSeparationFixer`
}

func (ClassAttributesSeparation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassAttributesSeparationFixer.php"
}

// Fix only ever ADJUSTS the newline count inside an existing whitespace token
// between two classified members; it never inserts, removes or moves code. It
// covers two gap shapes and skips everything else:
//   - a plain whitespace gap (already spanning a line) between two members;
//   - a member whose leading trivia is only docblocks/attributes: the blank line
//     is placed above that block, leaving the block attached.
//
// Gaps that glue members on one line, contain plain comments, or hold a detached
// docblock are left untouched. The first member (after "{") and the last member's
// gap to "}" are left to the blank-line-after-opening / brace fixers.
func (f ClassAttributesSeparation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		if kind, _ := classifyBrace(s, i); kind != braceClassLike {
			continue
		}
		starts := classMemberStarts(s, i)
		for k := 0; k+1 < len(starts); k++ {
			if f.fixMemberGap(s, starts[k], starts[k+1]) {
				changed = true
			}
		}
	}
	return changed
}

// fixMemberGap normalizes the whitespace separating the member at prevStart from
// the one at nextStart. Only whitespace values change, so token indices stay
// stable across calls. Returns whether it changed anything.
func (f ClassAttributesSeparation) fixMemberGap(s *tokens.Stream, prevStart, nextStart int) bool {
	prevEnd := memberSpanEnd(s, prevStart)
	if prevEnd < 0 || prevEnd >= nextStart {
		return false
	}
	nextType := classSepMemberType(s, nextStart)
	if nextType == sepUnknown {
		return false
	}
	spacing, ok := f.classSepSpacingFor(nextType)
	if !ok {
		return false // member type not targeted by configuration
	}

	firstTrivia := -1
	wsCount, wsIdx := 0, -1
	for j := prevEnd + 1; j < nextStart; j++ {
		t := s.At(j)
		switch {
		case t.Kind == token.Whitespace:
			wsCount++
			wsIdx = j
		case t.Kind == token.DocComment || isAttribute(t):
			if firstTrivia < 0 {
				firstTrivia = j
			}
		case t.Kind == token.Comment:
			return false // plain comment: ambiguous ownership, leave alone
		default:
			return false
		}
	}

	if firstTrivia < 0 {
		// pure whitespace gap
		if wsCount != 1 || !hasNewline(s.At(wsIdx).Value) {
			return false
		}
		prevType := classSepMemberType(s, prevStart)
		return setNewlineCount(s, wsIdx, requiredNewlines(s, prevStart, prevType, nextType, spacing))
	}

	// docblock/attribute block leading the next member: blank line goes above it.
	// Require exactly one whitespace token (ws0) between prevEnd and the block, and
	// the block itself contiguous (single-newline whitespace throughout) so it is
	// unambiguously attached to nextStart.
	if firstTrivia != prevEnd+2 {
		return false
	}
	ws0 := prevEnd + 1
	if s.At(ws0).Kind != token.Whitespace || !hasNewline(s.At(ws0).Value) {
		return false
	}
	for j := firstTrivia + 1; j < nextStart; j++ {
		if t := s.At(j); t.Kind == token.Whitespace && strings.Count(t.Value, "\n") != 1 {
			return false
		}
	}
	return setNewlineCount(s, ws0, 2)
}

// requiredNewlines returns the number of newlines wanted in a plain whitespace
// gap above a member of nextType, following the configured spacing.
func requiredNewlines(s *tokens.Stream, prevStart, prevType, nextType int, spacing string) int {
	switch spacing {
	case "one":
		return 2
	case "only_if_meta":
		// no docblock/attribute directly above the next member in a plain gap
		return 1
	default: // "none"
		if prevType == nextType && !hasDocOrAttrAbove(s, prevStart) {
			return 1
		}
		return 2
	}
}

// classSepMemberType classifies the member starting at m as one of the sep*
// kinds, or sepUnknown when it cannot be told apart safely.
func classSepMemberType(s *tokens.Stream, m int) int {
	end := memberSpanEnd(s, m)
	if end < 0 {
		return sepUnknown
	}
	k := m
	for k <= end && s.At(k).Kind == token.Keyword && memberModifiers[strings.ToLower(s.At(k).Value)] {
		k = skipWhitespace(s, k+1)
	}
	if k > end {
		return sepUnknown
	}
	if s.At(k).Kind == token.Keyword {
		switch strings.ToLower(s.At(k).Value) {
		case "use":
			return sepTraitImport
		case "const":
			return sepConst
		case "function":
			return sepMethod
		case "case":
			return sepCase
		default:
			return sepUnknown
		}
	}
	for x := k; x <= end; x++ {
		t := s.At(x)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				if mm := s.MatchForward(x); mm > 0 {
					x = mm
					continue
				}
			}
		}
		if t.Kind == token.Variable {
			return sepProperty
		}
	}
	return sepUnknown
}

// hasDocOrAttrAbove reports whether the token directly above start (skipping
// whitespace) is a docblock or an attribute.
func hasDocOrAttrAbove(s *tokens.Stream, start int) bool {
	j := start - 1
	for j >= 0 && s.At(j).Kind == token.Whitespace {
		j--
	}
	if j < 0 {
		return false
	}
	t := s.At(j)
	return t.Kind == token.DocComment || isAttribute(t)
}

func isAttribute(t token.Token) bool {
	return t.Kind == token.Comment && strings.HasPrefix(t.Value, "#[")
}

// setNewlineCount rewrites the whitespace token at idx to hold exactly req
// newlines while keeping the trailing line indentation. Returns whether it
// changed the value.
func setNewlineCount(s *tokens.Stream, idx, req int) bool {
	v := s.At(idx).Value
	indent := ""
	if nl := strings.LastIndexByte(v, '\n'); nl >= 0 {
		indent = v[nl+1:]
	}
	want := strings.Repeat("\n", req) + indent
	if v == want {
		return false
	}
	s.SetValue(idx, want)
	return true
}
