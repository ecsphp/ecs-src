package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// constructKeywords are the keywords PSR-12 requires to be followed by a single
// space (single_space_around_construct).
var constructKeywords = map[string]bool{
	"abstract": true, "as": true, "case": true, "catch": true, "class": true,
	"do": true, "else": true, "elseif": true, "final": true, "finally": true,
	"for": true, "foreach": true, "function": true, "if": true, "insteadof": true,
	"interface": true, "namespace": true, "new": true, "private": true,
	"protected": true, "public": true, "readonly": true, "static": true,
	"switch": true, "trait": true, "try": true, "use": true, "while": true,
}

// singleSpaceKnownKeywords are the plain keywords the options may name; virtual
// constructs of PHP-CS-Fixer (attribute, comment, type_colon, ...) have no
// keyword token in a flat stream and are ignored.
var singleSpaceKnownKeywords = map[string]bool{
	"abstract": true, "as": true, "break": true, "case": true, "catch": true, "class": true,
	"clone": true, "const": true, "continue": true, "do": true, "echo": true, "else": true,
	"elseif": true, "enum": true, "extends": true, "final": true, "finally": true, "for": true,
	"foreach": true, "function": true, "global": true, "goto": true, "if": true,
	"implements": true, "include": true, "include_once": true, "instanceof": true,
	"insteadof": true, "interface": true, "match": true, "namespace": true, "new": true,
	"print": true, "private": true, "protected": true, "public": true, "readonly": true,
	"require": true, "require_once": true, "return": true, "static": true, "switch": true,
	"throw": true, "trait": true, "try": true, "use": true, "var": true, "while": true,
	"yield": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/SingleSpaceAroundConstructFixer.php
//
// SingleSpaceAroundConstruct ensures a single space after a language construct
// keyword ("if(" -> "if (", "else{" -> "else {"). The zero value uses the
// built-in keyword set. Options: Followed replaces the keywords followed by a
// single space, Preceded lists keywords ("as", "else", "elseif", "use_lambda")
// preceded by one, YieldFrom collapses whitespace inside "yield from".
type SingleSpaceAroundConstruct struct {
	Followed  map[string]bool
	Preceded  map[string]bool
	YieldFrom bool
}

func (SingleSpaceAroundConstruct) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`
}

func (SingleSpaceAroundConstruct) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/SingleSpaceAroundConstructFixer.php"
}

func (f SingleSpaceAroundConstruct) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := singleSpaceStringList(config["constructs_followed_by_a_single_space"]); ok {
		f.Followed = map[string]bool{}
		for _, name := range list {
			if singleSpaceKnownKeywords[name] {
				f.Followed[name] = true
			}
		}
	}
	if list, ok := singleSpaceStringList(config["constructs_preceded_by_a_single_space"]); ok {
		f.Preceded = map[string]bool{}
		for _, name := range list {
			f.Preceded[name] = true
		}
	}
	if list, ok := singleSpaceStringList(config["constructs_contain_a_single_space"]); ok {
		f.YieldFrom = false
		for _, name := range list {
			if name == "yield_from" {
				f.YieldFrom = true
			}
		}
	}
	return f
}

func singleSpaceStringList(v any) ([]string, bool) {
	switch list := v.(type) {
	case []string:
		return list, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out, true
	}
	return nil, false
}

func (f SingleSpaceAroundConstruct) followedKeyword(kw string) bool {
	if f.Followed != nil {
		return f.Followed[kw]
	}
	return constructKeywords[kw]
}

func (f SingleSpaceAroundConstruct) fixYieldFrom(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 3; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "yield" ||
			s.At(i+1).Kind != token.Whitespace || s.At(i+1).Value == " " {
			continue
		}
		if n := s.At(i + 2); (n.Kind == token.Ident || n.Kind == token.Keyword) && strings.ToLower(n.Value) == "from" {
			s.SetValue(i+1, " ")
			changed = true
		}
	}
	return changed
}

func (f SingleSpaceAroundConstruct) fixPreceded(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i > 0; i-- {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		kw := strings.ToLower(t.Value)
		if !f.Preceded[kw] && (kw != "use" || !f.Preceded["use_lambda"] || nextSignificantValue(s, i) != "(") {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind == token.Whitespace {
			if !hasNewline(prev.Value) {
				if prev.Value != " " {
					s.SetValue(i-1, " ")
					changed = true
				}
			} else if p := i - 2; p >= 0 && isBlockOrDocComment(s.At(p)) {
				// a block/doc comment directly before the construct joins it onto the
				// comment's closing line ("*/ elseif"); php-cs-fixer collapses the
				// whitespace unless a "//"/"#" line comment precedes (code cannot follow)
				s.SetValue(i-1, " ")
				changed = true
			}
			continue
		}
		s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}

func (f SingleSpaceAroundConstruct) Fix(s *tokens.Stream) bool {
	changed := f.YieldFrom && f.fixYieldFrom(s)

	if len(f.Preceded) > 0 && f.fixPreceded(s) {
		changed = true
	}
	i := 0
	for i < s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !f.followedKeyword(strings.ToLower(t.Value)) || i+1 >= s.Len() {
			i++
			continue
		}
		next := s.At(i + 1)
		// "static" and "class" directly before "(" are a class reference, not a
		// construct: "new static(...)", "new class(...)". They take no space.
		kw := strings.ToLower(t.Value)
		classRef := kw == "static" || kw == "class"
		if next.Kind == token.Whitespace {
			if classRef && !hasNewline(next.Value) && i+2 < s.Len() &&
				s.At(i+2).Kind == token.Punct && s.At(i+2).Value == "(" {
				s.RemoveAt(i + 1) // "new static (" -> "new static("
				changed = true
				i++
				continue
			}
			if !hasNewline(next.Value) && next.Value != " " {
				s.SetValue(i+1, " ")
				changed = true
			}
			i++
			continue
		}
		switch next.Value {
		case "::", "->", "?->", ";", ",", ")", ":":
			// member access, statement end or label - not a construct body
		case "(":
			if !classRef {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
				i++
			}
		default:
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
			i++
		}
		i++
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoSpacesAfterFunctionNameFixer.php
//
// NoSpacesAfterFunctionName removes whitespace between a function name and its
// opening parenthesis ("foo ()" -> "foo()").
type NoSpacesAfterFunctionName struct{}

func (NoSpacesAfterFunctionName) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NoSpacesAfterFunctionNameFixer`
}

