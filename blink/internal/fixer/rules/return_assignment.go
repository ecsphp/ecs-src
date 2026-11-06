package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/ReturnAssignmentFixer.php
//
// ReturnAssignment removes a useless variable assignment made immediately before
// returning that variable: "$a = 1; return $a;" -> "return 1;". It only fixes
// when the scope cannot change the value indirectly (no references, static,
// global, eval, include, variable-variables or superglobals) and the variable is
// returned straight after being assigned.
//
// This is a conservative port: when the assigned value is itself a block
// expression ending in "}" (a closure, arrow-style `function` body, anonymous
// class or `match`), the statement is left unchanged rather than analysed.
type ReturnAssignment struct {
	SkipNamedVarTags bool
}

func (f ReturnAssignment) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["skip_named_var_tags"].(bool); ok {
		f.SkipNamedVarTags = v
	}
	return f
}

func (ReturnAssignment) Name() string {
	return `PhpCsFixer\Fixer\ReturnNotation\ReturnAssignmentFixer`
}

func (ReturnAssignment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/ReturnAssignmentFixer.php"
}

func (ReturnAssignment) IsCandidate(s *tokens.Stream) bool {
	var fn, ret, variable bool
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch {
		case t.Kind == token.Variable:
			variable = true
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "function"):
			fn = true
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "return"):
			ret = true
		}
		if fn && ret && variable {
			return true
		}
	}
	return false
}

func (f ReturnAssignment) Fix(s *tokens.Stream) bool {
	changed := false
	tokenCount := s.Len()

	for index := 1; index < tokenCount; index++ {
		if !isKeywordVal(s, index, "function") {
			continue
		}

		next := nextMeaningfulIndex(s, index)
		if isPunctVal(s, next, "&") { // function &foo(): returns by reference
			continue
		}

		functionOpenIndex := nextPunctOfKind(s, index, "{", ";")
		if functionOpenIndex < 0 {
			continue
		}
		if isPunctVal(s, functionOpenIndex, ";") { // abstract function
			index = functionOpenIndex - 1
			continue
		}

		functionCloseIndex := s.MatchForward(functionOpenIndex)
		if functionCloseIndex < 0 {
			continue
		}

		totalTokensAdded := 0
		for {
			tokensAdded := f.fixFunction(s, index, functionOpenIndex, functionCloseIndex)
			functionCloseIndex += tokensAdded
			totalTokensAdded += tokensAdded
			if tokensAdded == 0 {
				break
			}
		}

		index = functionCloseIndex
		tokenCount += totalTokensAdded
		if totalTokensAdded > 0 {
			changed = true
		}
	}

	s.ClearEmptyTokens()
	return changed
}

