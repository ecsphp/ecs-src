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

// commentType classifies a comment token the way PHP's getCommentType does:
// hash ("#"), double-slash ("//") or slash-asterisk ("/* */").
const (
	commentHash = iota
	commentDoubleSlash
	commentSlashAsterisk
)

func commentType(v string) int {
	if strings.HasPrefix(v, "#") {
		return commentHash
	}
	if len(v) > 1 && v[1] == '*' {
		return commentSlashAsterisk
	}
	return commentDoubleSlash
}

func lineBreakCount(v string) int {
	n := 0
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\n':
			n++
		case '\r':
			n++
			if i+1 < len(v) && v[i+1] == '\n' {
				i++
			}
		}
	}
	return n
}

// noEmptyCommentBlock walks the // or # comment block starting at index and
// returns its start, end and whether the whole block is empty, mirroring PHP's
// getCommentBlock. Consecutive // (or #) lines separated by a single line break
// form one block; a blank line (>1 line break) or a different comment type ends
// it. A /* */ comment is always a one-token block.
func noEmptyCommentBlock(s *tokens.Stream, index int) (start, end int, empty bool) {
	ct := commentType(s.At(index).Value)
	empty = commentBody(s.At(index).Value) == ""
	if ct == commentSlashAsterisk {
		return index, index, empty
	}

	start = index
	i := index + 1
	for ; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Comment {
			if strings.HasPrefix(t.Value, "#[") || commentType(t.Value) != ct {
				break
			}
			if empty {
				empty = commentBody(t.Value) == ""
			}
			continue
		}
		if t.Kind != token.Whitespace || lineBreakCount(t.Value) > 1 {
			break
		}
	}
	return start, i - 1, empty
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

		start, end, empty := noEmptyCommentBlock(s, i)
		if !empty {
			// a block with any non-empty comment line is left untouched, so an
			// empty // line inside a real // comment block is preserved.
			i = end
			continue
		}

		// clear the block and merge the whitespace around it into one token,
		// mirroring PHP's clearTokenAndMergeSurroundingWhitespace.
		lo, hi := start, end
		if lo > 0 && s.At(lo-1).Kind == token.Whitespace {
			lo--
		}
		if hi+1 < s.Len() && s.At(hi+1).Kind == token.Whitespace {
			hi++
		}
		var b strings.Builder
		for j := lo; j <= hi; j++ {
			if s.At(j).Kind == token.Whitespace {
				b.WriteString(s.At(j).Value)
			}
		}
		if merged := b.String(); merged != "" {
			s.ReplaceRange(lo, hi, []token.Token{{Kind: token.Whitespace, Value: merged}})
		} else {
			s.ReplaceRange(lo, hi, nil)
		}
		changed = true
		i = lo
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