func (NoSpacesAfterFunctionName) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoSpacesAfterFunctionNameFixer.php"
}

func (NoSpacesAfterFunctionName) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident {
			continue
		}
		if i+2 < s.Len() &&
			s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			s.At(i+2).Kind == token.Punct && s.At(i+2).Value == "(" {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SpacesInsideParenthesesFixer.php
//
// NoSpacesInsideParenthesis removes single-line whitespace just inside
// parentheses ("( $a )" -> "($a)"). Newlines are kept for multi-line calls. With
// Single set (space: single) it forces one space inside instead ("($a)" ->
// "( $a )"); empty parentheses lose their inner whitespace and casts are skipped.
type NoSpacesInsideParenthesis struct {
	Single bool
}

func (NoSpacesInsideParenthesis) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\SpacesInsideParenthesesFixer`
}

func (NoSpacesInsideParenthesis) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SpacesInsideParenthesesFixer.php"
}

func (f NoSpacesInsideParenthesis) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["space"].(string); ok {
		f.Single = v == "single"
	}
	return f
}

func (f NoSpacesInsideParenthesis) Fix(s *tokens.Stream) bool {
	if f.Single {
		return spacesInsideParenthesesSingle(s)
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == "(" &&
			i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			!commentAt(s, i+2) { // keep the space before a trailing comment
			s.RemoveAt(i + 1)
			changed = true
		}
		if s.At(i).Kind == token.Punct && s.At(i).Value == ")" &&
			i >= 1 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}

// spacesInsideParenthesesSingle walks right to left so insertions never shift
// the indices still to be visited.
func spacesInsideParenthesesSingle(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			p := prevSignificantIndex(s, i)
			if p < 0 || spacesInsideParenthesesIsCastClose(s, i, p) {
				continue
			}
			if s.At(p).Kind == token.Punct && s.At(p).Value == "(" {
				if p != i-1 && !hasNewline(s.At(i-1).Value) {
					s.RemoveAt(i - 1)
					changed = true
				}
				continue
			}
			if spacesInsideParenthesesEdge(s, i-1, i, p) {
				changed = true
			}
		case "(":
			if _, _, isCast := castAt(s, i); isCast {
				continue
			}
			n := nextSignificantIndex(s, i)
			if n < 0 || (s.At(n).Kind == token.Punct && s.At(n).Value == ")") {
				continue
			}
			if spacesInsideParenthesesEdge(s, i+1, i+1, n) {
				changed = true
			}
		}
	}
	return changed
}

func spacesInsideParenthesesIsCastClose(s *tokens.Stream, closeIdx, typeIdx int) bool {
	if !isCastType(s.At(typeIdx)) {
		return false
	}
	o := prevSignificantIndex(s, typeIdx)
	if o < 0 {
		return false
	}
	_, c, ok := castAt(s, o)
	return ok && c == closeIdx
}

// spacesInsideParenthesesEdge forces a single space at wsIdx, the slot between a
// parenthesis and the inner token at innerIdx; insertAt is where a missing space
// goes. A whitespace containing a newline, or one before a comment, is kept.
func spacesInsideParenthesesEdge(s *tokens.Stream, wsIdx, insertAt, innerIdx int) bool {
	if wsIdx >= 0 && wsIdx < s.Len() && s.At(wsIdx).Kind == token.Whitespace {
		ws := s.At(wsIdx).Value
		if ws == " " || hasNewline(ws) {
			return false
		}
		if k := s.At(innerIdx).Kind; k == token.Comment || k == token.DocComment {
			return false
		}
		s.SetValue(wsIdx, " ")
		return true
	}
	s.InsertAt(insertAt, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/UnaryOperatorSpacesFixer.php
//
// UnaryOperatorSpaces removes whitespace between an operand and ++ or --
// ("$i ++" -> "$i++"). With Full set (only_dec_inc: false) it also removes the
// whitespace after the unary "!", "~", "@", "+" and "-" ("! $a" -> "!$a").
type UnaryOperatorSpaces struct {
	Full bool
}

func (UnaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\UnaryOperatorSpacesFixer`
}