// fixFunction mirrors the PHP method: it analyses one function body, recurses
// into nested functions, decides whether fixing is risky, then simplifies every
// safe "$x = ...; return $x;" it finds. It returns the number of tokens inserted.
func (f ReturnAssignment) fixFunction(s *tokens.Stream, functionIndex, functionOpenIndex, functionCloseIndex int) int {
	inserted := 0
	var candidates []int

	// function returns by reference
	isRisky := isPunctVal(s, nextMeaningfulIndex(s, functionIndex), "&")

	// a "&" in the declaration means arguments are passed by reference
	for index := functionIndex + 1; index < functionOpenIndex; index++ {
		if isPunctVal(s, index, "&") {
			isRisky = true
			break
		}
	}

	for index := functionOpenIndex + 1; index < functionCloseIndex; index++ {
		if isKeywordVal(s, index, "function") {
			nestedOpenIndex := nextPunctOfKind(s, index, "{", ";")
			if nestedOpenIndex < 0 {
				break
			}
			if isPunctVal(s, nestedOpenIndex, ";") { // abstract function
				index = nestedOpenIndex - 1
				continue
			}

			nestedCloseIndex := s.MatchForward(nestedOpenIndex)
			if nestedCloseIndex < 0 {
				break
			}

			tokensAdded := f.fixFunction(s, index, nestedOpenIndex, nestedCloseIndex)

			index = nestedCloseIndex + tokensAdded
			functionCloseIndex += tokensAdded
			inserted += tokensAdded
		}

		if isRisky {
			continue // nothing but nested functions matters once this scope is risky
		}

		if isPunctVal(s, index, "&") {
			isRisky = true
			continue
		}

		if isKeywordVal(s, index, "return") {
			candidates = append(candidates, index)
			continue
		}

		// tokens that let the scope change state indirectly
		t := s.At(index)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "global", "include", "include_once", "require", "require_once":
				isRisky = true
				continue
			case "static":
				if !isKeywordVal(s, nextMeaningfulIndex(s, index), "function") {
					isRisky = true
				}
				continue
			}
		}

		// "eval" lexes as an identifier, not a keyword
		if t.Kind == token.Ident && strings.EqualFold(t.Value, "eval") {
			isRisky = true
			continue
		}

		// a bare "$" token is "$$a" (variable-variable) or "${...}" (dynamic)
		if t.Kind == token.Variable && t.Value == "$" {
			isRisky = true
			continue
		}

		if isSuperGlobalVar(t) {
			isRisky = true
			continue
		}
	}

	if isRisky {
		return inserted
	}

	for _, candidate := range slices.Backward(candidates) {
		inserted += f.simplifyCandidate(s, candidate, functionOpenIndex, functionCloseIndex)
	}

	return inserted
}

// simplifyCandidate validates the "$x = ...; return $x;" shape around a return at
// returnIndex and rewrites it when safe. It returns the number of tokens it
// inserted (0 when it left the code unchanged).
func (f ReturnAssignment) simplifyCandidate(s *tokens.Stream, returnIndex, functionOpenIndex, functionCloseIndex int) int {
	// returning only a variable (not "return 1;" nor "return $a + 1;")
	returnVarIndex := nextMeaningfulIndex(s, returnIndex)
	if returnVarIndex < 0 || s.At(returnVarIndex).Kind != token.Variable {
		return 0
	}

	endReturnVarIndex := nextMeaningfulIndex(s, returnVarIndex)
	if endReturnVarIndex < 0 {
		return 0
	}
	if !isPunctVal(s, endReturnVarIndex, ";") && s.At(endReturnVarIndex).Kind != token.CloseTag {
		return 0 // "return $a + 1;"
	}

	// the variable must be assigned just before it is returned
	assignVarEndIndex := prevMeaningfulIndex(s, returnIndex)
	if !isPunctVal(s, assignVarEndIndex, ";") {
		return 0 // "? return $a;"
	}

	// skip over any trailing "(...)" groups of the assigned expression
	for {
		prevM := prevMeaningfulIndex(s, assignVarEndIndex)
		if !isPunctVal(s, prevM, ")") {
			break
		}
		start := s.MatchBackward(prevM)
		if start < 0 {
			return 0
		}
		assignVarEndIndex = start
	}

	assignVarOperatorIndex := prevStatementOrAssign(s, assignVarEndIndex)
	if assignVarOperatorIndex < 0 {
		return 0
	}
	// a "}" here is a closure, `function` body, anonymous class or match value;
	// left unchanged by this conservative port
	if isPunctVal(s, assignVarOperatorIndex, "}") {
		return 0
	}
	if !isPunctVal(s, assignVarOperatorIndex, "=") {
		return 0
	}

	assignVarIndex := prevMeaningfulIndex(s, assignVarOperatorIndex)
	if assignVarIndex < 0 || !sameVariable(s.At(assignVarIndex), s.At(returnVarIndex)) {
		return 0
	}

	beforeAssignVarIndex := prevMeaningfulIndex(s, assignVarIndex)
	if beforeAssignVarIndex < 0 {
		return 0
	}
	if !isPunctVal(s, beforeAssignVarIndex, ";") &&
		!isPunctVal(s, beforeAssignVarIndex, "{") &&
		!isPunctVal(s, beforeAssignVarIndex, "}") {
		return 0
	}

	if f.SkipNamedVarTags && hasNamedVarTag(s, assignVarIndex, functionOpenIndex) {
		return 0
	}

	if isUsedInCatchOrFinally(s, returnVarIndex, functionOpenIndex, functionCloseIndex) {
		return 0
	}

	return f.simplifyReturnStatement(s, assignVarIndex, assignVarOperatorIndex, returnIndex, endReturnVarIndex)
}

