package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoUselessInheritdocFixer.php
//
// PhpdocNoUselessInheritdoc removes a "{@inheritDoc}" (or "@inheritdoc") line
// where there is nothing to inherit from, as PHP-CS-Fixer does: the docblock
// right before a class or interface that neither extends nor implements, and
// the docblocks inside one that also uses no trait. Docblocks anywhere else,
// and in traits and enums, are left alone. A block left empty is removed with
// its indentation.
type PhpdocNoUselessInheritdoc struct{}

func (PhpdocNoUselessInheritdoc) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoUselessInheritdocFixer`
}

func (PhpdocNoUselessInheritdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoUselessInheritdocFixer.php"
}

func (PhpdocNoUselessInheritdoc) Fix(s *tokens.Stream) bool {
	var docs []int
	for i := 0; i < s.Len(); i++ {
		if inheritdocIsClassy(s, i) {
			i = inheritdocCollectClassy(s, i, &docs)
		}
	}
	// docs are in ascending order; fixing from the end keeps earlier indices valid
	changed := false
	for k := len(docs) - 1; k >= 0; k-- {
		if inheritdocFixDoc(s, docs[k]) {
			changed = true
		}
	}
	return changed
}

// inheritdocIsClassy reports whether token i opens a class or interface
// declaration, not the "class" of a Foo::class constant.
func inheritdocIsClassy(s *tokens.Stream, i int) bool {
	if !isKeywordValue(s.At(i), "class", "interface") {
		return false
	}
	return !isPunctVal(s, prevMeaningfulIndex(s, i), "::")
}

// inheritdocCollectClassy mirrors PHP-CS-Fixer's fixClassy: it collects the
// docblocks of the class or interface at i that may lose their inheritdoc, and
// returns the index of its closing brace.
func inheritdocCollectClassy(s *tokens.Stream, i int, docs *[]int) int {
	open := nextPunctOfKind(s, i, "{")
	if open < 0 {
		return i
	}
	end := s.MatchForward(open)
	if end < 0 {
		return i
	}
	inherits := false
	for j := i; j < open; j++ {
		if isKeywordValue(s.At(j), "extends", "implements") {
			inherits = true
			break
		}
	}
	if !inherits {
		// a trait cannot document the class itself, so this one is still fixed
		if p := i - 1; p >= 0 {
			if s.At(p).Kind == token.Whitespace {
				p--
			}
			if p >= 0 && s.At(p).Kind == token.DocComment {
				*docs = append(*docs, p)
			}
		}
		if isKeywordValue(s.At(i), "class") && inheritdocUsesTrait(s, open, end) {
			inherits = true
		}
	}
	for j := open; j < end; j++ {
		if isKeywordValue(s.At(j), "class") && inheritdocIsClassy(s, j) {
			j = inheritdocCollectClassy(s, j, docs)
		} else if !inherits && s.At(j).Kind == token.DocComment {
			*docs = append(*docs, j)
		}
	}
	return end
}

// inheritdocUsesTrait reports whether the class body between open and end
// imports a trait: a "use" at the body's own brace depth.
func inheritdocUsesTrait(s *tokens.Stream, open, end int) bool {
	depth := 0
	for j := open; j < end; j++ {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				depth++
			case "}":
				depth--
			}
			continue
		}
		if depth == 1 && isKeywordValue(t, "use") {
			return true
		}
	}
	return false
}

// inheritdocFixDoc drops the inheritdoc lines of the docblock at i, removing the
// block and its indentation when nothing else is left in it.
func inheritdocFixDoc(s *tokens.Stream, i int) bool {
	d, ok := parseDoc(s.At(i).Value)
	if !ok {
		return false
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
		return false
	}
	if allBlankLines(kept) {
		// the block is now empty: drop it and the whitespace that indented
		// it so the following line keeps its original indentation
		s.RemoveAt(i)
		if i > 0 && s.At(i-1).Kind == token.Whitespace {
			s.RemoveAt(i - 1)
		}
		return true
	}
	d.inner = kept
	s.SetValue(i, d.render())
	return true
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
