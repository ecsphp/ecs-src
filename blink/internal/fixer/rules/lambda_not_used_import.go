package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/LambdaNotUsedImportFixer.php
//
// LambdaNotUsedImport removes variables imported into a closure via "use (...)"
// that the closure body never uses.
type LambdaNotUsedImport struct{}

func (LambdaNotUsedImport) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\LambdaNotUsedImportFixer`
}

func (LambdaNotUsedImport) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/LambdaNotUsedImportFixer.php"
}

var superGlobalNames = map[string]bool{
	"$GLOBALS": true, "$_SERVER": true, "$_GET": true, "$_POST": true,
	"$_REQUEST": true, "$_SESSION": true, "$_ENV": true, "$_COOKIE": true,
	"$_FILES": true,
}

func (LambdaNotUsedImport) Fix(s *tokens.Stream) bool {
	before := s.Render()
	for i := s.Len() - 4; i > 0; i-- {
		useIdx := lambdaUseIndex(s, i)
		if useIdx >= 0 {
			fixLambdaImports(s, useIdx)
		}
	}
	compactEmptyTokens(s)
	return s.Render() != before
}

// compactEmptyTokens drops the empty placeholder tokens left by clearing, so the
// rest of the pipeline sees a normal stream. It never changes the rendered bytes.
func compactEmptyTokens(s *tokens.Stream) {
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind == token.Whitespace && s.At(i).Value == "" {
			s.RemoveAt(i)
		}
	}
}

// lambdaUseIndex returns the "use" keyword index of the closure declared at
// index, or -1 when index is not a closure with a use-import list.
func lambdaUseIndex(s *tokens.Stream, index int) int {
	if !kwIs(s.At(index), "function") || !isLambdaFunction(s, index) {
		return -1
	}
	u := sigNext(s, index)
	if u < 0 {
		return -1
	}
	if isPunctVal(s, u, "&") { // by-reference return
		u = sigNext(s, u)
	}
	if u < 0 || !isPunctVal(s, u, "(") {
		return -1
	}
	closeIdx := s.MatchForward(u)
	if closeIdx < 0 {
		return -1
	}
	useIdx := sigNext(s, closeIdx)
	if useIdx < 0 || !kwIs(s.At(useIdx), "use") {
		return -1
	}
	return useIdx
}

// isLambdaFunction reports whether the "function" at index is anonymous.
func isLambdaFunction(s *tokens.Stream, index int) bool {
	n := sigNext(s, index)
	return n >= 0 && (isPunctVal(s, n, "(") || isPunctVal(s, n, "&"))
}

type lambdaImport struct {
	name string
	idx  int
}

func fixLambdaImports(s *tokens.Stream, useIdx int) {
	open := nextPunctOfKind(s, useIdx, "(")
	if open < 0 {
		return
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return
	}
	args := docArgumentSpans(s, open, closeIdx)
	imports := filterLambdaImports(s, args)
	if len(imports) == 0 {
		return
	}
	notUsed := findNotUsedLambdaImports(s, imports, closeIdx)
	if len(notUsed) == 0 {
		return
	}
	if len(notUsed) == len(args) {
		clearImportsAndUse(s, useIdx, closeIdx)
		return
	}
	// remove highest index first so lower indices stay valid
	for _, im := range slices.Backward(notUsed) {
		clearLambdaImport(s, im.idx)
	}
}

// docArgumentSpans splits the paren (open..closeIdx) into top-level argument
// spans (inclusive start/end of the meaningful content between commas).
func docArgumentSpans(s *tokens.Stream, open, closeIdx int) [][2]int {
	var spans [][2]int
	depth := 0
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		// trim trailing whitespace/comments
		e := end
		for e >= start && isTrivia(s, e) {
			e--
		}
		st := start
		for st <= e && isTrivia(s, st) {
			st++
		}
		if st <= e {
			spans = append(spans, [2]int{st, e})
		}
		start = -1
	}
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case ",":
				if depth == 0 {
					flush(j - 1)
					continue
				}
			}
		}
		if start < 0 && !isTrivia(s, j) {
			start = j
		}
	}
	flush(closeIdx - 1)
	return spans
}

func isTrivia(s *tokens.Stream, i int) bool {
	k := s.At(i).Kind
	return k == token.Whitespace || k == token.Comment || k == token.DocComment
}

func filterLambdaImports(s *tokens.Stream, args [][2]int) []lambdaImport {
	var imports []lambdaImport
	for _, sp := range args {
		varIdx := -1
		for j := sp[0]; j <= sp[1]; j++ {
			if s.At(j).Kind == token.Variable {
				varIdx = j
				break
			}
		}
		if varIdx < 0 {
			continue
		}
		if p := sigPrev(s, varIdx); p >= 0 && isPunctVal(s, p, "&") {
			continue // imported by reference
		}
		name := s.At(varIdx).Value
		if name == "$this" || superGlobalNames[name] {
			continue
		}
		imports = append(imports, lambdaImport{name: name, idx: varIdx})
	}
	return imports
}

func findNotUsedLambdaImports(s *tokens.Stream, imports []lambdaImport, useCloseBrace int) []lambdaImport {
	lambdaOpen := nextPunctOfKind(s, useCloseBrace, "{")
	if lambdaOpen < 0 {
		return nil
	}
	remaining := make(map[string]bool, len(imports))
	for _, im := range imports {
		remaining[im.name] = true
	}
	level := 0
	for i := lambdaOpen; i < s.Len(); i++ {
		t := s.At(i)
		if isPunctVal(s, i, "{") {
			level++
			continue
		}
		if isPunctVal(s, i, "}") {
			level--
			if level == 0 {
				break
			}
			continue
		}
		if t.Kind == token.Ident && strings.EqualFold(t.Value, "compact") && lambdaIsGlobalCall(s, i) {
			return nil
		}
		if isLambdaBailoutKeyword(t) {
			return nil
		}
		if isPunctVal(s, i, "$") {
			n := sigNext(s, i)
			if n >= 0 && (s.At(n).Kind == token.Variable || isPunctVal(s, n, "{")) {
				return nil // "$$a" or "${...}"
			}
		}
		if t.Kind == token.Variable && remaining[t.Value] {
			delete(remaining, t.Value)
			if len(remaining) == 0 {
				return nil
			}
		}
		if t.Kind == token.String {
			for _, name := range interpolatedVarNames(t.Value) {
				if remaining[name] {
					delete(remaining, name)
					if len(remaining) == 0 {
						return nil
					}
				}
			}
		}
		if isClassyKeyword(t) { // anonymous class
			j := nextPunctOfKind(s, i, "(", "{")
			if j < 0 {
				break
			}
			if isPunctVal(s, j, "(") {
				cb := s.MatchForward(j)
				if cb < 0 {
					break
				}
				countImportsUsedAsArgument(s, remaining, docArgumentSpans(s, j, cb))
				j = nextPunctOfKind(s, cb, "{")
				if j < 0 {
					break
				}
			}
			end := s.MatchForward(j)
			if end < 0 {
				break
			}
			i = end
			continue
		}
		if kwIs(t, "function") { // nested closure
			o := nextPunctOfKind(s, i, "(")
			if o < 0 {
				break
			}
			c := s.MatchForward(o)
			if c < 0 {
				break
			}
			countImportsUsedAsArgument(s, remaining, docArgumentSpans(s, o, c))
			j := nextUseOrBrace(s, i)
			if j < 0 {
				break
			}
			if kwIs(s.At(j), "use") {
				o2 := nextPunctOfKind(s, j, "(")
				if o2 < 0 {
					break
				}
				c2 := s.MatchForward(o2)
				if c2 < 0 {
					break
				}
				countImportsUsedAsArgument(s, remaining, docArgumentSpans(s, o2, c2))
				j = nextPunctOfKind(s, c2, "{")
				if j < 0 {
					break
				}
			}
			end := s.MatchForward(j)
			if end < 0 {
				break
			}
			i = end
			continue
		}
	}
	var notUsed []lambdaImport
	for _, im := range imports {
		if remaining[im.name] {
			notUsed = append(notUsed, im)
		}
	}
	return notUsed
}

func countImportsUsedAsArgument(s *tokens.Stream, remaining map[string]bool, args [][2]int) {
	for _, sp := range args {
		name := lambdaArgName(s, sp)
		if name != "" && remaining[name] {
			delete(remaining, name)
		}
	}
}

// lambdaArgName returns the argument's variable name when the span reduces to a
// plain (optionally typed / by-ref / defaulted) variable, else "".
func lambdaArgName(s *tokens.Stream, sp [2]int) string {
	varIdx := -1
	for j := sp[0]; j <= sp[1]; j++ {
		if s.At(j).Kind == token.Variable {
			if varIdx >= 0 {
				return "" // more than one variable: not a plain name
			}
			varIdx = j
		}
	}
	if varIdx < 0 {
		return ""
	}
	// anything after the variable other than "= default" disqualifies it
	n := sigNext(s, varIdx)
	if n >= 0 && n <= sp[1] && !isPunctVal(s, n, "=") {
		return ""
	}
	return s.At(varIdx).Value
}

// interpolatedVarNames returns the variable names ($name) referenced through
// interpolation inside a string token value. Single-quoted strings and nowdocs
// do not interpolate, so they yield nothing. Handles "$name", "{$name...}" and
// "${name}" forms and respects backslash escaping.
func interpolatedVarNames(value string) []string {
	if !stringInterpolates(value) {
		return nil
	}
	var names []string
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '\\' {
			i++ // skip the escaped character
			continue
		}
		if c != '$' {
			continue
		}
		j := i + 1
		if j < len(value) && value[j] == '{' { // ${name}
			j++
		}
		if j >= len(value) || !isVarNameStart(value[j]) {
			continue
		}
		k := j
		for k < len(value) && isVarNameByte(value[k]) {
			k++
		}
		names = append(names, "$"+value[j:k])
		i = k - 1
	}
	return names
}

// stringInterpolates reports whether a string token's raw value uses double
// quotes, backticks or a heredoc, all of which interpolate variables.
func stringInterpolates(value string) bool {
	if value == "" {
		return false
	}
	switch value[0] {
	case '"', '`':
		return true
	case '\'':
		return false
	case '<': // <<<LABEL heredoc / <<<'LABEL' nowdoc
		i := 3
		for i < len(value) && (value[i] == ' ' || value[i] == '\t') {
			i++
		}
		return i >= len(value) || value[i] != '\''
	}
	return false
}

func isVarNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isVarNameByte(c byte) bool {
	return isVarNameStart(c) || (c >= '0' && c <= '9')
}

func isLambdaBailoutKeyword(t token.Token) bool {
	if t.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "eval", "include", "include_once", "require", "require_once":
		return true
	}
	return false
}

func isClassyKeyword(t token.Token) bool {
	return t.Kind == token.Keyword && classyKeywords[strings.ToLower(t.Value)]
}

// lambdaIsGlobalCall reports whether the name at i is a global function call
// (reusing isGlobalFunctionCall from the alias fixer) with a following "(".
func lambdaIsGlobalCall(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	return n >= 0 && isPunctVal(s, n, "(") && isGlobalFunctionCall(s, i)
}

func nextUseOrBrace(s *tokens.Stream, idx int) int {
	for j := idx + 1; j < s.Len(); j++ {
		if kwIs(s.At(j), "use") {
			return j
		}
		if isPunctVal(s, j, "{") {
			return j
		}
	}
	return -1
}

// clearLambdaImport removes one imported variable token and its adjoining comma,
// mirroring PhpdocToComment's clearImports.
func clearLambdaImport(s *tokens.Stream, removeIdx int) {
	clearTokenAndMergeWS(s, removeIdx)
	prev := sigPrev(s, removeIdx)
	switch {
	case prev >= 0 && isPunctVal(s, prev, ","):
		clearTokenAndMergeWS(s, prev)
	case prev >= 0 && isPunctVal(s, prev, "("):
		clearTokenAndMergeWS(s, sigNext(s, removeIdx)) // the following ","
	}
}

