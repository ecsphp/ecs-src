package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
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

// Fix only ever ADJUSTS the newline count of the whitespace around classified
// members, inserting a newline where none separates a member from the docblock
// or attribute above it; it never removes or moves code. It covers three shapes
// and skips everything else:
//   - a member right below a docblock or an attribute: exactly one line break
//     between them, also for the first member;
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
		// last member first, as PHP-CS-Fixer does: a newline inserted above a
		// member only moves the members below it
		for k, m := range slices.Backward(starts) {
			start, fixed := f.fixMetaLineBreak(s, m)
			if fixed {
				changed = true
			}
			if k > 0 && f.fixMemberGap(s, starts[k-1], start) {
				changed = true
			}
		}
	}
	return changed
}

// fixMetaLineBreak leaves exactly one line break between the member at start and
// a docblock or attribute right above it, and returns the member's new start.
func (f ClassAttributesSeparation) fixMetaLineBreak(s *tokens.Stream, start int) (int, bool) {
	above := start - 1
	if above >= 0 && s.At(above).Kind == token.Whitespace {
		above--
	}
	if above < 0 || s.At(above).Kind != token.DocComment && s.At(above).Kind != token.AttributeEnd {
		return start, false
	}
	if _, ok := f.classSepSpacingFor(classSepMemberType(s, start)); !ok {
		return start, false
	}
	if above+1 == start {
		s.InsertAt(start, token.Token{Kind: token.Whitespace, Value: "\n"})
		return start + 1, true
	}
	return start, setNewlineCount(s, above+1, 1)
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
		switch t.Kind {
		case token.Whitespace:
			wsCount++
			wsIdx = j
		case token.DocComment, token.Comment:
			// a docblock or a line/block comment leading the next member is its
			// trivia: the blank line goes above the block. A trailing comment on the
			// previous member's line is rejected by the guards below (its whitespace
			// has no newline), so it is still left alone.
			if firstTrivia < 0 {
				firstTrivia = j
			}
		case token.AttributeStart:
			// an attribute (#[...]) leading the next member is trivia too; skip its
			// inner tokens so the blank line goes above the whole attribute block
			if firstTrivia < 0 {
				firstTrivia = j
			}
			if end := s.AttributeEnd(j); end > j {
				j = end
			}
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
	// the block itself attached to nextStart as PHP-CS-Fixer's findCommentBlockStart
	// walks it: no blank line between its items (whatever an attribute holds), and
	// one line break right above nextStart.
	if firstTrivia != prevEnd+2 {
		return false
	}
	ws0 := prevEnd + 1
	if s.At(ws0).Kind != token.Whitespace || !hasNewline(s.At(ws0).Value) {
		return false
	}
	for j := firstTrivia; j < nextStart; j++ {
		t := s.At(j)
		switch t.Kind {
		case token.AttributeStart:
			if end := s.AttributeEnd(j); end > j {
				j = end
			}
		case token.Whitespace:
			if n := strings.Count(t.Value, "\n"); n > 1 || n == 0 && j == nextStart-1 {
				return false
			}
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
		}
		// otherwise a keyword type ("array", "int", "string", ...) opens a typed
		// property; fall through to the variable scan below
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
	return s.At(j).Kind == token.DocComment || s.AttributeStartOf(j) >= 0
}

// setNewlineCount makes the whitespace token at idx hold req newlines, as
// PHP-CS-Fixer's correctLineBreaks does: a token that already has req is left
// as it is (even a blank line holding spaces), a missing newline is prepended,
// and surplus line breaks are dropped from the start. Returns whether it
// changed the value.
func setNewlineCount(s *tokens.Stream, idx, req int) bool {
	v := s.At(idx).Value
	have := strings.Count(v, "\n")
	switch {
	case have == req:
		return false
	case have < req:
		s.SetValue(idx, strings.Repeat("\n", req-have)+v)
	default:
		s.SetValue(idx, dropLineBreaks(v, have-req))
	}
	return true
}

// dropLineBreaks removes the first n line breaks ("\r\n" or "\n") from v.
func dropLineBreaks(v string, n int) string {
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); i++ {
		if n > 0 && v[i] == '\n' {
			n--
			continue
		}
		if n > 0 && v[i] == '\r' && i+1 < len(v) && v[i+1] == '\n' {
			n--
			i++
			continue
		}
		b.WriteByte(v[i])
	}
	return b.String()
}
