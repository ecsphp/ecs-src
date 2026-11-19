package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/CharacterBeforePHPOpeningTagSniff.php
//
// CharacterBeforePHPOpeningTag reports a first <?php tag that content other
// than a byte order mark or a shebang line precedes.
type CharacterBeforePHPOpeningTag struct{}

func (CharacterBeforePHPOpeningTag) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\CharacterBeforePHPOpeningTagSniff`
}

func (CharacterBeforePHPOpeningTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/CharacterBeforePHPOpeningTagSniff.php"
}

func (CharacterBeforePHPOpeningTag) Fix(*tokens.Stream) bool { return false }

// characterBeforePHPOpeningTagBOMs are the byte order marks allowed before the tag.
var characterBeforePHPOpeningTagBOMs = []string{"\xef\xbb\xbf", "\xfe\xff", "\xff\xfe"}

func (c CharacterBeforePHPOpeningTag) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	open := -1
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=") {
			open = i
			break
		}
	}
	if open <= 0 || characterBeforePHPOpeningTagAllowed(s, open) {
		return nil
	}
	return []fixer.CodingStandardError{{
		Line:    phpcsTokenLines(s)[open],
		Message: "The opening PHP tag must be the first content in the file",
		Checker: c.Name() + ".Found",
	}}
}

// characterBeforePHPOpeningTagAllowed reports whether what precedes the tag at
// open is a single line of inline HTML, PHP_CodeSniffer's token 0, that starts
// with a byte order mark or a shebang.
func characterBeforePHPOpeningTagAllowed(s *tokens.Stream, open int) bool {
	if open != 1 || s.At(0).Kind != token.InlineHTML {
		return false
	}
	html := s.At(0).Value
	// PHP_CodeSniffer splits inline HTML into a token per line
	if i := strings.IndexByte(html, '\n'); i >= 0 && i < len(html)-1 {
		return false
	}
	for _, bom := range characterBeforePHPOpeningTagBOMs {
		if strings.HasPrefix(html, bom) {
			return true
		}
	}
	return strings.HasPrefix(html, "#!")
}
