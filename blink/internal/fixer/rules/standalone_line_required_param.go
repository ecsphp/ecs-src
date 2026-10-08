package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLineRequiredParamFixer.php
//
// StandaloneLineRequiredParam puts every parameter of a public method marked
// with #[Required] or @required on its own line, easing git diffs when a
// dependency is added or removed from an autowired setter.
type StandaloneLineRequiredParam struct{}

var (
	requiredAnnotationRe = regexp.MustCompile(`(?i)@required\b`)
	// the lexer folds a #[Required] attribute into a single "#..." comment token
	requiredAttributeRe = regexp.MustCompile(`(?i)^#\[.*\bRequired\b`)
)

func (StandaloneLineRequiredParam) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLineRequiredParamFixer`
}

func (StandaloneLineRequiredParam) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLineRequiredParamFixer.php"
}

func (StandaloneLineRequiredParam) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || strings.ToLower(s.At(i).Value) != "function" {
			continue
		}
		if !isPublicRequiredMethod(s, i) {
			continue
		}
		name := sigNext(s, i)
		if name < 0 {
			continue
		}
		open := sigNext(s, name)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue // no parameters
		}
		if reflowParen(s, open, closeIdx, false) {
			changed = true
		}
	}
	return changed
}

// isPublicRequiredMethod walks back over the method head (modifiers, doc blocks
// and attributes) and reports whether it carries both a public modifier and a
// #[Required] attribute or an @required annotation.
func isPublicRequiredMethod(s *tokens.Stream, fnPos int) bool {
	isPublic := false
	isRequired := false
	for j := fnPos - 1; j >= 0; j-- {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace:
			continue
		case token.Comment:
			if requiredAttributeRe.MatchString(t.Value) {
				isRequired = true
			}
			continue
		case token.DocComment:
			if requiredAnnotationRe.MatchString(t.Value) {
				isRequired = true
			}
			continue
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "public":
				isPublic = true
				continue
			case "protected", "private", "static", "final", "abstract", "readonly":
				continue
			}
			return isPublic && isRequired
		default:
			return isPublic && isRequired
		}
	}
	return isPublic && isRequired
}
