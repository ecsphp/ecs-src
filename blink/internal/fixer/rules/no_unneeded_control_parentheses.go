package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php
//
// NoUnneededControlParentheses removes a redundant parenthesis pair wrapping the
// whole operand of a control statement: "return ($x);" -> "return $x;". It fires
// for return, echo, print, yield, break, continue and case, only when the "("
// directly follows the keyword and its matching ")" sits right before the
// statement terminator (";" or, for case, ":"). Nested pairs are peeled in one
// pass. The conservative boundary check leaves anything else (multiple arguments,
// a partial sub-expression) untouched. clone has its own rule, see unwrapClone.
type NoUnneededControlParentheses struct {
	// Statements limits the fixed statements; nil means the default set.
	Statements []string
}

func (f NoUnneededControlParentheses) WithConfig(config map[string]any) fixer.Fixer {
	switch list := config["statements"].(type) {
	case []string:
		f.Statements = append([]string{}, list...)
	case []any:
		f.Statements = []string{}
		for _, v := range list {
			if name, ok := v.(string); ok {
				f.Statements = append(f.Statements, name)
			}
		}
	}
	return f
}

func (f NoUnneededControlParentheses) enabled(key string) bool {
	if f.Statements == nil {
		return true
	}
	return slices.Contains(f.Statements, key)
}

func nucpStatementKey(keyword string) string {
	switch keyword {
	case "echo", "print":
		return "echo_print"
	case "case":
		return "switch_case"
	}
	return keyword
}

