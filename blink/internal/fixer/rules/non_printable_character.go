package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NonPrintableCharacterFixer.php
//
// NonPrintableCharacter removes or replaces a few invisible characters in
// comments, inline HTML, variable names and strings. With
// use_escape_sequences_in_strings (the default), strings get a "\u{...}" escape
// instead, and a single-quoted string or nowdoc becomes double-quoted first.
type NonPrintableCharacter struct {
	// KeepInStrings is !use_escape_sequences_in_strings, so the zero value is the default.
	KeepInStrings bool
}

func (NonPrintableCharacter) Name() string {
	return `PhpCsFixer\Fixer\Basic\NonPrintableCharacterFixer`
}

func (NonPrintableCharacter) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NonPrintableCharacterFixer.php"
}

func (f NonPrintableCharacter) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["use_escape_sequences_in_strings"].(bool); ok {
		f.KeepInStrings = !v
	}
	return f
}

// nonPrintableSymbols are the characters, their plain replacement and their escape code point.
var nonPrintableSymbols = []struct{ char, replacement, codepoint string }{
	{"​", "", "200b"},  // ZERO WIDTH SPACE
	{" ", " ", "2007"}, // FIGURE SPACE
	{" ", " ", "202f"}, // NARROW NO-BREAK SPACE
	{"⁠", "", "2060"},  // WORD JOINER
	{" ", " ", "a0"},   // NO-BREAK SPACE
}

var (
	nonPrintableReplacer = func() *strings.Replacer {
		var pairs []string
		for _, sym := range nonPrintableSymbols {
			pairs = append(pairs, sym.char, sym.replacement)
		}
		return strings.NewReplacer(pairs...)
	}()
	nonPrintableEscaper = func() *strings.Replacer {
		var pairs []string
		for _, sym := range nonPrintableSymbols {
			pairs = append(pairs, sym.char, `\u{`+sym.codepoint+`}`)
		}
		return strings.NewReplacer(pairs...)
	}()
)

func nonPrintableFound(v string) bool {
	for _, sym := range nonPrintableSymbols {
		if strings.Contains(v, sym.char) {
			return true
		}
	}
	return false
}

func (f NonPrintableCharacter) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if !nonPrintableFound(t.Value) {
			continue
		}
		v := t.Value
		switch t.Kind {
		case token.Comment, token.DocComment:
			v = nonPrintableInComment(v)
		case token.InlineHTML:
			v = nonPrintableReplacer.Replace(v)
		case token.Variable:
			v = nonPrintableInVariable(v)
		case token.String:
			v = f.fixString(v)
		}
		if v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// fixString applies the fixer to the PHP tokens inside one String token.
func (f NonPrintableCharacter) fixString(v string) string {
	parts := encapsedParts(v)
	if parts == nil {
		return v
	}
	contents := make([]string, len(parts))
	for k, p := range parts {
		contents[k] = v[p.start:p.end]
	}
	for k, p := range parts {
		content := contents[k]
		if !f.KeepInStrings && (p.kind == encConstString || p.kind == encText) {
			if !nonPrintableFound(content) {
				continue
			}
			stringTypeChanged, swapQuotes := false, false
			if k > 0 && parts[k-1].kind == encStartHeredoc {
				if strings.Contains(contents[k-1], "'") {
					contents[k-1] = strings.ReplaceAll(contents[k-1], "'", "")
					stringTypeChanged = true
				}
			} else if strings.HasPrefix(content, "'") {
				stringTypeChanged, swapQuotes = true, true
			}
			if swapQuotes {
				content = strings.ReplaceAll(content, `\'`, "'")
			}
			if stringTypeChanged {
				content = nonPrintableDoubleBackslashes(content)
				content = strings.ReplaceAll(content, "$", `\$`)
			}
			if swapQuotes {
				content = strings.ReplaceAll(content, `"`, `\"`)
				content = nonPrintableSwapQuotes(content)
			}
			contents[k] = nonPrintableEscaper.Replace(content)
			continue
		}
		switch p.kind {
		case encConstString, encText:
			contents[k] = nonPrintableReplacer.Replace(content)
		case encVariable, encVarname:
			contents[k] = nonPrintableInVariable(content)
		case encComment, encDocComment:
			contents[k] = nonPrintableInComment(content)
		}
	}
	return strings.Join(contents, "")
}

// nonPrintableInVariable replaces the characters in a variable name, unless
// that would put a space in it.
func nonPrintableInVariable(v string) string {
	if r := nonPrintableReplacer.Replace(v); !strings.Contains(r, " ") {
		return r
	}
	return v
}

// nonPrintableInComment replaces the characters in a comment, unless that would
// make a "*/" appear before the end of a block comment.
func nonPrintableInComment(v string) string {
	r := nonPrintableReplacer.Replace(v)
	if strings.HasPrefix(r, "/*") && strings.Index(r, "*/") != len(r)-2 {
		return v
	}
	return r
}

// nonPrintableDoubleBackslashes is Preg::replace('/(\\{1,2})/', '\\\\', $v):
// each run of one or two backslashes becomes two.
func nonPrintableDoubleBackslashes(v string) string {
	if !strings.Contains(v, `\`) {
		return v
	}
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] != '\\' {
			b.WriteByte(v[i])
			continue
		}
		if i+1 < len(v) && v[i+1] == '\\' {
			i++
		}
		b.WriteString(`\\`)
	}
	return b.String()
}

// nonPrintableSwapQuotes is Preg::replace('/^\'(.*)\'$/s', '"$1"', $v): "$"
// also matches before a final newline.
func nonPrintableSwapQuotes(v string) string {
	if !strings.HasPrefix(v, "'") {
		return v
	}
	if len(v) >= 2 && strings.HasSuffix(v, "'") {
		return `"` + v[1:len(v)-1] + `"`
	}
	if len(v) >= 3 && strings.HasSuffix(v, "'\n") {
		return `"` + v[1:len(v)-2] + `"` + "\n"
	}
	return v
}
