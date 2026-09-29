package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// commentBody returns the text of a // or # or /* */ comment without markers.
func commentBody(v string) string {
	switch {
	case strings.HasPrefix(v, "//"):
		return strings.TrimSpace(v[2:])
	case strings.HasPrefix(v, "#"):
		return strings.TrimSpace(v[1:])
	case strings.HasPrefix(v, "/*"):
		v = strings.TrimPrefix(v, "/*")
		v = strings.TrimSuffix(v, "*/")
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(v)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/NoEmptyCommentFixer.php
//
// NoEmptyComment removes a comment with no content ("//", "#", "/* */").
type NoEmptyComment struct{}

func (NoEmptyComment) Name() string {
	return `PhpCsFixer\Fixer\Comment\NoEmptyCommentFixer`
}

func (NoEmptyComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/NoEmptyCommentFixer.php"
}

func (NoEmptyComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment {
			continue
		}
		if strings.HasPrefix(t.Value, "#[") {
			continue // attribute, not a comment
		}
		if commentBody(t.Value) == "" {
			s.RemoveAt(i)
			changed = true
			// merge whitespace left adjacent by the removal, so a now-blank line
			// is a single token that no_whitespace_in_blank_line can clear.
			if i-1 >= 0 && i < s.Len() && s.At(i-1).Kind == token.Whitespace && s.At(i).Kind == token.Whitespace {
				s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
				s.RemoveAt(i)
			}
			i--
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/SingleLineCommentSpacingFixer.php
//
// SingleLineCommentSpacing puts a space after "//" or "#" in a single-line
// comment ("//foo" -> "// foo"). Attributes ("#[...]") are left alone.
type SingleLineCommentSpacing struct{}

func (SingleLineCommentSpacing) Name() string {
	return `PhpCsFixer\Fixer\Comment\SingleLineCommentSpacingFixer`
}

func (SingleLineCommentSpacing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/SingleLineCommentSpacingFixer.php"
}

func (SingleLineCommentSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Comment {
			continue
		}
		v := t.Value
		// single-line block comment: ensure one space inside "/* ... */"
		if strings.HasPrefix(v, "/*") && !strings.HasPrefix(v, "/**") &&
			strings.HasSuffix(v, "*/") && len(v) >= 4 && !strings.ContainsAny(v, "\n\r") {
			inner := v[2 : len(v)-2]
			nv := inner
			if len(nv) > 0 && nv[0] != ' ' && nv[0] != '\t' {
				nv = " " + nv
			}
			if len(nv) > 0 && nv[len(nv)-1] != ' ' && nv[len(nv)-1] != '\t' {
				nv = nv + " "
			}
			if nv != inner {
				s.SetValue(i, "/*"+nv+"*/")
				changed = true
			}
			continue
		}
		var marker string
		switch {
		case strings.HasPrefix(v, "//"):
			marker = "//"
		case strings.HasPrefix(v, "#") && !strings.HasPrefix(v, "#["):
			marker = "#"
		default:
			continue
		}
		rest := v[len(marker):]
		if rest == "" || rest[0] == ' ' || rest[0] == '\t' {
			continue
		}
		s.SetValue(i, marker+" "+rest)
		changed = true
	}
	return changed
}
