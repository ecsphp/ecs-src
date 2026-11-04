package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/VoidReturnFixer.php
//
// VoidReturn adds a ": void" return type to functions whose body only returns
// without a value (risky). A "@return" other than "void" keeps the function as
// is, and a method without a body is typed only from "@return void".
type VoidReturn struct {
	// SkipLambda is the inverse of the fix_lambda option (default true).
	SkipLambda bool
}

func (f VoidReturn) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["fix_lambda"].(bool); ok {
		f.SkipLambda = !v
	}
	return f
}

func (VoidReturn) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\VoidReturnFixer`
}

func (VoidReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/VoidReturnFixer.php"
}

// vrNonVoidMagic are the methods where ": void" is a syntax error.
var vrNonVoidMagic = map[string]bool{
	"__clone": true, "__construct": true, "__debuginfo": true, "__destruct": true,
	"__isset": true, "__serialize": true, "__set_state": true, "__sleep": true,
	"__tostring": true,
}

func (f VoidReturn) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		if !kwIs(s.At(index), "function") {
			continue
		}
		if f.SkipLambda && taIsLambda(s, index) {
			continue
		}
		name := sigNext(s, index)
		if name >= 0 && taIsTString(s, name) && vrNonVoidMagic[strings.ToLower(s.At(name).Value)] {
			continue
		}
		startIndex := nextPunctOfKind(s, index, "{", ";")
		if startIndex < 0 {
			continue
		}
		endFuncIndex := vrSignatureEnd(s, index, startIndex)
		if endFuncIndex < 0 {
			continue
		}
		if isPunctVal(s, sigNext(s, endFuncIndex), ":") {
			continue // has a return type
		}
		if isPunctVal(s, startIndex, ";") {
			// no body: fall back to the phpdoc
			if vrHasReturnAnnotation(s, index, true) {
				vrAddVoid(s, endFuncIndex)
				changed = true
			}
			continue
		}
		if vrHasReturnAnnotation(s, index, false) {
			continue
		}
		endIndex := s.MatchForward(startIndex)
		if endIndex < 0 {
			continue
		}
		if vrHasVoidReturn(s, startIndex, endIndex) {
			vrAddVoid(s, endFuncIndex)
			changed = true
		}
	}
	return changed
}

// vrSignatureEnd finds the ")" that PHP's getPrevTokenOfKind($startIndex, [')'])
// lands on. Scanning back would stop on a DNF type's ")", which PHP gives its own kind.
func vrSignatureEnd(s *tokens.Stream, index, startIndex int) int {
	open := nextPunctOfKind(s, index, "(")
	if open < 0 || open > startIndex {
		return -1
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 || closeIdx > startIndex {
		return -1
	}
	if use := sigNext(s, closeIdx); taKw(s, use, "use") {
		if useOpen := sigNext(s, use); isPunctVal(s, useOpen, "(") {
			if useClose := s.MatchForward(useOpen); useClose >= 0 && useClose < startIndex {
				return useClose
			}
		}
	}
	return closeIdx
}

func vrAddVoid(s *tokens.Stream, endFuncIndex int) {
	s.InsertSliceAt(endFuncIndex+1, []token.Token{
		{Kind: token.Punct, Value: ":"},
		{Kind: token.Whitespace, Value: " "},
		{Kind: token.Ident, Value: "void"},
	})
}

// vrHasVoidReturn mirrors hasVoidReturn: no "return <value>" and no yield in
// the body, nested closures and anonymous classes excluded.
func vrHasVoidReturn(s *tokens.Stream, startIndex, endIndex int) bool {
	for i := startIndex; i < endIndex; i++ {
		if (taIsClassKw(s, i) && taIsAnonymousClass(s, i)) || (kwIs(s.At(i), "function") && taIsLambda(s, i)) {
			open := nextPunctOfKind(s, i, "{")
			if open < 0 {
				return true
			}
			i = s.MatchForward(open)
			if i < 0 {
				return true
			}
			continue
		}
		if kwIs(s.At(i), "yield") || isYieldFrom(s.At(i)) {
			return false // generators cannot return void
		}
		if !kwIs(s.At(i), "return") {
			continue
		}
		i = sigNext(s, i)
		if !isPunctVal(s, i, ";") {
			return false
		}
	}
	return true
}

// vrHasReturnAnnotation mirrors hasVoidReturnAnnotation (wantVoid) and
// hasReturnAnnotation (!wantVoid): whether the function's docblock has a
// "@return" whose types are exactly ["void"], or one whose types are not.
func vrHasReturnAnnotation(s *tokens.Stream, index int, wantVoid bool) bool {
	return slices.Contains(vrReturnAnnotations(s, index), wantVoid)
}

// vrReturnAnnotations mirrors findReturnAnnotations: for each "@return" of the
// docblock before the function's modifiers and attributes, whether its types
// are exactly ["void"].
func vrReturnAnnotations(s *tokens.Stream, index int) []bool {
	for {
		index--
		for index >= 0 && s.At(index).Kind == token.Whitespace {
			index--
		}
		if index < 0 {
			return nil
		}
		if s.At(index).Kind == token.AttributeEnd {
			index = s.AttributeStartOf(index)
			if index < 0 {
				return nil
			}
		}
		if s.At(index).Kind == token.AttributeStart || taIsModifier(s, index) {
			continue
		}
		break
	}
	if s.At(index).Kind != token.DocComment {
		return nil
	}
	lines := splitDocLines(s.At(index).Value)
	var out []bool
	for _, a := range docAnnotations(lines) {
		if vrTagName(lines[a.start]) == "return" {
			out = append(out, vrTypesAreVoid(strings.Join(lines[a.start:a.end+1], "")))
		}
	}
	return out
}

var vrTagNameRe = regexp.MustCompile(`@[a-zA-Z0-9_-]+`)

// vrTagName mirrors Tag::getName: the first "@name" followed by whitespace or
// the end of the line.
func vrTagName(line string) string {
	for _, loc := range vrTagNameRe.FindAllStringIndex(line, -1) {
		end := loc[1]
		if end == len(line) || strings.ContainsRune(" \t\n\r\v\f", rune(line[end])) {
			return line[loc[0]+1 : end]
		}
	}
	return "other"
}

var vrReturnPrefixRe = regexp.MustCompile(`(?i)\A(?:[ \t]*\*|/\*\*)[ \t*]*@return[ \t]+`)

// vrTypesAreVoid reports whether Annotation::getTypes() of a "@return" is
// exactly ["void"]: TypeExpression::REGEX_TYPES must match "void" and nothing more.
func vrTypesAreVoid(content string) bool {
	m := vrReturnPrefixRe.FindStringIndex(content)
	if m == nil || !strings.HasPrefix(content[m[1]:], "void") {
		return false
	}
	rest := content[m[1]+len("void"):]
	if rest == "" {
		return true
	}
	c := rest[0]
	if vrIsIdentByte(c) || c == '\\' || c == '-' || c == ':' {
		return false
	}
	// a union or intersection glue followed by another type
	j := 0
	for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
		j++
	}
	if j < len(rest) && (rest[j] == '|' || rest[j] == '&') {
		k := j + 1
		for k < len(rest) && (rest[k] == ' ' || rest[k] == '\t') {
			k++
		}
		if vrStartsType(rest[k:]) {
			return false
		}
	}
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f', '*', '.', '$':
		return true
	case '&':
		return len(rest) > 1 && strings.ContainsRune(".$ \t\n\r\v\f", rune(rest[1]))
	}
	return false
}

func vrIsDigit(c byte) bool { return c >= '0' && c <= '9' }

// vrIsIdentByte mirrors TypeExpression::REGEX_IDENTIFIER's character class.
func vrIsIdentByte(c byte) bool {
	switch {
	case c <= 0x2f, c >= 0x3a && c <= 0x40, c >= 0x5b && c <= 0x5e, c == 0x60, c >= 0x7b && c <= 0x7f:
		return false
	}
	return true
}

// vrStartsType reports whether a PHPDoc type can start at the beginning of x.
func vrStartsType(x string) bool {
	if x == "" {
		return false
	}
	c := x[0]
	switch {
	case vrIsIdentByte(c), c == '\\', c == '?', c == '(', c == '\'', c == '"':
		return true
	case c == '+' || c == '-':
		return len(x) > 1 && (vrIsDigit(x[1]) || x[1] == '.' && len(x) > 2 && vrIsDigit(x[2]))
	case c == '.':
		return len(x) > 1 && vrIsDigit(x[1])
	case c == '$' || c == '@':
		l := strings.ToLower(x[1:])
		return strings.HasPrefix(l, "this") || strings.HasPrefix(l, "self") || strings.HasPrefix(l, "static")
	}
	return false
}
