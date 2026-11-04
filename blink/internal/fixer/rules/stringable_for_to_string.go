package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/StringableForToStringFixer.php
//
// StringableForToString makes a class that declares __toString() explicitly
// implement the Stringable interface, adding "\Stringable" to its implements
// clause (or a new clause) unless it already implements Stringable or an alias.
type StringableForToString struct{}

func (StringableForToString) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\StringableForToStringFixer`
}

func (StringableForToString) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/StringableForToStringFixer.php"
}

func (StringableForToString) Fix(s *tokens.Stream) bool {
	changed := false
	stringableInterfaces := []string{"stringable"}
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if kwIs(t, "namespace") {
			stringableInterfaces = nil
			continue
		}
		if kwIs(t, "use") {
			if name, ok := stringableUseName(s, i); ok {
				stringableInterfaces = append(stringableInterfaces, name)
			}
			continue
		}
		if !taIsClassKw(s, i) {
			continue
		}
		classOpen := nextPunctOfKind(s, i, "{")
		if classOpen < 0 {
			continue
		}
		classEnd := s.MatchForward(classOpen)
		if classEnd < 0 {
			continue
		}
		if !stringableHasToString(s, classOpen, classEnd) {
			continue
		}
		if stringableImplements(s, i, classOpen, stringableInterfaces) {
			continue
		}
		stringableAddInterface(s, i)
		changed = true
	}
	return changed
}

// stringableUseName returns the lowercased short name a "use" import binds, when
// it imports the global Stringable interface; ok is false otherwise.
func stringableUseName(s *tokens.Stream, useIndex int) (string, bool) {
	idx := sigNext(s, useIndex)
	if idx < 0 || isPunctVal(s, idx, "(") {
		return "", false // closure "use (...)"
	}
	if kwIs(s.At(idx), "function") || kwIs(s.At(idx), "const") {
		idx = sigNext(s, idx)
	}
	full, short := "", ""
	for idx >= 0 {
		t := s.At(idx)
		switch {
		case t.Kind == token.Ident:
			full += t.Value
			short = t.Value
		case isPunctVal(s, idx, `\`):
			full += `\`
		case kwIs(t, "as"):
			if a := sigNext(s, idx); a >= 0 && s.At(a).Kind == token.Ident {
				short = s.At(a).Value
			}
			idx = -1
			continue
		default:
			idx = -1
			continue
		}
		idx = sigNext(s, idx)
	}
	if lf := strings.ToLower(full); lf == "stringable" || lf == `\stringable` {
		return strings.ToLower(short), true
	}
	return "", false
}

// stringableHasToString reports whether the class body declares a __toString method.
func stringableHasToString(s *tokens.Stream, classStart, classEnd int) bool {
	for index := classStart + 1; index < classEnd; index++ {
		if isPunctVal(s, index, "{") {
			if end := s.MatchForward(index); end >= 0 {
				index = end
			}
			continue
		}
		if !kwIs(s.At(index), "function") {
			continue
		}
		if name := sigNext(s, index); name >= 0 && strings.EqualFold(s.At(name).Value, "__toString") {
			return true
		}
	}
	return false
}

// stringableImplements reports whether the class already implements the global
// Stringable interface or one of the known aliases.
func stringableImplements(s *tokens.Stream, classKw, classOpen int, aliases []string) bool {
	interfaces := stringableInterfaceList(s, classKw, classOpen)
	if len(interfaces) == 0 {
		return false
	}
	if slices.Contains(interfaces, `\stringable`) {
		return true
	}
	for _, alias := range aliases {
		if slices.Contains(interfaces, alias) {
			return true
		}
	}
	return false
}

// stringableInterfaceList returns the lowercased names in the class's implements
// clause, or nil when there is none.
func stringableInterfaceList(s *tokens.Stream, classKw, classOpen int) []string {
	impl := stringableImplementsIndex(s, classKw)
	if impl < 0 || !kwIs(s.At(impl), "implements") {
		return nil
	}
	var out []string
	cur := ""
	for idx := sigNext(s, impl); idx >= 0 && idx < classOpen; idx = sigNext(s, idx) {
		if isPunctVal(s, idx, ",") {
			out = append(out, strings.ToLower(cur))
			cur = ""
			continue
		}
		cur += s.At(idx).Value
	}
	if cur != "" {
		out = append(out, strings.ToLower(cur))
	}
	return out
}

// stringableImplementsIndex returns the index of the class's "implements" keyword
// or its body "{", whichever comes first.
func stringableImplementsIndex(s *tokens.Stream, classKw int) int {
	for i := classKw + 1; i < s.Len(); i++ {
		if kwIs(s.At(i), "implements") || isPunctVal(s, i, "{") {
			return i
		}
	}
	return -1
}

func stringableAddInterface(s *tokens.Stream, classKw int) {
	impl := stringableImplementsIndex(s, classKw)
	if impl < 0 {
		return
	}
	if isPunctVal(s, impl, "{") {
		prev := sigPrev(s, impl)
		if prev < 0 {
			return
		}
		s.InsertSliceAt(prev+1, []token.Token{
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Keyword, Value: "implements"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: `\`},
			{Kind: token.Ident, Value: "Stringable"},
		})
		return
	}
	after := sigNext(s, impl)
	if after < 0 {
		return
	}
	s.InsertSliceAt(after, []token.Token{
		{Kind: token.Punct, Value: `\`},
		{Kind: token.Ident, Value: "Stringable"},
		{Kind: token.Punct, Value: ","},
		{Kind: token.Whitespace, Value: " "},
	})
}
