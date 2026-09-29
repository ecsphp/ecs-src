package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfStaticAccessorFixer.php
//
// SelfStaticAccessor replaces "static" with "self" inside a final class or an
// enum (both cannot be extended, so late static binding is redundant): a static
// accessor "static::foo()" becomes "self::foo()" and "new static" becomes
// "new self". A "static" used as a type ("): static") or a closure/modifier
// ("static function") is left alone.
type SelfStaticAccessor struct{}

func (SelfStaticAccessor) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\SelfStaticAccessorFixer`
}

func (SelfStaticAccessor) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfStaticAccessorFixer.php"
}

type selfStaticFrame struct {
	classLike bool
	final     bool
}

func (SelfStaticAccessor) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []selfStaticFrame
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				if kind, kw := classifyBrace(s, i); kind == braceClassLike {
					stack = append(stack, selfStaticFrame{classLike: true, final: isFinalClassLike(s, kw)})
				} else {
					stack = append(stack, selfStaticFrame{})
				}
			case "}":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
			continue
		}
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "static") &&
			enclosingFinalClassLike(stack) && isStaticAccessor(s, i) {
			s.SetValue(i, "self")
			changed = true
		}
	}
	return changed
}

// isFinalClassLike reports whether the class-like whose keyword sits at kw cannot
// be extended: an enum, or a class carrying the "final" modifier. An anonymous
// class ("new class") is treated conservatively as non-final.
func isFinalClassLike(s *tokens.Stream, kw int) bool {
	if kw < 0 {
		return false
	}
	if strings.EqualFold(s.At(kw).Value, "enum") {
		return true
	}
	for j := kw - 1; j >= 0; j-- {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "final":
				return true
			case "new":
				return false // anonymous class
			case "abstract", "readonly":
				continue
			}
			return false
		}
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{" || t.Value == "}") {
			return false // statement boundary - no modifier seen
		}
		return false
	}
	return false
}

// enclosingFinalClassLike reports whether the nearest class-like frame on the
// stack is final.
func enclosingFinalClassLike(stack []selfStaticFrame) bool {
	for _, f := range slices.Backward(stack) {
		if f.classLike {
			return f.final
		}
	}
	return false
}

// isStaticAccessor reports whether the "static" at i is a class reference:
// "static::" (next significant is "::") or "new static" (previous is "new").
func isStaticAccessor(s *tokens.Stream, i int) bool {
	if nextSignificantValue(s, i) == "::" {
		return true
	}
	if prev, ok := prevSignificant(s, i); ok && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
		return true
	}
	return false
}
