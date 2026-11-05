package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocToCommentFixer.php
//
// PhpdocToComment turns a docblock into a regular comment when it does not
// document a structural element (and is not the file header). It reads
// ignored_tags (a docblock holding one of these tags stays a docblock, matched
// case insensitively; none by default) and allow_before_return_statement
// (false by default: true keeps a docblock right before a return).
type PhpdocToComment struct {
	ignoredTags                map[string]bool // lowercased
	allowBeforeReturnStatement bool
}

// phpdocToCommentTagRe is PHP's `~\@([a-zA-Z0-9_\\\-]+)\b~`; Go's leftmost-first
// matching backtracks the greedy name to the last word boundary as PCRE does.
var phpdocToCommentTagRe = regexp.MustCompile(`@([a-zA-Z0-9_\\\-]+)\b`)

func (PhpdocToComment) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocToCommentFixer`
}

func (PhpdocToComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocToCommentFixer.php"
}

func (f PhpdocToComment) WithConfig(config map[string]any) fixer.Fixer {
	if tags, ok := phpdocTagsStringList(config["ignored_tags"]); ok {
		f.ignoredTags = make(map[string]bool, len(tags))
		for _, tag := range tags {
			f.ignoredTags[asciiLower(tag)] = true
		}
	}
	if v, ok := config["allow_before_return_statement"].(bool); ok {
		f.allowBeforeReturnStatement = v
	}
	return f
}

func (f PhpdocToComment) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		if docIsHeaderComment(s, i) {
			continue
		}
		if f.allowBeforeReturnStatement && docIsBeforeReturn(s, i) {
			continue
		}
		if docIsBeforeStructuralElement(s, i) {
			continue
		}
		if f.hasIgnoredTag(s.At(i).Value) {
			continue
		}
		newVal := "/*" + strings.TrimLeft(s.At(i).Value, "/*")
		s.Set(i, token.Token{Kind: token.Comment, Value: newVal})
		changed = true
	}
	return changed
}

func (f PhpdocToComment) hasIgnoredTag(doc string) bool {
	if len(f.ignoredTags) == 0 {
		return false
	}
	for _, m := range phpdocToCommentTagRe.FindAllStringSubmatch(doc, -1) {
		if f.ignoredTags[asciiLower(m[1])] {
			return true
		}
	}
	return false
}

// docIsBeforeReturn mirrors CommentsAnalyzer::isBeforeReturn.
func docIsBeforeReturn(s *tokens.Stream, index int) bool {
	next := docNextTokenIndex(s, index)
	if next < 0 || isPunctVal(s, next, "}") {
		return false
	}
	return kwIs(s.At(next), "return")
}
