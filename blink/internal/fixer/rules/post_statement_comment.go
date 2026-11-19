package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// postStatementCommentOwners are the control structures whose multi-line
// conditions may carry trailing comments.
var postStatementCommentOwners = map[string]bool{
	"if": true, "elseif": true, "switch": true, "while": true, "for": true, "foreach": true, "match": true,
}

var postStatementAnnotation = regexp.MustCompile(`^//[ \t]*@[^\s]+`)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/PostStatementCommentSniff.php
//
// PostStatementComment moves a "//" comment that trails a statement onto the
// next line. Comments after "}", inside control structure conditions and
// annotations ("// @phpstan-ignore") stay put.
type PostStatementComment struct{}

func (PostStatementComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\PostStatementCommentSniff`
}

func (PostStatementComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/PostStatementCommentSniff.php"
}

func (PostStatementComment) Fix(s *tokens.Stream) bool {
	changed := false
	eol := phpcsEolChar(s)
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "//") || isPhpcsAnnotationComment(t.Value) {
			continue
		}
		last, sameLine := postStatementLastContent(s, i)
		if last < 0 || !sameLine || s.At(last).Value == "}" {
			continue
		}
		if v := s.At(last).Value; v == "," || v == ";" {
			before, _ := postStatementLastContent(s, last)
			if before < 0 || s.At(before).Value == "}" {
				continue
			}
		}
		if inControlCondition(s, i) || postStatementAnnotation.MatchString(t.Value) {
			continue
		}
		if i > 0 && s.At(i-1).Kind == token.Whitespace {
			s.SetValue(i-1, s.At(i-1).Value+eol)
		} else {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: eol})
			i++
		}
		changed = true
	}
	return changed
}

// postStatementLastContent is the last non-whitespace token before i and
// whether no line break separates it from i.
func postStatementLastContent(s *tokens.Stream, i int) (int, bool) {
	sameLine := true
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Whitespace {
			return j, sameLine
		}
		if strings.ContainsAny(t.Value, "\r\n") {
			sameLine = false
		}
	}
	return -1, false
}

// inControlCondition reports whether i sits inside the parentheses of an
// if/elseif/switch/while/for/foreach/match condition, at any depth.
func inControlCondition(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			depth++
		case "(":
			if depth > 0 {
				depth--
				continue
			}
			if p := prevSignificantIndex(s, j); p >= 0 && s.At(p).Kind == token.Keyword &&
				postStatementCommentOwners[strings.ToLower(s.At(p).Value)] {
				return true
			}
		}
	}
	return false
}
