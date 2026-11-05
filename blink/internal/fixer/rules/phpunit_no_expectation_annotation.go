package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitNoExpectationAnnotationFixer.php
//
// PhpUnitNoExpectationAnnotation rewrites the `@expectedException`,
// `@expectedExceptionMessage`, `@expectedExceptionCode` and
// `@expectedExceptionMessageRegExp` doc annotations of a PHPUnit test method
// into a `$this->setExpectedException*(...)` call at the top of its body. With
// the default config the class-const notation is used and the message-regexp
// annotation is honoured (newest target).
type PhpUnitNoExpectationAnnotation struct{}

var phpUnitNoExpTags = []string{
	"expectedException",
	"expectedExceptionCode",
	"expectedExceptionMessage",
	"expectedExceptionMessageRegExp",
}

func (PhpUnitNoExpectationAnnotation) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitNoExpectationAnnotationFixer`
}

func (PhpUnitNoExpectationAnnotation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitNoExpectationAnnotationFixer.php"
}

func (PhpUnitNoExpectationAnnotation) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if phpUnitNoExpClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func phpUnitNoExpClass(s *tokens.Stream, startIndex, endIndex int) bool {
	changed := false
	for i := endIndex - 1; i > startIndex; i-- {
		if !kwIs(s.At(i), "function") || ptaIsLambda(s, i) {
			continue
		}
		braceIdx := nextPunctOfKind(s, i, ";", "{")
		if !isPunctVal(s, braceIdx, "{") {
			continue
		}
		docIdx := phpUnitNoExpDocBlockIndex(s, i)
		if docIdx < 0 || s.At(docIdx).Kind != token.DocComment {
			continue
		}

		lines := splitDocLines(s.At(docIdx).Value)
		annotations := map[string]string{}
		for _, ann := range docAnnotations(lines) {
			if !phpUnitNoExpWanted(ann.name) {
				continue
			}
			annotations[ann.name] = phpUnitNoExpContent(lines[ann.start:ann.end+1], ann.name)
			for j := ann.start; j <= ann.end; j++ {
				lines[j] = ""
			}
		}
		if _, ok := annotations["expectedException"]; !ok {
			continue
		}

		originalIndent := detectIndent(s, docIdx)
		method := "setExpectedException"
		if _, ok := annotations["expectedExceptionMessageRegExp"]; ok {
			method = "setExpectedExceptionRegExp"
		}
		params := phpUnitNoExpParamList(annotations)

		callCode := "<?php $this->" + method + "(" + strings.Join(params, ", ") + ");"
		body := lexer.Lex(callCode)[2:] // drop the open tag and the single space after it
		newToks := append([]token.Token{{Kind: token.Whitespace, Value: "\n" + originalIndent + "    "}}, body...)

		docContent := strings.Join(lines, "")
		if docContent == "" {
			docContent = "/** */"
		}
		s.SetValue(docIdx, docContent)
		s.InsertSliceAt(braceIdx+1, newToks)
		if wsIdx := braceIdx + len(newToks) + 1; wsIdx < s.Len() && s.At(wsIdx).Kind == token.Whitespace {
			s.SetValue(wsIdx, "\n"+s.At(wsIdx).Value)
		}

		changed = true
		i = docIdx
	}
	return changed
}

func phpUnitNoExpWanted(name string) bool {
	return slices.Contains(phpUnitNoExpTags, name)
}

// phpUnitNoExpDocBlockIndex walks back over whitespace, the visibility/final/
// abstract modifiers and plain comments to the method's doc comment, mirroring
// the loop in applyPhpUnitClassFix.
func phpUnitNoExpDocBlockIndex(s *tokens.Stream, index int) int {
	for {
		index = getPrevNonWhitespace(s, index)
		if index < 0 {
			return -1
		}
		t := s.At(index)
		if t.Kind == token.Comment {
			continue
		}
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private", "final", "abstract":
				continue
			}
		}
		return index
	}
}

var (
	phpUnitNoExpDocEndRe   = regexp.MustCompile(`\*+/$`)
	phpUnitNoExpCollapseRe = regexp.MustCompile(`\s*(?:\r\n|\r|\n)+\s*\*\s*`)
)

// phpUnitNoExpContent mirrors extractContentFromAnnotation: the text after the
// "@tag" and its following whitespace, with a trailing "*/" dropped, interior
// line breaks folded to a single space and trailing whitespace trimmed.
func phpUnitNoExpContent(annLines []string, tag string) string {
	content := strings.Join(annLines, "")
	at := strings.Index(content, "@"+tag)
	if at < 0 {
		return ""
	}
	rest := content[at+len("@"+tag):]
	j := 0
	for j < len(rest) && phpUnitNoExpIsSpace(rest[j]) {
		j++
	}
	if j == 0 { // the pattern requires at least one whitespace after the tag
		return ""
	}
	captured := rest[j:]
	if captured == "" {
		return ""
	}
	captured = phpUnitNoExpDocEndRe.ReplaceAllString(captured, "")
	if strings.ContainsAny(captured, "\r\n") {
		captured = phpUnitNoExpCollapseRe.ReplaceAllString(captured, " ")
	}
	return strings.TrimRight(captured, " \t\n\r\x00\x0b")
}

func phpUnitNoExpIsSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// phpUnitNoExpParamList mirrors annotationsToParamList with the default config
// (use_class_const = true).
func phpUnitNoExpParamList(ann map[string]string) []string {
	exceptionClass := strings.TrimLeft(ann["expectedException"], "\\")
	if i := strings.IndexByte(exceptionClass, '*'); i >= 0 {
		exceptionClass = exceptionClass[:i]
	}
	exceptionClass = strings.TrimSpace(exceptionClass)

	params := []string{"\\" + exceptionClass + "::class"}

	if msg, ok := ann["expectedExceptionMessage"]; ok {
		params = append(params, phpVarExportString(msg))
	} else if msg, ok := ann["expectedExceptionMessageRegExp"]; ok {
		params = append(params, phpVarExportString(msg))
	} else if _, ok := ann["expectedExceptionCode"]; ok {
		params = append(params, "null")
	}

	if code, ok := ann["expectedExceptionCode"]; ok {
		params = append(params, code)
	}
	return params
}

// phpVarExportString mirrors var_export for a string: single quotes with the
// backslash and the single quote escaped.
func phpVarExportString(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "'", "\\'")
	return "'" + v + "'"
}
