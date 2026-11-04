package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// constructKeywords are the keywords PSR-12 requires to be followed by a single
// space (single_space_around_construct), the zero value's Followed.
var constructKeywords = map[string]bool{
	"abstract": true, "as": true, "case": true, "catch": true, "class": true,
	"do": true, "else": true, "elseif": true, "final": true, "finally": true,
	"for": true, "foreach": true, "function": true, "if": true, "insteadof": true,
	"interface": true, "namespace": true, "new": true, "private": true,
	"protected": true, "public": true, "readonly": true, "static": true,
	"switch": true, "trait": true, "try": true, "use": true, "while": true,
	"private_set": true, "protected_set": true, "public_set": true,
}

// singleSpaceKnownKeywords are the constructs that are a keyword token of the
// same name (T_ABSTRACT, T_AS, ...).
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
// SingleSpaceAroundConstruct is a port of SingleSpaceAroundConstructFixer. Its
// maps hold the construct names of the fixer's three options. The zero value
// follows constructKeywords and precedes and contains nothing.
type SingleSpaceAroundConstruct struct {
	Followed map[string]bool
	Preceded map[string]bool
	Contain  map[string]bool
}

func (SingleSpaceAroundConstruct) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`
}

func (SingleSpaceAroundConstruct) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/SingleSpaceAroundConstructFixer.php"
}

func (f SingleSpaceAroundConstruct) WithConfig(config map[string]any) fixer.Fixer {
	set := func(v any, into *map[string]bool) {
		if list, ok := singleSpaceStringList(v); ok {
			m := make(map[string]bool, len(list))
			for _, name := range list {
				m[name] = true
			}
			*into = m
		}
	}
	set(config["constructs_followed_by_a_single_space"], &f.Followed)
	set(config["constructs_preceded_by_a_single_space"], &f.Preceded)
	set(config["constructs_contain_a_single_space"], &f.Contain)
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

// ssacConstruct names the construct the token at i is, as PHP-CS-Fixer's token
// kinds tell them apart, or "" when it is none.
func ssacConstruct(s *tokens.Stream, i int) string {
	t := s.At(i)
	switch t.Kind {
	case token.AttributeEnd:
		return "attribute"
	case token.Comment:
		return "comment"
	case token.DocComment:
		return "php_doc"
	case token.OpenTag:
		if t.Value == "<?=" {
			return "open_tag_with_echo"
		}
		return "php_open"
	case token.Punct:
		if t.Value == ":" {
			if isTypeColon(s, i) {
				return "type_colon"
			}
			if isNamedArgumentColon(s, i) {
				return "named_argument"
			}
		}
		return ""
	case token.Ident:
		if !strings.EqualFold(t.Value, "clone") || memberPrev(s, i) {
			return ""
		}
		// a method or constant named "clone"
		if p := sigPrev(s, i); p >= 0 && (kwIs(s.At(p), "function") || kwIs(s.At(p), "const") || isReturnRef(s, p)) {
			return ""
		}
		return "clone"
	case token.Keyword:
	default:
		return ""
	}
	if isYieldFrom(t) {
		return "yield_from"
	}
	lv := strings.ToLower(t.Value)
	switch lv {
	case "private(set)", "protected(set)", "public(set)":
		return strings.TrimSuffix(lv, "(set)") + "_set"
	}
	if !singleSpaceKnownKeywords[lv] || memberPrev(s, i) || isNamedArgumentName(s, i) {
		return ""
	}
	// a function returning by reference named after a reserved word
	// ("function &print()") is a name
	if p := sigPrev(s, i); p >= 0 && isReturnRef(s, p) {
		return ""
	}
	switch lv {
	case "use":
		if n := sigNext(s, i); n >= 0 && isPunctVal(s, n, "(") {
			return "use_lambda"
		}
		if inClassLikeBody(s, i) {
			return "use_trait"
		}
	case "const", "function":
		if ssacIsImport(s, i) {
			return lv + "_import"
		}
	case "enum":
		if !cdIsClassy(s, i) {
			return ""
		}
	case "readonly":
		// "readonly(" is a function call, lexed as a name
		if n := sigNext(s, i); n >= 0 && isPunctVal(s, n, "(") {
			return ""
		}
	}
	return lv
}

// isYieldFrom reports whether t is the "yield from" keyword token.
func isYieldFrom(t token.Token) bool {
	return t.Kind == token.Keyword && len(t.Value) > len("yield") && strings.EqualFold(t.Value[:len("yield")], "yield")
}

// ssacIsImport is ImportTransformer: "const"/"function" right after "use", or
// inside a group import "use A\{const B, function c}".
func ssacIsImport(s *tokens.Stream, i int) bool {
	if p := sigPrev(s, i); p >= 0 && kwIs(s.At(p), "use") {
		return true
	}
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "=", "(":
			return false
		case "&":
			if isReturnRef(s, j) {
				return false
			}
		case "}":
			o := s.MatchBackward(j)
			p := -1
			if o >= 0 {
				p = sigPrev(s, o)
			}
			return p >= 0 && isPunctVal(s, p, `\`)
		}
	}
	return false
}

func (f SingleSpaceAroundConstruct) followedEnabled(name string) bool {
	followed := f.Followed
	if followed == nil {
		followed = constructKeywords
	}
	return name != "" && followed[name]
}

func (f SingleSpaceAroundConstruct) Fix(s *tokens.Stream) bool {
	before := s.Render()
	if f.Contain["yield_from"] {
		ssacFixYieldFromContent(s)
	}
	if len(f.Preceded) > 0 {
		f.fixPreceded(s)
	}
	f.fixFollowed(s)
	return s.Render() != before
}

// ssacFixYieldFromContent collapses each whitespace run inside a "yield from"
// token to one space, as PHP-CS-Fixer's Preg::replace('/\s+/', ' ', ...) does.
func ssacFixYieldFromContent(s *tokens.Stream) {
	for i := range s.Len() {
		t := s.At(i)
		if !isYieldFrom(t) || strings.ToLower(t.Value) == "yield from" {
			continue
		}
		if v := phpWhitespaceRunRe.ReplaceAllString(t.Value, " "); v != t.Value {
			s.SetValue(i, v)
		}
	}
}

// phpWhitespaceRunRe is PCRE's \s+ (space, \t, \n, \v, \f, \r).
var phpWhitespaceRunRe = regexp.MustCompile(`[ \t\n\v\f\r]+`)

func (f SingleSpaceAroundConstruct) fixPreceded(s *tokens.Stream) {
	for i := s.Len() - 1; i > 0; i-- {
		name := ""
		switch t := s.At(i); {
		case kwIs(t, "as"), kwIs(t, "else"), kwIs(t, "elseif"):
			if !memberPrev(s, i) {
				name = strings.ToLower(t.Value)
			}
		case kwIs(t, "use"):
			if ssacConstruct(s, i) == "use_lambda" {
				name = "use_lambda"
			}
		}
		if name == "" || !f.Preceded[name] || ssacIsFullLineCommentBefore(s, i) {
			continue
		}
		if s.At(i-1).Kind == token.Whitespace {
			s.SetValue(i-1, " ")
		} else {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
		}
	}
}

// ssacIsFullLineCommentBefore reports whether a "//" or "#" comment comes
// right before the token at i.
func ssacIsFullLineCommentBefore(s *tokens.Stream, i int) bool {
	p := prevSignificantIndex(s, i)
	return p >= 0 && s.At(p).Kind == token.Comment && cdIsLineComment(s.At(p))
}

func (f SingleSpaceAroundConstruct) fixFollowed(s *tokens.Stream) {
	for i := s.Len() - 2; i >= 0; i-- {
		name := ssacConstruct(s, i)
		if !f.followedEnabled(name) {
			continue
		}
		ws := i + 1
		next := s.At(ws)
		if next.Kind == token.Punct {
			switch next.Value {
			case ",", ":", ";", ")", "]":
				continue
			}
		}
		if name == "static" {
			n := sigNext(s, i)
			if n < 0 {
				continue
			}
			nt := s.At(n)
			if !kwIs(nt, "fn") && !kwIs(nt, "function") && !isPunctVal(s, n, `\`) && nt.Kind != token.Ident &&
				nt.Kind != token.Variable && !kwIs(nt, "array") && !isPunctVal(s, n, "?") {
				continue
			}
		}
		if name == "php_open" {
			// PHP-CS-Fixer's open tag holds the first whitespace character after
			// it; extra single-line spaces are dropped
			if next.Kind == token.Whitespace && len(next.Value) > 1 && next.Value[0] != '\n' && next.Value[0] != '\r' &&
				!strings.Contains(next.Value[1:], "\n") {
				s.SetValue(ws, next.Value[:1])
			}
			continue
		}
		if name == "class" {
			if n := sigNext(s, i); n >= 0 && isPunctVal(s, n, "(") {
				continue
			}
		}
		if (name == "extends" || name == "implements") && ssacIsMultilineExtendsWithMoreThanOneAncestor(s, i) {
			continue
		}
		if name == "return" && ssacIsMultiLineReturn(s, i) {
			continue
		}
		if name == "const" && ssacIsMultilineCommaSeparatedConstant(s, i) {
			continue
		}
		if name == "comment" || name == "php_doc" || name == "attribute" {
			if next.Kind == token.Whitespace && strings.Contains(next.Value, "\n") {
				continue
			}
		}
		if next.Kind == token.Whitespace && strings.Contains(next.Value, "\n") && ws+1 < s.Len() {
			nn := s.At(ws + 1)
			if nn.Kind == token.AttributeStart || nn.Kind == token.DocComment || (isComment(nn) && strings.HasPrefix(nn.Value, "#[")) {
				continue
			}
		}
		if next.Kind == token.Whitespace {
			if next.Value != " " {
				s.SetValue(ws, " ")
			}
			continue
		}
		s.InsertAt(ws, token.Token{Kind: token.Whitespace, Value: " "})
	}
}

// ssacIsMultiLineReturn is isMultiLineReturn.
func ssacIsMultiLineReturn(s *tokens.Stream, i int) bool {
	i++
	if i >= s.Len() || s.At(i).Kind != token.Whitespace || !strings.Contains(s.At(i).Value, "\n") {
		return false
	}
	nested := 0
	for i++; i < s.Len()-1; i++ {
		t := s.At(i)
		if strings.Contains(t.Value, "\n") {
			return true
		}
		switch {
		case isPunctVal(s, i, "{"):
			nested++
		case isPunctVal(s, i, "}"):
			nested--
		case nested == 0 && (isPunctVal(s, i, ";") || t.Kind == token.CloseTag):
			return false
		}
	}
	return false
}

// ssacIsMultilineExtendsWithMoreThanOneAncestor is
// isMultilineExtendsOrImplementsWithMoreThanOneAncestor.
func ssacIsMultilineExtendsWithMoreThanOneAncestor(s *tokens.Stream, i int) bool {
	more := false
	for i++; i < s.Len(); i++ {
		t := s.At(i)
		if isPunctVal(s, i, ",") {
			more = true
			continue
		}
		if isPunctVal(s, i, "{") {
			return false
		}
		if more && strings.Contains(t.Value, "\n") {
			return true
		}
	}
	return false
}

// ssacIsMultilineCommaSeparatedConstant is isMultilineCommaSeparatedConstant.
func ssacIsMultilineCommaSeparatedConstant(s *tokens.Stream, i int) bool {
	multiline, more := false, false
	for i < s.Len() && !isPunctVal(s, i, ";") && s.At(i).Kind != token.CloseTag {
		i++
		if i >= s.Len() {
			break
		}
		multiline = multiline || strings.Contains(s.At(i).Value, "\n")
		if isPunctVal(s, i, ",") {
			more = true
		}
		if isBlock, isStart := yodaBlock(s, i); isBlock && isStart {
			end := s.MatchForward(i)
			if s.At(i).Kind == token.AttributeStart {
				end = s.AttributeEnd(i)
			}
			if end < 0 {
				break
			}
			i = end
		}
	}
	return more && multiline
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

// language constructs php-cs-fixer treats as function-like calls (T_ARRAY,
// T_ECHO, T_EMPTY, T_INCLUDE[_ONCE], T_ISSET, T_LIST, T_PRINT, T_REQUIRE[_ONCE],
// T_UNSET). eval/exit/die are Ident in the lexer, handled by the name branch.
var noSpacesFunctionyKeywords = map[string]bool{
	"array": true, "echo": true, "empty": true, "include": true,
	"include_once": true, "isset": true, "list": true, "print": true,
	"require": true, "require_once": true, "unset": true,
}

// echo/print/include/require parentheses may be expression grouping rather than
// a call; the space is removed only when the matching ")" closes straight into ";".
var noSpacesGroupingKeywords = map[string]bool{
	"echo": true, "print": true, "include": true,
	"include_once": true, "require": true, "require_once": true,
}

func (NoSpacesAfterFunctionName) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "(" {
			continue
		}
		last := sigPrev(s, i)
		if last < 0 {
			continue
		}
		prev := s.At(last)

		if prev.Kind == token.Keyword && noSpacesGroupingKeywords[strings.ToLower(prev.Value)] {
			if end := s.MatchForward(i); end >= 0 {
				if n := sigNext(s, end); n >= 0 && (s.At(n).Kind != token.Punct || s.At(n).Value != ";") {
					continue
				}
			}
		}

		switch {
		case prev.Kind == token.Variable:
			// "$func ()"
		case prev.Kind == token.Keyword && noSpacesFunctionyKeywords[strings.ToLower(prev.Value)]:
		case prev.Kind == token.Ident:
			p := sigPrev(s, last)
			// "function foo ()" is a definition, not a call
			if p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "function") {
				continue
			}
		case prev.Kind == token.Punct && (prev.Value == ")" || prev.Value == "]"):
			// chained call "$f() ()" or "$a[0] ()"
		default:
			continue
		}

		if i >= 1 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			i--
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
// whitespace after the unary "!", "~", "@", "+", "-", "&" and "..." ("! $a" ->
// "!$a", "= & $b" -> "= &$b", "f(... $args)" -> "f(...$args)").
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

// unaryOperatorSpacesIsDynamicBraceClose reports whether the "}" at i closes a
// dynamic property, variable or index brace ("$a->{$b}", "${$b}", "$a{0}",
// "$a->b{0}", "$a[0]{1}"),
// which PHP-CS-Fixer's TokensAnalyzer counts as the end of an operand.
func unaryOperatorSpacesIsDynamicBraceClose(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	if open < 0 {
		return false
	}
	p := sigPrev(s, open)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Variable:
		return true
	case token.Ident:
		// "$a->b{0}": an index brace on a property
		pp := sigPrev(s, p)
		return pp >= 0 && s.At(pp).Kind == token.Punct && (s.At(pp).Value == "->" || s.At(pp).Value == "?->")
	case token.Punct:
		switch t.Value {
		case "->", "?->", "::", "$", "]":
			return true
		case "}":
			return unaryOperatorSpacesIsDynamicBraceClose(s, p)
		}
	}
	return false
}

// unaryOperatorSpacesIsSuccessor is TokensAnalyzer::isUnarySuccessorOperator:
// a "++" or "--" right after a variable, a name, "]" or a dynamic brace.
func unaryOperatorSpacesIsSuccessor(s *tokens.Stream, i int) bool {
	if v := s.At(i).Value; v != "++" && v != "--" {
		return false
	}
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Variable, token.Ident:
		return true
	case token.Punct:
		return t.Value == "]" || (t.Value == "}" && unaryOperatorSpacesIsDynamicBraceClose(s, p))
	}
	return false
}

// unaryOperatorSpacesIsPredecessor is TokensAnalyzer::isUnaryPredecessorOperator.
// "!", "~", "@" and "..." always are; "+", "-" and "&" are unless an operand
// ends right before them; "++"/"--" are when they are not a successor.
func unaryOperatorSpacesIsPredecessor(s *tokens.Stream, i int) bool {
	v := s.At(i).Value
	switch v {
	case "++", "--":
		return !unaryOperatorSpacesIsSuccessor(s, i)
	case "!", "~", "@", "...":
		return true
	case "+", "-", "&":
	default:
		return false
	}
	p := sigPrev(s, i)
	if p < 0 {
		return true
	}
	prev := s.At(p)
	operandEnd := false
	switch prev.Kind {
	case token.Variable, token.Ident, token.Number:
		operandEnd = true
	case token.String:
		// a string's end is: T_CONSTANT_ENCAPSED_STRING, the closing '"' or '`';
		// a heredoc/nowdoc ends with T_END_HEREDOC, which is not
		operandEnd = prev.ClosesString() && !prev.EndsHeredoc()
	case token.Punct:
		switch prev.Value {
		case "]", "}", "++", "--":
			operandEnd = true
		case ")":
			// a cast is a single token in PHP, after which the operator is unary
			open := s.MatchBackward(p)
			operandEnd = open < 0 || !isCastParen(s, open)
		}
	}
	if !operandEnd {
		return true
	}
	if v != "&" || prev.Kind != token.Ident {
		return false
	}
	// "Type & $param": a by-reference parameter after its type
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		switch {
		case t.Kind == token.OpenTag:
			return false
		case t.Kind == token.Punct && (t.Value == ";" || t.Value == "{" || t.Value == "}" || t.Value == "=>"):
			return false
		case t.Kind == token.Keyword && (strings.EqualFold(t.Value, "fn") || strings.EqualFold(t.Value, "function")):
			return true
		}
	}
	return false
}

// Fix ports UnaryOperatorSpacesFixer::applyFix: the whitespace (newlines
// included) before a unary successor and after a unary predecessor operator is
// removed, right to left.
func (f UnaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		if !f.Full && t.Value != "++" && t.Value != "--" {
			continue
		}
		if unaryOperatorSpacesIsSuccessor(s, i) {
			if i >= 1 && s.At(i-1).Kind == token.Whitespace {
				if p := i - 2; p < 0 || (s.At(p).Kind != token.Comment && s.At(p).Kind != token.DocComment) {
					s.RemoveAt(i - 1)
					changed = true
				}
			}
			continue
		}
		if unaryOperatorSpacesIsPredecessor(s, i) && i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}
