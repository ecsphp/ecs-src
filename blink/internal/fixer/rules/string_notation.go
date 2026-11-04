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
// no variables ($), no escape sequences (\), and no single quote in the content.
// WithSingleQuotes (strings_containing_single_quote_chars) also converts strings
// containing single quotes, escaping them.
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
		if !t.IsLiteral() {
			continue
		}
		prefix, v := token.SplitBinaryPrefix(t.Value) // b"x" keeps its prefix
		if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
			continue // not a plain double-quoted string (e.g. heredoc)
		}
		content := v[1 : len(v)-1]
		if strings.ContainsAny(content, "$\\") || (!f.WithSingleQuotes && strings.Contains(content, "'")) {
			continue // interpolation, escape, or a quote that would need escaping
		}
		content = strings.ReplaceAll(content, "'", `\'`)
		s.SetValue(i, prefix+"'"+content+"'")
		changed = true
	}
	return changed
}
