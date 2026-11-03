package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Port of PHP-CS-Fixer's FunctionsAnalyzer::isGlobalFunctionCall and the
// analyzers it uses (namespaces, function declarations, imports, attributes).

// gfcNamespace mirrors a NamespaceAnalysis.
type gfcNamespace struct {
	start, end int
	global     bool
}

// gfcImport mirrors a non-multi "use function" NamespaceUseAnalysis.
type gfcImport struct {
	start, end      int
	short, fullName string
}

// gfcAnalysis caches the per-file scans; callers invalidate it when they
// insert or remove tokens.
type gfcAnalysis struct {
	built        bool
	namespaces   []gfcNamespace
	declarations []int
	imports      []gfcImport
	hasAttribute bool
}

func (a *gfcAnalysis) invalidate() { a.built = false }

// gfcMagicNames are names blink lexes as identifiers that PHP tokenizes as a
// token kind of their own (magic constants, exit/die, eval, __halt_compiler),
// so they are never a T_STRING.
var gfcMagicNames = map[string]bool{
	"__line__": true, "__file__": true, "__dir__": true, "__function__": true,
	"__class__": true, "__trait__": true, "__method__": true, "__namespace__": true,
	"__property__": true,
	"exit":         true, "die": true, "eval": true, "__halt_compiler": true,
}

// gfcIsTString reports whether the token at i is a PHP T_STRING. The "from"
// of "yield from" is part of one T_YIELD_FROM token.
func gfcIsTString(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() || s.At(i).Kind != token.Ident {
		return false
	}
	name := strings.ToLower(s.At(i).Value)
	if name == "from" && gfcIsKeyword(s, sigPrev(s, i), "yield") {
		return false
	}
	return !gfcMagicNames[name]
}

// gfcIsTVariable reports a T_VARIABLE ("$name"), not the "$" of "$$name".
func gfcIsTVariable(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Variable && s.At(i).Value != "$"
}

func gfcIsKeyword(s *tokens.Stream, i int, name string) bool {
	return i >= 0 && i < s.Len() && kwIs(s.At(i), name)
}

