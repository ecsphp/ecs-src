package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseConstantSniff.php
//
// LowerCaseConstant lowercases true, false and null ("TRUE" -> "true"). Type
// declarations are left alone; parameter default values are not.
type LowerCaseConstant struct{}

func (LowerCaseConstant) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\LowerCaseConstantSniff`
}

func (LowerCaseConstant) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/LowerCaseConstantSniff.php"
}

func (LowerCaseConstant) Fix(s *tokens.Stream) bool {
	return fixConstantCase(s, lowerASCII)
}

// constantCaseModifiers precede a property type; the type is skipped
var constantCaseModifiers = map[string]bool{
	"public": true, "private": true, "protected": true,
	"public(set)": true, "private(set)": true, "protected(set)": true,
	"var": true, "static": true, "readonly": true, "final": true, "abstract": true,
}

// fixConstantCase is the shared LowerCaseConstantSniff walk: it recases every
// true/false/null outside type declarations with recase.
func fixConstantCase(s *tokens.Stream, recase func(string) string) bool {
	changed := false
	apply := func(i int) {
		if !isConstantKeyword(s, i) {
			return
		}
		if v := recase(s.At(i).Value); v != s.At(i).Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Keyword {
			lower := lowerASCII(t.Value)
			switch {
			case lower == "const" && !afterUse(s, i):
				// a constant always has a value: skip its type up to "="
				for j := i + 1; j < s.Len(); j++ {
					if s.At(j).Kind == token.Punct && s.At(j).Value == "=" {
						i = j - 1
						break
					}
				}
			case constantCaseModifiers[lower]:
				j := i + 1
				for j < s.Len() && (isPhpcsEmpty(s.At(j)) || isPropertyTypeToken(s.At(j))) {
					j++
				}
				i = j - 1
			case (lower == "function" || lower == "fn") && !afterUse(s, i):
				if end, ok := functionDefaults(s, i, apply); ok {
					i = end - 1
				}
			}
			continue
		}
		apply(i)
	}
	return changed
}

func afterUse(s *tokens.Stream, i int) bool {
	prev, ok := prevSignificant(s, i)
	return ok && strings.EqualFold(prev.Value, "use")
}

func isPropertyTypeToken(t token.Token) bool {
	switch t.Kind {
	case token.Ident:
		return true
	case token.Keyword:
		v := lowerASCII(t.Value)
		return v == "callable" || v == "array" || v == "namespace"
	case token.Punct:
		switch t.Value {
		case `\`, "|", "&", "(", ")", "?":
			return true
		}
	}
	return false
}

// functionDefaults applies fn to the tokens of each parameter default value of
// the function at kw and returns the index where its declaration ends (the
// body "{", the ";" of a bodiless method, or an arrow function's "=>").
func functionDefaults(s *tokens.Stream, kw int, fn func(int)) (int, bool) {
	open := kw + 1
	for open < s.Len() && (isPhpcsEmpty(s.At(open)) || s.At(open).Value == "&" || s.At(open).Kind == token.Ident || s.At(open).Kind == token.Keyword) {
		open++
	}
	if open >= s.Len() || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return 0, false
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return 0, false
	}
	end := closeIdx
	arrow := strings.EqualFold(s.At(kw).Value, "fn")
	for j := closeIdx + 1; j < s.Len(); j++ {
		v := s.At(j).Value
		if s.At(j).Kind == token.Punct && ((arrow && v == "=>") || (!arrow && (v == "{" || v == ";"))) {
			end = j
			break
		}
	}
	depth := 0
	inDefault := false
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case "=":
				if depth == 0 {
					inDefault = true
				}
			case ",":
				if depth == 0 {
					inDefault = false
				}
			}
		}
		if inDefault {
			fn(j)
		}
	}
	return end, true
}

// isConstantKeyword reports whether the token at i is the true/false/null
// constant, not a name that happens to be spelled that way.
func isConstantKeyword(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Ident {
		return false
	}
	switch lowerASCII(t.Value) {
	case "true", "false", "null":
	default:
		return false
	}
	if i+1 < s.Len() && s.At(i+1).Value == `\` {
		return false
	}
	next := nextSignificantValue(s, i)
	if next == "::" || next == "(" {
		return false
	}
	// "\null" is the constant; "Writer\Null" is a qualified class name
	if i > 0 && s.At(i-1).Value == `\` && i > 1 && (s.At(i-2).Kind == token.Ident || s.At(i-2).Kind == token.Keyword) {
		return false
	}
	prev, ok := prevSignificant(s, i)
	if !ok {
		return next != ":"
	}
	switch {
	// an enum case name ("case Null;"), not a switch case ("case NULL:")
	case strings.EqualFold(prev.Value, "case") && (next == ";" || next == "="):
		return false
	// a goto label ("null:")
	case next == ":" && (prev.Value == ";" || prev.Value == "{" || prev.Value == "}"):
		return false
	}
	switch lowerASCII(prev.Value) {
	case "->", "?->", "::", "function", "const", "class", "interface", "trait", "enum",
		"extends", "implements", "new", "namespace", "use", "insteadof", "goto":
		return false
	}
	return true
}
