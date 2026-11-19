package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FunctionCommentSniff.php
//
// SquizFunctionComment is PEARFunctionComment's blank-line fix plus the Squiz
// type spellings of @param and @return ("int" -> "integer") and the alignment
// of @param types and names.
type SquizFunctionComment struct{}

func (SquizFunctionComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\FunctionCommentSniff`
}

func (SquizFunctionComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FunctionCommentSniff.php"
}

func (SquizFunctionComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if !phpcsIsNamedFunction(f, i) {
				continue
			}
			if commentEnd, tags, ok := pearFunctionCommentPreamble(f, i); ok {
				squizProcessReturn(f, tags)
				squizProcessParams(f, commentEnd, tags)
			}
		}
	})
}

var squizReturnParts = regexp.MustCompile(`(?i)^((?:\|?(?:array\([^\)]*\)|[\\a-z0-9_\[\]]+))*)( .*)?`)

func squizSuggestTypes(typ string) string {
	var names []string
	for name := range strings.SplitSeq(typ, "|") {
		if s := phpcsSuggestType(name); !slices.Contains(names, s) {
			names = append(names, s)
		}
	}
	return strings.Join(names, "|")
}

func squizProcessReturn(f *slvFile, tags []int) {
	ret := -1
	for _, tag := range tags {
		if f.toks[tag].content == "@return" {
			if ret >= 0 {
				return
			}
			ret = tag
		}
	}
	if ret < 0 || ret+2 >= len(f.toks) {
		return
	}
	content := f.toks[ret+2].content
	if content == "" || content == "0" || f.toks[ret+2].doc != slvDocString {
		return
	}
	m := squizReturnParts.FindStringSubmatch(content)
	if want := squizSuggestTypes(m[1]); want != m[1] {
		f.replaceToken(ret+2, want+m[2])
	}
}

type squizCommentLine struct {
	comment string
	token   int
	indent  int
}

type squizParam struct {
	tag                 int
	typ, varName        string
	comment             string
	lines               []squizCommentLine
	typeSpace, varSpace int
}

func squizProcessParams(f *slvFile, commentCloser int, tags []int) {
	var params []squizParam
	maxType, maxVar := 0, 0
	for pos, tag := range tags {
		if f.toks[tag].content != "@param" {
			continue
		}
		p := squizParam{tag: tag}
		if tag+2 < len(f.toks) && f.toks[tag+2].doc == slvDocString {
			content := f.toks[tag+2].content
			m := pearParseParam(content)
			typeLen := len(m.typ)
			p.typ = strings.Trim(m.typ, " \t\n\r\x00\x0B")
			p.typeSpace = typeLen - len(p.typ)
			maxType = max(maxType, len(p.typ))
			if content[0] != '$' && m.hasVar {
				p.varName = m.varName
				maxVar = max(maxVar, len(p.varName))
				if m.hasComment {
					p.varSpace = len(m.space)
					p.comment = m.comment
					p.lines = append(p.lines, squizCommentLine{comment: m.comment, token: tag + 2, indent: p.varSpace})
					end := commentCloser
					if pos+1 < len(tags) {
						end = tags[pos+1]
					}
					for i := tag + 3; i < end; i++ {
						if f.toks[i].doc != slvDocString {
							continue
						}
						indent := 0
						if f.toks[i-1].doc == slvDocWhitespace {
							indent = f.toks[i-1].length
						}
						p.comment += " " + f.toks[i].content
						p.lines = append(p.lines, squizCommentLine{comment: f.toks[i].content, token: i, indent: indent})
					}
				} else {
					p.lines = append(p.lines, squizCommentLine{token: -1})
				}
			}
		}
		params = append(params, p)
	}

	for _, p := range params {
		if p.typ == "" {
			continue
		}
		var suggested []string
		for name := range strings.SplitSeq(p.typ, "|") {
			if name == "" {
				continue
			}
			suggested = append(suggested, phpcsSuggestType(strings.TrimPrefix(name, "?")))
		}
		if want := strings.Join(suggested, "|"); want != p.typ {
			first := ""
			if len(p.lines) > 0 {
				first = p.lines[0].comment
			}
			diff := len(p.typ) - len(want)
			f.beginChangeset()
			f.replaceToken(p.tag+2, want+strings.Repeat(" ", p.typeSpace)+p.varName+strings.Repeat(" ", p.varSpace)+first)
			squizReindentLines(f, p.lines, diff, false)
			f.endChangeset()
		}
		if p.varName == "" {
			continue
		}
		if spaces := maxType - len(p.typ) + 1; p.typeSpace != spaces {
			f.beginChangeset()
			f.replaceToken(p.tag+2, p.typ+strings.Repeat(" ", spaces)+p.varName+strings.Repeat(" ", p.varSpace)+p.lines[0].comment)
			squizReindentLines(f, p.lines, p.typeSpace-spaces, true)
			f.endChangeset()
		}
		if p.comment == "" {
			continue
		}
		if spaces := maxVar - len(p.varName) + 1; p.varSpace != spaces {
			f.beginChangeset()
			f.replaceToken(p.tag+2, p.typ+strings.Repeat(" ", p.typeSpace)+p.varName+strings.Repeat(" ", spaces)+p.lines[0].comment)
			squizReindentLines(f, p.lines, p.varSpace-spaces, true)
			f.endChangeset()
		}
	}
}

// squizReindentLines shifts the indent of a param comment's continuation
// lines by diff; skipNonPositive mirrors the checks that leave an indent
// that would drop to zero or below alone.
func squizReindentLines(f *slvFile, lines []squizCommentLine, diff int, skipNonPositive bool) {
	for n, l := range lines {
		if n == 0 || l.indent == 0 {
			continue
		}
		indent := l.indent - diff
		if indent <= 0 && skipNonPositive {
			continue
		}
		f.replaceToken(l.token-1, strings.Repeat(" ", max(indent, 0)))
	}
}

var phpcsAllowedTypes = map[string]bool{
	"array": true, "boolean": true, "float": true, "integer": true, "mixed": true,
	"object": true, "string": true, "resource": true, "callable": true,
}

var phpcsArrayType = regexp.MustCompile(`(?i)^array\(\s*([^\s^=^>]*)(\s*=>\s*(.*))?\s*\)`)

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
		type1, type2 := phpcsSuggestType(m[1]), phpcsSuggestType(m[3])
		if type2 != "" {
			type2 = " => " + type2
		}
		return "array(" + type1 + type2 + ")"
	}
	if phpcsAllowedTypes[lower] {
		return lower
	}
	return varType
}
