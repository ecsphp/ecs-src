package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/BlankLinesBeforeNamespaceFixer.php
//
// BlankLinesBeforeNamespace keeps the line breaks before a namespace declaration
// within the configured bounds (both default to 2, i.e. exactly one blank line).
type BlankLinesBeforeNamespace struct {
	// minLineBreaks / maxLineBreaks bound the line breaks before namespace; 0
	// means the default of 2.
	minLineBreaks int
	maxLineBreaks int
}

func (BlankLinesBeforeNamespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\BlankLinesBeforeNamespaceFixer`
}

func (BlankLinesBeforeNamespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/BlankLinesBeforeNamespaceFixer.php"
}

func (f BlankLinesBeforeNamespace) WithConfig(config map[string]any) fixer.Fixer {
	f.minLineBreaks = configInt(config, "min_line_breaks", 2)
	f.maxLineBreaks = configInt(config, "max_line_breaks", 2)
	return f
}

func (f BlankLinesBeforeNamespace) lineBreaks() (int, int) {
	minBreaks, maxBreaks := f.minLineBreaks, f.maxLineBreaks
	if minBreaks == 0 {
		minBreaks = 2
	}
	if maxBreaks == 0 {
		maxBreaks = 2
	}
	return minBreaks, maxBreaks
}

func (f BlankLinesBeforeNamespace) Fix(s *tokens.Stream) bool {
	minBreaks, maxBreaks := f.lineBreaks()
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "namespace" || memberPrev(s, i) {
			continue
		}
		if i == 0 {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind != token.Whitespace || !hasNewline(prev.Value) {
			continue
		}
		target := clampLineBreaks(strings.Count(prev.Value, "\n"), minBreaks, maxBreaks)
		indent := prev.Value
		if idx := strings.LastIndexByte(prev.Value, '\n'); idx >= 0 {
			indent = prev.Value[idx+1:]
		}
		want := strings.Repeat("\n", target) + indent
		if prev.Value != want {
			s.SetValue(i-1, want)
			changed = true
		}
	}
	return changed
}

func clampLineBreaks(n, minBreaks, maxBreaks int) int {
	if n < minBreaks {
		return minBreaks
	}
	if n > maxBreaks {
		return maxBreaks
	}
	return n
}

// configInt reads an int option (a JSON number arrives as float64), falling back
// to def when absent.
func configInt(config map[string]any, key string, def int) int {
	switch n := config[key].(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return def
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/BlankLineAfterNamespaceFixer.php
//
// BlankLineAfterNamespace ensures one blank line after a "namespace X;"
// declaration. Bracketed namespaces ("namespace X { }") are left alone.
type BlankLineAfterNamespace struct{}

func (BlankLineAfterNamespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\BlankLineAfterNamespaceFixer`
}

func (BlankLineAfterNamespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/BlankLineAfterNamespaceFixer.php"
}

func (BlankLineAfterNamespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "namespace" || memberPrev(s, i) {
			continue
		}
		semi := -1
		for j := i + 1; j < s.Len(); j++ {
			if s.At(j).Kind != token.Punct {
				continue
			}
			if s.At(j).Value == "{" {
				break // bracketed namespace
			}
			if s.At(j).Value == ";" {
				semi = j
				break
			}
		}
		if semi < 0 || semi+1 >= s.Len() {
			continue
		}
		if nextSignificantValue(s, semi) == "" {
			continue // namespace is the last statement; nothing to separate
		}
		if s.At(semi+1).Kind == token.Whitespace {
			if hasNewline(s.At(semi+1).Value) && s.At(semi+1).Value != "\n\n" {
				s.SetValue(semi+1, "\n\n")
				changed = true
			}
		} else {
			s.InsertAt(semi+1, token.Token{Kind: token.Whitespace, Value: "\n\n"})
			changed = true
		}
	}
	return changed
}

var classLikeKeywords = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoBlankLinesAfterClassOpeningFixer.php
//
// NoBlankLinesAfterClassOpening removes blank lines right after a class,
// interface, trait or enum opening brace.
type NoBlankLinesAfterClassOpening struct{}

func (NoBlankLinesAfterClassOpening) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\NoBlankLinesAfterClassOpeningFixer`
}

func (NoBlankLinesAfterClassOpening) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoBlankLinesAfterClassOpeningFixer.php"
}

func (NoBlankLinesAfterClassOpening) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !classLikeKeywords[strings.ToLower(t.Value)] || memberPrev(s, i) {
			continue
		}
		brace := -1
		for j := i + 1; j < s.Len(); j++ {
			if s.At(j).Kind != token.Punct {
				continue
			}
			if s.At(j).Value == "{" {
				brace = j
				break
			}
			if s.At(j).Value == ";" {
				break // e.g. no body
			}
		}
		if brace < 0 || brace+1 >= s.Len() {
			continue
		}
		ws := s.At(brace + 1)
		if ws.Kind != token.Whitespace || strings.Count(ws.Value, "\n") <= 1 {
			continue
		}
		idx := strings.LastIndexByte(ws.Value, '\n')
		want := "\n" + ws.Value[idx+1:]
		if ws.Value != want {
			s.SetValue(brace+1, want)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/IndentationTypeFixer.php
//
// IndentationType converts leading indentation tabs to four spaces.
type IndentationType struct{}

func (IndentationType) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\IndentationTypeFixer`
}

