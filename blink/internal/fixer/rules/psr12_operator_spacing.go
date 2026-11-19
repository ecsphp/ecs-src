package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Operators/OperatorSpacingSniff.php
//
// PSR12OperatorSpacing requires at least one space around binary, assignment,
// comparison, boolean, ternary and concatenation operators: "$a+$b" -> "$a + $b".
// Extra spaces are kept; a short ternary "?:" stays together, and an operator
// right after a line comment keeps its position.
type PSR12OperatorSpacing struct{}

func (PSR12OperatorSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Operators\OperatorSpacingSniff`
}

func (PSR12OperatorSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Operators/OperatorSpacingSniff.php"
}

func (PSR12OperatorSpacing) Fix(s *tokens.Stream) bool {
	ops := newPhpcsOperatorContext(s)
	var inserts []int
	for i := 0; i < s.Len(); i++ {
		if end, ok := ops.declare[i]; ok {
			if end < 0 {
				break
			}
			i = end
			continue
		}
		if !ops.isOperatorToken(s, i, true) || !ops.isOperator(s, i) {
			continue
		}
		v := s.At(i).Value
		checkBefore := i > 0
		if v == ":" && i > 0 && s.At(i-1).Value == "?" {
			checkBefore = false
		}
		// a line comment's newline belongs to the comment for PHPCS: an
		// operator starting the next line has no whitespace before it
		if i > 0 && !phpcsWhitespaceAt(s, i-1) && (isLineComment(s.At(i-1)) || (s.At(i-1).Kind == token.Whitespace && i > 1 && isLineComment(s.At(i-2)))) {
			checkBefore = false
		}
		checkAfter := i+1 < s.Len()
		if v == "?" && checkAfter && s.At(i+1).Value == ":" {
			checkAfter = false
		}
		if checkBefore && !phpcsWhitespaceAt(s, i-1) {
			inserts = append(inserts, i)
		}
		if checkAfter && s.At(i+1).Kind != token.Whitespace {
			inserts = append(inserts, i+1)
		}
	}
	for _, at := range slices.Backward(slices.Compact(inserts)) {
		s.InsertAt(at, token.Token{Kind: token.Whitespace, Value: " "})
	}
	return len(inserts) > 0
}
