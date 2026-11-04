package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocParamOrderFixer.php
//
// PhpdocParamOrder orders @param annotations in a docblock to match the order
// of the documented function's parameters. Annotations that do not match a
// parameter are appended after the matching ones, and non-param annotations
// found between the first and last @param keep their slots at the end.
type PhpdocParamOrder struct{}

func (PhpdocParamOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocParamOrderFixer`
}

func (PhpdocParamOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocParamOrderFixer.php"
}

func (PhpdocParamOrder) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		if !poHasFunctionAfter(s, i) {
			return changed // mirrors upstream `return` when no function follows
		}
		open := poNextParen(s, i)
		if open < 0 {
			return changed
		}
		lines := splitDocLines(s.At(i).Value)
		anns := docAnnotations(lines)
		paramIdx := poParamAnnotationIndices(anns)
		if len(paramIdx) == 0 {
			continue
		}
		paramNames := poFunctionParamNames(s, open)
		poRewriteDocBlock(lines, anns, paramIdx, paramNames)
		if newVal := strings.Join(lines, ""); newVal != s.At(i).Value {
			s.SetValue(i, newVal)
			changed = true
		}
	}
	return changed
}

// poHasFunctionAfter reports whether a function or fn keyword appears after the
// docblock at i.
func poHasFunctionAfter(s *tokens.Stream, i int) bool {
	for j := i + 1; j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.Keyword && (strings.EqualFold(t.Value, "function") || strings.EqualFold(t.Value, "fn")) {
			return true
		}
	}
	return false
}

// poNextParen returns the index of the first "(" after i.
func poNextParen(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.Punct && t.Value == "(" {
			return j
		}
	}
	return -1
}

// poFunctionParamNames collects every variable token between the parameter
// parentheses, in order (mirroring getFunctionParamNames).
func poFunctionParamNames(s *tokens.Stream, open int) []string {
	close := s.MatchForward(open)
	if close < 0 {
		return nil
	}
	var names []string
	for j := open + 1; j < close; j++ {
		if s.At(j).Kind == token.Variable {
			names = append(names, s.At(j).Value)
		}
	}
	return names
}

func poParamAnnotationIndices(anns []docAnnotation) []int {
	var out []int
	for i, a := range anns {
		if a.name == "param" {
			out = append(out, i)
		}
	}
	return out
}

// poRewriteDocBlock reorders the annotations between the first and last @param.
func poRewriteDocBlock(lines []string, anns []docAnnotation, paramIdx []int, paramNames []string) {
	paramAnns := make([]docAnnotation, len(paramIdx))
	for k, idx := range paramIdx {
		paramAnns[k] = anns[idx]
	}
	ordered := poSortParamAnnotations(lines, paramNames, paramAnns)
	// non-param annotations between the first and last @param
	paramsStart := paramAnns[0].start
	paramsEnd := paramAnns[len(paramAnns)-1].end
	for _, a := range anns {
		if a.start < paramsStart || a.end > paramsEnd {
			continue
		}
		if a.name != "param" {
			ordered = append(ordered, docAnnContent(lines, a))
		}
	}

	pos := 0
	for _, a := range anns {
		if a.start < paramsStart || a.end > paramsEnd {
			continue
		}
		if pos >= len(ordered) {
			break
		}
		docAnnRemove(lines, a)
		lines[a.start] = ordered[pos]
		pos++
	}
}

// poSortParamAnnotations orders the matched @param contents by signature order,
// then appends the unmatched ones in their original order.
func poSortParamAnnotations(lines []string, paramNames []string, paramAnns []docAnnotation) []string {
	valid := map[int]string{}
	var validOrder []int
	for _, name := range paramNames {
		for _, idx := range poFindParamByIdentifier(lines, paramAnns, name) {
			if _, seen := valid[idx]; !seen {
				validOrder = append(validOrder, idx)
			}
			valid[idx] = docAnnContent(lines, paramAnns[idx])
		}
	}
	ordered := make([]string, 0, len(paramAnns))
	for _, idx := range validOrder {
		ordered = append(ordered, valid[idx])
	}
	for idx, a := range paramAnns {
		if _, ok := valid[idx]; !ok {
			ordered = append(ordered, docAnnContent(lines, a))
		}
	}
	return ordered
}

// poFindParamByIdentifier ports findParamAnnotationByIdentifier: it returns the
// paramAnns indices matching identifier, tracking "{" / "}" blocks that let a
// matched annotation span several entries.
func poFindParamByIdentifier(lines []string, paramAnns []docAnnotation, identifier string) []int {
	blockLevel := 0
	blockMatch := false
	var blockIndices []int
	for i, a := range paramAnns {
		content := docAnnContent(lines, a)
		blockStart := strings.Contains(content, "{")
		blockEnds := strings.Count(content, "}")
		if blockLevel == 0 && poParamLineMatches(content, identifier) {
			if blockStart {
				blockMatch = true
			} else {
				return []int{i}
			}
		}
		if blockStart {
			blockLevel++
		}
		blockLevel -= blockEnds
		if blockMatch {
			blockIndices = append(blockIndices, i)
			if blockLevel == 0 {
				return blockIndices
			}
		}
	}
	return nil
}

// poParamLineMatches ports the paramRegex:
// /\*\h*@param\h*(?:|REGEX_TYPES\h*)&?(?=\$\b)<identifier>\b/.
func poParamLineMatches(content, identifier string) bool {
	for from := 0; ; {
		p := strings.Index(content[from:], "@param")
		if p < 0 {
			return false
		}
		at := from + p
		// require "*" then \h* immediately before "@param"
		b := at
		for b > 0 && poIsHByte(content[b-1]) {
			b--
		}
		if b > 0 && content[b-1] == '*' {
			if poParamTailMatches(content, at+len("@param"), identifier) {
				return true
			}
		}
		from = at + len("@param")
	}
}

// poParamTailMatches matches \h*(?:|REGEX_TYPES\h*)&?(?=\$\b)<identifier>\b.
func poParamTailMatches(content string, after int, identifier string) bool {
	p := poSkipH(content, after)
	if poIdentMatches(content, p, identifier) {
		return true
	}
	if ts, ok := parsePHPTypes(content[p:], 0); ok {
		q := poSkipH(content, p+ts.end)
		if poIdentMatches(content, q, identifier) {
			return true
		}
	}
	return false
}

// poIdentMatches matches &?<identifier>\b at p (identifier already starts "$").
func poIdentMatches(content string, p int, identifier string) bool {
	if p < len(content) && content[p] == '&' {
		p++
	}
	if !strings.HasPrefix(content[p:], identifier) {
		return false
	}
	end := p + len(identifier)
	return end >= len(content) || !addParamIsWordByte(content[end])
}

func poSkipH(s string, p int) int {
	for p < len(s) && poIsHByte(s[p]) {
		p++
	}
	return p
}

func poIsHByte(b byte) bool { return b == ' ' || b == '\t' }

// docAnnContent is Annotation::getContent: the joined lines of the annotation.
func docAnnContent(lines []string, a docAnnotation) string {
	return strings.Join(lines[a.start:a.end+1], "")
}

// docAnnRemove ports Annotation::remove: each line is cleared, except a line
// that carries the docblock's "/**" or "*/" keeps that delimiter.
func docAnnRemove(lines []string, a docAnnotation) {
	for idx := a.start; idx <= a.end; idx++ {
		line := lines[idx]
		start := strings.Contains(line, "/**")
		end := strings.Contains(line, "*/")
		switch {
		case start && end:
			lines[idx] = ""
		case start:
			lines[idx] = poKeepStart(line)
		case end:
			lines[idx] = poKeepEnd(line)
		default:
			lines[idx] = ""
		}
	}
}

// poKeepStart ports Preg::replace('#(\s*/\*\*).*#', '$1', $line).
func poKeepStart(line string) string {
	idx := strings.Index(line, "/**")
	if idx < 0 {
		return line
	}
	return line[:idx+3] + poTrailingNewline(line)
}

// poKeepEnd ports Preg::replace('#(\s*)\S.*(\*/.*)#', '$1$2', $line).
func poKeepEnd(line string) string {
	g := strings.LastIndex(line, "*/")
	if g < 0 {
		return line
	}
	f := 0
	for f < len(line) && poIsSByte(line[f]) {
		f++
	}
	if f >= g {
		return line
	}
	return line[:f] + line[g:]
}

func poTrailingNewline(line string) string {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(line, "\n"):
		return "\n"
	case strings.HasSuffix(line, "\r"):
		return "\r"
	}
	return ""
}

func poIsSByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}
