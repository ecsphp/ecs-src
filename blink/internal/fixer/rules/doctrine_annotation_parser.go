package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// A port of PHP-CS-Fixer's Doctrine\Annotation DocLexer + Tokens, enough for the
// DoctrineAnnotationArrayAssignment and DoctrineAnnotationIndentation fixers: it
// tokenizes the Doctrine-annotation grammar inside a /** */ docblock, keeping the
// non-annotation text as T_NONE gaps so getCode rebuilds the block byte for byte.

const (
	daTNone         = 1
	daTInteger      = 2
	daTString       = 3
	daTFloat        = 4
	daTIdentifier   = 100
	daTAt           = 101
	daTCloseCurly   = 102
	daTCloseParen   = 103
	daTComma        = 104
	daTEquals       = 105
	daTNamespaceSep = 107
	daTOpenCurly    = 108
	daTOpenParen    = 109
	daTColon        = 112
	daTMinus        = 113
)

var daNoCase = map[string]int{
	"@": daTAt, ",": daTComma, "(": daTOpenParen, ")": daTCloseParen,
	"{": daTOpenCurly, "}": daTCloseCurly, "=": daTEquals, ":": daTColon,
	"-": daTMinus, "\\": daTNamespaceSep,
}

// daToken mirrors Doctrine\Annotation\Token.
type daToken struct {
	typ     int
	content string
	pos     int
}

// docLexerRe mirrors DocLexer's scan regex: three catchable groups (identifier,
// number, string) plus a single-char catch; whitespace and "*" runs are the
// dropped delimiters.
var docLexerRe = regexp.MustCompile(
	`(?i)([a-z_\\][a-z0-9_:\\]*[a-z_][a-z0-9_]*)` +
		`|([+-]?[0-9]+(?:\.[0-9]+)*(?:[eE][+-]?[0-9]+)?)` +
		`|("(?:""|[^"])*")` +
		`|\s+|\*+|(.)`,
)

// daScan tokenizes input the way DocLexer::scan does.
func daScan(input string) []daToken {
	var toks []daToken
	for _, m := range docLexerRe.FindAllStringSubmatchIndex(input, -1) {
		// groups: 1 ident, 2 number, 3 string, 4 single char
		for g := 1; g <= 4; g++ {
			gs, ge := m[2*g], m[2*g+1]
			if gs < 0 {
				continue
			}
			value := input[gs:ge]
			typ := daGetType(value)
			// DocLexer::getType mutates a string value to its unescaped, unquoted
			// form; createFromDocComment re-wraps it, so lengths still line up
			if typ == daTString {
				// DocLexer uses substr($value, 1, strlen-2); a lone `"` (len < 2)
				// yields an empty string in PHP rather than an out-of-range slice
				if len(value) >= 2 {
					value = strings.ReplaceAll(value[1:len(value)-1], `""`, `"`)
				} else {
					value = ""
				}
			}
			toks = append(toks, daToken{typ: typ, content: value, pos: gs})
			break
		}
	}
	return toks
}

// daGetType mirrors DocLexer::getType; for a string it also returns the
// unescaped content (outer quotes removed, "" -> ").
func daGetType(value string) int {
	if value == "" {
		return daTNone
	}
	if value[0] == '"' {
		return daTString
	}
	if t, ok := daNoCase[value]; ok {
		return t
	}
	if value[0] == '_' || value[0] == '\\' || isAsciiLetter(value[0]) {
		return daTIdentifier
	}
	if daIsNumeric(value) {
		if strings.ContainsAny(value, ".eE") {
			return daTFloat
		}
		return daTInteger
	}
	return daTNone
}

func isAsciiLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func daIsNumeric(v string) bool {
	if v == "" {
		return false
	}
	i := 0
	if v[0] == '+' || v[0] == '-' {
		i++
	}
	digits, dot, e := false, false, false
	for ; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= '0' && c <= '9':
			digits = true
		case c == '.' && !dot && !e:
			dot = true
		case (c == 'e' || c == 'E') && !e && digits:
			e = true
			if i+1 < len(v) && (v[i+1] == '+' || v[i+1] == '-') {
				i++
			}
		default:
			return false
		}
	}
	return digits
}

