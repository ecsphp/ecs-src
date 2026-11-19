package rules

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/SyntaxSniff.php
//
// PHPSyntax reports the syntax error `php -l` finds in a file, like the sniff,
// which runs the php binary on the PATH. Without one it reports nothing.
type PHPSyntax struct{}

func (PHPSyntax) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\SyntaxSniff`
}

func (PHPSyntax) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/SyntaxSniff.php"
}

func (PHPSyntax) Fix(*tokens.Stream) bool { return false }

var phpSyntaxError = regexp.MustCompile(`(?m)^.*error:(.*) in .* on line ([0-9]+)`)

func (p PHPSyntax) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	// the sniff listens for open tags only
	hasOpenTag := false
	for i := range s.Len() {
		if s.At(i).Kind == token.OpenTag {
			hasOpenTag = true
			break
		}
	}
	if !hasOpenTag {
		return nil
	}
	php, err := exec.LookPath("php")
	if err != nil {
		return nil
	}
	cmd := exec.Command(php, "-l", "-d", "display_errors=1", "-d", "error_prepend_string=")
	cmd.Stdin = strings.NewReader(s.Render())
	out, _ := cmd.CombinedOutput()
	m := phpSyntaxError.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return nil
	}
	line, _ := strconv.Atoi(m[2])
	return []fixer.CodingStandardError{{
		Line:    line,
		Message: "PHP syntax error: " + strings.TrimSpace(m[1]),
		Checker: p.Name() + ".PHPSyntax",
	}}
}
