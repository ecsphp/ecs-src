package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/GotoTargetSpacingSniff.php
//
// GotoTargetSpacing removes whitespace between a goto label and its colon:
// "end :" -> "end:". A comment in between is left alone.
type GotoTargetSpacing struct{}

func (GotoTargetSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\GotoTargetSpacingSniff`
}

func (GotoTargetSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/GotoTargetSpacingSniff.php"
}

func (GotoTargetSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		colon, ok := phpcsGotoLabelColon(s, i)
		if !ok || colon == i+1 {
			continue
		}
		hasComment := false
		for j := i + 1; j < colon; j++ {
			if s.At(j).Kind != token.Whitespace {
				hasComment = true
			}
		}
		if hasComment {
			continue
		}
		for j := colon - 1; j > i; j-- {
			s.RemoveAt(j)
		}
		changed = true
	}
	return changed
}

// phpcsGotoLabelColon reports whether the token at i is a goto label as
// PHP_CodeSniffer's tokenizer marks one, returning the index of its colon.
func phpcsGotoLabelColon(s *tokens.Stream, i int) (int, bool) {
	t := s.At(i)
	if t.Kind != token.Ident {
		return 0, false
	}
	if i > 0 && s.At(i-1).Kind == token.Punct && s.At(i-1).Value == "::" {
		return 0, false
	}
	// a named argument ("foo(bar : 1)") is T_PARAM_NAME to PHP_CodeSniffer
	if p := prevMeaningfulIndex(s, i); p >= 0 && s.At(p).Kind == token.Punct && (s.At(p).Value == "(" || s.At(p).Value == ",") {
		return 0, false
	}
	colon := i + 1
	for colon < s.Len() && isPhpcsEmpty(s.At(colon)) {
		colon++
	}
	if colon >= s.Len() || s.At(colon).Kind != token.Punct || s.At(colon).Value != ":" {
		return 0, false
	}
	for x := i - 1; x >= 0; x-- {
		p := s.At(x)
		switch {
		case p.Kind == token.OpenTag && !strings.HasPrefix(p.Value, "<?="):
			return colon, true
		case p.Kind == token.Punct && (p.Value == ";" || p.Value == "{"):
			return colon, true
		case p.Kind == token.Punct && (p.Value == "::" || p.Value == "?"):
			return 0, false
		case p.Kind == token.Keyword && (strings.EqualFold(p.Value, "case") || strings.EqualFold(p.Value, "enum")):
			return 0, false
		}
	}
	return 0, false
}
