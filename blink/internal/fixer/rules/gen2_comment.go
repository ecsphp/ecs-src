package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/AlignMultilineCommentFixer.php
//
// AlignMultilineComment re-indents the continuation "*" lines of a multi-line
// block comment or docblock so each "*" sits one space to the right of the
// opening "/*". Only the leading whitespace of lines whose trimmed content
// starts with "*" is rewritten; free-form lines are left untouched.
type AlignMultilineComment struct{}

func (AlignMultilineComment) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\AlignMultilineCommentFixer`
}

func (AlignMultilineComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/AlignMultilineCommentFixer.php"
}

func (AlignMultilineComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		v := t.Value
		if !strings.Contains(v, "\n") {
			continue
		}
		indent, ok := commentOpeningIndent(s, i)
		if !ok {
			continue
		}
		nv := realignCommentLines(v, indent)
		if nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

// commentOpeningIndent returns the indentation of the comment's opening line,
// taken from the whitespace token right before it. ok is false when the comment
// is not preceded by a line break (an inline comment), which must be left alone.
func commentOpeningIndent(s *tokens.Stream, i int) (string, bool) {
	whitespace := ""
	if prev := i - 1; prev >= 0 && s.At(prev).Kind == token.Whitespace {
		whitespace = s.At(prev).Value
	}
	cut := -1
	for j := 0; j < len(whitespace); j++ {
		if whitespace[j] == '\n' || whitespace[j] == '\r' {
			cut = j
		}
	}
	if cut < 0 {
		return "", false
	}
	return whitespace[cut+1:], true
}

// realignCommentLines rewrites the leading whitespace of every interior line
// whose trimmed content starts with "*", aligning it to indent + one space.
func realignCommentLines(v, indent string) string {
	lines := strings.Split(v, "\n")
	for k := 1; k < len(lines); k++ {
		trimmed := strings.TrimLeft(lines[k], " \t")
		if !strings.HasPrefix(trimmed, "*") {
			continue
		}
		lines[k] = indent + " " + trimmed
	}
	return strings.Join(lines, "\n")
}
