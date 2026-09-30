package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/AlignMultilineCommentFixer.php
//
// AlignMultilineComment re-indents the continuation "*" lines of a multi-line
// block comment or docblock so each "*" sits one space to the right of the
// opening "/*". Only the leading whitespace of lines whose trimmed content
// starts with "*" is rewritten; free-form lines are left untouched.
//
// Option `comment_type` selects which comments to align: "phpdocs_only" (doc
// comments only), "phpdocs_like" (also block comments whose lines all start with
// "*") or "all_multiline" (any multi-line comment, inserting a missing "*"). The
// zero value keeps blink's existing behavior (both comment kinds, realigning only
// lines that already start with "*").
type AlignMultilineComment struct {
	commentType string
}

func (AlignMultilineComment) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\AlignMultilineCommentFixer`
}

func (AlignMultilineComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/AlignMultilineCommentFixer.php"
}

func (f AlignMultilineComment) WithConfig(config map[string]any) fixer.Fixer {
	if c, ok := config["comment_type"].(string); ok {
		switch c {
		case "phpdocs_only", "phpdocs_like", "all_multiline":
			f.commentType = c
		}
	}
	return f
}

func (f AlignMultilineComment) Fix(s *tokens.Stream) bool {
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
		isComment := t.Kind == token.Comment
		switch f.commentType {
		case "phpdocs_only":
			if isComment {
				continue
			}
		case "phpdocs_like":
			if isComment && !alignMultilineCommentDocLike(v) {
				continue
			}
		}
		indent, ok := commentOpeningIndent(s, i)
		if !ok {
			continue
		}
		var nv string
		if f.commentType == "" {
			nv = realignCommentLines(v, indent)
		} else {
			nv = alignMultilineCommentFull(v, indent)
		}
		if nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

// alignMultilineCommentDocLike reports whether every continuation line of a block
// comment starts with "*" and none is blank, matching the "phpdocs_like" filter.
func alignMultilineCommentDocLike(v string) bool {
	lines := strings.Split(v, "\n")
	for k := 1; k < len(lines); k++ {
		trimmed := strings.TrimRight(strings.TrimLeft(lines[k], " \t"), "\r")
		if trimmed == "" || trimmed[0] != '*' {
			return false
		}
	}
	return true
}

// alignMultilineCommentFull realigns every continuation line to indent + " *",
// inserting a missing "*" and normalizing blank lines, matching PHP-CS-Fixer's
// per-line handling for any configured comment_type.
func alignMultilineCommentFull(v, indent string) string {
	lines := strings.Split(v, "\n")
	for k := 1; k < len(lines); k++ {
		cr := strings.HasSuffix(lines[k], "\r")
		line := strings.TrimRight(strings.TrimLeft(lines[k], " \t"), "\r")
		if line == "" {
			line = "*"
		} else if line[0] != '*' {
			line = "* " + line
		}
		line = indent + " " + line
		if cr {
			line += "\r"
		}
		lines[k] = line
	}
	return strings.Join(lines, "\n")
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