// daStringPos scans "@" occurrences preceded by whitespace or start, the same
// filter Tokens::createFromDocComment applies.
func daIsSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
}

// createFromDocComment mirrors Tokens::createFromDocComment.
func daCreateFromDocComment(content string, ignoredTags map[string]bool) []daToken {
	var toks []daToken
	ignoredTextPosition := 0
	currentPosition := 0
	for {
		nextAt := strings.IndexByte(content[currentPosition:], '@')
		if nextAt < 0 {
			break
		}
		nextAt += currentPosition
		if nextAt != 0 && !daIsSpaceByte(content[nextAt-1]) {
			currentPosition = nextAt + 1
			continue
		}

		scanned := daScan(content[nextAt:])
		var used []daToken
		nbScannedTokensToUse := 0
		nbScopes := 0
		var lastToken daToken
		for index := range scanned {
			tk := scanned[index]
			lastToken = tk
			if index == 0 && tk.typ != daTAt {
				break
			}
			if index == 1 {
				if tk.typ != daTIdentifier || ignoredTags[tk.content] {
					break
				}
				nbScannedTokensToUse = 2
			}
			if index >= 2 && nbScopes == 0 && tk.typ != daTNone && tk.typ != daTOpenParen {
				break
			}
			used = append(used, tk)
			if tk.typ == daTOpenParen {
				nbScopes++
			} else if tk.typ == daTCloseParen {
				nbScopes--
				if nbScopes == 0 {
					nbScannedTokensToUse = len(used)
					break
				}
			}
		}

		if nbScopes != 0 {
			break
		}

		if nbScannedTokensToUse != 0 {
			if ignoredTextLength := nextAt - ignoredTextPosition; ignoredTextLength != 0 {
				toks = append(toks, daToken{typ: daTNone, content: content[ignoredTextPosition : ignoredTextPosition+ignoredTextLength]})
			}
			lastTokenEndIndex := 0
			var last daToken
			for _, st := range used[:nbScannedTokensToUse] {
				tk := st
				if st.typ == daTString {
					tk = daToken{typ: st.typ, content: `"` + strings.ReplaceAll(st.content, `"`, `""`) + `"`, pos: st.pos}
				}
				if missing := tk.pos - lastTokenEndIndex; missing > 0 {
					toks = append(toks, daToken{typ: daTNone, content: content[nextAt+lastTokenEndIndex : nextAt+lastTokenEndIndex+missing]})
				}
				toks = append(toks, daToken{typ: tk.typ, content: tk.content})
				lastTokenEndIndex = tk.pos + len(tk.content)
				last = tk
			}
			currentPosition = nextAt + last.pos + len(last.content)
			ignoredTextPosition = currentPosition
		} else {
			_ = lastToken
			currentPosition = nextAt + 1
		}
	}

	if ignoredTextPosition < len(content) {
		toks = append(toks, daToken{typ: daTNone, content: content[ignoredTextPosition:]})
	}
	return toks
}

func daGetCode(toks []daToken) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.content)
	}
	return b.String()
}

var daAnnotationEndRe = regexp.MustCompile(`^(\r?\n\s*\*\s*)*\s*$`)

// daGetAnnotationEnd mirrors Tokens::getAnnotationEnd; returns -1 for "null".
func daGetAnnotationEnd(toks []daToken, index int) int {
	current := -1
	if index+2 < len(toks) {
		if toks[index+2].typ == daTOpenParen {
			current = index + 2
		} else if index+3 < len(toks) && toks[index+2].typ == daTNone &&
			toks[index+3].typ == daTOpenParen && daAnnotationEndRe.MatchString(toks[index+2].content) {
			current = index + 3
		}
	}
	if current >= 0 {
		level := 0
		for ; current < len(toks); current++ {
			switch toks[current].typ {
			case daTOpenParen:
				level++
			case daTCloseParen:
				level--
			}
			if level == 0 {
				return current
			}
		}
		return -1
	}
	return index + 1
}