// gfcIsNamespaceOperator reports CT::T_NAMESPACE_OPERATOR: "namespace\".
func gfcIsNamespaceOperator(s *tokens.Stream, i int) bool {
	return gfcIsKeyword(s, i, "namespace") && isPunctVal(s, sigNext(s, i), `\`)
}

// gfcIsPossibleKind mirrors FunctionsAnalyzer::POSSIBLE_KINDS: the tokens that,
// before a name, make it something other than a global function call.
func gfcIsPossibleKind(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	switch t.Kind {
	case token.Punct:
		switch t.Value {
		case "::", "->", "?->":
			return true
		case "&": // CT::T_RETURN_REF
			p := sigPrev(s, i)
			return gfcIsKeyword(s, p, "function") || gfcIsKeyword(s, p, "fn")
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "function", "new":
			return true
		case "namespace":
			return gfcIsNamespaceOperator(s, i)
		}
	case token.AttributeStart:
		return true
	}
	return gfcIsTString(s, i)
}

// isGlobalFunctionCall is FunctionsAnalyzer::isGlobalFunctionCall.
func (a *gfcAnalysis) isGlobalFunctionCall(s *tokens.Stream, index int) bool {
	if !gfcIsTString(s, index) {
		return false
	}
	open := sigNext(s, index)
	if !isPunctVal(s, open, "(") {
		return false
	}
	previousIsNamespaceSeparator := false
	prev := sigPrev(s, index)
	if isPunctVal(s, prev, `\`) {
		previousIsNamespaceSeparator = true
		prev = sigPrev(s, prev)
	}
	if gfcIsPossibleKind(s, prev) {
		return false
	}
	// CT::T_FIRST_CLASS_CALLABLE: "foo(...)"
	if next := sigNext(s, open); isPunctVal(s, next, "...") && isPunctVal(s, sigNext(s, next), ")") {
		return false
	}
	if previousIsNamespaceSeparator {
		return true
	}
	functionName := strings.ToLower(s.At(index).Value)
	if functionName == "set" {
		if !gfcIsPropertyHookBraceOpen(s, prev) && !isPunctVal(s, prev, ";") && !isPunctVal(s, prev, "}") {
			return true
		}
		if closeIdx := s.MatchForward(open); closeIdx >= 0 {
			after := sigNext(s, closeIdx)
			if isPunctVal(s, after, "{") || isPunctVal(s, after, "=>") {
				return false
			}
		}
	}
	if !a.built {
		a.build(s)
	}

	scopeStart, scopeEnd := 0, s.Len()-1
	inGlobalNamespace := false
	for _, d := range a.namespaces {
		scopeStart, scopeEnd = d.start, d.end
		if index >= scopeStart && index <= scopeEnd {
			inGlobalNamespace = d.global
			break
		}
	}
	// a function declared in the same (non-global) namespace shadows the global one
	if !inGlobalNamespace {
		for _, decl := range a.declarations {
			if decl < scopeStart || decl > scopeEnd {
				continue
			}
			if strings.ToLower(s.At(decl).Value) == functionName {
				return false
			}
		}
	}
	for _, imp := range a.imports {
		if imp.start < scopeStart || imp.end > scopeEnd {
			continue
		}
		if functionName != strings.ToLower(imp.short) {
			continue
		}
		// global import like "use function \str_repeat;"
		return imp.short == strings.TrimLeft(imp.fullName, `\`)
	}
	return !a.hasAttribute || !gfcIsAttribute(s, index)
}

// gfcIsPropertyHookBraceOpen approximates CT::T_PROPERTY_HOOK_BRACE_OPEN: the
// "{" right after a property name.
func gfcIsPropertyHookBraceOpen(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "{") && gfcIsTVariable(s, sigPrev(s, i))
}

func gfcIsNamespaceDeclaration(s *tokens.Stream, i int) bool {
	return gfcIsKeyword(s, i, "namespace") && !gfcIsNamespaceOperator(s, i)
}

// gfcIsClassy reports a T_CLASS, T_INTERFACE, T_TRAIT or T_ENUM keyword ("enum"
// is a keyword only when a name follows).
func gfcIsClassy(s *tokens.Stream, i int) bool {
	if i < 0 || s.At(i).Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(s.At(i).Value) {
	case "class", "interface", "trait":
		return true
	case "enum":
		next := sigNext(s, i)
		return next >= 0 && s.At(next).Kind == token.Ident
	}
	return false
}

// gfcClassyBodyEnd returns the "}" closing the body of the classy keyword at i
// (stepping over an anonymous class's arguments), or -1.
func gfcClassyBodyEnd(s *tokens.Stream, i int) int {
	j := nextPunctOfKind(s, i, "(", "{")
	if j >= 0 && s.At(j).Value == "(" {
		j = nextPunctOfKind(s, s.MatchForward(j), "{")
	}
	if j < 0 {
		return -1
	}
	return s.MatchForward(j)
}

func (a *gfcAnalysis) build(s *tokens.Stream) {
	*a = gfcAnalysis{built: true}
	n := s.Len()
	hasOpenTag := false
	for i := range n {
		switch s.At(i).Kind {
		case token.AttributeStart:
			a.hasAttribute = true
		case token.OpenTag:
			if s.At(i).Value != "<?=" {
				hasOpenTag = true
			}
		}
	}

	// NamespacesAnalyzer::getDeclarations
	for i := 1; i < n; i++ {
		if !gfcIsNamespaceDeclaration(s, i) {
			continue
		}
		declEnd := nextPunctOfKind(s, i, ";", "{")
		if declEnd < 0 {
			break
		}
		var name strings.Builder
		for j := i + 1; j < declEnd; j++ {
			name.WriteString(s.At(j).Value)
		}
		var scopeEnd int
		if s.At(declEnd).Value == "{" {
			scopeEnd = s.MatchForward(declEnd)
			if scopeEnd < 0 {
				scopeEnd = n - 1
			}
		} else {
			scopeEnd = n
			for j := declEnd + 1; j < n; j++ {
				if gfcIsNamespaceDeclaration(s, j) {
					scopeEnd = j
					break
				}
			}
			scopeEnd--
		}
		a.namespaces = append(a.namespaces, gfcNamespace{start: i, end: scopeEnd, global: strings.TrimSpace(name.String()) == ""})
		i = scopeEnd
	}
	if len(a.namespaces) == 0 && hasOpenTag {
		start := 0
		if n > 0 && s.At(0).Kind == token.InlineHTML {
			start = 1
		}
		a.namespaces = append(a.namespaces, gfcNamespace{start: start, end: n - 1, global: true})
	}

	// function declarations, skipping classy bodies (methods are not functions)
	// and the "function" of "use function" imports
	for i := 0; i < n; i++ {
		if gfcIsClassy(s, i) {
			end := gfcClassyBodyEnd(s, i)
			if end < 0 {
				break
			}
			i = end
			continue
		}
		if gfcIsImportUse(s, i) {
			if end := nextPunctOfKind(s, i, ";"); end >= 0 {
				i = end
			}
			continue
		}
		if !gfcIsKeyword(s, i, "function") {
			continue
		}
		j := sigNext(s, i)
		if isPunctVal(s, j, "&") {
			j = sigNext(s, j)
		}
		if gfcIsTString(s, j) {
			a.declarations = append(a.declarations, j)
		}
	}

	// single (non-multi) "use function" imports
	for i := 0; i < n; i++ {
		if gfcIsClassy(s, i) {
			end := gfcClassyBodyEnd(s, i)
			if end < 0 {
				break
			}
			i = end
			continue
		}
		if !gfcIsImportUse(s, i) {
			continue
		}
		if imp, ok := gfcParseFunctionImport(s, i); ok {
			a.imports = append(a.imports, imp)
		}
	}
}

// gfcIsImportUse reports a "use" that is not a closure's "use (". Callers skip
// class bodies, where "use" imports traits.
func gfcIsImportUse(s *tokens.Stream, i int) bool {
	return gfcIsKeyword(s, i, "use") && !isPunctVal(s, sigNext(s, i), "(")
}

// gfcParseFunctionImport returns the "use function" import at useIdx, unless it
// is a multi or group import, which FunctionsAnalyzer ignores.
func gfcParseFunctionImport(s *tokens.Stream, useIdx int) (gfcImport, bool) {
	if !gfcIsKeyword(s, sigNext(s, useIdx), "function") {
		return gfcImport{}, false
	}
	end := -1
	for j := useIdx + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
			end = j
			break
		}
	}
	if end < 0 {
		end = s.Len() - 1
	}
	var full strings.Builder
	short := ""
	aliased := false
	for j := useIdx + 1; j < end; j++ {
		t := s.At(j)
		switch {
		case t.Kind == token.Punct && (t.Value == "," || t.Value == "{"):
			return gfcImport{}, false // multi or group import
		case gfcIsTString(s, j):
			short = t.Value
			if !aliased {
				full.WriteString(t.Value)
			}
		case t.Kind == token.Punct && t.Value == `\`:
			full.WriteString(t.Value)
		case kwIs(t, "as"):
			aliased = true
		}
	}
	if short == "" {
		return gfcImport{}, false
	}
	return gfcImport{start: useIdx, end: end, short: short, fullName: full.String()}, true
}

// gfcIsAttribute is AttributeAnalyzer::isAttribute.
func gfcIsAttribute(s *tokens.Stream, index int) bool {
	start := -1
	for j := index - 1; j >= 0; j-- {
		t := s.At(j)
		stop := false
		switch t.Kind {
		case token.Punct:
			stop = t.Value == ";" || t.Value == "{"
		case token.AttributeStart, token.AttributeEnd, token.OpenTag:
			stop = true
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "function", "private", "protected", "public", "return":
				stop = true
			}
		case token.Variable:
			stop = t.Value != "$"
		}
		if stop {
			start = j
			break
		}
	}
	if start < 0 || s.At(start).Kind != token.AttributeStart {
		return false
	}
	count := 0
	for j := start + 1; j < index; j++ {
		if isPunctVal(s, j, "(") {
			count++
		} else if isPunctVal(s, j, ")") {
			count--
		}
	}
	return count == 0
}
