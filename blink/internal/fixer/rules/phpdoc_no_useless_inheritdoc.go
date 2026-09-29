package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoUselessInheritdocFixer.php
//
// PhpdocNoUselessInheritdoc removes a "{@inheritDoc}" (or "@inheritdoc") line
// from a docblock; PHP already inherits the parent's doc, so the tag adds
// nothing. A block left empty is cleaned up by no_empty_phpdoc / phpdoc_trim.
type PhpdocNoUselessInheritdoc struct{}

func (PhpdocNoUselessInheritdoc) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoUselessInheritdocFixer`
}

func (PhpdocNoUselessInheritdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoUselessInheritdocFixer.php"
}

func (PhpdocNoUselessInheritdoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok {
			continue
		}
		kept := d.inner[:0:0]
		removed := false
		for _, l := range d.inner {
			if isInheritdoc(l.content) {
				removed = true
				continue
			}
			kept = append(kept, l)
		}
		if !removed {
			continue
		}
		changed = true
		if allBlankLines(kept) {
			// the block is now empty: drop it and the whitespace that indented
			// it so the following line keeps its original indentation
			s.RemoveAt(i)
			if i > 0 && s.At(i-1).Kind == token.Whitespace {
				s.RemoveAt(i - 1)
				i--
			}
			i--
			continue
		}
		d.inner = kept
		s.SetValue(i, d.render())
	}
	return changed
}

func allBlankLines(lines []docLine) bool {
	for _, l := range lines {
		if strings.TrimSpace(l.content) != "" {
			return false
		}
	}
	return true
}

// isInheritdoc reports whether a docblock line is nothing but an inheritdoc tag.
func isInheritdoc(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	return c == "{@inheritdoc}" || c == "@inheritdoc"
}
