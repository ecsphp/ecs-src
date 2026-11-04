package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NullableTypeDeclarationFixer.php
//
// NullableTypeDeclaration standardises a nullable single type to the question
// mark syntax (the fixer default): "int|null" and "null|int" become "?int".
// With `syntax: union` it goes the other way: "?int" becomes "null|int".
// Like the PHP fixer, it only looks at parameter and return types and at
// class properties.
type NullableTypeDeclaration struct {
	// syntax is "question_mark" (the default when empty) or "union".
	syntax string
}

func (NullableTypeDeclaration) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\NullableTypeDeclarationFixer`
}

func (NullableTypeDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NullableTypeDeclarationFixer.php"
}

func (f NullableTypeDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.syntax = v
	}
	return f
}

// ntdElement is one entry of NullableTypeDeclarationFixer::getElements: a
// property (index of its variable) or a function / fn keyword.
type ntdElement struct {
	index    int
	property bool
}

// IsCandidate rules out a file without the trigger: the question_mark syntax only rewrites a union with "null", the union one any "?".
func (f NullableTypeDeclaration) IsCandidate(s *tokens.Stream) bool {
	return f.syntax == "union" || s.Index().HasWord("null")
}

func (f NullableTypeDeclaration) Fix(s *tokens.Stream) bool {
	union := f.syntax == "union"
	elements := ntdElements(s)
	changed := false
	// Right-to-left, as the PHP fixer does, so a rewrite never shifts an
	// element still to be visited.
	for _, e := range slices.Backward(elements) {

		if e.property {
			changed = ntdNormalize(s, ntdPropertyType(s, e.index), union) || changed
			continue
		}
		open := ntdFunctionParamsOpen(s, e.index)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		changed = ntdNormalize(s, ntdReturnType(s, closeIdx), union) || changed
		args := ntdArguments(s, open, closeIdx)
		for _, arg := range slices.Backward(args) {
			changed = ntdNormalize(s, ntdArgumentType(s, arg[0], arg[1]), union) || changed
		}
	}
	return changed
}

// ntdElements lists the class properties (TokensAnalyzer::getClassyElements)
// and every function / fn keyword, in source order.
func ntdElements(s *tokens.Stream) []ntdElement {
	properties := map[int]bool{}
	for i := 0; i < s.Len(); i++ {
		if ntdIsClassy(s, i) {
			ntdCollectProperties(s, i, properties)
		}
	}
	var out []ntdElement
	for i := 0; i < s.Len(); i++ {
		switch {
		case properties[i]:
			out = append(out, ntdElement{index: i, property: true})
		case kwIs(s.At(i), "function") || kwIs(s.At(i), "fn"):
			out = append(out, ntdElement{index: i})
		}
	}
	return out
}

func ntdIsClassy(s *tokens.Stream, i int) bool {
	t := s.At(i)
	return kwIs(t, "class") || kwIs(t, "interface") || kwIs(t, "trait") || caseStructureKind(s, i) == "enum"
}

// ntdCollectProperties marks the variables declared at the top level of the
// body of the class-like at classIdx (not inside parentheses, a method body
// or a property hook block).
func ntdCollectProperties(s *tokens.Stream, classIdx int, properties map[int]bool) {
	open := -1
	for j := classIdx + 1; j < s.Len() && open < 0; j++ {
		switch {
		case isPunctVal(s, j, "("): // anonymous class arguments
			end := s.MatchForward(j)
			if end < 0 {
				return
			}
			j = end
		case isPunctVal(s, j, "{"):
			open = j
		case isPunctVal(s, j, ";"):
			return
		}
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return
	}
	curly, parens := 0, 0
	for j := open + 1; j < closeIdx; j++ {
		switch {
		case s.Kind(j) == token.AttributeStart:
			if end := s.AttributeEnd(j); end > j {
				j = end
			}
		case isPunctVal(s, j, "("):
			parens++
		case isPunctVal(s, j, ")"):
			parens--
		case isPunctVal(s, j, "{"):
			curly++
		case isPunctVal(s, j, "}"):
			curly--
		case curly == 0 && parens == 0 && s.Kind(j) == token.Variable:
			properties[j] = true
		}
	}
}

// ntdFunctionParamsOpen returns the "(" opening the parameter list of the
// function / fn keyword at fi ("function [&] [name] (", "fn [&] ("), or -1.
func ntdFunctionParamsOpen(s *tokens.Stream, fi int) int {
	j := sigNext(s, fi)
	if isPunctVal(s, j, "&") {
		j = sigNext(s, j)
	}
	if j >= 0 && (s.Kind(j) == token.Ident || s.Kind(j) == token.Keyword) {
		j = sigNext(s, j)
	}
	if !isPunctVal(s, j, "(") {
		return -1
	}
	return j
}

// ntdReturnType mirrors FunctionsAnalyzer::getFunctionReturnType: the
// significant tokens of the type after the ":" that follows the parameter list
// (or the closure's "use (...)"), up to the body, ";" or "=>".
func ntdReturnType(s *tokens.Stream, closeIdx int) []int {
	colon := sigNext(s, closeIdx)
	if colon >= 0 && kwIs(s.At(colon), "use") {
		useClose := s.MatchForward(nextPunctOfKind(s, colon, "("))
		if useClose < 0 {
			return nil
		}
		colon = sigNext(s, useClose)
	}
	if !isPunctVal(s, colon, ":") {
		return nil
	}
	var idxs []int
	for j := colon + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, "{") || isPunctVal(s, j, ";") || isPunctVal(s, j, "=>") {
			break
		}
		if !isTrivia(s, j) {
			idxs = append(idxs, j)
		}
	}
	return idxs
}

// ntdArguments mirrors ArgumentsAnalyzer::getArguments: the [start, end]
// token ranges of the parameters between open and closeIdx.
func ntdArguments(s *tokens.Stream, open, closeIdx int) [][2]int {
	if sigNext(s, open) == closeIdx {
		return nil
	}
	var args [][2]int
	start := open + 1
	j := open + 1
	for ; j < closeIdx; j++ {
		if s.Kind(j) == token.AttributeStart {
			if end := s.AttributeEnd(j); end > j {
				j = end
			}
			continue
		}
		if isPunctVal(s, j, "(") || isPunctVal(s, j, "[") || isPunctVal(s, j, "{") {
			if end := s.MatchForward(j); end > j {
				j = end
			}
			continue
		}
		if isPunctVal(s, j, ",") {
			if sigNext(s, j) == closeIdx {
				break // trailing comma
			}
			args = append(args, [2]int{start, j - 1})
			start = j + 1
		}
	}
	return append(args, [2]int{start, j - 1})
}

// ntdArgumentType mirrors ArgumentsAnalyzer::getArgumentInfo: the tokens
// before the parameter's variable, minus attributes, modifiers, a by-reference
// "&" and "...".
// A parameter without a variable has no type.
func ntdArgumentType(s *tokens.Stream, start, end int) []int {
	var idxs []int
	for j := start; j <= end; j++ {
		switch {
		case s.Kind(j) == token.AttributeStart:
			if e := s.AttributeEnd(j); e > j {
				j = e
			}
			continue
		case isTrivia(s, j), isPunctVal(s, j, "..."), ntdIsReferenceAmp(s, j):
			continue
		case s.Kind(j) == token.Variable:
			return idxs
		}
		if ntdIsArgumentModifier(s, j) {
			continue
		}
		idxs = append(idxs, j)
	}
	return nil
}

// ntdIsReferenceAmp reports whether j is a by-reference "&" ("int &$x",
// "int &...$x"). An intersection "&" ("(A&B)|null") is CT::T_TYPE_INTERSECTION
// in PHP-CS-Fixer, which getArgumentInfo keeps as part of the type.
func ntdIsReferenceAmp(s *tokens.Stream, j int) bool {
	if !isPunctVal(s, j, "&") {
		return false
	}
	n := sigNext(s, j)
	return n >= 0 && (s.Kind(n) == token.Variable || isPunctVal(s, n, "..."))
}

// ntdIsArgumentModifier reports whether j is a promoted-property modifier
// (ARGUMENT_INFO_SKIP_TYPES: final, public, protected, private, readonly and
// the asymmetric "*(set)" ones, which the lexer emits as one token).
func ntdIsArgumentModifier(s *tokens.Stream, j int) bool {
	switch olKwLower(s.At(j)) {
	case "final", "readonly", "public", "protected", "private", "private(set)", "protected(set)", "public(set)":
		return true
	}
	return false
}

// ntdPropertyType mirrors normalizePropertyType's type collection: the tokens
// between the nearest property modifier before the variable and the variable.
func ntdPropertyType(s *tokens.Stream, varIdx int) []int {
	mod := sigPrev(s, varIdx)
	for ; mod >= 0; mod = sigPrev(s, mod) {
		if ntdIsPropertyModifier(s, mod) {
			break
		}
		if isPunctVal(s, mod, "{") || isPunctVal(s, mod, "}") || isPunctVal(s, mod, ";") {
			return nil
		}
	}
	if mod < 0 {
		return nil
	}
	var idxs []int
	for j := mod + 1; j < varIdx; j++ {
		if !isTrivia(s, j) {
			idxs = append(idxs, j)
		}
	}
	return idxs
}

// ntdIsPropertyModifier reports whether j is one of PROPERTY_MODIFIERS.
func ntdIsPropertyModifier(s *tokens.Stream, j int) bool {
	switch olKwLower(s.At(j)) {
	case "private", "protected", "public", "static", "abstract", "final", "var", "readonly",
		"private(set)", "protected(set)", "public(set)":
		return true
	}
	return false
}

// ntdNormalize ports isTypeNormalizable and normalizeNullableType for the type
// made of the significant tokens idxs.
func ntdNormalize(s *tokens.Stream, idxs []int, union bool) bool {
	if len(idxs) == 0 {
		return false
	}
	var b strings.Builder
	for _, j := range idxs {
		b.WriteString(s.At(j).Value)
	}
	name, nullable := b.String(), false
	if strings.HasPrefix(name, "?") {
		name, nullable = name[1:], true
	} else {
		for m := range strings.SplitSeq(strings.ToLower(name), "|") {
			if strings.TrimSpace(m) == "null" {
				nullable = true
			}
		}
	}
	if !nullable || strings.EqualFold(name, "null") || strings.Contains(name, "&") {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(name, "|") &&
		(strings.Count(name, "|") != 1 || (!strings.HasSuffix(lower, "|null") && !strings.HasPrefix(lower, "null|"))) {
		return false
	}

	var repl []token.Token
	if union {
		if strings.Contains(name, "|") {
			return false // already a union
		}
		// "?T" -> "null|T"; idxs[0] is the "?"
		repl = []token.Token{{Kind: token.Ident, Value: "null"}, {Kind: token.Punct, Value: "|"}}
		for _, j := range idxs[1:] {
			repl = append(repl, s.At(j))
		}
	} else {
		if !strings.Contains(name, "|") {
			return false // already "?T"
		}
		// "T|null" / "null|T" -> "?T": drop the null member and its "|"
		pipe := 0
		for pipe < len(idxs) && !isPunctVal(s, idxs[pipe], "|") {
			pipe++
		}
		keep := idxs[:pipe]
		if !strings.HasSuffix(lower, "|null") {
			keep = idxs[pipe+1:]
		}
		repl = []token.Token{{Kind: token.Punct, Value: "?"}}
		for _, j := range keep {
			repl = append(repl, s.At(j))
		}
	}
	start := idxs[0]
	s.ReplaceRange(start, idxs[len(idxs)-1], repl)
	if start > 0 && s.Kind(start-1) != token.Whitespace && !isPunctVal(s, start-1, "(") {
		s.InsertAt(start, token.Token{Kind: token.Whitespace, Value: " "})
	}
	return true
}
