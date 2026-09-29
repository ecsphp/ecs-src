package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocLineSpanFixer.php
//
// PhpdocLineSpan expands a single-line docblock that documents a class member
// into a multi-line one, matching the fixer's default config (const, method and
// property all default to "multi").
type PhpdocLineSpan struct{}

func (PhpdocLineSpan) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocLineSpanFixer`
}

func (PhpdocLineSpan) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocLineSpanFixer.php"
}

func (PhpdocLineSpan) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(t.Value)
		if !ok || !d.single {
			continue
		}
		if !documentsMember(s, i) {
			continue
		}
		indent, ok := docblockLineIndent(s, i)
		if !ok {
			continue
		}
		expandDocblock(&d, indent)
		s.SetValue(i, d.render())
		changed = true
	}
	return changed
}

// documentsMember reports whether the token after the docblock at index i is a
// visibility keyword (public/private/protected) or "var", which unambiguously
// introduce a class member (property or method). Only whitespace is skipped, so
// an intervening attribute is a no-op. Bare const/function without visibility is
// intentionally excluded - on flat tokens it cannot be told from a top-level
// const or a free function, which the real fixer (class-scope only) leaves alone.
func documentsMember(s *tokens.Stream, i int) bool {
	j := skipWhitespace(s, i+1)
	if j >= s.Len() {
		return false
	}
	next := s.At(j)
	if next.Kind != token.Keyword {
		return false
	}
	lw := strings.ToLower(next.Value)
	return visibilityModifiers[lw] || lw == "var"
}

// expandDocblock rewrites a single-line docblock into a multi-line one, keeping
// the content verbatim and aligning the "*" one column past the given indent.
func expandDocblock(d *docblock, indent string) {
	content := ""
	if len(d.inner) > 0 {
		content = d.inner[0].content
	}
	d.single = false
	d.open = "/**"
	d.inner = []docLine{{prefix: indent + " * ", content: content}}
	d.close = indent + " */"
}
