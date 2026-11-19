package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/DocCommentAlignmentSniff.php
//
// DocCommentAlignment aligns the stars of a declaration or file docblock one
// column right of its "/**" and puts one space after each star: none or
// several before a tag become one, a star glued to text gets one.
type DocCommentAlignment struct{}

func (DocCommentAlignment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\DocCommentAlignmentSniff`
}

func (DocCommentAlignment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/DocCommentAlignmentSniff.php"
}

func (DocCommentAlignment) Fix(s *tokens.Stream) bool {
	changed := false
	eol := phpcsEolChar(s)
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment || !docAlignmentTarget(s, i) {
			continue
		}
		if fixed := alignDocComment(s.At(i).Value, eol, phpcsColumn(s, i)+1); fixed != s.At(i).Value {
			s.SetValue(i, fixed)
			changed = true
		}
	}
	return changed
}

// docAlignmentTarget reports whether the docblock at i documents a declaration
// the sniff checks, or is a file comment right after "<?php".
func docAlignmentTarget(s *tokens.Stream, i int) bool {
	if n := phpcsNextNonEmpty(s, i+1); n < s.Len() && s.At(n).Kind == token.Keyword {
		kw := strings.ToLower(s.At(n).Value)
		switch {
		case strings.HasPrefix(kw, "public"), strings.HasPrefix(kw, "protected"), strings.HasPrefix(kw, "private"):
			return true
		case kw == "case":
			if enumCase(s, n) {
				return true
			}
		case slices.Contains([]string{"class", "interface", "enum", "function", "static", "abstract", "final", "var", "readonly"}, kw):
			return true
		}
	}
	p := i - 1
	for p >= 0 && isPhpcsEmpty(s.At(p)) {
		p--
	}
	return p >= 0 && s.At(p).Kind == token.OpenTag && strings.EqualFold(s.At(p).Value, "<?php")
}

// enumCase reports whether the "case" at i declares an enum case.
func enumCase(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			for k := j - 1; k >= 0 && s.At(k).Value != ";" && s.At(k).Value != "}" && s.At(k).Value != "{"; k-- {
				if phpcsOOScopeOpener(s, k) == j {
					return strings.EqualFold(s.At(k).Value, "enum")
				}
			}
			return false
		}
	}
	return false
}

// alignDocComment fixes the star columns and star spacing of doc, following
// PHP_CodeSniffer's docblock tokenizer: the open tag is the leading "/*" run,
// the close tag the trailing one, and each line after the first is indent,
// optional "*", padding, then a tag or text.
func alignDocComment(doc, eol string, required int) string {
	open := len(doc) - len(strings.TrimLeft(doc, "/*"))
	if open == len(doc) {
		return doc
	}
	rest := doc[open:]
	body := strings.TrimRight(rest, "/*")
	closeTag := rest[len(body):]
	lines := strings.Split(body, eol)
	padding := strings.Repeat(" ", required-1)
	for n := 1; n < len(lines); n++ {
		line := lines[n]
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		content := line[len(indent):]
		if content == "" {
			// the close tag, alone on the last line
			if n == len(lines)-1 && strings.TrimSpace(closeTag) != "" && 1+expandedWidth(indent, 0) != required {
				lines[n] = padding
			}
			continue
		}
		if content[0] != '*' {
			continue
		}
		if 1+expandedWidth(indent, 0) != required {
			indent = padding
		}
		after := content[1:]
		gap := after[:len(after)-len(strings.TrimLeft(after, " \t"))]
		text := after[len(gap):]
		switch {
		case after == "":
			// an empty line
		case gap == "":
			after = " " + after
		case docTagStart(text) && gap != " ":
			after = " " + text
		}
		lines[n] = indent + "*" + after
	}
	return doc[:open] + strings.Join(lines, eol) + closeTag
}

// docTagStart reports whether text starts with a docblock tag as PHP_CodeSniffer
// tokenizes one: "@" plus non-space characters, other than "@phpcs:".
func docTagStart(text string) bool {
	return len(text) > 1 && text[0] == '@' && !strings.ContainsAny(text[1:2], " \t\r\n") &&
		!strings.HasPrefix(strings.ToLower(text), "@phpcs:")
}

// phpcsColumn is the 1-based column PHP_CodeSniffer gives the token at i, tabs
// expanded to ECS's tab width of 4.
func phpcsColumn(s *tokens.Stream, i int) int {
	var parts []string
	for j := i - 1; j >= 0; j-- {
		v := s.At(j).Value
		if nl := strings.LastIndexAny(v, "\r\n"); nl >= 0 {
			parts = append(parts, v[nl+1:])
			break
		}
		parts = append(parts, v)
	}
	slices.Reverse(parts)
	return 1 + expandedWidth(strings.Join(parts, ""), 0)
}
