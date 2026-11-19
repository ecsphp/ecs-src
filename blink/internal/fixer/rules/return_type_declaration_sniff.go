package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Functions/ReturnTypeDeclarationSniff.php
//
// ReturnTypeDeclarationSniff formats a return type as "): type": no space
// before the colon, one space after it. A comment in either gap is left alone.
type ReturnTypeDeclarationSniff struct{}

func (ReturnTypeDeclarationSniff) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Functions\ReturnTypeDeclarationSniff`
}

func (ReturnTypeDeclarationSniff) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Functions/ReturnTypeDeclarationSniff.php"
}

func (ReturnTypeDeclarationSniff) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		colon, ok := phpcsReturnTypeColon(s, i)
		if !ok {
			continue
		}
		typ := phpcsNextNonEmpty(s, colon+1)
		if typ >= s.Len() {
			continue
		}
		// one space between the colon and the type, unless a comment sits there
		if (typ != colon+2 || s.At(colon+1).Value != " ") && !hasNonWhitespace(s, colon+1, typ) {
			for j := typ - 1; j > colon; j-- {
				s.RemoveAt(j)
			}
			s.InsertAt(colon+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
		// no space before the colon, unless a comment sits there
		if prev := colon - 1; s.At(prev).Kind == token.Whitespace {
			p := prev
			for p >= 0 && s.At(p).Kind == token.Whitespace {
				p--
			}
			if p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == ")" {
				for j := colon - 1; j > p; j-- {
					s.RemoveAt(j)
				}
				changed = true
			}
		}
	}
	return changed
}

// phpcsReturnTypeColon finds the return type colon of the function, closure or
// arrow function whose keyword is at i.
func phpcsReturnTypeColon(s *tokens.Stream, i int) (int, bool) {
	t := s.At(i)
	if t.Kind != token.Keyword || (!strings.EqualFold(t.Value, "function") && !strings.EqualFold(t.Value, "fn")) {
		return 0, false
	}
	j := phpcsNextNonEmpty(s, i+1)
	if j < s.Len() && s.At(j).Value == "&" {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j < s.Len() && s.At(j).Kind != token.Punct {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
		return 0, false
	}
	closeIdx := s.MatchForward(j)
	if closeIdx < 0 {
		return 0, false
	}
	next := phpcsNextNonEmpty(s, closeIdx+1)
	// closure "use (...)" comes before the return type
	if next < s.Len() && s.At(next).Kind == token.Keyword && strings.EqualFold(s.At(next).Value, "use") {
		open := phpcsNextNonEmpty(s, next+1)
		if open >= s.Len() || s.At(open).Value != "(" {
			return 0, false
		}
		if closeIdx = s.MatchForward(open); closeIdx < 0 {
			return 0, false
		}
		next = phpcsNextNonEmpty(s, closeIdx+1)
	}
	if next >= s.Len() || s.At(next).Kind != token.Punct || s.At(next).Value != ":" {
		return 0, false
	}
	return next, true
}

func hasNonWhitespace(s *tokens.Stream, from, to int) bool {
	for j := from; j < to; j++ {
		if s.At(j).Kind != token.Whitespace {
			return true
		}
	}
	return false
}
