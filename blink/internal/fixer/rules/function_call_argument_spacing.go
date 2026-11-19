package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/FunctionCallArgumentSpacingSniff.php
//
// FunctionCallArgumentSpacing removes the space before each comma of a call's
// arguments and puts one space after it: "foo($a ,$b)" -> "foo($a, $b)".
type FunctionCallArgumentSpacing struct{}

func (FunctionCallArgumentSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Functions\FunctionCallArgumentSpacingSniff`
}

func (FunctionCallArgumentSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/FunctionCallArgumentSpacingSniff.php"
}

func (FunctionCallArgumentSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if t := s.At(open); t.Kind != token.Punct || t.Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !argumentSpacingCall(s, open) {
			continue
		}
		var commas []int
		for i := open + 1; i < closeIdx; i++ {
			t := s.At(i)
			if t.Kind == token.AttributeStart {
				if e := s.AttributeEnd(i); e > i {
					i = e
				}
				continue
			}
			if t.Kind != token.Punct {
				continue
			}
			switch t.Value {
			case "(", "[", "{":
				if m := s.MatchForward(i); m > i {
					i = m
				}
			case ",":
				commas = append(commas, i)
			}
		}
		for _, comma := range slices.Backward(commas) {
			if fixArgumentComma(s, comma) {
				changed = true
			}
		}
	}
	return changed
}

// argumentSpacingCall reports whether the "(" at open opens the arguments of
// a call the sniff checks: after a name, isset/unset/self/static/parent, a
// variable, a ")" or a "}" closing a dynamic name.
func argumentSpacingCall(s *tokens.Stream, open int) bool {
	p := phpcsPrevNonEmpty(s, open-1)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Variable:
		return true
	case token.Ident, token.Keyword:
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "isset", "unset", "self", "static", "parent":
			default:
				return false
			}
		}
		k := p - 1
		for k >= 0 && (isPhpcsEmpty(s.At(k)) || s.At(k).Value == "&") {
			k--
		}
		return k < 0 || s.At(k).Kind != token.Keyword ||
			(!strings.EqualFold(s.At(k).Value, "function") && !strings.EqualFold(s.At(k).Value, "class"))
	case token.Punct:
		switch t.Value {
		case ")":
			return true
		case "}":
			o := s.MatchBackward(p)
			if o < 0 {
				return false
			}
			po := phpcsPrevNonEmpty(s, o-1)
			return po >= 0 && isPunctIn(s.At(po), "->", "?->", "::", "$")
		}
	}
	return false
}

func fixArgumentComma(s *tokens.Stream, comma int) bool {
	changed := false
	if phpcsWhitespaceAt(s, comma-1) {
		prev := phpcsPrevNonEmpty(s, comma-2)
		if prev >= 0 && !isHeredocEnd(s, prev) {
			if argumentSpanHasNewline(s, prev, comma) {
				s.RemoveAt(comma)
				s.InsertAt(prev+1, token.Token{Kind: token.Punct, Value: ","})
				// PHP_CodeSniffer changes a token once per pass: the space
				// after the moved comma waits for the next pass
				return true
			}
			if dropPhpcsWhitespace(s, comma-1) {
				comma--
			}
			changed = true
		}
	}
	if comma+1 >= s.Len() {
		return changed
	}
	if s.At(comma+1).Kind != token.Whitespace {
		if s.At(comma+1).Kind == token.Punct && s.At(comma+1).Value == ")" {
			return changed
		}
		s.InsertAt(comma+1, token.Token{Kind: token.Whitespace, Value: " "})
		return true
	}
	next := phpcsNextNonEmpty(s, comma+1)
	if next >= s.Len() || argumentSpanHasNewline(s, comma, next) {
		return changed
	}
	if phpcsTabWidth(s, comma+1) > 1 {
		s.SetValue(comma+1, " ")
		changed = true
	}
	return changed
}

// argumentSpanHasNewline reports whether a line break lies between the tokens
// at from and to.
func argumentSpanHasNewline(s *tokens.Stream, from, to int) bool {
	for i := from + 1; i < to; i++ {
		if strings.ContainsAny(s.At(i).Value, "\r\n") {
			return true
		}
	}
	return false
}

// isHeredocEnd reports whether the token at i ends a heredoc or nowdoc.
func isHeredocEnd(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.String || !t.ClosesString() {
		return false
	}
	for i >= 0 && !s.At(i).OpensString() {
		i--
	}
	if i < 0 {
		return false
	}
	_, v := token.SplitBinaryPrefix(s.At(i).Value)
	return strings.HasPrefix(v, "<<<")
}
