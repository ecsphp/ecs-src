package rules

import (
	"regexp"
	"strings"

	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationIndentationFixer.php
//
// DoctrineAnnotationIndentation indents Doctrine annotation continuation lines
// with four spaces per brace level (indent_mixed_lines is false by default).
type DoctrineAnnotationIndentation struct{}

func (DoctrineAnnotationIndentation) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationIndentationFixer`
}

func (DoctrineAnnotationIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationIndentationFixer.php"
}

// the trailing (\r?\n)? mirrors PCRE's `$`, which also matches before a final newline
var daIndentRe = regexp.MustCompile(`(\n( +\*)?) *(\r?\n)?$`)

func (DoctrineAnnotationIndentation) Fix(s *tokens.Stream) bool {
	return applyToDoctrineAnnotations(s, func(toks []daToken) bool {
		var positions []daSpan
		for index := 0; index < len(toks); index++ {
			if toks[index].typ != daTAt {
				continue
			}
			end := daGetAnnotationEnd(toks, index)
			if end < 0 {
				return false
			}
			positions = append(positions, daSpan{index, end})
			index = end
		}

		changed := false
		indentLevel := 0
		for index := range toks {
			if toks[index].typ != daTNone || !strings.Contains(toks[index].content, "\n") {
				continue
			}
			if !daIndentationCanBeFixed(toks, index, positions) {
				continue
			}
			opening, closing := daLineBracesCount(toks, index)
			delta := opening - closing
			mixedBraces := delta == 0 && opening > 0

			if indentLevel > 0 && (delta < 0 || mixedBraces) {
				indentLevel--
			}

			repl := "${1}" + strings.Repeat(" ", 4*indentLevel+1)
			if daIndentRe.MatchString(toks[index].content) {
				fixed := daIndentRe.ReplaceAllString(toks[index].content, repl)
				if fixed != toks[index].content {
					toks[index].content = fixed
					changed = true
				}
			}

			if delta > 0 || mixedBraces {
				indentLevel++
			}
		}
		return changed
	})
}

func daLineBracesCount(toks []daToken, index int) (int, int) {
	opening, closing := 0, 0
	for i := index + 1; i < len(toks); i++ {
		if toks[i].typ == daTNone && strings.Contains(toks[i].content, "\n") {
			break
		}
		switch toks[i].typ {
		case daTOpenParen, daTOpenCurly:
			opening++
		case daTCloseParen, daTCloseCurly:
			if opening > 0 {
				opening--
			} else {
				closing++
			}
		}
	}
	return opening, closing
}

type daSpan struct{ start, end int }

func daIndentationCanBeFixed(toks []daToken, newLineTokenIndex int, positions []daSpan) bool {
	for _, p := range positions {
		if newLineTokenIndex >= p.start && newLineTokenIndex <= p.end {
			return true
		}
	}
	if newLineTokenIndex+1 < len(toks) {
		if strings.Contains(toks[newLineTokenIndex+1].content, "\n") {
			return false
		}
		return toks[newLineTokenIndex+1].typ == daTAt
	}
	return false
}
