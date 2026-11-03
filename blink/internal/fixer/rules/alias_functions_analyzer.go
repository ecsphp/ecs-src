package rules

import (
	"strings"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// This file ports the PHP-CS-Fixer tokenizer analysis the Alias fixers
// (ArrayPush, EregToPreg, ModernizeStrpos, PowToExponentiation, SetTypeToCast)
// share: FunctionsAnalyzer::isGlobalFunctionCall with the namespace, function
// declaration and "use function" import scans it relies on, plus
// ArgumentsAnalyzer::getArguments and the Tokens sibling/block helpers. Cleared
// tokens are empty Whitespace placeholders (see clearTokenAt), skipped like
// PHP-CS-Fixer skips empty tokens.

// aliasIsMeaningless reports whether the token at i is whitespace, a comment or
// a cleared placeholder: what Tokens::getMeaningfulTokenSibling skips.
func aliasIsMeaningless(s *tokens.Stream, i int) bool {
	switch s.At(i).Kind {
	case token.Whitespace, token.Comment, token.DocComment:
		return true
	}
	return s.At(i).Value == ""
}

// aliasNextMeaningful returns the next meaningful token index after i, or -1.
func aliasNextMeaningful(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if !aliasIsMeaningless(s, j) {
			return j
		}
	}
	return -1
}

// aliasPrevMeaningful returns the previous meaningful token index before i, or -1.
func aliasPrevMeaningful(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if !aliasIsMeaningless(s, j) {
			return j
		}
	}
	return -1
}

// aliasMeaningfulSibling is Tokens::getMeaningfulTokenSibling.
func aliasMeaningfulSibling(s *tokens.Stream, i, dir int) int {
	if dir < 0 {
		return aliasPrevMeaningful(s, i)
	}
	return aliasNextMeaningful(s, i)
}

func aliasIsKeyword(s *tokens.Stream, i int, name string) bool {
	return i >= 0 && i < s.Len() && kwIs(s.At(i), name)
}

