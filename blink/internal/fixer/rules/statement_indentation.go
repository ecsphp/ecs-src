package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/StatementIndentationFixer.php
//
// StatementIndentation reindents statement lines to four spaces per brace level.
// It only touches lines that begin a new statement at brace scope (after ";",
// "{" or "}" and not inside "(...)"/"[...]"), so continuation lines - multiline
// arguments, arrays and method chains - keep their own alignment. Heredoc bodies
// and comment interiors are single tokens and are never reindented.
type StatementIndentation struct{}

func (StatementIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\StatementIndentationFixer`
}

func (StatementIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/StatementIndentationFixer.php"
}

func (StatementIndentation) Fix(s *tokens.Stream) bool {
	changed := false
	brace, paren := 0, 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				brace++
			case "}":
				if brace > 0 {
					brace--
				}
			case "(", "[":
				paren++
			case ")", "]":
				if paren > 0 {
					paren--
				}
			}
		}
		if t.Kind != token.Whitespace || !hasNewline(t.Value) || paren != 0 {
			continue
		}
		prev, ok := prevSignificant(s, i)
		if !ok || (prev.Value != ";" && prev.Value != "{" && prev.Value != "}") {
			continue // continuation line - leave its alignment alone
		}
		if nextSignificantValue(s, i) == "" {
			continue
		}

		level := brace
		if nextSignificantValue(s, i) == "}" {
			level-- // a closing brace dedents to the outer level
		}
		if level < 0 {
			level = 0
		}
		target := strings.Repeat("    ", level)

		v := t.Value
		nl := strings.LastIndexByte(v, '\n')
		if v[nl+1:] != target {
			s.SetValue(i, v[:nl+1]+target)
			changed = true
		}
	}
	return changed
}
