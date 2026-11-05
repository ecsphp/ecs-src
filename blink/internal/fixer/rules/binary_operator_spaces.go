package rules

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php

// The fix strategies of BinaryOperatorSpacesFixer.
const (
	bosSingleSpace                    = "single_space"
	bosAtLeastSingleSpace             = "at_least_single_space"
	bosNoSpace                        = "no_space"
	bosAlign                          = "align"
	bosAlignByScope                   = "align_by_scope"
	bosAlignSingleSpace               = "align_single_space"
	bosAlignSingleSpaceByScope        = "align_single_space_by_scope"
	bosAlignSingleSpaceMinimal        = "align_single_space_minimal"
	bosAlignSingleSpaceMinimalByScope = "align_single_space_minimal_by_scope"
)

// bosSupportedOperators is SUPPORTED_OPERATORS: the operators "default" applies
// to and the only keys "operators" accepts.
var bosSupportedOperators = []string{
	"=", "*", "/", "%", "<", ">", "|", "^", "+", "-", "&", "&=", "&&", "||", ".=", "/=", "=>", "==", ">=",
	"===", "!=", "<>", "!==", "<=", "and", "or", "xor", "-=", "%=", "*=", "|=", "+=", "<<", "<<=", ">>",
	">>=", "^=", "**", "**=", "<=>", "??", "??=",
}

var bosStrategies = map[string]bool{
	bosSingleSpace: true, bosAtLeastSingleSpace: true, bosNoSpace: true, bosAlign: true, bosAlignByScope: true,
	bosAlignSingleSpace: true, bosAlignSingleSpaceByScope: true, bosAlignSingleSpaceMinimal: true,
	bosAlignSingleSpaceMinimalByScope: true,
}

// bosDefaultOperators is the configuration without options: every supported
// operator single spaced.
var bosDefaultOperators = bosResolve(bosSingleSpace, nil)

// BinaryOperatorSpaces ports BinaryOperatorSpacesFixer: binary operators are
// surrounded by space as configured, or aligned in columns.
//
// Options, as PHP-CS-Fixer: "default" is the strategy of every supported
// operator (single_space when unset, null leaving them all alone), and
// "operators" maps an operator to its own strategy, null leaving that operator
// alone. The strategies are single_space, at_least_single_space, no_space,
// align, align_by_scope, align_single_space, align_single_space_by_scope,
// align_single_space_minimal and align_single_space_minimal_by_scope. Where
// PHP rejects the configuration (an unsupported operator or strategy), blink
// ignores that entry.
type BinaryOperatorSpaces struct {
	// operators maps each handled operator to its strategy, as
	// resolveOperatorsFromConfig; nil is the default configuration.
	operators map[string]string
}

