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
		// skip anonymous classes and "::class"
		if selfAccessorIsAnonymousClass(s, i) {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok && prev.Kind == token.Punct &&
			(prev.Value == "->" || prev.Value == "?->" || prev.Value == "::") {
			continue
		}
		nameIdx := nextSignificantIndex(s, i)
		if nameIdx < 0 || s.At(nameIdx).Kind != token.Ident {
			continue
		}
		open := findBodyBrace(s, nameIdx)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		var c bool
		closeIdx, c = selfAccessorReplace(s, s.At(nameIdx).Value, selfAccessorNamespace(s, i), open, closeIdx)
		changed = changed || c
		// like PHP-CS-Fixer, go on after the class body: a class declared in
		// one of its methods is never visited as a class of its own
		i = closeIdx
	}
	return changed
}

// selfAccessorReplace is PHP-CS-Fixer's replaceNameOccurrences over the body
// open..closeIdx of the class className. It returns the body's closing index,
// shifted by any removed namespace qualifiers.
func selfAccessorReplace(s *tokens.Stream, className string, namespace []string, open, closeIdx int) (int, bool) {
	changed := false
	// index of the "{" or ";" ending the method signature being walked, or -1
	sigUntil := -1
	for k := open + 1; k < closeIdx; k++ {
		if k == sigUntil {
			sigUntil = -1
		}
		if s.At(k).Kind == token.Keyword {
			switch strings.ToLower(s.At(k).Value) {
			case "class":
				// skip anonymous classes; named ones nested in a method are
				// walked as part of this body, as PHP-CS-Fixer does
				if selfAccessorIsAnonymousClass(s, k) {
					if bodyOpen := findBodyBrace(s, k); bodyOpen >= 0 {
						if bodyClose := s.MatchForward(bodyOpen); bodyClose >= 0 {
							k = bodyClose
							continue
						}
					}
				}
			case "fn":
				// skip arrow functions, and the token after them
				if end := arrowFunctionEnd(s, k); end > k {
					k = end
					if n := sigNext(s, end); n >= 0 && n < closeIdx {
						k = n
					}
					continue
				}
			case "function":
				// skip closures; a method's signature runs from its "(" to
				// the next "{" or ";"
				if isLambdaFunc(s, k) {
					if bodyOpen := findBodyBrace(s, k); bodyOpen >= 0 {
						if bodyClose := s.MatchForward(bodyOpen); bodyClose >= 0 {
							k = bodyClose
							continue
						}
					}
					continue
				}
				paren := nextPunctOfKind(s, k, "(")
				if paren < 0 || paren >= closeIdx {
					continue
				}
				sigUntil = nextPunctOfKind(s, paren, "{", ";")
				k = paren
				continue
			}
		}
		if s.At(k).Kind != token.Ident || !strings.EqualFold(s.At(k).Value, className) {
			continue
		}
		if start, ok := selfAccessorRef(s, k, namespace, sigUntil >= 0 && k < sigUntil); ok {
			s.SetValue(k, "self")
			if start < k {
				s.ReplaceRange(start, k-1, nil)
				closeIdx -= k - start
				if sigUntil >= 0 {
					sigUntil -= k - start
				}
				k = start
			}
			changed = true
		}
	}
	return closeIdx, changed
}

// selfAccessorIsAnonymousClass is TokensAnalyzer::isAnonymousClass: "new",
// optional attributes, optional "readonly", then the "class" at i.
func selfAccessorIsAnonymousClass(s *tokens.Stream, i int) bool {
	if p := sigPrev(s, i); p >= 0 && kwIs(s.At(p), "readonly") {
		i = p
	}
	return anonClassNewIndex(s, i) >= 0
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
func selfAccessorRef(s *tokens.Stream, k int, namespace []string, inSignature bool) (start int, ok bool) {
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
	// a type only counts in a method signature, right after "(", ",", a
	// nullable "?", a union "|" or the return-type ":"; property types are
	// left untouched
	if !inSignature || p < 0 || s.At(p).Kind != token.Punct {
		return 0, false
	}
	switch s.At(p).Value {
	case "(":
		return start, !selfAccessorIsDNFOpen(s, p)
	case ",", "?", "|":
		return start, true
	case ":":
		return start, isReturnTypeColon(s, p)
	}
	return 0, false
}

// selfAccessorIsDNFOpen reports whether the "(" at i groups a DNF type such as
// "(A&B)|null", which PHP-CS-Fixer lexes as
// CT::T_DISJUNCTIVE_NORMAL_FORM_TYPE_PARENTHESIS_OPEN: it stands where a type
// starts, not after a function name or a callee.
func selfAccessorIsDNFOpen(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	switch t := s.At(p); t.Kind {
	case token.AttributeEnd:
		return true
	case token.Punct:
		switch t.Value {
		case "(", ",", "|", "?":
			return true
		case ":":
			return isReturnTypeColon(s, p)
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "public", "protected", "private", "readonly":
			return true
		}
	}
	return false
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
