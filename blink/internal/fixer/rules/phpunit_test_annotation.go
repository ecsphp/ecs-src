package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestAnnotationFixer.php
//
// PhpUnitTestAnnotation marks PHPUnit tests either with the "test" name prefix
// (style "prefix", the default) or with the @test annotation (style
// "annotation"). @depends references follow the rename.
type PhpUnitTestAnnotation struct {
	annotation bool
}

func (PhpUnitTestAnnotation) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitTestAnnotationFixer`
}

func (PhpUnitTestAnnotation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestAnnotationFixer.php"
}

func (f PhpUnitTestAnnotation) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["style"].(string); ok {
		f.annotation = v == "annotation"
	}
	return f
}

func (f PhpUnitTestAnnotation) Fix(s *tokens.Stream) bool {
	classes := puClassesLastFirst(s)
	if len(classes) == 0 {
		return false
	}
	before := s.Render()
	for _, cls := range classes {
		if f.annotation {
			ptaApplyTestAnnotation(s, cls[0], cls[1])
		} else {
			ptaApplyTestPrefix(s, cls[0], cls[1])
		}
	}
	return s.Render() != before
}

func ptaApplyTestAnnotation(s *tokens.Stream, start, end int) {
	for i := end - 1; i > start; i-- {
		if !ptaIsTestMethod(s, i) {
			continue
		}
		nameIdx := sigNext(s, i)
		if name := s.At(nameIdx).Value; ptaHasTestPrefix(name) && !ptaHasProperTestAnnotation(s, i) {
			s.Set(nameIdx, token.Token{Kind: token.Ident, Value: ptaRemoveTestPrefix(name), Pos: s.At(nameIdx).Pos})
		}
		doc := puAnnotationDocBlockIndex(s, i)
		if doc >= 0 && s.At(doc).Kind == token.DocComment {
			lines := ptaUpdateLines(splitDocLines(s.At(doc).Value), s, doc, true)
			lines = ptaAddTestAnnotation(lines, s, doc)
			s.SetValue(doc, strings.Join(lines, ""))
		} else {
			ptaCreateDocBlock(s, doc, "test")
		}
	}
}

func ptaApplyTestPrefix(s *tokens.Stream, start, end int) {
	for i := end - 1; i > start; i-- {
		if !ptaIsTestMethod(s, i) {
			continue
		}
		doc := puAnnotationDocBlockIndex(s, i)
		if doc < 0 || s.At(doc).Kind != token.DocComment {
			continue
		}
		lines := ptaUpdateLines(splitDocLines(s.At(doc).Value), s, doc, false)
		s.SetValue(doc, strings.Join(lines, ""))
		nameIdx := sigNext(s, i)
		name := s.At(nameIdx).Value
		if ptaHasTestPrefix(name) {
			continue
		}
		s.Set(nameIdx, token.Token{Kind: token.Ident, Value: ptaAddTestPrefix(name), Pos: s.At(nameIdx).Pos})
	}
}

// ptaIsTestMethod mirrors isTestMethod: a non-lambda function whose name starts
// with "test" or whose docblock mentions "@test" anywhere.
func ptaIsTestMethod(s *tokens.Stream, index int) bool {
	if !kwIs(s.At(index), "function") || ptaIsLambda(s, index) {
		return false
	}
	nameIdx := sigNext(s, index)
	if nameIdx < 0 {
		return false
	}
	if ptaHasTestPrefix(s.At(nameIdx).Value) {
		return true
	}
	doc := puAnnotationDocBlockIndex(s, index)
	return doc >= 0 && s.At(doc).Kind == token.DocComment && strings.Contains(s.At(doc).Value, "@test")
}

// ptaIsLambda mirrors TokensAnalyzer::isLambda: "function", an optional "&",
// then "(".
func ptaIsLambda(s *tokens.Stream, index int) bool {
	n := sigNext(s, index)
	if isPunctVal(s, n, "&") {
		n = sigNext(s, n)
	}
	return isPunctVal(s, n, "(")
}

func ptaHasTestPrefix(name string) bool {
	return strings.HasPrefix(name, "test")
}

// ptaHasProperTestAnnotation mirrors hasProperTestAnnotation: the token before
// the method (its docblock or not) matches /\*\s+@test\b/.
func ptaHasProperTestAnnotation(s *tokens.Stream, index int) bool {
	doc := puAnnotationDocBlockIndex(s, index)
	if doc < 0 {
		return false
	}
	v := s.At(doc).Value
	for k := strings.IndexByte(v, '*'); k >= 0; {
		j := k + 1
		for j < len(v) && phpAlignIsWS(v[j]) {
			j++
		}
		if j > k+1 && strings.HasPrefix(v[j:], "@test") && (j+5 == len(v) || !isIdentByte(v[j+5])) {
			return true
		}
		next := strings.IndexByte(v[k+1:], '*')
		if next < 0 {
			break
		}
		k += 1 + next
	}
	return false
}

// ptaRemoveTestPrefix mirrors removeTestPrefix: drop /^test(?=[A-Z_])_?/ and
// lowercase the first letter, keeping the name when nothing would remain.
func ptaRemoveTestPrefix(name string) string {
	remainder := name
	if len(name) > 4 && strings.HasPrefix(name, "test") && (name[4] == '_' || name[4] >= 'A' && name[4] <= 'Z') {
		remainder = strings.TrimPrefix(name[4:], "_")
	}
	if remainder == "" {
		return name
	}
	return ptaLcfirst(remainder)
}

func ptaAddTestPrefix(name string) string {
	return "test" + ptaUcfirst(name)
}

func ptaUcfirst(v string) string {
	if v != "" && v[0] >= 'a' && v[0] <= 'z' {
		return string(v[0]-'a'+'A') + v[1:]
	}
	return v
}

func ptaLcfirst(v string) string {
	if v != "" && v[0] >= 'A' && v[0] <= 'Z' {
		return string(v[0]-'A'+'a') + v[1:]
	}
	return v
}

// ptaUpdateLines mirrors updateLines.
func ptaUpdateLines(lines []string, s *tokens.Stream, doc int, needsAnnotation bool) []string {
	original := s.At(doc).Value
	for i, line := range lines {
		if needsAnnotation && strings.Contains(line, "/**") && strings.Contains(line, "*/") {
			if !ptaDocContainsTest(original) {
				return ptaUpdateLines(ptaSplitUpDocBlock(lines, s, doc), s, doc, needsAnnotation)
			}
		}
		if !needsAnnotation && strings.Contains(line, " @test") && !strings.Contains(line, "@testWith") && !strings.Contains(line, "@testdox") {
			line = strings.ReplaceAll(line, " @test", "")
			lines[i] = line
		}
		if !strings.Contains(line, "@depends") {
			continue
		}
		lines[i] = ptaUpdateDepends(line, needsAnnotation)
	}
	return lines
}

// ptaUpdateDepends mirrors updateDependsAnnotation: the name runs to the end of
// the line, newline included.
func ptaUpdateDepends(line string, annotation bool) string {
	index := ptaIndexFold(line, "@depends") + len("@depends")
	for index < len(line) && line[index] == ' ' {
		index++
	}
	name := line[index:]
	if annotation {
		if ptaHasTestPrefix(name) {
			name = ptaRemoveTestPrefix(name)
		}
	} else if !ptaHasTestPrefix(name) {
		name = ptaAddTestPrefix(name)
	}
	return line[:index] + name
}

// ptaIndexFold mirrors stripos for an ASCII needle, returning 0 when absent
// (PHP's (int) false).
func ptaIndexFold(v, needle string) int {
	lower := func(c byte) byte {
		if c >= 'A' && c <= 'Z' {
			return c + 'a' - 'A'
		}
		return c
	}
outer:
	for i := 0; i+len(needle) <= len(v); i++ {
		for j := 0; j < len(needle); j++ {
			if lower(v[i+j]) != lower(needle[j]) {
				continue outer
			}
		}
		return i
	}
	return 0
}

// ptaSplitUpDocBlock mirrors splitUpDocBlock.
func ptaSplitUpDocBlock(lines []string, s *tokens.Stream, doc int) []string {
	content := ptaSingleLineDocBlockEntry(lines[0])
	indent := detectIndent(s, puNextNonWhitespace(s, doc))
	return []string{"/**\n", indent + " * " + content + "\n", indent + " */"}
}

// ptaSingleLineDocBlockEntry mirrors getSingleLineDocBlockEntry.
func ptaSingleLineDocBlockEntry(line string) string {
	b := []byte(strings.Trim(strings.ReplaceAll(line, "*/", ""), " \t\n\r\x00\x0B"))
	at := func(k int) byte {
		if k < 0 || k >= len(b) {
			return 0
		}
		return b[k]
	}
	i := len(b)
	for {
		i--
		if i < 0 || at(i) == '*' || at(i-1) == '*' || at(i-2) == '/' {
			break
		}
	}
	if at(i) == ' ' {
		i++
	}
	if i < 0 {
		i = 0
	}
	return string(b[i:])
}

// ptaAddTestAnnotation mirrors addTestAnnotation.
func ptaAddTestAnnotation(lines []string, s *tokens.Stream, doc int) []string {
	if ptaDocContainsTest(s.At(doc).Value) {
		return lines
	}
	indent := detectIndent(s, doc)
	out := append([]string{}, lines[:len(lines)-1]...)
	out = append(out, indent+" *\n"+indent+" * @test\n")
	return append(out, lines[len(lines)-1])
}

// ptaDocContainsTest mirrors doesDocBlockContainTest.
func ptaDocContainsTest(content string) bool {
	lines := splitDocLines(content)
	for _, ann := range docAnnotations(lines) {
		if ptaTagName(lines[ann.start]) == "test" {
			return true
		}
	}
	return false
}

// ptaTagName mirrors Tag::getName: the first /@[a-zA-Z0-9_-]+(?=\s|$)/ of the
// line, or "other".
func ptaTagName(line string) string {
	for k := strings.IndexByte(line, '@'); k >= 0; {
		j := k + 1
		for j < len(line) && (isIdentByte(line[j]) || line[j] == '-') {
			j++
		}
		if j > k+1 && (j == len(line) || phpAlignIsWS(line[j])) {
			return line[k+1 : j]
		}
		next := strings.IndexByte(line[k+1:], '@')
		if next < 0 {
			break
		}
		k += 1 + next
	}
	return "other"
}

// ptaCreateDocBlock mirrors DocBlockAnnotationTrait::createDocBlock. PHP's
// open tag and line comments own their trailing newline, so a lone "\n" after
// them is no whitespace token for PHP.
func ptaCreateDocBlock(s *tokens.Stream, docBlockIndex int, annotation string) {
	if docBlockIndex < 0 {
		return
	}
	indent := detectIndent(s, puNextNonWhitespace(s, docBlockIndex))
	index := sigNext(s, docBlockIndex)
	s.InsertSliceAt(index, []token.Token{
		{Kind: token.DocComment, Value: "/**\n" + indent + " * @" + annotation + "\n" + indent + " */"},
		{Kind: token.Whitespace, Value: "\n" + indent},
	})
	prev := index - 1
	if prev >= 0 && s.At(prev).Kind == token.Whitespace {
		owner := -1
		if prev > 0 && s.At(prev).Value == "\n" {
			if t := s.At(prev - 1); t.Kind == token.OpenTag || t.Kind == token.Comment && !strings.HasPrefix(t.Value, "/*") {
				owner = prev - 1
			}
		}
		if owner < 0 {
			return
		}
		extra := "\n"
		if s.At(owner).Kind != token.OpenTag {
			extra += "\n"
		}
		s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: extra + detectIndent(s, owner)})
		return
	}
	extra := "\n"
	if prev >= 0 && s.At(prev).Kind != token.OpenTag {
		extra += "\n"
	}
	s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: extra + detectIndent(s, index)})
}

// puAnnotationDocBlockIndex mirrors DocBlockAnnotationTrait::getDocBlockIndex.
// Unlike puDocBlockIndex it does not skip "static", as upstream.
func puAnnotationDocBlockIndex(s *tokens.Stream, index int) int {
	for {
		index = getPrevNonWhitespace(s, index)
		if index < 0 {
			return -1
		}
		if s.At(index).Kind == token.AttributeEnd {
			index = s.AttributeStartOf(index)
			if index < 0 {
				return -1
			}
		}
		t := s.At(index)
		switch t.Kind {
		case token.Comment, token.AttributeStart:
			continue
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private", "final", "abstract", "readonly":
				continue
			}
		}
		return index
	}
}
