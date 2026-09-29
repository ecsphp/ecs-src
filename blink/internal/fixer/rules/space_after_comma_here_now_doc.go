package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/SpaceAfterCommaHereNowDocFixer.php
//
// SpaceAfterCommaHereNowDoc puts a newline between a heredoc/nowdoc closing
// marker and a following "," or "]" (a heredoc argument in a call/array). The
// blink lexer keeps a whole heredoc/nowdoc as a single string token, so the
// closing marker is the end of that token.
type SpaceAfterCommaHereNowDoc struct{}

func (SpaceAfterCommaHereNowDoc) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\SpaceAfterCommaHereNowDocFixer`
}

func (SpaceAfterCommaHereNowDoc) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/SpaceAfterCommaHereNowDocFixer.php"
}

func (SpaceAfterCommaHereNowDoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind != token.String || !strings.HasPrefix(s.At(i).Value, "<<<") {
			continue
		}
		n := i + 1
		if n >= s.Len() {
			continue
		}
		if s.At(n).Kind == token.Punct && (s.At(n).Value == "," || s.At(n).Value == "]") {
			s.InsertAt(n, token.Token{Kind: token.Whitespace, Value: "\n"})
			changed = true
		}
	}
	return changed
}
