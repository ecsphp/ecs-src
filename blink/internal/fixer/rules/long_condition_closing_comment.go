package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/LongConditionClosingCommentSniff.php
//
// LongConditionClosingComment puts "//end if", "//end foreach" and the like
// after the closing brace of a long control structure, and corrects a wrong
// comment there.
type LongConditionClosingComment struct {
	LineLimit     int
	CommentFormat string
}

func (LongConditionClosingComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\LongConditionClosingCommentSniff`
}

func (LongConditionClosingComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/LongConditionClosingCommentSniff.php"
}

func (f LongConditionClosingComment) WithConfig(config map[string]any) fixer.Fixer {
	f.LineLimit = configInt(config, "lineLimit", f.LineLimit)
	if v, ok := config["commentFormat"].(string); ok {
		f.CommentFormat = v
	}
	return f
}

func (f LongConditionClosingComment) Fix(s *tokens.Stream) bool {
	return phpcsFixLoop(s, f.pass)
}

func (f LongConditionClosingComment) pass(s *tokens.Stream) bool {
	eol := phpcsEolChar(s)
	lines := phpcsLines(s, eol)
	var edits []phpcsDocEdit
	for i := range s.Len() {
		open, closeIdx := longConditionBody(s, i)
		if closeIdx < 0 {
			continue
		}
		kw := strings.ToLower(s.At(i).Value)
		end, ok := longConditionEnd(s, kw, closeIdx)
		if !ok {
			continue
		}
		at := end
		if kw == "match" {
			if n := nextNonWhitespaceIndex(s, end); n >= 0 && (isPunctVal(s, n, ";") || isPunctVal(s, n, ",")) {
				at = n
			}
		}
		expected := strings.Replace(f.CommentFormat, "%s", s.At(i).Value, 1)
		comment := at
		for comment < s.Len() && !isPhpcsComment(s.At(comment)) {
			comment++
		}
		if comment == s.Len() || lines[comment] != lines[end] {
			if lines[end]-lines[open] < f.LineLimit {
				continue
			}
			content := expected
			if n := nextNonWhitespaceIndex(s, at); n >= 0 && lines[n] == lines[at] {
				content += eol
			}
			edits = append(edits, phpcsDocEdit{at, func(s *tokens.Stream) {
				s.InsertAt(at+1, token.Token{Kind: token.Comment, Value: content})
			}})
			continue
		}
		if phpTrim(s.At(comment).Value) == expected {
			continue
		}
		keepsBreak := isLineComment(s.At(comment)) && comment+1 < s.Len() &&
			s.At(comment+1).Kind == token.Whitespace && strings.ContainsAny(s.At(comment + 1).Value[:1], "\r\n")
		edits = append(edits, phpcsDocEdit{comment, func(s *tokens.Stream) {
			if keepsBreak {
				s.SetValue(comment, expected)
				return
			}
			s.SetValue(comment, expected+eol)
		}})
	}
	return applyPhpcsEdits(s, edits)
}

// longConditionBody returns the braces of the switch, if, for, foreach, while,
// try, case or match whose keyword is at i, closeIdx -1 for anything else.
func longConditionBody(s *tokens.Stream, i int) (open, closeIdx int) {
	t := s.At(i)
	if t.Kind != token.Keyword {
		return -1, -1
	}
	switch strings.ToLower(t.Value) {
	case "if":
		if p := prevNonWhitespaceIdx(s, i); p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "else") {
			return -1, -1
		}
		return scopeBraces(s, i, true)
	case "switch", "for", "foreach", "while", "match":
		return scopeBraces(s, i, true)
	case "try":
		return scopeBraces(s, i, false)
	case "case":
		return caseBraces(s, i)
	}
	return -1, -1
}

// scopeBraces returns the "{...}" body right after the keyword at i, past its
// parenthesized condition when withParens.
func scopeBraces(s *tokens.Stream, i int, withParens bool) (open, closeIdx int) {
	o := phpcsNextNonEmpty(s, i+1)
	if withParens {
		if !isPunctVal(s, o, "(") {
			return -1, -1
		}
		c := s.MatchForward(o)
		if c < 0 {
			return -1, -1
		}
		o = phpcsNextNonEmpty(s, c+1)
	}
	if !isPunctVal(s, o, "{") {
		return -1, -1
	}
	if c := s.MatchForward(o); c >= 0 {
		return o, c
	}
	return -1, -1
}

// caseBraces returns the braces right after a case's ":" or ";", which
// PHP_CodeSniffer makes the case's own scope.
func caseBraces(s *tokens.Stream, i int) (open, closeIdx int) {
	depth := 0
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.CloseTag {
			return -1, -1
		}
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth < 0 {
				return -1, -1
			}
		case ":", ";":
			if depth > 0 {
				continue
			}
			o := phpcsNextNonEmpty(s, j+1)
			if !isPunctVal(s, o, "{") {
				return -1, -1
			}
			if c := s.MatchForward(o); c >= 0 {
				return o, c
			}
			return -1, -1
		}
	}
	return -1, -1
}

// longConditionEnd follows an if through its elseif and else blocks, and a try
// through its catch and finally blocks, to the last closing brace; ok is false
// when an elseif has no braces.
func longConditionEnd(s *tokens.Stream, kw string, closeIdx int) (end int, ok bool) {
	end = closeIdx
	switch kw {
	case "if":
		for {
			n := nextNonWhitespaceIndex(s, end)
			if n < 0 || s.At(n).Kind != token.Keyword {
				return end, true
			}
			switch strings.ToLower(s.At(n).Value) {
			case "elseif":
				_, c := scopeBraces(s, n, true)
				if c < 0 {
					return end, false
				}
				end = c
			case "else":
				_, c := scopeBraces(s, n, false)
				if c < 0 {
					m := nextNonWhitespaceIndex(s, n)
					if m < 0 || s.At(m).Kind != token.Keyword || !strings.EqualFold(s.At(m).Value, "if") {
						return end, true
					}
					if _, c = scopeBraces(s, m, true); c < 0 {
						return end, true
					}
				}
				end = c
			default:
				return end, true
			}
		}
	case "try":
		for {
			n := nextNonWhitespaceIndex(s, end)
			if n < 0 || s.At(n).Kind != token.Keyword {
				return end, true
			}
			var c int
			switch strings.ToLower(s.At(n).Value) {
			case "catch":
				_, c = scopeBraces(s, n, true)
			case "finally":
				_, c = scopeBraces(s, n, false)
			default:
				return end, true
			}
			if c < 0 {
				return end, true
			}
			end = c
		}
	}
	return end, true
}
