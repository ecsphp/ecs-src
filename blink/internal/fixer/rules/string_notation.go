package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SingleQuoteFixer.php
//
// SingleQuote converts a double-quoted string to single quotes when it is safe:
// no interpolation, no escape sequence other than \\, \" and \$, and no single
// quote in the content. WithSingleQuotes (strings_containing_single_quote_chars)
// also converts strings containing single quotes, escaping them.
type SingleQuote struct {
	WithSingleQuotes bool
}

func (SingleQuote) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\SingleQuoteFixer`
}

func (SingleQuote) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SingleQuoteFixer.php"
}

func (f SingleQuote) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["strings_containing_single_quote_chars"].(bool); ok {
		f.WithSingleQuotes = v
	}
	return f
}

func (f SingleQuote) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if !aliasIsConstantString(t) {
			continue
		}
		prefix, v := token.SplitBinaryPrefix(t.Value) // b"x" keeps its prefix
		if v[0] != '"' {
			continue
		}
		content := v[1 : len(v)-1]
		if singleQuoteKeepsEscape(content) || (!f.WithSingleQuotes && strings.Contains(content, "'")) {
			continue // an escape sequence single quotes lack, or a quote that would need escaping
		}
		content = strings.NewReplacer(`\"`, `"`, `\$`, `$`, `'`, `\'`).Replace(content)
		s.SetValue(i, prefix+"'"+content+"'")
		changed = true
	}
	return changed
}

// singleQuoteKeepsEscape reports whether a double-quoted string body has a
// backslash that escapes none of \, " and $: PHP-CS-Fixer's
// /(?<!\\)(?:\\{2})*\\(?!["$\\])/. A backslash ending the body escapes the
// closing quote of an unterminated literal, which PHP never makes a
// T_CONSTANT_ENCAPSED_STRING.
func singleQuoteKeepsEscape(content string) bool {
	for i := 0; i < len(content); i++ {
		if content[i] != '\\' {
			continue
		}
		i++
		if i == len(content) || strings.IndexByte(`\"$`, content[i]) < 0 {
			return true
		}
	}
	return false
}
