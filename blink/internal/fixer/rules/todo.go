package rules

import (
	"regexp"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/TodoSniff.php
//
// Todo reports every comment that mentions a TODO task. Upstream warns; ECS
// escalates this sniff's warnings to errors. It fixes nothing.
type Todo struct{}

func (Todo) Name() string { return `PHP_CodeSniffer\Standards\Generic\Sniffs\Commenting\TodoSniff` }

func (Todo) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/TodoSniff.php"
}

func (Todo) Fix(*tokens.Stream) bool { return false }

var todoTask = regexp.MustCompile(`(?i)(?:\A|[^\pL]+)todo(?:([^\pL]+(.*))|\n?\z)`)

func (t Todo) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return fixmeTaskErrors(s, todoTask, "TODO", t.Name())
}
