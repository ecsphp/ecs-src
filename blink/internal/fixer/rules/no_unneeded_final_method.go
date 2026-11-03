package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoUnneededFinalMethodFixer.php
//
// NoUnneededFinalMethod removes "final" from methods where it has no effect
// (risky): every method of a final class or an enum, and, with private_methods
// (default true), private methods of other classes except the constructor.
type NoUnneededFinalMethod struct {
	// KeepPrivate is the inverse of the private_methods option (default true).
	KeepPrivate bool
}

func (f NoUnneededFinalMethod) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["private_methods"].(bool); ok {
		f.KeepPrivate = !v
	}
	return f
}

func (NoUnneededFinalMethod) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\NoUnneededFinalMethodFixer`
}

func (NoUnneededFinalMethod) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoUnneededFinalMethodFixer.php"
}

func (f NoUnneededFinalMethod) Fix(s *tokens.Stream) bool {
	if !nufmIsCandidate(s) {
		return false
	}
	var finals []int
	isEnum := map[int]bool{}
	isFinal := map[int]bool{}
	for _, m := range taClassyMethods(s) {
		classIndex := m.classIndex
		if _, ok := isEnum[classIndex]; !ok {
			isEnum[classIndex] = taKw(s, classIndex, "enum")
		}
		finalIndex, private := -1, false
		for previous := m.index; ; {
			previous = sigPrev(s, previous)
			if previous < 0 {
				break
			}
			if taKw(s, previous, "private") {
				private = true
			} else if taKw(s, previous, "final") {
				finalIndex = previous
			}
			if !taIsModifier(s, previous) {
				break
			}
		}
		if finalIndex < 0 {
			continue
		}
		if isEnum[classIndex] {
			finals = append(finals, finalIndex)
			continue
		}
		if _, ok := isFinal[classIndex]; !ok {
			isFinal[classIndex] = taClassyIsFinal(s, classIndex)
		}
		if isFinal[classIndex] {
			finals = append(finals, finalIndex)
			continue
		}
		name := sigNext(s, m.index)
		constructor := name >= 0 && strings.ToLower(s.At(name).Value) == "__construct"
		if !private || f.KeepPrivate || constructor {
			continue
		}
		finals = append(finals, finalIndex)
	}
	if len(finals) == 0 {
		return false
	}
	slices.Sort(finals)
	for _, index := range slices.Backward(finals) {
		// clearFinal: the "final" and the whitespace right after it
		if index+1 < s.Len() && s.At(index+1).Kind == token.Whitespace {
			s.RemoveAt(index + 1)
		}
		s.RemoveAt(index)
	}
	return true
}

// nufmIsCandidate mirrors isCandidate: "final" and "function" are found, and
// so is a T_CLASS or T_ENUM.
func nufmIsCandidate(s *tokens.Stream) bool {
	final, function, class := false, false, false
	for i := 0; i < s.Len(); i++ {
		switch {
		case kwIs(s.At(i), "final"):
			final = true
		case kwIs(s.At(i), "function"):
			function = true
		case taIsClassKw(s, i), kwIs(s.At(i), "enum") && taIsClassy(s, i):
			class = true
		}
	}
	return final && function && class
}

// taClassyIsFinal mirrors getClassyModifiers()['final'] for the classy at i.
func taClassyIsFinal(s *tokens.Stream, i int) bool {
	for {
		i = sigPrev(s, i)
		if i < 0 {
			return false
		}
		switch {
		case taKw(s, i, "final"):
			return true
		case taKw(s, i, "abstract"), taKw(s, i, "readonly"):
			continue
		}
		return false
	}
}

// taClassyElement is a method found by taClassyMethods: the index of its
// "function" token and of the classy token that owns it.
type taClassyElement struct {
	index      int
	classIndex int
}

// taClassyMethods mirrors the "method" elements of
// TokensAnalyzer::getClassyElements, in source order.
func taClassyMethods(s *tokens.Stream) []taClassyElement {
	var elements []taClassyElement
	for index := 1; index < s.Len()-2; index++ {
		if taIsClassy(s, index) {
			index = taFindClassyElements(s, index, index, &elements)
		}
	}
	slices.SortFunc(elements, func(a, b taClassyElement) int { return a.index - b.index })
	return elements
}

// taFindClassyElements mirrors TokensAnalyzer::findClassyElements (methods
// only): it scans the body of the classy at classIndex from index and returns
// the index where the scan stopped (the body's closing brace).
func taFindClassyElements(s *tokens.Stream, classIndex, index int, elements *[]taClassyElement) int {
	curlyBracesLevel := 0
	index++
	for count := s.Len(); index < count; index++ {
		t := s.At(index)
		if taIsClassKw(s, index) {
			// anonymous class in class; skip its constructor arguments first, as
			// they may hold functions of their own
			nestedClassIndex := index
			index = sigNext(s, index)
			if index < 0 {
				return count
			}
			if isPunctVal(s, index, "(") {
				index++
				for nestedBracesLevel := 1; index < count; index++ {
					if isPunctVal(s, index, "(") {
						nestedBracesLevel++
						continue
					}
					if isPunctVal(s, index, ")") {
						nestedBracesLevel--
						if nestedBracesLevel == 0 {
							index = taFindClassyElements(s, nestedClassIndex, index, elements)
							break
						}
						continue
					}
					if taIsClassKw(s, index) {
						index = taFindClassyElements(s, index, index, elements)
					}
				}
			} else {
				index = taFindClassyElements(s, nestedClassIndex, nestedClassIndex, elements)
			}
			continue
		}
		if isPunctVal(s, index, "{") {
			curlyBracesLevel++
			continue
		}
		if isPunctVal(s, index, "}") {
			curlyBracesLevel--
			if curlyBracesLevel == 0 {
				break
			}
			continue
		}
		if curlyBracesLevel != 1 {
			continue
		}
		if t.Kind == token.AttributeStart {
			if end := s.AttributeEnd(index); end >= 0 {
				index = end
			}
			continue
		}
		if kwIs(t, "function") {
			if isPunctVal(s, sigNext(s, index), "(") {
				continue
			}
			*elements = append(*elements, taClassyElement{index: index, classIndex: classIndex})
		}
	}
	return index
}