func (f ReturnAssignment) simplifyReturnStatement(s *tokens.Stream, assignVarIndex, assignVarOperatorIndex, returnIndex, returnVarEndIndex int) int {
	var originalIndent string
	hasOriginalIndent := false
	if assignVarIndex-1 >= 0 && s.At(assignVarIndex-1).Kind == token.Whitespace {
		originalIndent = s.At(assignVarIndex - 1).Value
		hasOriginalIndent = true
	}

	// remove the return statement, but keep a PHP close tag
	if isPunctVal(s, returnVarEndIndex, ";") {
		clearTokenAndMergeWS(s, returnVarEndIndex)
	}
	for i := returnIndex; i <= returnVarEndIndex-1; i++ {
		clearIfSave(s, i)
	}

	// drop the now-useless indentation of the removed return statement
	if returnIndex-1 >= 0 && s.At(returnIndex-1).Kind == token.Whitespace {
		content := s.At(returnIndex - 1).Value
		if pos := strings.LastIndex(content, "\n"); pos < 0 {
			content = " "
		} else {
			content = content[pos:]
		}
		s.Set(returnIndex-1, token.Token{Kind: token.Whitespace, Value: content})
	}

	// remove the variable and the assignment operator
	for i := assignVarIndex; i <= assignVarOperatorIndex; i++ {
		clearIfSave(s, i)
	}

	s.InsertAt(assignVarIndex, token.Token{Kind: token.Keyword, Value: "return"})
	inserted := 1

	// restore the assignment's original indentation for the new return
	if hasOriginalIndent && assignVarIndex-1 >= 0 && s.At(assignVarIndex-1).Kind == token.Whitespace &&
		originalIndent != s.At(assignVarIndex-1).Value {
		s.Set(assignVarIndex-1, token.Token{Kind: token.Whitespace, Value: originalIndent})
	}

	// make sure a single space separates the new return from its value
	nextIndex := nonEmptySibling(s, assignVarIndex, 1)
	if nextIndex >= 0 && s.At(nextIndex).Kind != token.Whitespace {
		s.InsertAt(nextIndex, token.Token{Kind: token.Whitespace, Value: " "})
		inserted++
	}

	return inserted
}

// clearIfSave clears a token unless it is a comment, or whitespace trailing a
// comment, so comments survive the rewrite.
func clearIfSave(s *tokens.Stream, index int) {
	if index < 0 || index >= s.Len() {
		return
	}
	t := s.At(index)
	if isComment(t) {
		return
	}
	if t.Kind == token.Whitespace {
		if pv := getPrevNonWhitespace(s, index); pv >= 0 && isComment(s.At(pv)) {
			return
		}
	}
	clearTokenAndMergeWS(s, index)
}

// prevStatementOrAssign scans back for the "=" that begins the assigned
// expression, stopping at a statement boundary ("=", ";", "{", "}" or a PHP
// open tag). It returns that token's index, or -1.
func prevStatementOrAssign(s *tokens.Stream, index int) int {
	for j := index - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.OpenTag {
			return j
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "=", ";", "{", "}":
				return j
			}
		}
	}
	return -1
}

// sameVariable reports whether two tokens are the same variable. PHP-CS-Fixer
// compares the names case-insensitively here (Token::equals with caseSensitive
// false), so "$A" matches "$a".
func sameVariable(a, b token.Token) bool {
	return a.Kind == token.Variable && b.Kind == token.Variable && strings.EqualFold(a.Value, b.Value)
}

