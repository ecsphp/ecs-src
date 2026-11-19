package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LineEndingsSniff.php
//
// LineEndings rewrites every line ending to EolChar ("\n" by default) when the
// file's first line ending differs. Like the sniff, it trims all trailing CR/LF
// of each line and also terminates the last line.
type LineEndings struct {
	EolChar string
}

func (LineEndings) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\LineEndingsSniff`
}

func (LineEndings) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LineEndingsSniff.php"
}

func (f LineEndings) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["eolChar"].(string); ok {
		f.EolChar = strings.NewReplacer(`\r`, "\r", `\n`, "\n").Replace(v)
	}
	return f
}

func (f LineEndings) Fix(s *tokens.Stream) bool {
	want := f.EolChar
	if want == "" {
		want = "\n"
	}
	if !phpcsHasOpenTag(s, true) {
		return false
	}
	found := phpcsEolChar(s)
	text := s.Render()
	// a one-line file without a line ending reads as "\n" and is left alone
	if found == want || (found == "\n" && !strings.Contains(text, "\n")) {
		return false
	}
	lines := strings.Split(text, found)
	var b strings.Builder
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			break
		}
		b.WriteString(strings.TrimRight(line, "\r\n"))
		b.WriteString(want)
	}
	if b.String() == text {
		return false
	}
	s.ReplaceRange(0, s.Len()-1, lexer.Lex(b.String()))
	return true
}
