package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowAlternativePHPTagsSniff.php
//
// DisallowAlternativePHPTags reports <script language="php"> opening tags in
// inline HTML. ASP style tags only get warnings, which ECS drops for this sniff.
type DisallowAlternativePHPTags struct{}

func (DisallowAlternativePHPTags) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\DisallowAlternativePHPTagsSniff`
}

func (DisallowAlternativePHPTags) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowAlternativePHPTagsSniff.php"
}

func (DisallowAlternativePHPTags) Fix(*tokens.Stream) bool { return false }

var disallowAlternativePHPTagsScript = regexp.MustCompile(`(?i)(<script (?:[^>]+)?language=['"]?php['"]?(?:[^>]+)?>)`)

func (d DisallowAlternativePHPTags) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.InlineHTML || !strings.Contains(strings.ToLower(t.Value), "<script ") {
			continue
		}
		if lines == nil {
			lines = phpcsTokenLines(s)
		}
		// PHP_CodeSniffer splits inline HTML into a token per line
		line := lines[i]
		for content := range strings.SplitAfterSeq(t.Value, "\n") {
			if m := disallowAlternativePHPTagsScript.FindStringSubmatch(content); m != nil && phpTrim(content) != "" {
				errs = append(errs, fixer.CodingStandardError{
					Line:    line,
					Message: `Script style opening tag used; expected "<?php" but found "` + m[1] + disallowAlternativePHPTagsSnippet(content, m[1]) + `"`,
					Checker: d.Name() + ".ScriptOpenTagFound",
				})
			}
			line++
		}
	}
	return errs
}

// disallowAlternativePHPTagsSnippet is the sniff's getSnippet: up to 40 bytes
// of content after start, with "..." when more follow.
func disallowAlternativePHPTagsSnippet(content, start string) string {
	from := strings.Index(content, start) + len(start)
	snippet := content[from:min(from+40, len(content))]
	if len(content)-from > 40 {
		snippet += "..."
	}
	return snippet
}
