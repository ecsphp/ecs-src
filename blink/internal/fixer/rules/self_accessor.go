package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfAccessorFixer.php
//
// SelfAccessor replaces a reference to the containing class's own name with
// "self" inside the class body: "new Foo()" -> "new self()",
// "Foo::CONST" -> "self::CONST", type declarations "Foo $x" -> "self $x".
// Like PHP-CS-Fixer it only visits classes and interfaces (not enums or
// traits), compares the name case-insensitively and also rewrites the name
// qualified with the current namespace ("\App\Foo::CONST").
type SelfAccessor struct{}

func (SelfAccessor) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\SelfAccessorFixer`
}

func (SelfAccessor) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SelfAccessorFixer.php"
}

func (SelfAccessor) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword {
			continue
		}
		if lw := strings.ToLower(s.At(i).Value); lw != "class" && lw != "interface" {
			continue
		}
		// skip "new class" (anonymous) and "::class"
		if prev, ok := prevSignificant(s, i); ok {
			if prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
				continue
			}
			if prev.Kind == token.Punct && (prev.Value == "->" || prev.Value == "?->" || prev.Value == "::") {
				continue
			}
		}
		nameIdx := nextSignificantIndex(s, i)
		if nameIdx < 0 || s.At(nameIdx).Kind != token.Ident {
			continue
		}
		className := s.At(nameIdx).Value
		namespace := selfAccessorNamespace(s, i)
		open := findBodyBrace(s, nameIdx)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		for k := open + 1; k < closeIdx; k++ {
			// Skip lambdas/closures and arrow functions entirely: PHP's fixer
			// never rewrites inside them (isLambda / T_FN branches), so a class
			// name in a closure's parameter type or body is left untouched.
			if s.At(k).Kind == token.Keyword {
				switch strings.ToLower(s.At(k).Value) {
				case "fn":
					if end := arrowFunctionEnd(s, k); end > k {
						k = end
						continue
					}
				case "function":
					if isLambdaFunc(s, k) {
						if bodyOpen := findBodyBrace(s, k); bodyOpen >= 0 {
							if bodyClose := s.MatchForward(bodyOpen); bodyClose >= 0 {
								k = bodyClose
								continue
							}
						}
					}
				}
			}
			// Skip nested class scopes (named or anonymous). A reference inside
			// them belongs to a different innermost class, so the outer class
			// name must be left untouched; nested named classes are rewritten
			// when the outer loop reaches them as their own scope.
			if s.At(k).Kind == token.Keyword && isClassLikeKeyword(strings.ToLower(s.At(k).Value)) {
				if prev, ok := prevSignificant(s, k); !ok || prev.Kind != token.Punct || prev.Value != "::" {
					if nestedOpen := findBodyBrace(s, k); nestedOpen >= 0 {
						if nestedClose := s.MatchForward(nestedOpen); nestedClose >= 0 {
							k = nestedClose
							continue
						}
					}
				}
			}
			if s.At(k).Kind != token.Ident || !strings.EqualFold(s.At(k).Value, className) {
				continue
			}
			if start, ok := selfAccessorRef(s, k, namespace); ok {
				s.SetValue(k, "self")
				if start < k {
					s.ReplaceRange(start, k-1, nil)
					closeIdx -= k - start
					k = start
				}
				changed = true
			}
		}
	}
	return changed
}

// isLambdaFunc reports whether the "function" keyword at i opens an anonymous
// function (closure): it is followed by an optional "&" and then "(", with no
// method name in between.
func isLambdaFunc(s *tokens.Stream, i int) bool {
	n := nextSignificantIndex(s, i)
	if n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "&" {
		n = nextSignificantIndex(s, n)
	}
	return n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "("
}

// arrowFunctionEnd returns the index of the last token of the arrow function
// that starts at the "fn" keyword at i (its body is a single expression ending
// at the first top-level ",", ";", ")", "]" or "}"). Returns i unchanged if the
// shape cannot be parsed.
func arrowFunctionEnd(s *tokens.Stream, i int) int {
	open := nextSignificantIndex(s, i)
	if open >= 0 && s.At(open).Kind == token.Punct && s.At(open).Value == "&" {
		open = nextSignificantIndex(s, open)
	}
	if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return i
	}
	paramClose := s.MatchForward(open)
	if paramClose < 0 {
		return i
	}
	arrow := -1
	for j := paramClose + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.Punct && t.Value == "=>" {
			arrow = j
			break
		}
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{") {
			return i
		}
	}
	if arrow < 0 {
		return i
	}
	depth := 0
	for j := arrow + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return j - 1
			}
			depth--
		case ",", ";":
			if depth == 0 {
				return j - 1
			}
		}
	}
	return s.Len() - 1
}

func isClassLikeKeyword(lw string) bool {
	switch lw {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

// findBodyBrace returns the index of the "{" opening a class body declared at
// nameIdx, scanning past any "extends"/"implements" clause. Returns -1 if none.
func findBodyBrace(s *tokens.Stream, nameIdx int) int {
	for k := nameIdx + 1; k < s.Len(); k++ {
		if s.At(k).Kind == token.Punct {
			switch s.At(k).Value {
			case "{":
				return k
			case ";":
				return -1
			}
		}
	}
	return -1
}

// selfAccessorNamespace returns the segments of the namespace declared before
// the class keyword at i (nil for the global namespace).
func selfAccessorNamespace(s *tokens.Stream, i int) []string {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Keyword || !strings.EqualFold(s.At(j).Value, "namespace") {
			continue
		}
		var segs []string
		k := sigNext(s, j)
		for k >= 0 && s.At(k).Kind == token.Ident {
			segs = append(segs, s.At(k).Value)
			k = sigNext(s, k)
			if k < 0 || s.At(k).Kind != token.Punct || s.At(k).Value != `\` {
				break
			}
			k = sigNext(s, k)
		}
		if k >= 0 && s.At(k).Kind == token.Punct && (s.At(k).Value == ";" || s.At(k).Value == "{") {
			return segs
		}
		// "namespace\Foo" is a relative name, not a declaration
	}
	return nil
}

// selfAccessorRef reports whether the class-name token at k is a reference
// PHP-CS-Fixer turns into "self": a "new"/"instanceof" operand, a "::" access,
// or a type in a method signature. A name qualified with the current namespace
// counts too; start is then the index of its leading "\", else k.
func selfAccessorRef(s *tokens.Stream, k int, namespace []string) (start int, ok bool) {
	nextIdx := sigNext(s, k)
	if nextIdx >= 0 && s.At(nextIdx).Kind == token.Punct && s.At(nextIdx).Value == `\` {
		return 0, false // heads a namespaced name
	}
	start = k
	p := sigPrev(s, k)
	if p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == `\` {
		if start = selfAccessorQualifiedStart(s, k, namespace); start < 0 {
			return 0, false
		}
		p = sigPrev(s, start)
	}
	if p >= 0 {
		prev := s.At(p)
		if prev.Kind == token.Ident {
			return 0, false
		}
		if prev.Kind == token.Punct {
			switch prev.Value {
			case "->", "?->", "::":
				return 0, false
			}
		}
		if prev.Kind == token.Keyword {
			switch strings.ToLower(prev.Value) {
			case "new", "instanceof":
				return start, true
			}
		}
	}
	if nextIdx >= 0 && s.At(nextIdx).Kind == token.Punct && s.At(nextIdx).Value == "::" {
		return start, true
	}
	// a type only counts in a method signature, right after "(", ",", "?", "|"
	// or the return-type ":"; property types are left untouched
	if p < 0 || s.At(p).Kind != token.Punct {
		return 0, false
	}
	switch s.At(p).Value {
	case "(", ",":
		return start, enclosingFuncParamOpen(s, start) >= 0
	case "?", "|":
		return start, enclosingFuncParamOpen(s, start) >= 0 || selfAccessorInReturnType(s, start)
	case ":":
		return start, isReturnTypeColon(s, p)
	}
	return 0, false
}

// selfAccessorQualifiedStart walks back from the name at k over "\"-separated
// segments that must spell the current namespace with a leading "\", and
// returns the index of that leading "\", or -1.
func selfAccessorQualifiedStart(s *tokens.Stream, k int, namespace []string) int {
	idx := k
	for i := len(namespace) - 1; i >= -1; i-- {
		idx = sigPrev(s, idx)
		if idx < 0 || s.At(idx).Kind != token.Punct || s.At(idx).Value != `\` {
			return -1
		}
		if i < 0 {
			return idx
		}
		idx = sigPrev(s, idx)
		if idx < 0 || s.At(idx).Kind != token.Ident || !strings.EqualFold(s.At(idx).Value, namespace[i]) {
			return -1
		}
	}
	return -1
}

// selfAccessorInReturnType reports whether the type token at k sits in a return
// type, walking back over "?", "|", "&", "\" to a return-type colon.
func selfAccessorInReturnType(s *tokens.Stream, k int) bool {
	j := k
	for {
		p := prevSignificantIndex(s, j)
		if p < 0 {
			return false
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == ":" {
			return isReturnTypeColon(s, p)
		}
		if s.At(p).Kind == token.Ident ||
			(s.At(p).Kind == token.Punct && (s.At(p).Value == "?" || s.At(p).Value == "|" || s.At(p).Value == "&" || s.At(p).Value == `\`)) {
			j = p
			continue
		}
		return false
	}
}
