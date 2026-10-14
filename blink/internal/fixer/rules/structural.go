package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
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
