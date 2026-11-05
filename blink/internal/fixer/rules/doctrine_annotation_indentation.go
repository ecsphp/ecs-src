package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationIndentationFixer.php
//
// DoctrineAnnotationIndentation indents Doctrine annotation continuation lines
// with four spaces per brace level. It reads ignored_tags (the tags left alone,
// shared with the other Doctrine annotation fixers) and indent_mixed_lines
// (false by default: true indents one level deeper a closing line that starts
// with content, as in `foo="bar")`).
type DoctrineAnnotationIndentation struct {
	indentMixedLines bool
	ignoredTags      map[string]bool // nil keeps the default list
}

func (DoctrineAnnotationIndentation) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationIndentationFixer`
}

func (DoctrineAnnotationIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationIndentationFixer.php"
}

// the trailing (\r?\n)? mirrors PCRE's `$`, which also matches before a final newline
var daIndentRe = regexp.MustCompile(`(\n( +\*)?) *(\r?\n)?$`)

func (f DoctrineAnnotationIndentation) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["indent_mixed_lines"].(bool); ok {
		f.indentMixedLines = v
	}
	if tags, ok := doctrineIgnoredTagsOption(config); ok {
		f.ignoredTags = tags
	}
	return f
}

func (f DoctrineAnnotationIndentation) Fix(s *tokens.Stream) bool {
	ignored := f.ignoredTags
	if ignored == nil {
		ignored = doctrineIgnoredTags
	}
	return applyToDoctrineAnnotations(s, ignored, func(toks []daToken) []daToken {
		var positions []daSpan
		for index := 0; index < len(toks); index++ {
			if toks[index].typ != daTAt {
				continue
			}
			end := daGetAnnotationEnd(toks, index)
			if end < 0 {
				return toks
			}
			positions = append(positions, daSpan{index, end})
			index = end
		}

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
			extraIndentLevel := 0

			if indentLevel > 0 && (delta < 0 || mixedBraces) {
				indentLevel--
				if f.indentMixedLines && daIsClosingLineWithMeaningfulContent(toks, index) {
					extraIndentLevel = 1
				}
			}

			repl := "${1}" + strings.Repeat(" ", 4*(indentLevel+extraIndentLevel)+1)
			if daIndentRe.MatchString(toks[index].content) {
				toks[index].content = daIndentRe.ReplaceAllString(toks[index].content, repl)
			}

			if delta > 0 || mixedBraces {
				indentLevel++
			}
		}
		return toks
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

// daIsClosingLineWithMeaningfulContent reports whether the line starting at the
// newline token index begins with something other than a closing brace.
func daIsClosingLineWithMeaningfulContent(toks []daToken, index int) bool {
	for i := index + 1; i < len(toks); i++ {
		if toks[i].typ == daTNone {
			if strings.Contains(toks[i].content, "\n") {
				return false
			}
			continue
		}
		return toks[i].typ != daTCloseParen && toks[i].typ != daTCloseCurly
	}
	return false
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
