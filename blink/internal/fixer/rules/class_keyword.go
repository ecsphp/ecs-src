package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer (experimental, risky): https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordFixer.php
//
// ClassKeyword converts an FQCN string literal into `\Name::class`. PHP-CS-Fixer
// only converts a name whose class is autoloadable at runtime; blink has no
// runtime, so it converts any string that is a valid namespaced class name.
type ClassKeyword struct{}

func (ClassKeyword) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\ClassKeywordFixer`
}

func (ClassKeyword) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordFixer.php"
}

func (ClassKeyword) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.String || !t.IsLiteral() || t.IsShellExec() {
			continue
		}
		v := t.Value
		if len(v) < 2 {
			continue
		}
		q := v[0]
		if (q != '\'' && q != '"') || v[len(v)-1] != q {
			continue
		}
		name := v[1 : len(v)-1]
		name = strings.TrimLeft(name, `\`)
		name = strings.ReplaceAll(name, `\\`, `\`)
		if !ckIsQualifiedName(name) {
			continue
		}
		s.ReplaceRange(i, i, ckClassTokens(name))
		changed = true
	}
	return changed
}

// ckIsQualifiedName reports whether name is a namespaced class name: two or more
// PHP identifier segments separated by single backslashes.
func ckIsQualifiedName(name string) bool {
	if !strings.Contains(name, `\`) {
		return false
	}
	for _, seg := range strings.Split(name, `\`) {
		if !ckIsIdentifier(seg) {
			return false
		}
	}
	return true
}

func ckIsIdentifier(seg string) bool {
	if seg == "" {
		return false
	}
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		ok := c == '_' || c >= 0x80 ||
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(i > 0 && c >= '0' && c <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// ckClassTokens builds the `\Name::class` token run for a qualified name.
func ckClassTokens(name string) []token.Token {
	out := []token.Token{{Kind: token.Punct, Value: `\`}}
	for i, seg := range strings.Split(name, `\`) {
		if i > 0 {
			out = append(out, token.Token{Kind: token.Punct, Value: `\`})
		}
		out = append(out, token.Token{Kind: token.Ident, Value: seg})
	}
	out = append(out, token.Token{Kind: token.Punct, Value: "::"}, token.Token{Kind: token.Ident, Value: "class"})
	return out
}
