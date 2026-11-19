package rules

import (
	"path/filepath"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LowercasedFilenameSniff.php
//
// LowercasedFilename reports a file name with uppercase letters, on the first
// open tag. It fixes nothing.
type LowercasedFilename struct{}

func (LowercasedFilename) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\LowercasedFilenameSniff`
}

func (LowercasedFilename) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LowercasedFilenameSniff.php"
}

func (LowercasedFilename) Fix(*tokens.Stream) bool { return false }

func (l LowercasedFilename) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	if s.Path() == "" {
		return nil
	}
	name := filepath.Base(s.Path())
	lower := lowercasedFilenameASCII(name)
	if name == lower {
		return nil
	}
	for _, t := range newSlvFile(lexer.Lex(s.Render())).toks {
		if t.kind == token.OpenTag {
			return []fixer.CodingStandardError{{
				Line:    t.line,
				Message: `Filename "` + name + `" doesn't match the expected filename "` + lower + `"`,
				Checker: l.Name() + ".NotFound",
			}}
		}
	}
	return nil
}

// lowercasedFilenameASCII is PHP 8's strtolower, which lowers ASCII letters only.
func lowercasedFilenameASCII(v string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, v)
}