func (UnaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/UnaryOperatorSpacesFixer.php"
}

func (f UnaryOperatorSpaces) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["only_dec_inc"].(bool); ok {
		f.Full = !v
	}
	return f
}

// unaryOperatorSpacesIsPredecessor reports whether the "!", "~", "@", "+" or "-"
// at i is a prefix (unary) operator.
func unaryOperatorSpacesIsPredecessor(s *tokens.Stream, i int) bool {
	switch s.At(i).Value {
	case "!", "~", "@":
		return true
	case "+", "-":
		prev, ok := prevSignificant(s, i)
		if !ok {
			return true
		}
		switch prev.Kind {
		case token.Variable, token.Ident, token.Number, token.String:
			return false
		}
		switch prev.Value {
		case ")", "]", "}", "++", "--":
			return false
		}
		return true
	}
	return false
}

func (f UnaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if f.Full && t.Kind == token.Punct && unaryOperatorSpacesIsPredecessor(s, i) {
			if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
				s.RemoveAt(i + 1)
				changed = true
			}
			continue
		}
		if t.Kind != token.Punct || (t.Value != "++" && t.Value != "--") {
			continue
		}
		if i >= 2 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) && isOperand(s.At(i-2)) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
		if i+2 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) && isOperand(s.At(i+2)) {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}
