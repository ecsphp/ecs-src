package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModernSerializationMethodsFixer.php
//
// ModernSerializationMethods renames the deprecated __sleep and __wakeup
// methods of a class-like to __serialize and __unserialize(array $data), unless
// the class-like already declares the new method. Names are matched exactly
// (case-sensitive), as PHP-CS-Fixer does.
type ModernSerializationMethods struct{}

func (ModernSerializationMethods) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ModernSerializationMethodsFixer`
}

func (ModernSerializationMethods) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModernSerializationMethodsFixer.php"
}

// IsCandidate rules out a file without the trigger: only a "__sleep" or "__wakeup" method is renamed.
func (ModernSerializationMethods) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("__sleep", "__wakeup")
}

func (ModernSerializationMethods) Fix(s *tokens.Stream) bool {
	// method name -> name index, per class-like
	methodsByClass := map[int]map[string]int{}
	for _, m := range classyMethods(s) {
		nameIdx := nextMeaningfulIndex(s, m.function)
		if nameIdx < 0 {
			continue
		}
		methods, ok := methodsByClass[m.class]
		if !ok {
			methods = map[string]int{}
			methodsByClass[m.class] = methods
		}
		methods[s.At(nameIdx).Value] = nameIdx
	}

	var inserts []int // "(" of each renamed __wakeup
	changed := false
	for _, methods := range methodsByClass {
		if idx, ok := methods["__sleep"]; ok {
			if _, has := methods["__serialize"]; !has {
				s.Set(idx, token.Token{Kind: token.Ident, Value: "__serialize"})
				changed = true
			}
		}
		if idx, ok := methods["__wakeup"]; ok {
			if _, has := methods["__unserialize"]; !has {
				s.Set(idx, token.Token{Kind: token.Ident, Value: "__unserialize"})
				changed = true
				for j := idx + 1; j < s.Len(); j++ {
					if isPunctVal(s, j, "(") {
						inserts = append(inserts, j)
						break
					}
				}
			}
		}
	}
	// insert "array $data" after each "(", last first so indexes stay valid
	slices.Sort(inserts)
	for _, insert := range slices.Backward(inserts) {
		at := insert + 1
		s.InsertSliceAt(at, []token.Token{
			{Kind: token.Keyword, Value: "array"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Variable, Value: "$data"},
		})
	}
	return changed
}

// classyMethod is a method element of a class-like: its "function" keyword and
// the index of the class-like keyword it belongs to.
type classyMethod struct{ function, class int }

// classyMethods ports the "method" elements of TokensAnalyzer::getClassyElements:
// every named "function" at the first brace level of a class, interface, trait
// or enum body, nested anonymous classes included (attributed to themselves).
func classyMethods(s *tokens.Stream) []classyMethod {
	var out []classyMethod
	for i := 1; i < s.Len()-2; i++ {
		if isClassyDeclaration(s, i) {
			i = classyMethodsFind(s, i, i, &out)
		}
	}
	// getClassyElements sorts the elements by index
	slices.SortFunc(out, func(a, b classyMethod) int { return a.function - b.function })
	return out
}

// classyMethodsFind ports TokensAnalyzer::findClassyElements (methods only)
// for the class-like at classIdx, scanning from index; it returns the index it
// stopped at (the body's closing brace).
func classyMethodsFind(s *tokens.Stream, classIdx, index int, out *[]classyMethod) int {
	curly, braces := 0, 0
	n := s.Len()
	for index++; index < n; index++ {
		t := s.At(index)
		if kwIs(t, "class") {
			// anonymous class in the class: skip its constructor arguments
			nested := index
			index = nextMeaningfulIndex(s, index)
			if index < 0 {
				return n
			}
			if isPunctVal(s, index, "(") {
				index++
				for level := 1; index < n; index++ {
					if isPunctVal(s, index, "(") {
						level++
						continue
					}
					if isPunctVal(s, index, ")") {
						level--
						if level == 0 {
							index = classyMethodsFind(s, nested, index, out)
							break
						}
						continue
					}
					if kwIs(s.At(index), "class") {
						index = classyMethodsFind(s, index, index, out)
					}
				}
			} else {
				index = classyMethodsFind(s, nested, nested, out)
			}
			continue
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "(":
				braces++
			case ")":
				braces--
			case "{":
				curly++
			case "}":
				curly--
				if curly == 0 {
					return index
				}
			}
			continue
		}
		if curly != 1 {
			continue
		}
		if t.Kind == token.AttributeStart {
			if end := s.AttributeEnd(index); end > index {
				index = end
			}
			continue
		}
		if kwIs(t, "function") && !isPunctVal(s, nextMeaningfulIndex(s, index), "(") {
			*out = append(*out, classyMethod{function: index, class: classIdx})
		}
	}
	return index
}
