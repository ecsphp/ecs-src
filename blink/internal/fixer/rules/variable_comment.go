package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/VariableCommentSniff.php
//
// VariableComment normalizes the type of a property's @var tag the way
// PHP_CodeSniffer suggests it: "int" becomes "integer", "Boolean" "boolean",
// "Array()" "array" and so on.
type VariableComment struct{}

func (VariableComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\VariableCommentSniff`
}

func (VariableComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/VariableCommentSniff.php"
}

var varTagParts = regexp.MustCompile(`(?i)^((?:\|?(?:array\([^)]*\)|[\\a-z0-9\[\]]+))*)( .*)?`)

// memberVarSkip are the words PHP_CodeSniffer steps over between a property and
// its docblock.
var memberVarSkip = map[string]bool{
	"public": true, "private": true, "protected": true, "var": true, "static": true,
	"readonly": true, "final": true, "abstract": true, "null": true, "true": true,
	"false": true, "self": true, "parent": true, "array": true,
}

func (VariableComment) Fix(s *tokens.Stream) bool {
	eol := phpcsEolChar(s)
	docs := map[int]bool{}
	for _, body := range phpcsOOBodies(s) {
		braces, parens := 0, 0
		for i := body[0] + 1; i < body[1]; i++ {
			t := s.At(i)
			switch {
			case t.Kind == token.Punct && t.Value == "{":
				braces++
			case t.Kind == token.Punct && t.Value == "}":
				braces--
			case t.Kind == token.Punct && t.Value == "(":
				parens++
			case t.Kind == token.Punct && t.Value == ")":
				parens--
			case t.Kind == token.Variable && braces == 0 && parens == 0:
				if d := memberVarDoc(s, i); d >= 0 {
					docs[d] = true
				}
			}
		}
	}
	changed := false
	for d := range docs {
		if v, ok := fixVarTag(s.At(d).Value, eol); ok {
			s.SetValue(d, v)
			changed = true
		}
	}
	return changed
}

// memberVarDoc returns the docblock of the property at i, -1 when another
// token comes first.
func memberVarDoc(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Ident:
			continue
		case token.Keyword:
			if memberVarSkip[strings.ToLower(t.Value)] {
				continue
			}
		case token.Punct:
			if len(t.Value) == 1 && strings.Contains("?|&()\\", t.Value) {
				continue
			}
		case token.AttributeEnd:
			if a := s.AttributeStartOf(j); a >= 0 {
				j = a
				continue
			}
		case token.DocComment:
			return j
		}
		return -1
	}
	return -1
}

// fixVarTag rewrites the type of the first @var tag in doc to its suggested form.
func fixVarTag(doc, eol string) (string, bool) {
	for _, tag := range phpcsDocTags(doc, eol) {
		if tag.Name != "@var" {
			continue
		}
		if tag.StrPos < 0 {
			return doc, false
		}
		parts := varTagParts.FindStringSubmatch(tag.Str)
		var suggested []string
		for name := range strings.SplitSeq(parts[1], "|") {
			if n := phpcsSuggestType(name); !slices.Contains(suggested, n) {
				suggested = append(suggested, n)
			}
		}
		want := strings.Join(suggested, "|")
		if want == parts[1] {
			return doc, false
		}
		return replaceDocStrings(doc, map[int][2]string{tag.StrPos: {tag.Str, want + parts[2]}}), true
	}
	return doc, false
}

var (
	phpcsAllowedTypes = map[string]bool{
		"array": true, "boolean": true, "float": true, "integer": true, "mixed": true,
		"object": true, "string": true, "resource": true, "callable": true,
	}
	phpcsArrayType = regexp.MustCompile(`(?i)^array\(\s*([^\s^=>]*)(\s*=>\s*(.*))?\s*\)`)
)

// phpcsSuggestType is PHP_CodeSniffer's Common::suggestType.
func phpcsSuggestType(varType string) string {
	if varType == "" || phpcsAllowedTypes[varType] {
		return varType
	}
	lower := strings.ToLower(varType)
	switch lower {
	case "bool", "boolean":
		return "boolean"
	case "double", "real", "float":
		return "float"
	case "int", "integer":
		return "integer"
	case "array()", "array":
		return "array"
	}
	if strings.Contains(lower, "array(") {
		m := phpcsArrayType.FindStringSubmatch(varType)
		if m == nil {
			return "array"
		}
		type2 := phpcsSuggestType(m[3])
		if type2 != "" {
			type2 = " => " + type2
		}
		return "array(" + phpcsSuggestType(m[1]) + type2 + ")"
	}
	if phpcsAllowedTypes[lower] {
		return lower
	}
	return varType
}
