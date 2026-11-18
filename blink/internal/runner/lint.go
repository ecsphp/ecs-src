package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// LintChecker is the checker named on a file left unfixed because the fixed
// code did not parse.
const LintChecker = "blink"

var parseErrorRegex = regexp.MustCompile(`Parse error:\s+(.+?) in Standard input code on line (\d+)`)

// lintFixed reports false, with the error to show, when the fixed code does not
// parse but the original does, as PHP-CS-Fixer lints each file after fixing it.
// An original that does not parse either is not the fixers' fault.
func lintFixed(phpBinary, original string, res FileResult) (fixer.CodingStandardError, bool) {
	if phpBinary == "" {
		return fixer.CodingStandardError{}, true
	}
	message, line, ok := lint(phpBinary, res.After)
	if ok {
		return fixer.CodingStandardError{}, true
	}
	if _, _, originalOk := lint(phpBinary, original); !originalOk {
		return fixer.CodingStandardError{}, true
	}
	return fixer.CodingStandardError{
		Line: line,
		Message: fmt.Sprintf(
			"File left unchanged: the fixed code does not parse (%s). Applied checkers: %s",
			message, strings.Join(res.AppliedRules, ", "),
		),
		Checker: LintChecker,
	}, false
}

// lint runs "php -l" on code and returns the parse error and its line when it
// fails. A PHP binary that cannot run counts as a pass, so it never blocks fixing.
func lint(phpBinary, code string) (string, int, bool) {
	cmd := exec.Command(phpBinary, "-l")
	cmd.Stdin = strings.NewReader(code)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		return "", 0, true
	} else if _, isExit := err.(*exec.ExitError); !isExit {
		return "", 0, true
	}
	match := parseErrorRegex.FindStringSubmatch(out.String())
	if match == nil {
		return strings.TrimSpace(out.String()), 0, false
	}
	line, _ := strconv.Atoi(match[2])
	return match[1], line, false
}