func (NoUnneededControlParentheses) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUnneededControlParenthesesFixer`
}

func (NoUnneededControlParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php"
}

func (f NoUnneededControlParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		var term string
		kw := strings.ToLower(t.Value)
		switch kw {
		case "clone":
			if f.enabled("clone") && unwrapClone(s, i) {
				changed = true
			}
			continue
		case "return", "echo", "print", "yield", "break", "continue":
			term = ";"
		case "case":
			term = ":"
		default:
			continue
		}
		if !f.enabled(nucpStatementKey(kw)) {
			continue
		}
		// a keyword after "->"/"::" is a member name (Foo::case(...)), not a control
		// statement
		if memberPrev(s, i) {
			continue
		}
		// peel redundant pairs: "return (($x));" -> "return $x;"
		for {
			j := nextSignificantIndex(s, i)
			if j < 0 || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
				break
			}
			k := s.MatchForward(j)
			if k < 0 {
				break
			}
			// the matching ")" must sit right before the terminator, and the pair
			// must not be empty
			nj := nextSignificantIndex(s, k)
			if nj < 0 || s.At(nj).Kind != token.Punct || s.At(nj).Value != term {
				break
			}
			if nextSignificantIndex(s, j) >= k {
				break
			}
			s.RemoveAt(k)
			if j == i+1 {
				// "return($x)" -> keep a separator so the keyword stays a keyword
				s.RemoveAt(j)
				s.InsertAt(j, token.Token{Kind: token.Whitespace, Value: " "})
			} else {
				s.RemoveAt(j)
			}
			changed = true
		}
	}
	return changed
}

// unwrapClone removes the pairs wrapping the operand of the clone at index i, like
// PHP-CS-Fixer's isWrappedCloneArgument: only where the clone stands alone in its
// statement or sequence element, and only around a plain operand. The parentheses
// around anything else are needed: "clone ($a ?: $b)" clones the result of the
// ternary while "clone $a ?: $b" does not, and "clone ($a, [...])" and "clone(...)"
// are a call.
func unwrapClone(s *tokens.Stream, i int) bool {
	changed := false
	for {
		open := nextMeaningfulIndex(s, i)
		if open < 0 || !csbIsPunct(s.At(open), "(") {
			return changed
		}
		closing := s.MatchForward(open)
		if closing < 0 || nextMeaningfulIndex(s, open) == closing {
			return changed
		}
		before := prevMeaningfulIndex(s, i)
		after := nextMeaningfulIndex(s, closing)
		if before < 0 || after < 0 || !cloneStandsAlone(s, before, after) || cloneOperandHasOperation(s, open, closing) {
			return changed
		}
		removeParenthesis(s, closing, csbIsPunct(s.At(after), ":"))
		removeParenthesis(s, open, true)
		changed = true
	}
}

func cloneStandsAlone(s *tokens.Stream, before, after int) bool {
	b, a := s.At(before), s.At(after)
	switch {
	case csbIsPunct(b, "?"):
		// Only before ":". Anything else can then apply to another expression, where
		// PHP-CS-Fixer unwraps anyway: "clone ($o->b)()" would call a method, and
		// "clone (new Foo)->bar" and "clone (enum) and $b" become parse errors.
		return csbIsPunct(a, ":")
	case csbIsPunct(b, "=") && endsStatement(a):
		return true
	case isCloneStatementStart(s, before) && endsStatement(a):
		return true
	case csbIsKeyword(b, "case") && (csbIsPunct(a, ":") || csbIsPunct(a, ";")):
		return true
	case isFnBodyStart(s, before) && (endsStatement(a) || csbIsPunct(a, ",")):
		return true
	case isForElement(s, before, after):
		return true
	}
	return isSequenceElement(b, a)
}

func endsStatement(t token.Token) bool {
	return csbIsPunct(t, ";") || t.Kind == token.CloseTag
}

func isCloneStatementStart(s *tokens.Stream, before int) bool {
	b := s.At(before)
	if b.Kind == token.OpenTag || csbIsPunct(b, ";") || csbIsPunct(b, "{") {
		return true
	}
	if csbIsPunct(b, "}") {
		return !closesDynamicElement(s, before)
	}
	if b.Kind != token.Keyword {
		return false
	}
	// yield from is one token that keeps its inner spacing and comments
	word := strings.ToLower(b.Value)
	return strings.HasPrefix(word, "yield") || slices.Contains([]string{
		"echo", "print", "return", "throw", "break", "continue", "require", "require_once", "include", "include_once",
	}, word)
}

// closesDynamicElement follows PHP-CS-Fixer's closeCurlyBelongsToDynamicElement: a
// "}" whose "{" follows a ":" with no case closer to it than a "?".
func closesDynamicElement(s *tokens.Stream, closing int) bool {
	open := s.MatchBackward(closing)
	if open < 0 {
		return false
	}
	prev := prevMeaningfulIndex(s, open)
	if prev < 0 {
		return false
	}
	if !csbIsPunct(s.At(prev), ":") {
		return false
	}
	for j := prev - 1; j >= 0; j-- {
		if csbIsPunct(s.At(j), "?") {
			return true
		}
		if csbIsKeyword(s.At(j), "case") {
			return false
		}
	}
	// PHP-CS-Fixer throws when it finds neither: keep the parentheses
	return true
}

// isFnBodyStart reports whether the token at index i is the "=>" of an arrow
// function, with at most a plain return type: "fn () =>", "fn (): A|B =>".
func isFnBodyStart(s *tokens.Stream, i int) bool {
	if !csbIsPunct(s.At(i), "=>") {
		return false
	}
	j := prevMeaningfulIndex(s, i)
	if j >= 0 && s.At(j).Kind == token.Ident {
		for j >= 0 && (s.At(j).Kind == token.Ident || csbIsPunct(s.At(j), "|") || csbIsPunct(s.At(j), "&")) {
			j = prevMeaningfulIndex(s, j)
		}
		if j < 0 || !csbIsPunct(s.At(j), ":") {
			return false
		}
		j = prevMeaningfulIndex(s, j)
	}
	if j < 0 || !csbIsPunct(s.At(j), ")") {
		return false
	}
	open := s.MatchBackward(j)
	if open < 0 {
		return false
	}
	j = prevMeaningfulIndex(s, open)
	if j >= 0 && csbIsPunct(s.At(j), "&") {
		j = prevMeaningfulIndex(s, j)
	}
	return j >= 0 && csbIsKeyword(s.At(j), "fn")
}

// isForElement matches the first and last expressions of a for head; the middle
// one is a statement start already: for (X; Y; Z).
func isForElement(s *tokens.Stream, before, after int) bool {
	candidate := -1
	switch {
	case csbIsPunct(s.At(before), "(") && csbIsPunct(s.At(after), ";"):
		candidate = prevMeaningfulIndex(s, before)
	case csbIsPunct(s.At(after), ")") && csbIsPunct(s.At(before), ";"):
		if open := s.MatchBackward(after); open >= 0 {
			candidate = prevMeaningfulIndex(s, open)
		}
	}
	return candidate >= 0 && csbIsKeyword(s.At(candidate), "for")
}

func isSequenceElement(before, after token.Token) bool {
	beforeComma, afterComma := csbIsPunct(before, ","), csbIsPunct(after, ",")
	beforeOpen := csbIsPunct(before, "(") || csbIsPunct(before, "[") || csbIsPunct(before, "{")
	afterClose := csbIsPunct(after, ")") || csbIsPunct(after, "]") || csbIsPunct(after, "}")
	return (beforeComma || beforeOpen) && (afterComma || afterClose)
}

// cloneOperandHasOperation scans the content of the pair at open..closing, a nested
// block counting as one operand. A leading new belongs to the operand.
func cloneOperandHasOperation(s *tokens.Stream, open, closing int) bool {
	start := open
	if first := nextMeaningfulIndex(s, open); first >= 0 && csbIsKeyword(s.At(first), "new") {
		start = first
	}
	for j := nextMeaningfulIndex(s, start); j >= 0 && j != closing; j = nextMeaningfulIndex(s, j) {
		t := s.At(j)
		if _, _, isCast := castAt(s, j); isCast {
			return true
		}
		if prev := prevMeaningfulIndex(s, j); strings.EqualFold(t.Value, "class") && prev >= 0 && csbIsPunct(s.At(prev), "::") {
			return true
		}
		if t.Kind == token.Ident {
			if !isCloneOperandName(s, j) {
				return true
			}
			continue
		}
		if csbIsPunct(t, "(") || csbIsPunct(t, "[") || csbIsPunct(t, "{") {
			end := s.MatchForward(j)
			if end < 0 {
				return true
			}
			j = end
			continue
		}
		if !isCloneOperandToken(t) {
			return true
		}
	}
	return false
}

// isCloneOperandName rejects the names that are language constructs to PHP but plain
// names to the lexer, unless they are a member or a segment of a qualified name.
func isCloneOperandName(s *tokens.Stream, i int) bool {
	word := strings.ToLower(s.At(i).Value)
	if prev := prevMeaningfulIndex(s, i); prev >= 0 && s.At(prev).Kind == token.Punct {
		switch s.At(prev).Value {
		case "->", "?->", "::", `\`:
			return true
		}
	}
	if next := nextMeaningfulIndex(s, i); next >= 0 && csbIsPunct(s.At(next), `\`) {
		return word != "namespace"
	}
	switch word {
	case "eval", "exit", "die", "__property__":
		return false
	}
	return true
}

func isCloneOperandToken(t token.Token) bool {
	switch t.Kind {
	case token.Variable, token.Number:
		return true
	case token.String:
		return t.IsLiteral() && !t.StartsHeredoc() && !t.IsShellExec()
	case token.Punct:
		return t.Value == "$" || t.Value == "::" || t.Value == `\` || t.Value == "->" || t.Value == "?->"
	case token.Keyword:
		return strings.EqualFold(t.Value, "static")
	}
	return false
}

func removeParenthesis(s *tokens.Stream, i int, needsSpace bool) {
	spaced := func(j int) bool {
		if j < 0 || j >= s.Len() {
			return false
		}
		k := s.At(j).Kind
		return k == token.Whitespace || k == token.Comment || k == token.DocComment
	}
	if needsSpace && !spaced(i-1) && !spaced(i+1) {
		s.Set(i, token.Token{Kind: token.Whitespace, Value: " "})
		return
	}
	s.RemoveAt(i)
	if i > 0 && i < s.Len() && s.Kind(i-1) == token.Whitespace && s.Kind(i) == token.Whitespace {
		s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
		s.RemoveAt(i)
	}
}
