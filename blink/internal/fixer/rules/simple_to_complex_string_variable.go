package rules

import (
	"strings"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SimpleToComplexStringVariableFixer.php
//
// SimpleToComplexStringVariable rewrites the "${name}" interpolation of
// double-quoted strings and heredocs to "{$name}" ("${name[...]}" too). An
// expression such as "${$a}" or "${a . b}" is left alone. A literal "$" right
// before it is escaped, so "$${a}" does not become "${$a}".
type SimpleToComplexStringVariable struct{}

func (SimpleToComplexStringVariable) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\SimpleToComplexStringVariableFixer`
}

func (SimpleToComplexStringVariable) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SimpleToComplexStringVariableFixer.php"
}

func (SimpleToComplexStringVariable) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || !strings.Contains(t.Value, "${") {
			continue
		}
		parts := encapsedParts(t.Value)
		repl := map[int]string{}
		for k, p := range parts {
			if p.kind != encDollarOpen || k+1 >= len(parts) || parts[k+1].kind != encVarname {
				continue
			}
			// the literal before: a trailing "$" (not "\$") would join the new "{$"
			if k > 0 {
				prev := parts[k-1]
				content := t.Value[prev.start:prev.end]
				if strings.HasSuffix(content, "$") && !strings.HasSuffix(content, `\$`) {
					repl[k-1] = content[:len(content)-1] + `\$`
				}
			}
			repl[k] = "{"
			next := parts[k+1]
			repl[k+1] = "$" + t.Value[next.start:next.end]
		}
		if len(repl) == 0 {
			continue
		}
		s.SetValue(i, encRebuild(t.Value, parts, repl))
		changed = true
	}
	if changed {
		// the new "{$...}" interpolations are split into chunks by the lexer
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(s.Render()))
	}
	return changed
}
