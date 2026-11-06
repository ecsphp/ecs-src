package rules

import (
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderStaticFixer.php
//
// PhpUnitDataProviderStatic makes PHPUnit data providers static. With the
// default (force disabled) a provider using $this is left alone. Risky.
type PhpUnitDataProviderStatic struct {
	force bool
}

func (PhpUnitDataProviderStatic) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDataProviderStaticFixer`
}

func (PhpUnitDataProviderStatic) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderStaticFixer.php"
}

func (f PhpUnitDataProviderStatic) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["force"].(bool); ok {
		f.force = v
	}
	return f
}

func (f PhpUnitDataProviderStatic) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if f.staticClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func (f PhpUnitDataProviderStatic) staticClass(s *tokens.Stream, start, end int) bool {
	var functionIndexes []int
	for _, dp := range puDataProviders(s, start, end) {
		if !f.force {
			if bodyOpen := nextPunctOfKind(s, dp.nameIndex, "{"); bodyOpen >= 0 {
				bodyEnd := s.MatchForward(bodyOpen)
				if bodyEnd > bodyOpen && dataProviderUsesThis(s, bodyOpen, bodyEnd) {
					continue
				}
			}
		}
		fnIdx := sigPrev(s, dp.nameIndex)
		if fnIdx < 0 || !kwIs(s.At(fnIdx), "function") {
			continue
		}
		if dataProviderMethodIsStatic(s, fnIdx) {
			continue
		}
		functionIndexes = append(functionIndexes, fnIdx)
	}
	if len(functionIndexes) == 0 {
		return false
	}
	// insert from the last function back, so earlier indexes stay valid
	sort.Sort(sort.Reverse(sort.IntSlice(functionIndexes)))
	for _, fnIdx := range functionIndexes {
		s.InsertSliceAt(fnIdx, []token.Token{
			{Kind: token.Keyword, Value: "static"},
			{Kind: token.Whitespace, Value: " "},
		})
	}
	return true
}

// dataProviderUsesThis reports whether a $this variable appears in (start, end).
func dataProviderUsesThis(s *tokens.Stream, start, end int) bool {
	for i := start + 1; i < end && i < s.Len(); i++ {
		if s.At(i).Kind == token.Variable && s.At(i).Value == "$this" {
			return true
		}
	}
	return false
}

// dataProviderMethodIsStatic mirrors getMethodAttributes()['static']: a `static`
// modifier in the method's modifier run before the `function` keyword.
func dataProviderMethodIsStatic(s *tokens.Stream, fnIdx int) bool {
	for p := getPrevNonWhitespace(s, fnIdx); p >= 0; p = getPrevNonWhitespace(s, p) {
		t := s.At(p)
		if t.Kind == token.Comment {
			continue
		}
		if t.Kind == token.Keyword && puModifierKw[strings.ToLower(t.Value)] {
			if strings.EqualFold(t.Value, "static") {
				return true
			}
			continue
		}
		break
	}
	return false
}
