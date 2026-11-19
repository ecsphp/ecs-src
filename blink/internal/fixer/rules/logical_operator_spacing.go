package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/LogicalOperatorSpacingSniff.php
//
// LogicalOperatorSpacing puts one space on each side of "&&", "||", "and", "or"
// and "xor"; spacing that spans a line break is left alone.
type LogicalOperatorSpacing struct{}

func (LogicalOperatorSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\LogicalOperatorSpacingSniff`
}

func (LogicalOperatorSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/LogicalOperatorSpacingSniff.php"
}

func (LogicalOperatorSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if !isBooleanOperator(s.At(i)) {
			continue
		}
		// after
		if i+1 < s.Len() {
			if s.At(i+1).Kind != token.Whitespace {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			} else if v := s.At(i + 1).Value; !strings.ContainsAny(v, "\r\n") && phpcsTabWidth(s, i+1) != 1 {
				s.SetValue(i+1, " ")
				changed = true
			}
		}
		// before
		if i > 0 {
			if !phpcsWhitespaceAt(s, i-1) {
				if s.At(i-1).Kind == token.Whitespace {
					s.SetValue(i-1, s.At(i-1).Value+" ")
				} else {
					s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
					i++
				}
				changed = true
			} else if v := s.At(i - 1).Value; !strings.ContainsAny(v, "\r\n") && phpcsTabWidth(s, i-1) != 1 {
				s.SetValue(i-1, " ")
				changed = true
			}
		}
	}
	return changed
}

func isBooleanOperator(t token.Token) bool {
	switch t.Kind {
	case token.Punct:
		return t.Value == "&&" || t.Value == "||"
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
	}
	return false
}
