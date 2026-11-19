package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/BacktickOperatorSniff.php
//
// BacktickOperator reports both backticks of every shell-exec literal.
type BacktickOperator struct{}

func (BacktickOperator) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\BacktickOperatorSniff`
}

func (BacktickOperator) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/BacktickOperatorSniff.php"
}

func (BacktickOperator) Fix(*tokens.Stream) bool { return false }

func (b BacktickOperator) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	report := func(line int) {
		errs = append(errs, fixer.CodingStandardError{
			Line:    line,
			Message: "Use of the backtick operator is forbidden",
			Checker: b.Name() + ".Found",
		})
	}
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || !strings.Contains(t.Value, "`") {
			continue
		}
		opens := (t.Part == token.Whole || t.Part == token.Head) && strings.HasPrefix(t.Value, "`")
		closes := strings.HasSuffix(t.Value, "`") && (t.Part == token.Tail || opens && len(t.Value) > 1)
		if !opens && !closes {
			continue
		}
		if lines == nil {
			lines = phpcsTokenLines(s)
		}
		if opens {
			report(lines[i])
		}
		if closes {
			report(lines[i] + phpcsLineBreaks(t.Value[:len(t.Value)-1]))
		}
	}
	return errs
}
