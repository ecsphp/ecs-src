package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
	"slices"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderFixer.php
//
// PhpdocOrder orders a docblock's annotations by tag, following the `order`
// option. Moved annotations leave blank lines behind, as in PHP.
type PhpdocOrder struct {
	order []string
}

var phpdocOrderDefault = []string{"param", "throws", "return"}

func (PhpdocOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocOrderFixer`
}

func (PhpdocOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderFixer.php"
}

// WithConfig keeps the default for an `order` PHP rejects: fewer than two tags,
// or duplicates.
func (f PhpdocOrder) WithConfig(config map[string]any) fixer.Fixer {
	list, ok := phpdocTagsStringList(config["order"])
	if !ok || len(list) < 2 {
		return f
	}
	seen := map[string]bool{}
	for _, tag := range list {
		if seen[tag] {
			return f
		}
		seen[tag] = true
	}
	f.order = list
	return f
}

// configurationOrder ports configurePostNormalisation: each tag followed by
// its phpstan- and psalm- variants, unless the variant is configured itself.
func (f PhpdocOrder) configurationOrder() []string {
	order := f.order
	if order == nil {
		order = phpdocOrderDefault
	}
	configured := map[string]bool{}
	for _, tag := range order {
		configured[tag] = true
	}
	var out []string
	for _, tag := range order {
		out = append(out, tag)
		for _, prefix := range []string{"phpstan-", "psalm-"} {
			if !configured[prefix+tag] {
				out = append(out, prefix+tag)
			}
		}
	}
	return out
}

// IsCandidate rules out a file without the trigger: only docblocks are reordered.
func (PhpdocOrder) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasKind(token.DocComment)
}

func (f PhpdocOrder) Fix(s *tokens.Stream) bool {
	order := f.configurationOrder()
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		content := t.Value
		successors := order
		for len(successors) >= 3 {
			predecessor := successors[0]
			successors = successors[1:]
			content = phpdocMoveAnnotationsBefore(predecessor, successors, content)
		}
		last := order[len(order)-1]
		content = phpdocMoveAnnotationsAfter(last, order[:len(order)-1], content)
		if content != t.Value {
			s.SetValue(i, content)
			changed = true
		}
	}
	return changed
}

// phpdocMoveAnnotationsBefore ports moveAnnotationsBefore: every `before`
// annotation that starts before the end of the last `move` annotation is
// appended, in order, to that annotation's last line.
func phpdocMoveAnnotationsBefore(move string, before []string, content string) string {
	doc := newPHPDocBlock(content)
	toBeMoved := doc.annotationsOfType([]string{move})
	if len(toBeMoved) == 0 {
		return content
	}
	others := doc.annotationsOfType(before)
	if len(others) == 0 {
		return content
	}
	end := toBeMoved[len(toBeMoved)-1].end
	line := doc.lines[end]
	for _, other := range others {
		if other.start < end {
			line.content += other.content()
			other.remove()
		}
	}
	return doc.content()
}

// phpdocMoveAnnotationsAfter ports moveAnnotationsAfter: every `after`
// annotation that ends after the start of the first `move` annotation is
// prepended, in order, to that annotation's first line.
func phpdocMoveAnnotationsAfter(move string, after []string, content string) string {
	doc := newPHPDocBlock(content)
	toBeMoved := doc.annotationsOfType([]string{move})
	if len(toBeMoved) == 0 {
		return content
	}
	others := doc.annotationsOfType(after)
	if len(others) == 0 {
		return content
	}
	start := toBeMoved[0].start
	line := doc.lines[start]
	for _, other := range slices.Backward(others) {
		if other.end > start {
			line.content = other.content() + line.content
			other.remove()
		}
	}
	return doc.content()
}
