package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasLanguageConstructCallFixer.php
//
// NoAliasLanguageConstructCall replaces the "die" alias with the master language
// construct "exit". Only the bare construct is touched: a member call ($x->die())
// or a namespaced/name position is left alone.
type NoAliasLanguageConstructCall struct{}

func (NoAliasLanguageConstructCall) Name() string {
	return `PhpCsFixer\Fixer\Alias\NoAliasLanguageConstructCallFixer`
}

func (NoAliasLanguageConstructCall) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasLanguageConstructCallFixer.php"
}

func (NoAliasLanguageConstructCall) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		// "die"/"exit" are language constructs; this lexer tokenizes them as Ident
		if t.Kind != token.Ident || strings.ToLower(t.Value) != "die" {
			continue
		}
		// a member name ($x->die, Foo::die) or a namespaced/declared name is not
		// the language construct
		if memberPrev(s, i) {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok {
			if prev.Kind == token.Punct && prev.Value == `\` {
				continue
			}
			if prev.Kind == token.Keyword {
				switch strings.ToLower(prev.Value) {
				case "function", "const", "class", "namespace", "use":
					continue
				}
			}
		}
		s.SetValue(i, "exit")
		changed = true
	}
	return changed
}
