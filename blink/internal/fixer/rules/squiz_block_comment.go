package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

const phpTrimChars = " \t\n\r\x00\x0B"

// phpcsIsPlainComment reports whether i is a T_COMMENT: a comment that is
// neither a docblock piece nor a phpcs: annotation.
func phpcsIsPlainComment(f *slvFile, i int) bool {
	return f.toks[i].kind == token.Comment && !phpcsIsAnnotation(f, i)
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/BlockCommentSniff.php
//
// SquizBlockComment turns an inline "/** */" docblock into "/* */", a
// one-line "/* text */" ending its line into "// text", drops empty block
// comments and aligns the lines of a multi-line block comment to its star.
type SquizBlockComment struct{}

func (SquizBlockComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\BlockCommentSniff`
}

func (SquizBlockComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/BlockCommentSniff.php"
}

func (SquizBlockComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			switch {
			case f.toks[i].doc == slvDocOpen:
				squizBlockDocComment(f, i)
			case phpcsIsPlainComment(f, i):
				squizBlockComment(f, i)
			}
		}
	})
}

// squizBlockCommentSkip lists the tokens a docblock may document; one before
// any other is an inline docblock.
var squizBlockCommentSkip = phpcsSet("public", "private", "protected", "public(set)", "protected(set)", "private(set)",
	"class", "interface", "trait", "enum", "final", "static", "abstract", "const", "var", "readonly")

func squizBlockDocComment(f *slvFile, ptr int) {
	next := ptr
	for {
		next = f.nextEffective(next+1, len(f.toks))
		if next >= 0 && f.toks[next].kind == token.AttributeStart && f.toks[next].closer > next {
			next = f.toks[next].closer
			continue
		}
		break
	}
	if next >= 0 && f.toks[next].kind == token.Keyword {
		if squizBlockCommentSkip[strings.ToLower(f.toks[next].orig)] || phpcsIsNamedFunction(f, next) {
			return
		}
	}
	if prev := max(f.prevEffective(ptr-1), 0); f.toks[prev].kind == token.OpenTag && !phpcsIsEchoTag(f, prev) {
		return
	}
	f.replaceToken(ptr, "/*")
	end := ptr + 1
	for end < len(f.toks) && f.toks[end].doc != slvDocClose {
		end++
	}
	if end < len(f.toks) && f.toks[end].content != "*/" {
		f.replaceToken(end, "*/")
	}
}

func phpcsIsEchoTag(f *slvFile, i int) bool {
	return strings.HasPrefix(f.toks[i].orig, "<?=")
}

var squizBlockCommentOpener = regexp.MustCompile(`^(\s*/\*\*?)`)

func squizBlockComment(f *slvFile, ptr int) {
	if !strings.HasPrefix(f.toks[ptr].content, "/*") {
		return
	}
	commentLines := []int{ptr}
	lastLine := f.toks[ptr].line
	var commentString strings.Builder
	commentString.WriteString(f.toks[ptr].content)
	for next := ptr; ; {
		next = f.nextNot(func(i int) bool { return f.isWS(i) }, next+1, len(f.toks))
		if next < 0 || f.toks[next].kind != token.Comment || f.toks[next].line-1 != lastLine {
			break
		}
		lastLine = f.toks[next].line
		commentLines = append(commentLines, next)
		commentString.WriteString(f.toks[next].content)
		if strings.HasSuffix(f.toks[next].content, "*/") {
			break
		}
	}

	commentText := strings.Trim(strings.ReplaceAll(commentString.String(), f.eol, ""), "/* \t")
	if commentText == "" {
		f.beginChangeset()
		f.replaceToken(ptr, "")
		for i := ptr + 1; i <= commentLines[len(commentLines)-1]; i++ {
			f.replaceToken(i, "")
		}
		f.endChangeset()
		return
	}

	if len(commentLines) == 1 {
		next := max(f.nextEffective(ptr+1, len(f.toks)), 0)
		if f.toks[ptr].line != f.toks[next].line {
			f.replaceToken(ptr, "// "+commentText+f.eol)
		}
		return
	}

	if content := strings.Trim(f.toks[ptr].content, phpTrimChars); content != "/*" && content != "/**" {
		indent := ""
		if ptr > 0 && f.isWS(ptr-1) {
			indent = f.toks[ptr-1].orig
		}
		comment := f.toks[ptr].content
		if m := squizBlockCommentOpener.FindStringIndex(comment); m != nil {
			comment = comment[:m[1]] + f.eol + indent + comment[m[1]:]
		}
		f.replaceToken(ptr, comment)
		return
	}

	starColumn := f.toks[ptr].column
	hasStars := false
	padding := func(line, expected int) string {
		if strings.HasPrefix(f.toks[line].orig, "\t") {
			return strings.Repeat("\t", expected/phpcsTabStop) + strings.Repeat(" ", expected%phpcsTabStop)
		}
		return strings.Repeat(" ", expected)
	}
	expectedFor := func(text string) int {
		if text[0] == '*' {
			hasStars = true
			return starColumn
		}
		return starColumn + 3
	}

	first := commentLines[1]
	if strings.Trim(f.toks[first].content, phpTrimChars) == "" {
		f.replaceToken(first, "")
	} else {
		content := f.toks[first].content
		text := strings.TrimLeft(content, phpTrimChars)
		expected := expectedFor(text)
		if len(content)-len(text) != expected {
			f.replaceToken(first, padding(first, expected)+text)
		}
	}

	lastIndex := len(commentLines) - 1
	for n, line := range commentLines {
		if n == 0 || n == 1 || n == lastIndex {
			continue
		}
		content := f.toks[line].content
		if strings.Trim(content, phpTrimChars) == "" {
			continue
		}
		text := strings.TrimLeft(content, phpTrimChars)
		expected := expectedFor(text)
		if len(content)-len(text) < expected {
			f.replaceToken(line, padding(line, expected)+text)
		}
	}

	last := commentLines[lastIndex]
	content := f.toks[last].content
	text := strings.TrimLeft(content, phpTrimChars)
	if text != "*/" && text != "**/" {
		return
	}
	expected := starColumn - 1
	if hasStars {
		expected = starColumn
	}
	if len(content)-len(text) != expected {
		f.replaceToken(last, padding(last, expected)+text)
	}
}