func (IndentationType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/IndentationTypeFixer.php"
}

func (IndentationType) Fix(s *tokens.Stream) bool {
	if s.Indent() == "\t" {
		return indentWithTabs(s)
	}
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if !strings.Contains(t.Value, "\t") {
			continue
		}
		switch t.Kind {
		case token.Whitespace:
			// segments after a newline are line indentation; the first segment is
			// trailing whitespace of the previous line and is left alone
			parts := strings.Split(t.Value, "\n")
			touched := false
			for k := 1; k < len(parts); k++ {
				if strings.Contains(parts[k], "\t") {
					parts[k] = strings.ReplaceAll(parts[k], "\t", "    ")
					touched = true
				}
			}
			if touched {
				s.SetValue(i, strings.Join(parts, "\n"))
				changed = true
			}
		case token.Comment, token.DocComment:
			if v := reindentCommentTabs(t.Value); v != t.Value {
				s.SetValue(i, v)
				changed = true
			}
		}
	}
	return changed
}

// reindentCommentTabs converts leading-indentation tabs to four spaces on each
// continuation line of a multi-line comment or docblock, matching ECS.
func reindentCommentTabs(v string) string {
	lines := strings.Split(v, "\n")
	for k := 1; k < len(lines); k++ {
		line := lines[k]
		j := 0
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		if strings.Contains(line[:j], "\t") {
			lines[k] = strings.ReplaceAll(line[:j], "\t", "    ") + line[j:]
		}
	}
	return strings.Join(lines, "\n")
}

// indentWithTabs ports IndentationTypeFixer for a tab indent: tabs in an indent
// are normalized to four spaces, then every four spaces become one tab.
func indentWithTabs(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		var v string
		switch t.Kind {
		case token.Whitespace:
			v = tabIndentWhitespace(t.Value, i > 0 && strings.Contains(s.At(i-1).Value, "\n"))
		case token.Comment, token.DocComment:
			v = tabIndentComment(t.Value)
		default:
			continue
		}
		if v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// tabIndentWhitespace rewrites the indent after each line break; the first
// segment is an indent too when the previous token ended the line.
func tabIndentWhitespace(v string, afterLineBreak bool) string {
	parts := strings.Split(v, "\n")
	start := 1
	if afterLineBreak {
		start = 0
	}
	for k := start; k < len(parts); k++ {
		part := parts[k]
		j := 0
		for j < len(part) && (part[j] == ' ' || part[j] == '\t') {
			j++
		}
		if j > 0 {
			parts[k] = strings.ReplaceAll(normalizeIndentTabs(part[:j]), "    ", "\t") + part[j:]
		}
	}
	return strings.Join(parts, "\n")
}

// normalizeIndentTabs is Preg::replace('/(?:(?<! ) {1,3})?\t/', '    ', $indent):
// a tab, with up to three spaces before it that follow no other space, becomes
// four spaces.
func normalizeIndentTabs(indent string) string {
	var b strings.Builder
	for p := 0; p < len(indent); {
		if p == 0 || indent[p-1] != ' ' {
			n := 0
			for n < 3 && p+n < len(indent) && indent[p+n] == ' ' {
				n++
			}
			if n > 0 && p+n < len(indent) && indent[p+n] == '\t' {
				b.WriteString("    ")
				p += n + 1
				continue
			}
		}
		if indent[p] == '\t' {
			b.WriteString("    ")
		} else {
			b.WriteByte(indent[p])
		}
		p++
	}
	return b.String()
}

// tabIndentComment ports fixIndentInComment: tabs in each line's leading
// whitespace become four spaces, then each leading run of four spaces a tab.
func tabIndentComment(v string) string {
	lines := strings.Split(v, "\n")
	for k, line := range lines {
		// '/^(?:(?<! ) {1,3})?\t/m' until nothing matches
		for {
			n := 0
			for n < 3 && n < len(line) && line[n] == ' ' {
				n++
			}
			if n < len(line) && line[n] == '\t' {
				line = "    " + line[n+1:]
				continue
			}
			if len(line) > 0 && line[0] == '\t' {
				line = "    " + line[1:]
				continue
			}
			break
		}
		// '/^(\ +)?\t/m' until nothing matches
		j := 0
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		line = strings.ReplaceAll(line[:j], "\t", "    ") + line[j:]
		// '/^(?:    )+/m' to tabs
		j = 0
		for j+4 <= len(line) && line[j:j+4] == "    " {
			j += 4
		}
		lines[k] = strings.Repeat("\t", j/4) + line[j:]
	}
	return strings.Join(lines, "\n")
}
