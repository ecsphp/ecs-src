package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoPhp4ConstructorFixer.php
//
// NoPhp4Constructor renames a PHP4-style constructor (a method named after its
// class) to __construct, and calls to the parent's one to parent::__construct()
// (risky).
type NoPhp4Constructor struct{}

func (NoPhp4Constructor) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\NoPhp4ConstructorFixer`
}

func (NoPhp4Constructor) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoPhp4ConstructorFixer.php"
}

var np4ObjectOperators = []string{"->", "?->"}

func (NoPhp4Constructor) Fix(s *tokens.Stream) bool {
	var classes []int
	for i := 0; i < s.Len(); i++ {
		if taIsClassKw(s, i) {
			classes = append(classes, i)
		}
	}
	if len(classes) == 0 {
		return false
	}
	before := s.Render()
	numClasses := len(classes)
	for i := 0; i < numClasses; i++ {
		index := classes[i]
		if taIsAnonymousClass(s, index) {
			continue
		}
		// is it inside a namespace?
		if nspIndex := np4PrevNamespace(s, index); nspIndex >= 0 {
			nspIndex = sigNext(s, nspIndex)
			// the global namespace "namespace {" allows PHP4 constructors
			if nspIndex >= 0 && !isPunctVal(s, nspIndex, "{") {
				nspIndex = nextPunctOfKind(s, nspIndex, ";", "{")
				if nspIndex < 0 || isPunctVal(s, nspIndex, ";") {
					break // a non-block namespace: no PHP4 code in there
				}
				nspEnd := s.MatchForward(nspIndex)
				if index < nspEnd {
					// skip the other classes of this block namespace
					for j := i + 1; j < numClasses; j++ {
						if classes[j] < nspEnd {
							i++
						}
					}
					continue
				}
			}
		}
		classNameIndex := sigNext(s, index)
		if classNameIndex < 0 {
			continue
		}
		className := s.At(classNameIndex).Value
		classStart := nextPunctOfKind(s, classNameIndex, "{")
		if classStart < 0 {
			continue
		}
		classEnd := s.MatchForward(classStart)
		if classEnd < 0 {
			continue
		}
		np4FixConstructor(s, className, classStart, classEnd)
		np4FixParent(s, classStart, classEnd)
	}
	compactEmptyTokens(s)
	return s.Render() != before
}

// np4PrevNamespace mirrors getPrevTokenOfKind($index, [[T_NAMESPACE,
// 'namespace']]): the "namespace" keyword, in that exact case. A relative name
// ("namespace\foo", an Ident here) does not count, as in the PHP engine.
func np4PrevNamespace(s *tokens.Stream, index int) int {
	for j := index - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Keyword && t.Value == "namespace" {
			return j
		}
	}
	return -1
}

type np4Function struct {
	nameIndex  int
	startIndex int
	endIndex   int // -1 when unknown
	bodyIndex  int // -1 for an abstract method
	modifiers  map[string]bool
}

// np4FindFunction mirrors findFunction: the first method named name (any case)
// within [startIndex, endIndex].
func np4FindFunction(s *tokens.Stream, name string, startIndex, endIndex int) *np4Function {
	function := taFindSequence(s, []taSeqItem{taSeqKeyword("function"), taSeqTString(name, true), taSeqPunct("(")}, startIndex, endIndex)
	if function == nil {
		return nil
	}
	modifiers := map[string]bool{}
	prevBlock := sigPrev(s, function[0])
	for prevBlock >= 0 && taIsModifier(s, prevBlock) {
		modifiers[strings.ToLower(s.At(prevBlock).Value)] = true
		prevBlock = sigPrev(s, prevBlock)
	}
	f := &np4Function{nameIndex: function[1], startIndex: prevBlock + 1, endIndex: -1, bodyIndex: -1, modifiers: modifiers}
	if modifiers["abstract"] {
		f.endIndex = nextPunctOfKind(s, function[2], ";")
	} else {
		f.bodyIndex = nextPunctOfKind(s, function[2], "{")
		if f.bodyIndex >= 0 {
			f.endIndex = s.MatchForward(f.bodyIndex)
		}
	}
	return f
}

func np4FixConstructor(s *tokens.Stream, className string, classStart, classEnd int) {
	php4 := np4FindFunction(s, className, classStart, classEnd)
	if php4 == nil {
		return // no PHP4 constructor
	}
	if php4.modifiers["abstract"] || php4.modifiers["static"] {
		return // a PHP4 constructor can't be abstract or static
	}
	php5 := np4FindFunction(s, "__construct", classStart, classEnd)
	if php5 == nil {
		// no PHP5 constructor: rename the PHP4 one
		s.SetValue(php4.nameIndex, "__construct")
		np4FixInfiniteRecursion(s, php4.bodyIndex, php4.endIndex)
		return
	}
	if php4.bodyIndex < 0 || php4.endIndex < 0 {
		return
	}
	// does the PHP4 constructor only call $this->__construct($args, ...)?
	for _, seq := range np4WrapperMethodSequences(s, "__construct", php4.startIndex, php4.bodyIndex) {
		if taFindSequence(s, seq, php4.bodyIndex-1, php4.endIndex) != nil {
			for i := php4.startIndex; i <= php4.endIndex; i++ {
				clearTokenAt(s, i)
			}
			return
		}
	}
	if php5.bodyIndex < 0 || php5.endIndex < 0 {
		return
	}
	// does __construct only call the PHP4 constructor with the same arguments?
	for _, seq := range np4WrapperMethodSequences(s, className, php4.startIndex, php4.bodyIndex) {
		if taFindSequence(s, seq, php5.bodyIndex-1, php5.endIndex) != nil {
			for i := php5.startIndex; i <= php5.endIndex; i++ {
				clearTokenAt(s, i)
			}
			s.SetValue(php4.nameIndex, "__construct")
			return
		}
	}
}

// np4FixParent mirrors fixParent: for every "extends" of the file, calls to
// that parent's PHP4 constructor inside the class become parent::__construct().
func np4FixParent(s *tokens.Stream, classStart, classEnd int) {
	for index := 0; index < s.Len(); index++ {
		if !kwIs(s.At(index), "extends") {
			continue
		}
		parentIndex := sigNext(s, index)
		if parentIndex < 0 {
			continue
		}
		parentClass := s.At(parentIndex).Value
		// parent::ParentClassName() or ParentClassName::ParentClassName()
		if seq := taFindSequence(s, []taSeqItem{taSeqTString("", false), taSeqPunct("::"), taSeqTString(parentClass, true), taSeqPunct("(")}, classStart, classEnd); seq != nil {
			if first := s.At(seq[0]).Value; strings.EqualFold(first, "parent") || strings.EqualFold(first, parentClass) {
				s.SetValue(seq[0], "parent")
				s.SetValue(seq[2], "__construct")
			}
		}
		// $this->ParentClassName()
		for _, op := range np4ObjectOperators {
			if seq := taFindSequence(s, []taSeqItem{taSeqVariable("$this"), taSeqPunct(op), taSeqTString(parentClass, true), taSeqPunct("(")}, classStart, classEnd); seq != nil {
				s.Set(seq[0], token.Token{Kind: token.Ident, Value: "parent", Pos: s.At(seq[0]).Pos})
				s.SetValue(seq[1], "::")
				s.SetValue(seq[2], "__construct")
			}
		}
	}
}

// np4FixInfiniteRecursion points $this->__construct() calls of a renamed
// constructor at parent::__construct().
func np4FixInfiniteRecursion(s *tokens.Stream, start, end int) {
	if start < 0 || end < 0 {
		return
	}
	for _, op := range np4ObjectOperators {
		seq := []taSeqItem{taSeqVariable("$this"), taSeqPunct(op), taSeqTString("__construct", true)}
		for {
			callSeq := taFindSequence(s, seq, start, end)
			if callSeq == nil {
				return
			}
			s.Set(callSeq[0], token.Token{Kind: token.Ident, Value: "parent", Pos: s.At(callSeq[0]).Pos})
			s.SetValue(callSeq[1], "::")
		}
	}
}

// np4WrapperMethodSequences mirrors getWrapperMethodSequence: for each object
// operator, "{ $this->method($a, $b); }" with the variables of the signature
// between startIndex and bodyIndex. The method name matches in any case.
func np4WrapperMethodSequences(s *tokens.Stream, method string, startIndex, bodyIndex int) [][]taSeqItem {
	var sequences [][]taSeqItem
	for _, op := range np4ObjectOperators {
		seq := []taSeqItem{taSeqPunct("{"), taSeqVariable("$this"), taSeqPunct(op), taSeqTString(method, true), taSeqPunct("(")}
		for index := startIndex; ; {
			index = np4NextVariable(s, index)
			if index < 0 || index >= bodyIndex {
				break
			}
			if len(seq) > 5 {
				seq = append(seq, taSeqPunct(","))
			}
			seq = append(seq, taSeqVariable(s.At(index).Value))
		}
		seq = append(seq, taSeqPunct(")"), taSeqPunct(";"), taSeqPunct("}"))
		sequences = append(sequences, seq)
	}
	return sequences
}

func np4NextVariable(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.Variable {
			return j
		}
	}
	return -1
}
