package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/IncrementDecrementSpacingSniff.php
//
// IncrementDecrementSpacing removes whitespace between ++/-- and its operand:
// "++ $i" -> "++$i", "$i ++" -> "$i++". A comment in between is left alone.
type IncrementDecrementSpacing struct{}

func (IncrementDecrementSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\IncrementDecrementSpacingSniff`
}

func (IncrementDecrementSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/IncrementDecrementSpacingSniff.php"
}

func (IncrementDecrementSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || (t.Value != "++" && t.Value != "--") {
			continue
		}
		next := i + 1
		for next < s.Len() && isPhpcsEmpty(s.At(next)) {
			next++
		}
		if next < s.Len() && idsIsNameStart(s, next) {
			if next == i+2 && s.At(i+1).Kind == token.Whitespace {
				s.RemoveAt(i + 1)
				changed = true
			}
			continue
		}
		prev := i - 1
		for prev >= 0 && isPhpcsEmpty(s.At(prev)) {
			prev--
		}
		if prev < 0 || prev != i-2 || s.At(i-1).Kind != token.Whitespace {
			continue
		}
		if p := s.At(prev); p.Kind == token.Variable || p.Kind == token.Ident || (p.Kind == token.Punct && p.Value == "]") {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}

// idsIsNameStart reports whether a variable or a (qualified) name starts at i.
func idsIsNameStart(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind == token.Variable || t.Kind == token.Ident {
		return true
	}
	return t.Kind == token.Punct && t.Value == `\` && i+1 < s.Len() && s.At(i+1).Kind == token.Ident
}