func (BinaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\BinaryOperatorSpacesFixer`
}

func (BinaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php"
}

func (f BinaryOperatorSpaces) WithConfig(config map[string]any) fixer.Fixer {
	def := bosSingleSpace
	if v, ok := config["default"]; ok {
		if v == nil {
			def = ""
		} else if str, ok := v.(string); ok && bosStrategies[str] {
			def = str
		}
	}
	ops, _ := config["operators"].(map[string]any)
	f.operators = bosResolve(def, ops)
	return f
}

// bosResolve is resolveOperatorsFromConfig: the default strategy ("" for null)
// for every supported operator, then the per-operator overrides, null removing
// the operator.
func bosResolve(def string, overrides map[string]any) map[string]string {
	ops := make(map[string]string, len(bosSupportedOperators))
	if def != "" {
		for _, op := range bosSupportedOperators {
			ops[op] = def
		}
	}
	for op, v := range overrides {
		if !bosIsSupported(op) {
			continue
		}
		if v == nil {
			delete(ops, op)
			continue
		}
		if str, ok := v.(string); ok && bosStrategies[str] {
			ops[op] = str
		}
	}
	return ops
}

func bosIsSupported(op string) bool {
	for _, o := range bosSupportedOperators {
		if o == op {
			return true
		}
	}
	return false
}

func bosIsAlignSingleSpace(strategy string) bool {
	switch strategy {
	case bosAlignSingleSpace, bosAlignSingleSpaceMinimal, bosAlignSingleSpaceByScope, bosAlignSingleSpaceMinimalByScope:
		return true
	}
	return false
}

func bosIsMinimal(strategy string) bool {
	return strategy == bosAlignSingleSpaceMinimal || strategy == bosAlignSingleSpaceMinimalByScope
}

func bosIsByScope(strategy string) bool {
	switch strategy {
	case bosAlignByScope, bosAlignSingleSpaceByScope, bosAlignSingleSpaceMinimalByScope:
		return true
	}
	return false
}

// bosFixer holds one run's state, as the PHP fixer's properties.
type bosFixer struct {
	s   *tokens.Stream
	ops map[string]string
	// alignOps and alignStrategy are alignOperatorTokens: the operators to
	// align, in the order the backward scan met them.
	alignOps      []string
	alignStrategy map[string]string
	deepestLevel  int
	currentLevel  int
}

// Fix ports applyFix: right to left, every binary operator is spaced as its
// strategy says or scheduled for alignment, then each scheduled operator is
// aligned.
func (f BinaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	ops := f.operators
	if ops == nil {
		ops = bosDefaultOperators
	}
	if len(ops) == 0 {
		return false
	}
	b := &bosFixer{s: s, ops: ops}
	changed := false
	// last and first tokens cannot be an operator
	for index := s.Len() - 2; index > 0; index-- {
		if !bosIsBinaryOperator(s, index) {
			continue
		}
		if s.At(index).Value == "=" {
			if d := bosDeclareOfEqual(s, index); d >= 0 {
				// skip `declare(foo ==bar)`, see `declare_equal_normalize`
				index = d
			} else if b.fixAround(index) {
				changed = true
			}
		} else if b.fixAround(index) {
			changed = true
		}
		// previous of binary operator is now never an operator
		index--
	}
	s.ClearEmptyTokens()
	if len(b.alignOps) > 0 && b.fixAlignment() {
		changed = true
	}
	return changed
}

// fixAround is fixWhiteSpaceAroundOperator.
func (b *bosFixer) fixAround(index int) bool {
	s := b.s
	op := strings.ToLower(s.At(index).Value)
	strategy, ok := b.ops[op]
	if !ok {
		return false // not configured to be changed
	}
	switch strategy {
	case bosSingleSpace:
		return bosFixSingleSpace(s, index)
	case bosAtLeastSingleSpace:
		changed := false
		if !bosIsWhitespace(s, index+1) {
			s.InsertAt(index+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
		if !bosIsWhitespace(s, index-1) {
			s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
		return changed
	case bosNoSpace:
		// whitespace is cleared (left empty, as PHP's clearAt) so indexes hold
		changed := false
		if bosIsWhitespace(s, index+1) {
			v := s.At(index + 1).Value
			if v != "" && !strings.Contains(v, "\n") && !bosIsCommentAt(s, bosNextNonWhitespace(s, index+1)) {
				s.SetValue(index+1, "")
				changed = true
			}
		}
		if bosIsWhitespace(s, index-1) {
			v := s.At(index - 1).Value
			if v != "" && !strings.Contains(v, "\n") && !bosIsCommentAt(s, bosPrevNonWhitespace(s, index-1)) {
				s.SetValue(index-1, "")
				changed = true
			}
		}
		return changed
	}
	// schedule for alignment
	if b.alignStrategy == nil {
		b.alignStrategy = map[string]string{}
	}
	if _, seen := b.alignStrategy[op]; !seen {
		b.alignOps = append(b.alignOps, op)
	}
	b.alignStrategy[op] = strategy
	if strategy == bosAlign || strategy == bosAlignByScope {
		return false
	}
	// fix white space after operator (a newline included, for the minimal ones)
	if bosIsWhitespace(s, index+1) {
		if bosIsMinimal(strategy) && s.At(index+1).Value != " " {
			s.SetValue(index+1, " ")
			return true
		}
		return false
	}
	s.InsertAt(index+1, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}

// bosFixSingleSpace is fixWhiteSpaceAroundOperatorToSingleSpace: whitespace
// holding a newline or next to a comment is kept.
func bosFixSingleSpace(s *tokens.Stream, index int) bool {
	changed := false
	if bosIsWhitespace(s, index+1) {
		v := s.At(index + 1).Value
		if v != " " && !strings.Contains(v, "\n") && !bosIsCommentAt(s, bosNextNonWhitespace(s, index+1)) {
			s.SetValue(index+1, " ")
			changed = true
		}
	} else {
		s.InsertAt(index+1, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	if bosIsWhitespace(s, index-1) {
		v := s.At(index - 1).Value
		if v != " " && !strings.Contains(v, "\n") && !bosIsCommentAt(s, bosPrevNonWhitespace(s, index-1)) {
			s.SetValue(index-1, " ")
			changed = true
		}
	} else {
		s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}

func bosIsWhitespace(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Whitespace
}

func bosIsCommentAt(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	k := s.At(i).Kind
	return k == token.Comment || k == token.DocComment
}

func bosNextNonWhitespace(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

func bosPrevNonWhitespace(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

// bosIsBinaryOperator mirrors TokensAnalyzer::isBinaryOperator.
func bosIsBinaryOperator(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return !taIsNamedArgument(s, i)
		}
		return false
	case token.Punct:
	default:
		return false
	}
	switch t.Value {
	case "+", "-":
		return !unaryOperatorSpacesIsPredecessor(s, i)
	case "&":
		// CT::T_TYPE_INTERSECTION is not an operator
		if bosIsTypeOperator(s, i) {
			return false
		}
		return !unaryOperatorSpacesIsPredecessor(s, i)
	case "|":
		// CT::T_TYPE_ALTERNATION is not an operator
		return !bosIsTypeOperator(s, i)
	case "=", "*", "/", "%", "<", ">", "^", ".",
		"&=", "&&", "||", ".=", "/=", "=>", "==", ">=", "===", "!=", "<>", "!==", "<=", "-=", "%=", "*=",
		"|=", "+=", "**", "**=", "<<", "<<=", ">>", ">>=", "^=", "<=>", "??", "??=":
		return true
	}
	return false
}

// bosIsTypeOperator reports whether the "|" or "&" at i joins types
// (CT::T_TYPE_ALTERNATION, CT::T_TYPE_INTERSECTION), as TypeAlternationTransformer
// and TypeIntersectionTransformer decide. An intersection "&" is only a "&" not
// followed by a variable or "..." (T_AMPERSAND_NOT_FOLLOWED_BY_VAR_OR_VARARG).
func bosIsTypeOperator(s *tokens.Stream, i int) bool {
	if s.At(i).Value == "&" {
		n := sigNext(s, i)
		if n >= 0 && (s.At(n).Kind == token.Variable || isPunctVal(s, n, "...")) {
			return false
		}
	}
	return bosIsPartOfType(s, i)
}

// bosIsPartOfType is AbstractTypeTransformer::isPartOfType.
func bosIsPartOfType(s *tokens.Stream, index int) bool {
	// return types and non-capturing catches
	before := index - 1
	for before >= 0 && bosIsTypeToken(s, before) {
		before--
	}
	if before >= 0 && (kwIs(s.At(before), "catch") || kwIs(s.At(before), "const") || bosIsTypeColon(s, before)) {
		return true
	}
	// for parameter there will be splat operator or variable after the type
	after := index + 1
	for after < s.Len() && bosIsTypeToken(s, after) {
		after++
	}
	if after >= s.Len() {
		return false
	}
	if isPunctVal(s, after, "...") {
		return true
	}
	if !bosIsVariable(s.At(after)) {
		return false
	}
	beforeVariable := sigPrev(s, after)
	if isPunctVal(s, beforeVariable, "&") {
		// "&" is ambiguous: a reference after a parameter type, or bitwise and
		for j := index - 1; j >= 0; j-- {
			t := s.At(j)
			switch {
			case kwIs(t, "fn"), kwIs(t, "function"):
				return true
			case t.Kind == token.CloseTag, t.Kind == token.Punct && (t.Value == "{" || t.Value == "}" || t.Value == ";"):
				return false
			}
		}
		return false
	}
	return beforeVariable >= 0 && bosIsTypeEndToken(s, beforeVariable)
}

// bosIsVariable reports a T_VARIABLE ("$name", not the "$" of "${...}").
func bosIsVariable(t token.Token) bool {
	return t.Kind == token.Variable && len(t.Value) > 1
}

// bosIsTypeEndToken is AbstractTypeTransformer's TYPE_END_TOKENS: ")",
// T_CALLABLE, T_NS_SEPARATOR, T_STATIC, T_STRING and CT::T_ARRAY_TYPEHINT.
func bosIsTypeEndToken(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Punct:
		return t.Value == ")" || t.Value == `\`
	case token.Ident:
		return taIsTString(s, i)
	case token.Keyword:
		if kwIs(t, "array") {
			return !isPunctVal(s, sigNext(s, i), "(")
		}
		// static, callable, and the type names blink lexes as keywords
		return nativeTypeNames[strings.ToLower(t.Value)]
	}
	return false
}

// bosIsTypeToken is AbstractTypeTransformer's TYPE_TOKENS: "|", "&", "(", the
// end tokens, whitespace and comments.
func bosIsTypeToken(s *tokens.Stream, i int) bool {
	switch t := s.At(i); t.Kind {
	case token.Whitespace, token.Comment, token.DocComment:
		return true
	case token.Punct:
		if t.Value == "|" || t.Value == "&" || t.Value == "(" {
			return true
		}
	}
	return bosIsTypeEndToken(s, i)
}

// bosIsTypeColon mirrors TypeColonTransformer: the ":" of a return type (after
// the parameters of a function, a closure's use clause or an fn) or of an
// enum's backing type.
func bosIsTypeColon(s *tokens.Stream, i int) bool {
	if !isPunctVal(s, i, ":") {
		return false
	}
	end := sigPrev(s, i)
	if end < 0 {
		return false
	}
	if p := sigPrev(s, end); p >= 0 && kwIs(s.At(p), "enum") {
		return true
	}
	if !isPunctVal(s, end, ")") {
		return false
	}
	start := s.MatchBackward(end)
	if start < 0 {
		return false
	}
	prev := sigPrev(s, start)
	// if this could be a function name we need to take one more step
	if prev >= 0 && s.At(prev).Kind == token.Ident && taIsTString(s, prev) {
		prev = sigPrev(s, prev)
	}
	if prev < 0 {
		return false
	}
	t := s.At(prev)
	if kwIs(t, "function") || kwIs(t, "fn") || kwIs(t, "use") {
		return true
	}
	// CT::T_RETURN_REF: "function &" or "fn &"
	if isPunctVal(s, prev, "&") {
		r := sigPrev(s, prev)
		return r >= 0 && (kwIs(s.At(r), "function") || kwIs(s.At(r), "fn"))
	}
	return false
}

// bosDeclareOfEqual is isEqualPartOfDeclareStatement: the index of the
// "declare" when the "=" at index follows "declare(name", else -1. A second
// directive ("declare(a=1, b=2)") is not skipped, as in PHP.
func bosDeclareOfEqual(s *tokens.Stream, index int) int {
	p := sigPrev(s, index)
	if p < 0 || !taIsTString(s, p) {
		return -1
	}
	p = sigPrev(s, p)
	if !isPunctVal(s, p, "(") {
		return -1
	}
	p = sigPrev(s, p)
	if p >= 0 && kwIs(s.At(p), "declare") {
		return p
	}
	return -1
}

// fixAlignment aligns each scheduled operator in turn. As PHP, it injects a
// placeholder before the operators to align into a copy of the tokens, renders
// it, pads the placeholders into columns line by line, and re-tokenizes.
func (b *bosFixer) fixAlignment() bool {
	b.deepestLevel = 0
	b.currentLevel = 0
	changed := false
	for _, op := range b.alignOps {
		strategy := b.alignStrategy[op]
		s := b.s
		clone := tokens.New(append([]token.Token(nil), s.Tokens()...))
		sq := bosSquareKinds(s)
		if op == "=>" {
			b.injectArrow(clone, sq, 0, s.Len())
		} else {
			b.injectDefault(clone, sq, 0, s.Len(), op)
		}
		// for all tokens that should be aligned but do not have anything to
		// align with, fix spacing if needed
		if bosIsAlignSingleSpace(strategy) {
			for index := s.Len() - 2; index > 0; index-- {
				t := s.At(index)
				var match bool
				switch op {
				case "=>":
					match = t.Kind == token.Punct && t.Value == "=>"
				case "=":
					match = t.Kind == token.Punct && t.Value == "=" && bosDeclareOfEqual(s, index) < 0 && bosIsBinaryOperator(s, index)
				default:
					match = strings.ToLower(t.Value) == op && bosIsBinaryOperator(s, index)
				}
				if match {
					bosFixWhiteSpaceBefore(clone, index, strategy)
				}
			}
		}
		code := bosReplacePlaceholders(clone.Render(), strategy, b.deepestLevel)
		if code != s.Render() {
			s.ReplaceRange(0, s.Len()-1, lexer.Lex(code))
			changed = true
		}
	}
	return changed
}

// bosPlaceholder is ALIGN_PLACEHOLDER for a level.
func bosPlaceholder(level int) string {
	return "\x02 ALIGNABLE" + strconv.Itoa(level) + " \x03"
}

// injectDefault is injectAlignmentPlaceholdersDefault: a placeholder before the
// first operator of each line, at a deeper level inside each multi-line block.
func (b *bosFixer) injectDefault(clone *tokens.Stream, sq []uint8, startAt, endAt int, op string) {
	s := b.s
	newLineFound := true
	for index := startAt; index < endAt; index++ {
		t := s.At(index)
		if strings.Contains(clone.At(index).Value, "\n") {
			newLineFound = true
		}
		if newLineFound && strings.ToLower(t.Value) == op && bosIsBinaryOperator(s, index) &&
			(t.Value != "=" || bosDeclareOfEqual(s, index) < 0) {
			clone.SetValue(index, bosPlaceholder(b.currentLevel)+t.Value)
			newLineFound = false
			continue
		}
		if kwIs(t, "fn") {
			from := sigNext(s, index)
			until := taArrowFunctionEnd(s, index)
			if from < 0 || until < 0 {
				continue
			}
			b.injectNestedDefault(clone, sq, from+1, until-1, op)
			index = until
			continue
		}
		if kwIs(t, "function") || taIsClassKw(s, index) {
			index = bosNextTokenOfKind(s, index)
			if index < 0 {
				return
			}
			// We don't align `=` on multi-line definition of function parameters with default values
			if isPunctVal(s, index, "(") {
				if end := s.MatchForward(index); end >= 0 {
					index = end
				}
				continue
			}
			if isPunctVal(s, index, ";") {
				continue
			}
			t = s.At(index)
		}
		if t.Kind != token.Punct {
			continue
		}
		var until int
		switch {
		case t.Value == "{" && bosIsPlainBrace(s, index),
			t.Value == "(" && !bosIsClassInstantiationParen(s, index),
			t.Value == "[" && sq[index] == bosSquareArray:
			until = s.MatchForward(index)
			if until < 0 {
				continue
			}
			b.injectNestedDefault(clone, sq, index+1, until-1, op)
		case t.Value == "[" && sq[index] == bosSquareIndex:
			until = s.MatchForward(index)
			if until < 0 {
				continue
			}
		default:
			continue
		}
		index = until
	}
}

// injectNestedDefault is injectAlignmentPlaceholders: only multi-line code
// gets placeholders, at a new level.
func (b *bosFixer) injectNestedDefault(clone *tokens.Stream, sq []uint8, from, until int, op string) {
	if bosIsMultiline(clone, from, until) {
		b.deepestLevel++
		current := b.currentLevel
		b.currentLevel = b.deepestLevel
		b.injectDefault(clone, sq, from, until, op)
		b.currentLevel = current
	}
}

// injectArrow is injectAlignmentPlaceholdersForArrow: a placeholder before the
// first "=>" of each line, at a deeper level inside each multi-line array,
// block or parenthesis and after each ";" or "yield"; the whitespace after
// the arrow becomes one space unless it holds a newline.
func (b *bosFixer) injectArrow(clone *tokens.Stream, sq []uint8, startAt, endAt int) {
	s := b.s
	newLineFound := true
	yieldFound := false
	for index := startAt; index < endAt; index++ {
		t := s.At(index)
		if strings.Contains(clone.At(index).Value, "\n") {
			newLineFound = true
		}
		if kwIs(t, "yield") {
			yieldFound = true
		}
		if kwIs(t, "fn") {
			yieldFound = false
			from := sigNext(s, index)
			until := taArrowFunctionEnd(s, index)
			if from < 0 || until < 0 {
				continue
			}
			b.injectNestedArrow(clone, sq, from+1, until-1)
			index = until
			continue
		}
		// array( and the short array [, not an index nor a destructuring
		if bosIsArrayKeyword(s, index) || (isPunctVal(s, index, "[") && sq[index] == bosSquareArray) {
			yieldFound = false
			from := index
			if t.Kind == token.Keyword {
				from = sigNext(s, index)
			}
			until := s.MatchForward(from)
			if until < 0 {
				continue
			}
			index = until
			b.injectNestedArrow(clone, sq, from+1, until-1)
			continue
		}
		if t.Kind != token.Punct {
			continue
		}
		// there is no need to analyse the second arrow of a line
		if t.Value == "=>" && newLineFound {
			if yieldFound {
				b.deepestLevel++
				b.currentLevel = b.deepestLevel
			}
			content := bosPlaceholder(b.currentLevel) + t.Value
			if !bosIsWhitespace(clone, index+1) {
				content += " "
			} else if strings.Trim(clone.At(index+1).Value, " \t") == "" {
				clone.SetValue(index+1, " ")
			}
			clone.SetValue(index, content)
			newLineFound = false
			yieldFound = false
			continue
		}
		switch t.Value {
		case ";":
			b.deepestLevel++
			b.currentLevel++
		case ",":
			// skip to the end of the line, the next arrow or a multi-line array
			for i := index; i < endAt-1; i++ {
				if strings.Contains(clone.At(i).Value, "\n") {
					newLineFound = true
					break
				}
				if isPunctVal(s, i+1, "=>") {
					break
				}
				if bosIsArrayKeyword(s, i+1) || (isPunctVal(s, i+1, "[") && sq[i+1] == bosSquareArray) {
					start := i + 1
					if bosIsArrayKeyword(s, i+1) {
						start = sigNext(s, i+1)
					}
					if end := s.MatchForward(start); end >= 0 && bosIsMultiline(clone, start, end) {
						break
					}
				}
				index++
			}
		case "{", "(":
			if (t.Value == "{" && !bosIsPlainBrace(s, index)) || (t.Value == "(" && bosIsClassInstantiationParen(s, index)) {
				continue
			}
			until := s.MatchForward(index)
			if until < 0 {
				continue
			}
			b.injectNestedArrow(clone, sq, index+1, until-1)
			index = until
		}
	}
}

// injectNestedArrow is injectArrayAlignmentPlaceholders.
func (b *bosFixer) injectNestedArrow(clone *tokens.Stream, sq []uint8, from, until int) {
	if bosIsMultiline(clone, from, until) {
		b.deepestLevel++
		current := b.currentLevel
		b.currentLevel = b.deepestLevel
		b.injectArrow(clone, sq, from, until)
		b.currentLevel = current
	}
}

// bosIsMultiline is Tokens::isPartialCodeMultiline over [start, end].
func bosIsMultiline(s *tokens.Stream, start, end int) bool {
	for i := max(start, 0); i <= end && i < s.Len(); i++ {
		if strings.Contains(s.At(i).Value, "\n") {
			return true
		}
	}
	return false
}

// bosNextTokenOfKind is getNextTokenOfKind($index, ['{', ';', '(']): the next
// plain brace, semicolon or plain parenthesis, or -1.
func bosNextTokenOfKind(s *tokens.Stream, index int) int {
	for j := index + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch {
		case t.Value == ";",
			t.Value == "{" && bosIsPlainBrace(s, j),
			t.Value == "(" && !bosIsClassInstantiationParen(s, j):
			return j
		}
	}
	return -1
}

// bosFixWhiteSpaceBefore is fixWhiteSpaceBeforeOperator.
func bosFixWhiteSpaceBefore(s *tokens.Stream, index int, strategy string) {
	if !bosIsWhitespace(s, index-1) {
		s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: " "})
		return
	}
	if !bosIsMinimal(strategy) || bosIsCommentAt(s, bosPrevNonWhitespace(s, index-1)) {
		return
	}
	if v := s.At(index - 1).Value; v != " " && !strings.Contains(v, "\n") {
		s.SetValue(index-1, " ")
	}
}

// bosReplacePlaceholders is replacePlaceholders: level by level, the
// placeholders of consecutive lines (of the whole level for the *_by_scope
// strategies) are padded to the rightmost one, then removed.
func bosReplacePlaceholders(code, strategy string, deepestLevel int) string {
	for j := 0; j <= deepestLevel; j++ {
		placeholder := bosPlaceholder(j)
		if !strings.Contains(code, placeholder) {
			continue
		}
		lines := strings.Split(code, "\n")
		groups := [][]int{nil}
		for index, line := range lines {
			if strings.Contains(line, placeholder) {
				groups[len(groups)-1] = append(groups[len(groups)-1], index)
			} else if !bosIsByScope(strategy) {
				groups = append(groups, nil)
			}
		}
		for _, group := range groups {
			if len(group) == 0 {
				continue
			}
			if strategy != bosAlign {
				// move placeholders to match strategy
				for _, index := range group {
					pos := strings.Index(lines[index], placeholder)
					before := lines[index][:pos]
					switch {
					case strategy == bosAlignSingleSpace || strategy == bosAlignSingleSpaceByScope:
						if !strings.HasSuffix(before, " ") {
							before += " "
						}
					case bosIsMinimal(strategy):
						// if indent, do not move, leave to other fixer
						if !bosIsHorizontalSpace(before) {
							before = strings.TrimRight(before, " \t\n\r\x00\x0B") + " "
						}
					}
					lines[index] = before + lines[index][pos:]
				}
			}
			rightmost := 0
			for _, index := range group {
				rightmost = max(rightmost, bosSubstringWidth(lines[index], placeholder))
			}
			for _, index := range group {
				if delta := rightmost - bosSubstringWidth(lines[index], placeholder); delta > 0 {
					lines[index] = strings.ReplaceAll(lines[index], placeholder, strings.Repeat(" ", delta)+placeholder)
				}
			}
		}
		code = strings.ReplaceAll(strings.Join(lines, "\n"), placeholder, "")
	}
	return code
}

// bosIsHorizontalSpace is Preg::match('/^\h+$/', $s), which uses the u modifier
// on valid UTF-8.
func bosIsHorizontalSpace(s string) bool {
	if s == "" {
		return false
	}
	if !utf8.ValidString(s) {
		for i := 0; i < len(s); i++ {
			if c := s[i]; c != ' ' && c != '\t' && c != 0xA0 {
				return false
			}
		}
		return true
	}
	for _, r := range s {
		switch {
		case r == ' ', r == '\t', r == 0xA0, r == 0x1680, r == 0x180E, r >= 0x2000 && r <= 0x200A,
			r == 0x202F, r == 0x205F, r == 0x3000:
		default:
			return false
		}
	}
	return true
}

// bosSubstringWidth is getSubstringWidth: the display width (mb_strwidth) of
// haystack up to needle.
func bosSubstringWidth(haystack, needle string) int {
	return mbStrWidth(haystack[:strings.Index(haystack, needle)])
}

// The kinds of "[" PHP-CS-Fixer's SquareBraceTransformer tells apart.
const (
	bosSquareIndex         uint8 = iota // a plain "[" (an index), or no "[" at all
	bosSquareArray                      // CT::T_ARRAY_BRACKET_OPEN
	bosSquareDestructuring              // CT::T_DESTRUCTURING_BRACKET_OPEN
)

// bosSquareKinds mirrors SquareBraceTransformer::process: the kind of each
// "[" of the stream, by index.
func bosSquareKinds(s *tokens.Stream) []uint8 {
	kinds := make([]uint8, s.Len())
	for i := 0; i < s.Len(); i++ {
		if !isPunctVal(s, i, "[") || kinds[i] != bosSquareIndex {
			continue
		}
		if bosIsArrayDestructuring(s, i) {
			kinds[i] = bosSquareDestructuring
			end := s.MatchForward(i)
			prev := i
			for j := sigNext(s, i); j >= 0 && j < end; j = sigNext(s, j) {
				if isPunctVal(s, j, "[") && kinds[j] == bosSquareIndex &&
					((isPunctVal(s, prev, "[") && kinds[prev] == bosSquareDestructuring) || isPunctVal(s, prev, ",")) {
					kinds[j] = bosSquareDestructuring
				}
				prev = j
			}
			continue
		}
		if bosIsShortArray(s, i) {
			kinds[i] = bosSquareArray
		}
	}
	return kinds
}

// bosIsShortArray is SquareBraceTransformer::isShortArray for a "[" that is not
// a destructuring.
func bosIsShortArray(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return true
	}
	if isPunctVal(s, p, "}") {
		return false
	}
	return !bosEndsIndexable(s, p)
}

// bosIsArrayDestructuring is SquareBraceTransformer::isArrayDestructing.
func bosIsArrayDestructuring(s *tokens.Stream, i int) bool {
	if p := sigPrev(s, i); p >= 0 {
		t := s.At(p)
		if (isPunctVal(s, p, "}") && bosIsDynamicBraceClose(s, p)) || bosEndsIndexable(s, p) {
			return false
		}
		if kwIs(t, "as") {
			return true
		}
		if isPunctVal(s, p, "=>") {
			v := sigPrev(s, p)
			if v < 0 || s.At(v).Kind != token.Variable {
				return false
			}
			if pv := sigPrev(s, v); pv >= 0 && kwIs(s.At(pv), "as") {
				return true
			}
		}
	}
	end := s.MatchForward(i)
	return end >= 0 && isPunctVal(s, sigNext(s, end), "=")
}

// bosEndsIndexable reports the tokens both SquareBraceTransformer checks rule
// out before a "[" besides braces: ")", "]", a closing double quote, a
// constant string, a T_STRING and a variable.
func bosEndsIndexable(s *tokens.Stream, p int) bool {
	t := s.At(p)
	switch t.Kind {
	case token.Variable:
		return true
	case token.Ident:
		return taIsTString(s, p)
	case token.String:
		return t.ClosesString() && !t.IsShellExec() && !t.EndsHeredoc()
	case token.Punct:
		switch t.Value {
		case "]":
			return true
		case ")":
			// a cast is one token in PHP, and "(new Foo)" closes with
			// CT::T_CLASS_INSTANTIATION_PARENTHESIS_CLOSE
			open := s.MatchBackward(p)
			return open < 0 || (!isCastParen(s, open) && !bosIsClassInstantiationParen(s, open))
		}
	}
	return false
}

// bosIsDynamicBraceClose reports whether the "}" at i closes a "->{...}" or a
// "${...}" (CT::T_DYNAMIC_PROP_BRACE_CLOSE, CT::T_DYNAMIC_VAR_BRACE_CLOSE).
func bosIsDynamicBraceClose(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	return open >= 0 && bosIsDynamicBraceOpen(s, open)
}

func bosIsDynamicBraceOpen(s *tokens.Stream, open int) bool {
	if open > 0 && (isPunctVal(s, open-1, "->") || isPunctVal(s, open-1, "?->")) {
		return true
	}
	p := sigPrev(s, open)
	return p >= 0 && s.At(p).Kind == token.Variable && s.At(p).Value == "$"
}

// bosIsPlainBrace reports whether the "{" at i stays a plain brace through
// BraceTransformer: not a dynamic property or variable brace, a group import
// brace, a dynamic class constant fetch brace nor a property hook brace.
func bosIsPlainBrace(s *tokens.Stream, i int) bool {
	if bosIsDynamicBraceOpen(s, i) {
		return false
	}
	p := sigPrev(s, i)
	if isPunctVal(s, p, `\`) {
		return false
	}
	if isPunctVal(s, p, "::") {
		if end := s.MatchForward(i); end >= 0 {
			n := sigNext(s, end)
			if isPunctVal(s, n, ";") || isPunctVal(s, n, "::") || (n >= 0 && s.At(n).Kind == token.CloseTag) {
				return false
			}
		}
	}
	return !bosIsPropertyHookBrace(s, i)
}

// bosIsPropertyHookBrace mirrors transformIntoPropertyHookBraces.
func bosIsPropertyHookBrace(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	for n >= 0 && s.At(n).Kind == token.AttributeStart {
		n = sigNext(s, s.AttributeEnd(n))
	}
	if n < 0 || s.At(n).Kind != token.Ident {
		return false
	}
	if v := strings.ToLower(s.At(n).Value); v != "get" && v != "set" {
		return false
	}
	nn := sigNext(s, n)
	switch {
	case isPunctVal(s, nn, "{"), isPunctVal(s, nn, ";"), isPunctVal(s, nn, "=>"):
		return true
	case isPunctVal(s, nn, "("):
		end := s.MatchForward(nn)
		if end < 0 {
			return false
		}
		a := sigNext(s, end)
		return isPunctVal(s, a, "{") || isPunctVal(s, a, "=>")
	}
	return false
}

// bosIsClassInstantiationParen mirrors BraceClassInstantiationTransformer: the
// "(" of "(new Foo)", CT::T_CLASS_INSTANTIATION_PARENTHESIS_OPEN.
func bosIsClassInstantiationParen(s *tokens.Stream, i int) bool {
	if !isPunctVal(s, i, "(") || !taKw(s, sigNext(s, i), "new") {
		return false
	}
	p := sigPrev(s, i)
	if p < 0 {
		return true
	}
	t := s.At(p)
	switch t.Kind {
	case token.Punct:
		return t.Value != ")" && t.Value != "]"
	case token.Variable:
		return false
	case token.Ident:
		return !taIsTString(s, p)
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "array", "class", "elseif", "for", "foreach", "if", "static", "switch", "while":
			return false
		}
	}
	return true
}

// bosIsArrayKeyword reports a T_ARRAY: "array(" opening an array literal.
func bosIsArrayKeyword(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && kwIs(s.At(i), "array") && isPunctVal(s, sigNext(s, i), "(")
}
