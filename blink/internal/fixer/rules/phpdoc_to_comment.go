package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocToCommentFixer.php
//
// PhpdocToComment turns a docblock into a regular comment when it does not
// document a structural element (and is not the file header). Default config:
// no ignored tags, allow_before_return_statement false.
type PhpdocToComment struct{}

func (PhpdocToComment) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocToCommentFixer`
}

func (PhpdocToComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocToCommentFixer.php"
}

func (PhpdocToComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		if docIsHeaderComment(s, i) {
			continue
		}
		if docIsBeforeStructuralElement(s, i) {
			continue
		}
		// default ignored_tags is empty, so no tag exempts the docblock
		newVal := "/*" + strings.TrimLeft(s.At(i).Value, "/*")
		s.Set(i, token.Token{Kind: token.Comment, Value: newVal})
		changed = true
	}
	return changed
}
