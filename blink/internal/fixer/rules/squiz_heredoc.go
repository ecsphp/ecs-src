package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/HeredocSniff.php
//
// SquizHeredoc reports every heredoc and nowdoc.
type SquizHeredoc struct{}

func (SquizHeredoc) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\HeredocSniff`
}

func (SquizHeredoc) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/HeredocSniff.php"
}

func (SquizHeredoc) Fix(*tokens.Stream) bool { return false }

func (h SquizHeredoc) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	var encapsed []bool
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || (t.Part != token.Whole && t.Part != token.Head) {
			continue
		}
		opener := strings.TrimLeft(t.Value, "bB")
		if !strings.HasPrefix(opener, "<<<") {
			continue
		}
		if encapsed == nil {
			encapsed = encapsedTokens(s)
			lines = phpcsTokenLines(s)
		}
		// T_START_HEREDOC / T_START_NOWDOC: the opener through its line break
		start, _, _ := strings.Cut(t.Value, "\n")
		if encapsed[i] || !squizHeredocTerminated(s, i, start) {
			continue
		}
		kind, code := "heredoc", "HeredocNotAllowed"
		if strings.Contains(start, "'") {
			kind, code = "nowdoc", "NowdocNotAllowed"
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    lines[i],
			Message: fmt.Sprintf("Use of %s syntax (%s) is not allowed; use standard strings or inline HTML instead", kind, phpTrim(start)),
			Checker: h.Name() + "." + code,
		})
	}
	return errs
}

// squizHeredocTerminated reports whether the heredoc or nowdoc opening at i
// has a closing label: PHP_CodeSniffer turns one running to the end of the
// file, such as "<<< HEAD" in a git conflict marker, into a T_STRING.
func squizHeredocTerminated(s *tokens.Stream, i int, start string) bool {
	label := strings.TrimLeft(strings.TrimLeft(start, "bB")[3:], " \t")
	label = strings.TrimRight(strings.Trim(strings.TrimRight(label, "\r"), `"'`), " \t")
	if s.At(i).Part == token.Whole {
		return literalCloses(s.At(i).Value, label)
	}
	for j := i + 1; j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.String && t.Part == token.Tail && literalCloses(t.Value, label) {
			return true
		}
	}
	return false
}
