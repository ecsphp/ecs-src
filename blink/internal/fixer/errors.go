package fixer

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// CodingStandardError is an error a rule reports but cannot fix, as ECS's
// CodingStandardError.
type CodingStandardError struct {
	Line    int
	Message string
	Checker string // "<sniff class>.<code>"
}

// ErrorReporter is a rule that reports errors it cannot fix. The runner calls
// Errors on each file's original contents, as ECS runs sniffs before fixers.
type ErrorReporter interface {
	Fixer
	Errors(s *tokens.Stream) []CodingStandardError
}