// clearImportsAndUse removes the whole "use (...)" clause.
func clearImportsAndUse(s *tokens.Stream, useIdx, useCloseBrace int) {
	for i := useCloseBrace; i >= useIdx; i-- {
		if s.At(i).Kind == token.Comment || s.At(i).Kind == token.DocComment {
			continue
		}
		if s.At(i).Kind == token.Whitespace {
			pv := getPrevNonWhitespace(s, i)
			if pv >= 0 && (s.At(pv).Kind == token.Comment || s.At(pv).Kind == token.DocComment) {
				continue
			}
		}
		clearTokenAndMergeWS(s, i)
	}
}

// clearTokenAndMergeWS mirrors Tokens::clearTokenAndMergeSurroundingWhitespace:
// the token becomes an empty whitespace token (renders as nothing) and adjacent
// whitespace is merged, so token indices stay stable.
func clearTokenAndMergeWS(s *tokens.Stream, index int) {
	if index < 0 || index >= s.Len() {
		return
	}
	count := s.Len()
	clearTokenAt(s, index)
	if index == count-1 {
		return
	}
	nextIdx := nonEmptySibling(s, index, 1)
	if nextIdx < 0 || s.At(nextIdx).Kind != token.Whitespace {
		return
	}
	prevIdx := nonEmptySibling(s, index, -1)
	if prevIdx >= 0 && s.At(prevIdx).Kind == token.Whitespace {
		s.SetValue(prevIdx, s.At(prevIdx).Value+s.At(nextIdx).Value)
	} else if prevIdx+1 < s.Len() && isEmptyToken(s, prevIdx+1) {
		s.Set(prevIdx+1, token.Token{Kind: token.Whitespace, Value: s.At(nextIdx).Value})
	}
	clearTokenAt(s, nextIdx)
}

func clearTokenAt(s *tokens.Stream, i int) {
	s.Set(i, token.Token{Kind: token.Whitespace, Value: ""})
}

func isEmptyToken(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Value == ""
}

// nonEmptySibling returns the nearest non-empty token index in the given
// direction, skipping empty (cleared) tokens.
func nonEmptySibling(s *tokens.Stream, i, dir int) int {
	j := i + dir
	for j >= 0 && j < s.Len() {
		if s.At(j).Value != "" {
			return j
		}
		j += dir
	}
	return -1
}
