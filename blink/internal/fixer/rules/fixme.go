package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/FixmeSniff.php
//
// Fixme reports every comment that mentions a FIXME task. It fixes nothing.
type Fixme struct{}

func (Fixme) Name() string { return `PHP_CodeSniffer\Standards\Generic\Sniffs\Commenting\FixmeSniff` }

func (Fixme) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/FixmeSniff.php"
}

func (Fixme) Fix(*tokens.Stream) bool { return false }

var fixmeTask = regexp.MustCompile(`(?i)(?:\A|[^\pL]+)fixme(?:([^\pL]+(.*))|\n?\z)`)

func (f Fixme) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return fixmeTaskErrors(s, fixmeTask, "FIXME", f.Name())
}

// fixmeTaskErrors reports the comment tokens, as PHP_CodeSniffer splits them,
// that re matches, for the Fixme and Todo sniffs.
func fixmeTaskErrors(s *tokens.Stream, re *regexp.Regexp, word, sniff string) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	for _, t := range newSlvFile(lexer.Lex(s.Render())).toks {
		if t.kind != token.Comment && t.kind != token.DocComment || t.kind == token.Comment && isPhpcsAnnotationComment(t.orig) {
			continue
		}
		m := re.FindStringSubmatch(t.content)
		if m == nil {
			continue
		}
		task := strings.Trim(strings.Trim(m[1], " \t\n\r\x00\x0B"), "-:[](). ")
		code, msg := "CommentFound", "Comment refers to a "+word+" task"
		if task != "" {
			code, msg = "TaskFound", msg+` "`+task+`"`
		}
		errs = append(errs, fixer.CodingStandardError{Line: t.line, Message: msg, Checker: sniff + "." + code})
	}
	return errs
}
