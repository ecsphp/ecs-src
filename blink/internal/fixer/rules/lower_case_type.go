package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseTypeSniff.php
//
// LowerCaseType lowercases casts and the native types in parameter, return,
// property and class constant declarations: "(INT) $a", "function (Int $a):
// VOID" -> "(int) $a", "function (int $a): void".
type LowerCaseType struct{}

func (LowerCaseType) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\LowerCaseTypeSniff`
}

func (LowerCaseType) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseTypeSniff.php"
}

func (LowerCaseType) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if _, typeIdx, ok := phpCastAt(s, i); ok {
			changed = lowerCaseToken(s, typeIdx) || changed
			continue
		}
		if t.Kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.Value) {
		case "function", "fn":
			changed = lowerCaseFunctionTypes(s, i) || changed
		case "class", "interface", "trait", "enum":
			changed = lowerCaseMemberTypes(s, i) || changed
		}
	}
	return changed
}

func lowerCaseToken(s *tokens.Stream, i int) bool {
	v := s.At(i).Value
	lower := strings.ToLower(v)
	if lower == v {
		return false
	}
	s.SetValue(i, lower)
	return true
}

// lowerCaseTypeRange lowercases each part of the type from start to end that
// is a single native type word, as LowerCaseTypeSniff::processUnionType does.
func lowerCaseTypeRange(s *tokens.Stream, start, end int) bool {
	changed := false
	part, count := -1, 0
	flush := func() {
		if count == 1 && nativeTypeNames[strings.ToLower(s.At(part).Value)] {
			changed = lowerCaseToken(s, part) || changed
		}
		part, count = -1, 0
	}
	for i := start; i <= end && i < s.Len(); i++ {
		t := s.At(i)
		if isPhpcsEmpty(t) {
			continue
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "|", "&", "(", ")", "?":
				flush()
				continue
			}
		}
		if part < 0 {
			part = i
		}
		count++
	}
	flush()
	return changed
}

func lowerCaseFunctionTypes(s *tokens.Stream, fn int) bool {
	open := phpcsNextNonEmpty(s, fn+1)
	if open < s.Len() && s.At(open).Value == "&" {
		open = phpcsNextNonEmpty(s, open+1)
	}
	if open < s.Len() && s.At(open).Kind == token.Ident {
		open = phpcsNextNonEmpty(s, open+1)
	}
	if open >= s.Len() || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return false
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	changed := false
	start := open + 1
	depth := 0
	for i := open + 1; i <= closeIdx; i++ {
		t := s.At(i)
		if t.Kind == token.AttributeStart {
			if e := s.AttributeEnd(i); e > i {
				i = e
				start = i + 1
			}
			continue
		}
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if i == closeIdx {
				changed = lowerCaseParamType(s, start, i) || changed
			} else {
				depth--
			}
		case ",":
			if depth == 0 {
				changed = lowerCaseParamType(s, start, i) || changed
				start = i + 1
			}
		}
	}
	j := phpcsNextNonEmpty(s, closeIdx+1)
	if j < s.Len() && s.At(j).Kind == token.Keyword && strings.EqualFold(s.At(j).Value, "use") {
		j = phpcsNextNonEmpty(s, j+1)
		if j >= s.Len() || s.At(j).Value != "(" {
			return changed
		}
		j = phpcsNextNonEmpty(s, s.MatchForward(j)+1)
	}
	if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != ":" {
		return changed
	}
	typeStart := j + 1
	end := typeStart
	for ; end < s.Len(); end++ {
		t := s.At(end)
		if t.Kind == token.Punct && (t.Value == "{" || t.Value == ";" || t.Value == "=>") {
			break
		}
	}
	return lowerCaseTypeRange(s, typeStart, end-1) || changed
}

// lowerCaseParamType handles the parameter from start up to (not including)
// end: its type runs from after the modifiers to before "&", "..." and the
// variable.
func lowerCaseParamType(s *tokens.Stream, start, end int) bool {
	v := -1
	for i := start; i < end; i++ {
		if s.At(i).Kind == token.Variable {
			v = i
			break
		}
	}
	if v < 0 {
		return false
	}
	typeStart := phpcsNextNonEmpty(s, start)
	for typeStart < v && s.At(typeStart).Kind == token.Keyword && isMemberModifier(s.At(typeStart).Value) {
		typeStart = phpcsNextNonEmpty(s, typeStart+1)
	}
	typeEnd := phpcsPrevNonEmpty(s, v-1)
	if typeEnd > typeStart && s.At(typeEnd).Value == "..." {
		typeEnd = phpcsPrevNonEmpty(s, typeEnd-1)
	}
	if typeEnd > typeStart && s.At(typeEnd).Value == "&" {
		typeEnd = phpcsPrevNonEmpty(s, typeEnd-1)
	}
	if typeEnd < typeStart {
		return false
	}
	return lowerCaseTypeRange(s, typeStart, typeEnd)
}

func isMemberModifier(v string) bool {
	switch strings.ToLower(v) {
	case "public", "protected", "private", "readonly", "final",
		"public(set)", "protected(set)", "private(set)":
		return true
	}
	return false
}

// lowerCaseMemberTypes handles the property and constant types declared
// directly in the body of the class-like whose keyword is at kw.
func lowerCaseMemberTypes(s *tokens.Stream, kw int) bool {
	if p := phpcsPrevNonEmpty(s, kw-1); p >= 0 && s.At(p).Value == "::" {
		return false
	}
	isEnum := strings.EqualFold(s.At(kw).Value, "enum")
	open := kw + 1
	for ; open < s.Len(); open++ {
		t := s.At(open)
		if t.Kind != token.Punct {
			continue
		}
		if t.Value == "(" {
			open = s.MatchForward(open)
			if open < 0 {
				return false
			}
			continue
		}
		if t.Value == ";" {
			return false
		}
		if t.Value == "{" {
			break
		}
	}
	if open >= s.Len() {
		return false
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	changed := false
	for i := open + 1; i < closeIdx; i++ {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == "{" || t.Value == "(" || t.Value == "[") {
			if m := s.MatchForward(i); m > i {
				i = m
			}
			continue
		}
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "const") {
			changed = lowerCaseConstType(s, i) || changed
			continue
		}
		if t.Kind == token.Variable && !isEnum {
			changed = lowerCasePropertyType(s, i) || changed
		}
	}
	return changed
}

func lowerCaseConstType(s *tokens.Stream, c int) bool {
	start := c + 1
	for start < s.Len() && (isPhpcsEmpty(s.At(start)) || s.At(start).Value == "?") {
		start++
	}
	assign := -1
	for i := start + 1; i < s.Len(); i++ {
		if v := s.At(i); v.Kind == token.Punct && (v.Value == "=" || v.Value == ";") {
			if v.Value == "=" {
				assign = i
			}
			break
		}
	}
	if assign < 0 {
		return false
	}
	name := phpcsPrevNonEmpty(s, assign-1)
	if name == start {
		return false
	}
	return lowerCaseTypeRange(s, start, phpcsPrevNonEmpty(s, name-1))
}

// lowerCasePropertyType walks back from the property variable at v over its
// type, up to the modifiers.
func lowerCasePropertyType(s *tokens.Stream, v int) bool {
	end := phpcsPrevNonEmpty(s, v-1)
	start := end + 1
	for i := end; i >= 0; i-- {
		t := s.At(i)
		if isPhpcsEmpty(t) {
			continue
		}
		if t.Kind == token.Ident || (t.Kind == token.Keyword && !isMemberModifier(t.Value) && !strings.EqualFold(t.Value, "static") && !strings.EqualFold(t.Value, "var")) {
			start = i
			continue
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "|", "&", "(", ")", "?", `\`:
				start = i
				continue
			}
		}
		break
	}
	if start > end {
		return false
	}
	return lowerCaseTypeRange(s, start, end)
}
