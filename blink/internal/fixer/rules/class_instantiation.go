package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/ClassInstantiationSniff.php
//
// ClassInstantiation adds the missing parentheses to "new Foo" -> "new Foo()".
// Anonymous classes are left alone.
type ClassInstantiation struct{}

func (ClassInstantiation) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Classes\ClassInstantiationSniff`
}

func (ClassInstantiation) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/ClassInstantiationSniff.php"
}

func (ClassInstantiation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Keyword || !strings.EqualFold(t.Value, "new") {
			continue
		}
		end := classInstantiationNameEnd(s, i)
		if end < 0 || s.At(end).Kind == token.Punct && s.At(end).Value == "(" {
			continue
		}
		prev := phpcsPrevNonEmpty(s, end-1)
		if prev <= i {
			continue
		}
		s.InsertSliceAt(prev+1, []token.Token{{Kind: token.Punct, Value: "("}, {Kind: token.Punct, Value: ")"}})
		changed = true
	}
	return changed
}

// classInstantiationNameEnd is the first token after the class name of the
// "new" at i, -1 for an anonymous class or when the file ends first.
func classInstantiationNameEnd(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		switch {
		case isPhpcsEmpty(t), t.Kind == token.Ident, t.Kind == token.Variable:
			continue
		case t.Kind == token.Keyword:
			switch strings.ToLower(t.Value) {
			case "static":
				continue
			case "class", "readonly":
				return -1
			}
			return j
		case t.Kind == token.AttributeStart:
			return -1
		case t.Kind == token.Punct:
			switch t.Value {
			case `\`, "$", "->", "?->", "::":
				continue
			case "[", "{":
				m := s.MatchForward(j)
				if m < 0 {
					return -1
				}
				j = m
				continue
			}
		}
		return j
	}
	return -1
}
