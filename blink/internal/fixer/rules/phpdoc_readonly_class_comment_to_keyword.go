package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/PhpdocReadonlyClassCommentToKeywordFixer.php
//
// PhpdocReadonlyClassCommentToKeyword turns a "@readonly" class docblock
// annotation into the readonly keyword (risky), dropping the annotation and, for
// an empty single-line docblock, the whole comment.
type PhpdocReadonlyClassCommentToKeyword struct{}

func (PhpdocReadonlyClassCommentToKeyword) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\PhpdocReadonlyClassCommentToKeywordFixer`
}

func (PhpdocReadonlyClassCommentToKeyword) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/PhpdocReadonlyClassCommentToKeywordFixer.php"
}

// prcModifiers are the class modifiers walked over between the docblock and the
// class keyword.
func prcIsModifier(s *tokens.Stream, i int) bool {
	if i < 0 || s.At(i).Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(s.At(i).Value) {
	case "abstract", "final", "private", "public", "protected", "readonly":
		return true
	}
	return false
}

func (PhpdocReadonlyClassCommentToKeyword) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok {
			continue
		}
		if !prcHasReadonlyAnnotation(d) {
			continue
		}

		// the docblock is only rewritten when it documents a class
		index := sigNext(s, i)
		addReadonly := true
		for index >= 0 && prcIsModifier(s, index) {
			if kwIs(s.At(index), "readonly") {
				addReadonly = false
			}
			index = sigNext(s, index)
		}
		if index < 0 || !taIsClassKw(s, index) {
			continue
		}

		if addReadonly {
			s.InsertSliceAt(index, []token.Token{
				{Kind: token.Keyword, Value: "readonly"},
				{Kind: token.Whitespace, Value: " "},
			})
		}

		if d.single {
			// a single-line docblock collapses to nothing once its annotation goes
			clearTokenAndMergeWS(s, i)
			changed = true
			continue
		}
		kept := d.inner[:0:0]
		for _, l := range d.inner {
			if generalPhpdocLineHasTag(l.content, []string{"readonly"}, false) {
				continue
			}
			kept = append(kept, l)
		}
		d.inner = kept
		s.SetValue(i, d.render())
		changed = true
	}
	return changed
}

// prcHasReadonlyAnnotation reports whether the docblock carries a "@readonly" tag.
func prcHasReadonlyAnnotation(d docblock) bool {
	for _, l := range d.inner {
		if generalPhpdocLineHasTag(l.content, []string{"readonly"}, false) {
			return true
		}
	}
	return false
}