// doctrineIgnoredTags is the default ignored_tags list shared by the fixers.
var doctrineIgnoredTags = buildStringSet([]string{
	"abstract", "access", "code", "deprec", "encode", "exception", "final", "ingroup",
	"inheritdoc", "inheritDoc", "magic", "name", "toc", "tutorial", "private", "static",
	"staticvar", "staticVar", "throw", "api", "author", "category", "copyright", "deprecated",
	"example", "filesource", "global", "ignore", "internal", "license", "link", "method",
	"package", "param", "property", "property-read", "property-write", "return", "see", "since",
	"source", "subpackage", "throws", "todo", "TODO", "usedBy", "uses", "var", "version",
	"after", "afterClass", "backupGlobals", "backupStaticAttributes", "before", "beforeClass",
	"codeCoverageIgnore", "codeCoverageIgnoreStart", "codeCoverageIgnoreEnd", "covers",
	"coversDefaultClass", "coversNothing", "dataProvider", "depends", "expectedException",
	"expectedExceptionCode", "expectedExceptionMessage", "expectedExceptionMessageRegExp",
	"group", "large", "medium", "preserveGlobalState", "requires", "runTestsInSeparateProcesses",
	"runInSeparateProcess", "small", "test", "testdox", "ticket", "SuppressWarnings",
	"noinspection", "package_version", "enduml", "startuml", "psalm", "phpstan", "template",
	"fix", "FIXME", "fixme", "override",
})

func buildStringSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// classModifierKw are the class modifiers skipped before a "class" keyword.
var classModifierKw = map[string]bool{"abstract": true, "final": true, "readonly": true}

// memberModifierKw mirrors MODIFIER_KINDS for skipping to a class member.
var memberModifierKw = map[string]bool{
	"public": true, "protected": true, "private": true, "final": true,
	"abstract": true, "readonly": true,
}

// docAnnotationEligible mirrors nextElementAcceptsDoctrineAnnotations: the
// docblock at index must precede a class or a member of a class (not interface,
// trait or enum).
func docAnnotationEligible(s *tokens.Stream, index int) bool {
	i := sigNext(s, index)
	for i >= 0 && s.At(i).Kind == token.Keyword && classModifierKw[strings.ToLower(s.At(i).Value)] {
		i = sigNext(s, i)
	}
	if i < 0 {
		return false
	}
	if s.At(i).Kind == token.Keyword && strings.ToLower(s.At(i).Value) == "class" {
		return true
	}
	// skip member modifiers / type tokens, then require an enclosing "class" body
	for i >= 0 && s.At(i).Kind == token.Keyword && memberModifierKw[strings.ToLower(s.At(i).Value)] {
		i = sigNext(s, i)
	}
	if i < 0 {
		return false
	}
	return enclosingClassIsClass(s, i)
}

// enclosingClassIsClass reports whether the nearest enclosing "{ }" block around
// idx is a class body (not interface/trait/enum).
func enclosingClassIsClass(s *tokens.Stream, idx int) bool {
	depth := 0
	for j := idx - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth == 0 {
				kind, kw := classifyBrace(s, j)
				return kind == braceClassLike && kw >= 0 && strings.ToLower(s.At(kw).Value) == "class"
			}
			depth--
		}
	}
	return false
}

// applyToDoctrineAnnotations runs fn over each eligible docblock's parsed
// annotation tokens, replacing the docblock when fn mutates them.
func applyToDoctrineAnnotations(s *tokens.Stream, fn func(toks []daToken) bool) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		if !docAnnotationEligible(s, i) {
			continue
		}
		toks := daCreateFromDocComment(s.At(i).Value, doctrineIgnoredTags)
		if fn(toks) {
			rebuilt := daGetCode(toks)
			if rebuilt != s.At(i).Value {
				s.SetValue(i, rebuilt)
				changed = true
			}
		}
	}
	return changed
}