// aliasIsNamespaceOperator reports whether the "namespace" keyword at i is the
// relative-name operator ("namespace\foo"), CT::T_NAMESPACE_OPERATOR.
func aliasIsNamespaceOperator(s *tokens.Stream, i int) bool {
	return aliasIsKeyword(s, i, "namespace") && isPunctVal(s, aliasNextMeaningful(s, i), `\`)
}

// aliasIsOpenTag reports whether the token at i is a T_OPEN_TAG ("<?php" or
// "<?"), not the T_OPEN_TAG_WITH_ECHO "<?=".
func aliasIsOpenTag(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.OpenTag && s.At(i).Value != "<?="
}

// aliasIsVariable reports a T_VARIABLE ("$name"), not the lone "$" of a
// variable variable.
func aliasIsVariable(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Variable && s.At(i).Value != "$"
}

// aliasPhpMagicConstants are the magic constants PHP lexes as their own token
// kinds (T_LINE, T_FILE, ...) where blink keeps an Ident.
var aliasPhpMagicConstants = map[string]bool{
	"__line__": true, "__file__": true, "__dir__": true, "__function__": true,
	"__class__": true, "__trait__": true, "__method__": true, "__namespace__": true,
	"__property__": true,
}

// aliasIsTString reports whether the token at i is a PHP T_STRING (a name).
func aliasIsTString(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Ident && !aliasPhpMagicConstants[strings.ToLower(s.At(i).Value)]
}

// aliasIsConstantString reports whether a String token is a
// T_CONSTANT_ENCAPSED_STRING: a complete single- or double-quoted literal
// without interpolation (heredocs, backticks and interpolated strings are not).
func aliasIsConstantString(t token.Token) bool {
	v := t.Value
	if t.Kind != token.String || len(v) < 2 || (v[0] != '\'' && v[0] != '"') || v[len(v)-1] != v[0] {
		return false
	}
	if v[0] == '\'' {
		return true
	}
	return !aliasHasInterpolation(v[1 : len(v)-1])
}

// aliasHasInterpolation reports whether double-quoted string content would
// interpolate: an unescaped "$name", "{$" or "${".
func aliasHasInterpolation(body string) bool {
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\\':
			i++
		case '$':
			if i+1 < len(body) && (body[i+1] == '{' || isPHPLabelStart(body[i+1])) {
				return true
			}
		case '{':
			if i+1 < len(body) && body[i+1] == '$' {
				return true
			}
		}
	}
	return false
}

func isPHPLabelStart(c byte) bool {
	return c == '_' || c >= 0x80 || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// aliasCastTypes are the words PHP's lexer turns "(word)" into a cast token for.
var aliasCastTypes = map[string]bool{
	"int": true, "integer": true, "bool": true, "boolean": true, "float": true,
	"double": true, "real": true, "string": true, "binary": true, "array": true,
	"object": true, "unset": true, "void": true,
}

// aliasCastAt reports whether the "(" at open starts a cast the way PHP's lexer
// sees one ("(" [ \t]* type [ \t]* ")"), returning the closing ")" index and
// the lowercased type word.
func aliasCastAt(s *tokens.Stream, open int) (closeIdx int, typ string, ok bool) {
	if !isPunctVal(s, open, "(") {
		return 0, "", false
	}
	j := open + 1
	if j < s.Len() && s.At(j).Kind == token.Whitespace && strings.Trim(s.At(j).Value, " \t") == "" {
		j++
	}
	if j >= s.Len() || (s.At(j).Kind != token.Ident && s.At(j).Kind != token.Keyword) || !aliasCastTypes[strings.ToLower(s.At(j).Value)] {
		return 0, "", false
	}
	typ = strings.ToLower(s.At(j).Value)
	j++
	if j < s.Len() && s.At(j).Kind == token.Whitespace && strings.Trim(s.At(j).Value, " \t") == "" {
		j++
	}
	if !isPunctVal(s, j, ")") {
		return 0, "", false
	}
	return j, typ, true
}

// aliasIsEllipsis reports whether the "..." at i is a spread/variadic
// T_ELLIPSIS, not the CT::T_FIRST_CLASS_CALLABLE of "name(...)".
func aliasIsEllipsis(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "...") &&
		!(isPunctVal(s, aliasPrevMeaningful(s, i), "(") && isPunctVal(s, aliasNextMeaningful(s, i), ")"))
}

// aliasBlockEnd returns the end of the block opened at i (Tokens::detectBlockType
// plus findBlockEnd), or -1 when the token does not open a block.
func aliasBlockEnd(s *tokens.Stream, i int) int {
	t := s.At(i)
	if t.Kind == token.AttributeStart {
		return s.AttributeEnd(i)
	}
	if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
		return s.MatchForward(i)
	}
	return -1
}

// aliasArguments ports ArgumentsAnalyzer::getArguments: each argument's first
// and last token index between the parentheses at open and closeIdx,
// including surrounding whitespace and comments, without the separators.
func aliasArguments(s *tokens.Stream, open, closeIdx int) [][2]int {
	var args [][2]int
	if first := aliasNextMeaningful(s, open); first < 0 || isPunctVal(s, first, ")") {
		return args
	}
	start := open + 1
	i := open + 1
	for ; i < closeIdx; i++ {
		if end := aliasBlockEnd(s, i); end >= 0 {
			i = end
			continue
		}
		if isPunctVal(s, i, ",") {
			if aliasNextMeaningful(s, i) == closeIdx {
				break // trailing comma
			}
			args = append(args, [2]int{start, i - 1})
			start = i + 1
		}
	}
	return append(args, [2]int{start, i - 1})
}

// aliasFind ports AbstractFunctionReferenceFixer::find: the next global call
// to name ("name" then "(") whose name and "(" lie within [start, end], as the
// name, "(" and matching ")" indices.
func aliasFind(s *tokens.Stream, name string, start, end int) (fn, open, closeIdx int, ok bool) {
	end = min(end, s.Len()-1)
	for start <= end {
		fn, open = -1, -1
		for i := start; i <= end; i++ {
			if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, name) {
				continue
			}
			if o := aliasNextMeaningful(s, i); o >= 0 && o <= end && isPunctVal(s, o, "(") {
				fn, open = i, o
				break
			}
		}
		if fn < 0 {
			return 0, 0, 0, false
		}
		if aliasIsGlobalFunctionCall(s, fn) {
			return fn, open, s.MatchForward(open), true
		}
		start = open
	}
	return 0, 0, 0, false
}

// aliasIsGlobalFunctionCall ports FunctionsAnalyzer::isGlobalFunctionCall: the
// name at index is called as a global function. It is not a method, static or
// namespaced call, a declaration, an instantiation, an attribute or a
// first-class callable, and the current namespace neither declares a function
// of that name nor imports one with "use function".
func aliasIsGlobalFunctionCall(s *tokens.Stream, index int) bool {
	if s.At(index).Kind != token.Ident {
		return false
	}
	open := aliasNextMeaningful(s, index)
	if !isPunctVal(s, open, "(") {
		return false
	}
	previousIsNamespaceSeparator := false
	prev := aliasPrevMeaningful(s, index)
	if isPunctVal(s, prev, `\`) {
		previousIsNamespaceSeparator = true
		prev = aliasPrevMeaningful(s, prev)
	}
	if prev >= 0 && aliasIsPossibleKind(s, prev) {
		return false
	}
	// first-class callable "name(...)"
	if n := aliasNextMeaningful(s, open); isPunctVal(s, n, "...") && !aliasIsEllipsis(s, n) {
		return false
	}
	if previousIsNamespaceSeparator {
		return true
	}
	functionName := strings.ToLower(s.At(index).Value)
	if functionName == "set" {
		// a property hook "set(...) {" or "set(...) =>" is no call; the hook
		// brace is approximated by a plain "{"
		if !(isPunctVal(s, prev, "{") || isPunctVal(s, prev, ";") || isPunctVal(s, prev, "}")) {
			return true
		}
		if c := s.MatchForward(open); c >= 0 {
			after := aliasNextMeaningful(s, c)
			if isPunctVal(s, after, "{") || isPunctVal(s, after, "=>") {
				return false
			}
		}
	}

	// figure out in which namespace we are
	scopeStart, scopeEnd := 0, s.Len()-1
	inGlobalNamespace := false
	for _, d := range aliasNamespaceDeclarations(s) {
		scopeStart, scopeEnd = d.scopeStart, d.scopeEnd
		if index >= scopeStart && index <= scopeEnd {
			inGlobalNamespace = d.global
			break
		}
	}

	// a function declared in the same (non-global) namespace shadows the global one
	if !inGlobalNamespace {
		for _, decl := range aliasFunctionDeclarations(s) {
			if decl < scopeStart || decl > scopeEnd {
				continue
			}
			if strings.ToLower(s.At(decl).Value) == functionName {
				return false
			}
		}
	}

	for _, imp := range aliasFunctionImports(s) {
		if imp.start < scopeStart || imp.end > scopeEnd {
			continue
		}
		if functionName != strings.ToLower(imp.shortName) {
			continue
		}
		// a global import like "use function \str_repeat;"
		return imp.shortName == strings.TrimLeft(imp.fullName, `\`)
	}

	return !aliasIsAttribute(s, index)
}

// aliasIsPossibleKind mirrors FunctionsAnalyzer::POSSIBLE_KINDS: a token before
// a name that makes it something other than a global function call.
func aliasIsPossibleKind(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Ident:
		// PHP lexes "yield from" as one T_YIELD_FROM token, so its "from" is no name
		if strings.EqualFold(t.Value, "from") && aliasIsKeyword(s, aliasPrevMeaningful(s, i), "yield") {
			return false
		}
		return aliasIsTString(s, i)
	case token.AttributeStart:
		return true
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "function", "new":
			return true
		case "namespace":
			return aliasIsNamespaceOperator(s, i)
		}
	case token.Punct:
		switch t.Value {
		case "::", "->", "?->":
			return true
		case "&":
			// CT::T_RETURN_REF in "function &name("
			return aliasIsKeyword(s, aliasPrevMeaningful(s, i), "function")
		}
	}
	return false
}

// aliasIsAttribute ports AttributeAnalyzer::isAttribute: the name at index is
// an attribute class inside "#[...]" (not an argument of one).
func aliasIsAttribute(s *tokens.Stream, index int) bool {
	if !aliasIsTString(s, index) {
		return false
	}
	found := false
	for j := 0; j < s.Len(); j++ {
		if s.At(j).Kind == token.AttributeStart {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	start := -1
	for j := index - 1; j >= 0; j-- {
		t := s.At(j)
		stop := false
		switch t.Kind {
		case token.AttributeStart, token.AttributeEnd, token.OpenTag:
			stop = true
		case token.Variable:
			stop = t.Value != "$"
		case token.Punct:
			stop = t.Value == ";" || t.Value == "{"
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "private", "protected", "public", "return":
				stop = true
			case "function":
				stop = !aliasIsKeyword(s, aliasPrevMeaningful(s, j), "use")
			}
		}
		if stop {
			start = j
			break
		}
	}
	if start < 0 || s.At(start).Kind != token.AttributeStart {
		return false
	}
	depth := 0
	for j := start + 1; j < index; j++ {
		if isPunctVal(s, j, "(") {
			depth++
		} else if isPunctVal(s, j, ")") {
			depth--
		}
	}
	return depth == 0
}

type aliasNamespace struct {
	scopeStart, scopeEnd int
	global               bool
}

// aliasIsNamespaceDeclaration reports whether the "namespace" keyword at i
// declares a namespace (T_NAMESPACE), as opposed to the "namespace\" operator.
func aliasIsNamespaceDeclaration(s *tokens.Stream, i int) bool {
	return aliasIsKeyword(s, i, "namespace") && !aliasIsNamespaceOperator(s, i)
}

// aliasNamespaceDeclarations ports NamespacesAnalyzer::getDeclarations.
func aliasNamespaceDeclarations(s *tokens.Stream) []aliasNamespace {
	var out []aliasNamespace
	n := s.Len()
	for index := 1; index < n; index++ {
		if !aliasIsNamespaceDeclaration(s, index) {
			continue
		}
		declEnd := -1
		for j := index + 1; j < n; j++ {
			if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") {
				declEnd = j
				break
			}
		}
		if declEnd < 0 {
			break
		}
		var name strings.Builder
		for j := index + 1; j < declEnd; j++ {
			name.WriteString(s.At(j).Value)
		}
		scopeEnd := -1
		if isPunctVal(s, declEnd, "{") {
			scopeEnd = s.MatchForward(declEnd)
		} else {
			scopeEnd = n
			for j := declEnd + 1; j < n; j++ {
				if aliasIsNamespaceDeclaration(s, j) {
					scopeEnd = j
					break
				}
			}
			scopeEnd--
		}
		if scopeEnd < 0 {
			scopeEnd = n - 1
		}
		out = append(out, aliasNamespace{scopeStart: index, scopeEnd: scopeEnd, global: strings.TrimSpace(name.String()) == ""})
		index = scopeEnd
	}
	if len(out) == 0 {
		for j := range n {
			if aliasIsOpenTag(s, j) {
				openTag := 0
				if s.At(0).Kind == token.InlineHTML {
					openTag = 1
				}
				out = append(out, aliasNamespace{scopeStart: openTag, scopeEnd: n - 1, global: true})
				break
			}
		}
	}
	return out
}

// aliasIsClassy reports whether the keyword at i opens a class-like
// declaration (class, interface, trait, enum; anonymous classes included).
func aliasIsClassy(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "class", "interface", "trait":
		return true
	case "enum":
		// T_ENUM only when an enum is declared ("enum Name")
		return aliasIsTString(s, aliasNextMeaningful(s, i))
	}
	return false
}

// aliasClassyBodyEnd returns the closing "}" of the classy declared at i, or -1.
func aliasClassyBodyEnd(s *tokens.Stream, i int) int {
	j := i + 1
	for ; j < s.Len(); j++ {
		if isPunctVal(s, j, "(") || isPunctVal(s, j, "{") {
			break
		}
	}
	if j >= s.Len() {
		return -1
	}
	if isPunctVal(s, j, "(") {
		// anonymous class constructor arguments
		if j = s.MatchForward(j); j < 0 {
			return -1
		}
		for ; j < s.Len() && !isPunctVal(s, j, "{"); j++ {
		}
		if j >= s.Len() {
			return -1
		}
	}
	return s.MatchForward(j)
}

// aliasFunctionDeclarations returns the name index of every named function
// declared outside class-like bodies (the "declarations" of
// FunctionsAnalyzer::buildFunctionsAnalysis).
func aliasFunctionDeclarations(s *tokens.Stream) []int {
	var out []int
	for i := 0; i < s.Len(); i++ {
		if aliasIsClassy(s, i) {
			end := aliasClassyBodyEnd(s, i)
			if end < 0 {
				return out
			}
			i = end
			continue
		}
		if !aliasIsKeyword(s, i, "function") || aliasIsKeyword(s, aliasPrevMeaningful(s, i), "use") {
			continue
		}
		i = aliasNextMeaningful(s, i)
		if i < 0 {
			return out
		}
		if isPunctVal(s, i, "&") {
			if i = aliasNextMeaningful(s, i); i < 0 {
				return out
			}
		}
		if !aliasIsTString(s, i) {
			continue
		}
		out = append(out, i)
	}
	return out
}

type aliasFunctionImport struct {
	fullName, shortName string
	start, end          int
}

// aliasImportUseIndexes returns the "use" keywords that import (T_USE), not a
// closure's "use (...)" nor a trait "use" inside a class-like body.
func aliasImportUseIndexes(s *tokens.Stream) []int {
	var out []int
	classyEnd := -1
	for i := 0; i < s.Len(); i++ {
		if i > classyEnd && aliasIsClassy(s, i) {
			classyEnd = aliasClassyBodyEnd(s, i)
			if classyEnd < 0 {
				classyEnd = s.Len()
			}
			continue
		}
		if i <= classyEnd || !aliasIsKeyword(s, i, "use") {
			continue
		}
		if isPunctVal(s, aliasPrevMeaningful(s, i), ")") {
			continue // closure use
		}
		out = append(out, i)
	}
	return out
}

// aliasFunctionImports ports the single (non-group, non-multi) "use function"
// imports FunctionsAnalyzer gets from NamespaceUsesAnalyzer.
func aliasFunctionImports(s *tokens.Stream) []aliasFunctionImport {
	var out []aliasFunctionImport
	for _, use := range aliasImportUseIndexes(s) {
		if !aliasIsKeyword(s, aliasNextMeaningful(s, use), "function") {
			continue
		}
		end := -1
		for j := use + 1; j < s.Len(); j++ {
			if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		multi := false
		for j := use + 1; j < end; j++ {
			if isPunctVal(s, j, ",") || isPunctVal(s, j, "{") {
				multi = true
				break
			}
		}
		if multi {
			continue // NamespaceUsesAnalyzer drops multi imports by default
		}
		var full strings.Builder
		short := ""
		aliased := false
		for j := aliasNextMeaningful(s, aliasNextMeaningful(s, use)); j >= 0 && j < end; j = aliasNextMeaningful(s, j) {
			t := s.At(j)
			switch {
			case kwIs(t, "as"):
				aliased = true
			case t.Kind == token.Ident || t.Kind == token.Keyword:
				short = t.Value
				if !aliased {
					full.WriteString(t.Value)
				}
			case t.Kind == token.Punct && t.Value == `\`:
				full.WriteString(t.Value)
			}
		}
		if short == "" {
			continue
		}
		out = append(out, aliasFunctionImport{fullName: full.String(), shortName: short, start: use, end: end})
	}
	return out
}

// aliasRelex re-tokenizes the stream from its rendered source, so the fixers
// below can edit PHP-CS-Fixer style (cleared placeholders, inserted tokens) and
// still hand later fixers a canonical stream.
func aliasRelex(s *tokens.Stream) {
	s.ReplaceRange(0, s.Len()-1, lexer.Lex(s.Render()))
}

func aliasPunct(v string) token.Token { return token.Token{Kind: token.Punct, Value: v} }

func aliasSpace() token.Token { return token.Token{Kind: token.Whitespace, Value: " "} }
