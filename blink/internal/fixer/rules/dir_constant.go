package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DirConstantFixer.php
//
// DirConstant replaces a global "dirname(__FILE__)" call with the equivalent
// __DIR__ constant. Risky: dirname may be overridden.
type DirConstant struct{}

func (DirConstant) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\DirConstantFixer`
}

func (DirConstant) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DirConstantFixer.php"
}

func isFileConstant(s *tokens.Stream, i int) bool {
	return i >= 0 && s.At(i).Kind == token.Ident && strings.EqualFold(s.At(i).Value, "__FILE__")
}

// IsCandidate rules out a file without the trigger: only a "dirname" call is replaced.
func (DirConstant) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("dirname")
}

func (DirConstant) Fix(s *tokens.Stream) bool {
	changed := false
	curr := 0
	for {
		name, open, closeIdx, ok := aliasFind(s, "dirname", curr, s.Len()-1)
		if !ok || closeIdx < 0 {
			return changed
		}
		// keep searching inside the call, so nested expressions are processed
		curr = open

		// __FILE__ must be the only argument
		right := prevMeaningfulIndex(s, closeIdx)
		trailingComma := -1
		if isPunctVal(s, right, ",") {
			trailingComma = right
			right = prevMeaningfulIndex(s, right)
		}
		if !isFileConstant(s, right) {
			continue
		}
		left := nextMeaningfulIndex(s, open)
		if !isFileConstant(s, left) {
			continue
		}

		if ns := prevMeaningfulIndex(s, name); isPunctVal(s, ns, `\`) {
			removeWhitespaceSafely(s, ns, 1)
			clearTokenAt(s, ns)
		}
		if trailingComma >= 0 {
			if !commentAt(s, nextSignificantIndex(s, trailingComma)) {
				removeWhitespaceSafely(s, trailingComma, 1)
			}
			clearTokenAndMergeWS(s, trailingComma)
		}
		if !commentAt(s, nextSignificantIndex(s, closeIdx)) {
			removeWhitespaceSafely(s, closeIdx, -1)
		}
		clearTokenAndMergeWS(s, closeIdx)
		if !commentAt(s, nextSignificantIndex(s, open)) {
			removeWhitespaceSafely(s, open, -1)
		}
		removeWhitespaceSafely(s, open, 1)
		clearTokenAndMergeWS(s, open)

		s.Set(left, token.Token{Kind: token.Ident, Value: "__DIR__", Pos: s.At(left).Pos})
		clearTokenAndMergeWS(s, name)
		changed = true
	}
}
