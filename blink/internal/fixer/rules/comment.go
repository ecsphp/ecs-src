package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/NoTrailingWhitespaceInCommentFixer.php
//
// NoTrailingWhitespaceInComment trims trailing spaces and tabs from each line of
// a comment or doc comment.
type NoTrailingWhitespaceInComment struct{}

func (NoTrailingWhitespaceInComment) Name() string {
	return `PhpCsFixer\Fixer\Comment\NoTrailingWhitespaceInCommentFixer`
}

func (NoTrailingWhitespaceInComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/NoTrailingWhitespaceInCommentFixer.php"
}

func (NoTrailingWhitespaceInComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		// "#[...]" attributes are captured as comment tokens but are real code in
		// PHP (and may hold heredoc bodies); this fixer must not touch them
		if strings.HasPrefix(t.Value, "#[") {
			continue
		}
		lines := strings.Split(t.Value, "\n")
		touched := false
		for k, line := range lines {
			trimmed := strings.TrimRight(line, " \t")
			if trimmed != line {
				lines[k] = trimmed
				touched = true
			}
		}
		if touched {
			s.SetValue(i, strings.Join(lines, "\n"))
			changed = true
		}
	}
	return changed
}