func isKeywordVal(s *tokens.Stream, i int, v string) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, v)
}

// isSuperGlobalVar mirrors TokensAnalyzer::isSuperGlobal: the comparison is
// case-insensitive, so "$_server" counts as "$_SERVER".
func isSuperGlobalVar(t token.Token) bool {
	return t.Kind == token.Variable && superGlobalNames[strings.ToUpper(t.Value)]
}

var namedVarTagRe = regexp.MustCompile(`@(?:psalm-|phpstan-)?var\b([^@]*)`)
var docVarNameRe = regexp.MustCompile(`\$[A-Za-z_]`)

// hasNamedVarTag reports whether a doc comment between the function body opener
// and the assignment carries a @var/@psalm-var/@phpstan-var tag that names a
// variable (e.g. "@var int[] $a").
func hasNamedVarTag(s *tokens.Stream, assignVarIndex, functionOpenIndex int) bool {
	docIndex := -1
	for j := assignVarIndex - 1; j >= 0; j-- {
		if s.At(j).Kind == token.DocComment {
			docIndex = j
			break
		}
	}
	if docIndex < 0 || docIndex <= functionOpenIndex {
		return false
	}
	for _, m := range namedVarTagRe.FindAllStringSubmatch(s.At(docIndex).Value, -1) {
		if docVarNameRe.MatchString(m[1]) {
			return true
		}
	}
	return false
}

// isUsedInCatchOrFinally reports whether the returned variable is referenced in a
// catch or finally block that belongs to a try preceding the return, in which
// case removing the assignment would change behaviour.
func isUsedInCatchOrFinally(s *tokens.Stream, returnVarIndex, functionOpenIndex, functionCloseIndex int) bool {
	tryIndex := prevKeywordIndex(s, returnVarIndex, "try")
	if tryIndex < 0 || tryIndex <= functionOpenIndex {
		return false
	}
	tryOpenIndex := nextPunctOfKind(s, tryIndex, "{")
	if tryOpenIndex < 0 {
		return false
	}
	tryCloseIndex := s.MatchForward(tryOpenIndex)
	if tryCloseIndex < 0 {
		return false
	}

	nextIndex := nextMeaningfulIndex(s, tryCloseIndex)
	if nextIndex < 0 {
		return false
	}

	for isKeywordVal(s, nextIndex, "catch") {
		catchOpenIndex := nextPunctOfKind(s, nextIndex, "{")
		if catchOpenIndex < 0 {
			return false
		}
		catchCloseIndex := s.MatchForward(catchOpenIndex)
		if catchCloseIndex < 0 || catchCloseIndex >= functionCloseIndex {
			return false
		}
		if varIndex := nextVariableOfName(s, catchOpenIndex, s.At(returnVarIndex).Value); varIndex >= 0 && varIndex < catchCloseIndex {
			return true
		}
		nextIndex = nextMeaningfulIndex(s, catchCloseIndex)
		if nextIndex < 0 {
			return false
		}
	}

	if !isKeywordVal(s, nextIndex, "finally") {
		return false
	}
	if nextIndex >= functionCloseIndex {
		return false
	}
	finallyOpenIndex := nextPunctOfKind(s, nextIndex, "{")
	if finallyOpenIndex < 0 {
		return false
	}
	finallyCloseIndex := s.MatchForward(finallyOpenIndex)
	if finallyCloseIndex < 0 {
		return false
	}
	varIndex := nextVariableOfName(s, finallyOpenIndex, s.At(returnVarIndex).Value)
	return varIndex >= 0 && varIndex < finallyCloseIndex
}

func prevKeywordIndex(s *tokens.Stream, from int, kw string) int {
	for j := from - 1; j >= 0; j-- {
		if isKeywordVal(s, j, kw) {
			return j
		}
	}
	return -1
}

func nextVariableOfName(s *tokens.Stream, from int, name string) int {
	for j := from + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.Variable && s.At(j).Value == name {
			return j
		}
	}
	return -1
}
