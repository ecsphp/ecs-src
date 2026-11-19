package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/ClosingDeclarationCommentSniff.php
//
// ClosingDeclarationComment puts "//end class", "//end foo()" and the like
// right after the closing brace of a class, interface, trait, enum or named
// function. A matching comment further down is pulled up to the brace.
type ClosingDeclarationComment struct{}

func (ClosingDeclarationComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\ClosingDeclarationCommentSniff`
}

func (ClosingDeclarationComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/ClosingDeclarationCommentSniff.php"
}

func (ClosingDeclarationComment) Fix(s *tokens.Stream) bool {
	changed := false
	eol := phpcsEolChar(s)
	for i := 0; i < s.Len(); i++ {
		closeIdx, comment := closingDeclaration(s, i)
		if closeIdx < 0 {
			continue
		}
		if closingCommentFix(s, closeIdx, comment, eol) {
			changed = true
		}
	}
	return changed
}

// closingDeclaration returns the closing brace and expected comment of the
// declaration whose keyword is at i, closeIdx -1 when i starts none.
func closingDeclaration(s *tokens.Stream, i int) (closeIdx int, comment string) {
	t := s.At(i)
	if t.Kind != token.Keyword {
		return -1, ""
	}
	kw := strings.ToLower(t.Value)
	if kw == "function" {
		name := phpcsNextNonEmpty(s, i+1)
		if name < s.Len() && s.At(name).Value == "&" {
			name = phpcsNextNonEmpty(s, name+1)
		}
		if name >= s.Len() || s.At(name).Kind != token.Ident {
			return -1, ""
		}
		_, c, ok := phpcsFunctionBody(s, i)
		if !ok || inInterfaceBody(s, i) {
			return -1, ""
		}
		return c, "//end " + s.At(name).Value + "()"
	}
	if !cdIsClassy(s, i) || (kw == "class" && anonClassNewIndex(s, i) >= 0) {
		return -1, ""
	}
	open := phpcsOOScopeOpener(s, i)
	if open < 0 {
		return -1, ""
	}
	c := s.MatchForward(open)
	if c < 0 {
		return -1, ""
	}
	return c, "//end " + kw
}

// inInterfaceBody reports whether i lies inside an interface body, at any depth.
func inInterfaceBody(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			for k := j - 1; k >= 0; k-- {
				if phpcsOOScopeOpener(s, k) == j {
					if strings.EqualFold(s.At(k).Value, "interface") {
						return true
					}
					break
				}
				if s.At(k).Value == "{" || s.At(k).Value == "}" || s.At(k).Value == ";" {
					break
				}
			}
		}
	}
	return false
}

// closingCommentFix makes the token right after the brace at c the comment.
func closingCommentFix(s *tokens.Stream, c int, comment, eol string) bool {
	if n := c + 1; n < s.Len() && isPhpcsComment(s.At(n)) {
		if phpRtrim(s.At(n).Value) == comment {
			return false
		}
		setClosingComment(s, n, comment, eol)
		return true
	}
	next := c + 1
	for next < s.Len() && s.At(next).Kind == token.Whitespace {
		next++
	}
	if next < s.Len() && s.At(next).Kind == token.Comment && phpRtrim(s.At(next).Value) == comment {
		for k := next - 1; k > c; k-- {
			s.RemoveAt(k)
		}
		setClosingComment(s, c+1, comment, eol)
		return true
	}
	s.InsertAt(c+1, token.Token{Kind: token.Comment, Value: comment})
	swallowLineIntoComment(s, c+1, eol)
	return true
}

// swallowLineIntoComment mirrors the next pass after a comment is glued to a
// brace: PHP lexes the rest of the line into that "//" comment, and the sniff
// then replaces the whole comment with itself plus eol. Trailing whitespace
// alone is left as is, since the comment then still matches.
func swallowLineIntoComment(s *tokens.Stream, n int, eol string) {
	end, rest, cut := n+1, "", -1
	for ; end < s.Len(); end++ {
		t := s.At(end)
		if t.Kind == token.CloseTag {
			break
		}
		if k := strings.IndexAny(t.Value, "\r\n"); k >= 0 && t.Kind == token.Whitespace {
			rest += t.Value[:k]
			cut = k
			break
		}
		rest += t.Value
	}
	if phpRtrim(rest) == "" {
		return
	}
	if cut >= 0 {
		s.SetValue(end, s.At(end).Value[cut:])
	}
	for k := end - 1; k > n; k-- {
		s.RemoveAt(k)
	}
	if cut < 0 {
		s.InsertAt(n+1, token.Token{Kind: token.Whitespace, Value: eol})
	}
}

// setClosingComment replaces the comment at n the way PHP_CodeSniffer
// replaces it with comment + eol: its own newline (part of a line comment
// there) is kept, any other comment gains one.
func setClosingComment(s *tokens.Stream, n int, comment, eol string) {
	keepsNewline := isLineComment(s.At(n)) && n+1 < s.Len() && s.At(n+1).Kind == token.Whitespace &&
		strings.HasPrefix(s.At(n+1).Value, eol)
	s.SetValue(n, comment)
	if !keepsNewline {
		s.InsertAt(n+1, token.Token{Kind: token.Whitespace, Value: eol})
	}
}

// isPhpcsComment reports whether PHP_CodeSniffer tokenizes t as T_COMMENT.
func isPhpcsComment(t token.Token) bool {
	return t.Kind == token.Comment && !strings.HasPrefix(t.Value, "#[") && !isPhpcsAnnotationComment(t.Value)
}

// phpRtrim is PHP's rtrim() with its default character list.
func phpRtrim(v string) string {
	return strings.TrimRight(v, " \t\n\r\x00\x0B")
}
