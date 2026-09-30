package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLineSymfonyAttributeParamFixer.php
//
// StandaloneLineSymfonyAttributeParam puts each argument of an allow-listed
// Symfony attribute on its own line. #[AsCommand] always breaks; every other
// listed attribute only when it has 2 or more arguments. The lexer keeps an
// attribute as a single comment token, so the reformat happens on that string.
type StandaloneLineSymfonyAttributeParam struct{}

const alwaysBreakAttributeShortName = "AsCommand"

var symfonyAttributeShortNames = map[string]bool{
	"AsCommand":         true,
	"Route":             true,
	"Autowire":          true,
	"AutowireIterator":  true,
	"AutowireLocator":   true,
	"AsAlias":           true,
	"AsDecorator":       true,
	"AsTaggedItem":      true,
	"When":              true,
	"AsEventListener":   true,
	"AsMessageHandler":  true,
	"AsController":      true,
	"MapRequestPayload": true,
	"MapQueryParameter": true,
	"MapQueryString":    true,
	"MapEntity":         true,
	"IsGranted":         true,
}

func (StandaloneLineSymfonyAttributeParam) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer`
}

func (StandaloneLineSymfonyAttributeParam) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLineSymfonyAttributeParamFixer.php"
}

func (StandaloneLineSymfonyAttributeParam) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "#[") {
			continue
		}
		reflowed, ok := reflowSymfonyAttribute(t.Value, lineIndentBefore(s, i))
		if ok && reflowed != t.Value {
			s.SetValue(i, reflowed)
			changed = true
		}
	}
	return changed
}

// reflowSymfonyAttribute rewrites the attribute string so each top-level
// argument sits on its own line. base is the indentation of the attribute line.
func reflowSymfonyAttribute(attr, base string) (string, bool) {
	open := strings.IndexByte(attr, '(')
	if open < 0 {
		return attr, false
	}
	shortName := attributeShortName(attr, open)
	if !symfonyAttributeShortNames[shortName] {
		return attr, false
	}
	closeIdx := matchParen(attr, open)
	if closeIdx < 0 {
		return attr, false
	}
	inner := attr[open+1 : closeIdx]
	args, trailingComma := splitTopLevelArgs(inner)
	if len(args) == 0 {
		return attr, false // empty argument list
	}
	if shortName != alwaysBreakAttributeShortName && len(args) < 2 {
		return attr, false
	}

	indent := base + "    "
	var b strings.Builder
	b.WriteString(attr[:open+1])
	for idx, arg := range args {
		b.WriteString("\n")
		b.WriteString(indent)
		b.WriteString(arg)
		// the fixer only newlines existing commas; a trailing comma is kept only
		// when the source had one, never added, to mirror the PHP fixer
		if idx < len(args)-1 || trailingComma {
			b.WriteString(",")
		}
	}
	b.WriteString("\n")
	b.WriteString(base)
	b.WriteString(attr[closeIdx:])
	return b.String(), true
}

// attributeShortName returns the identifier immediately before the "(" at open.
func attributeShortName(attr string, open int) string {
	end := open
	start := end
	for start > 0 {
		c := attr[start-1]
		if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			start--
			continue
		}
		break
	}
	return attr[start:end]
}

// matchParen returns the index of the ")" matching the "(" at open, or -1.
// Heredoc/nowdoc bodies are opaque, so brackets inside them do not shift depth.
func matchParen(attr string, open int) int {
	depth := 0
	for j := open; j < len(attr); j++ {
		if attr[j] == '<' && strings.HasPrefix(attr[j:], "<<<") {
			if end := heredocEnd(attr, j); end > j {
				j = end - 1
				continue
			}
		}
		switch attr[j] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// heredocEnd returns the index just past the closing label of the heredoc or
// nowdoc that starts at s[start:] (which must begin with "<<<"), or -1 if the
// document is malformed. The body between the opening line and the closing
// label is opaque: its commas, quotes and brackets are never PHP tokens.
func heredocEnd(s string, start int) int {
	i := start + 3
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	quote := byte(0)
	if i < len(s) && (s[i] == '\'' || s[i] == '"') {
		quote = s[i]
		i++
	}
	labelStart := i
	for i < len(s) && isLabelChar(s[i]) {
		i++
	}
	label := s[labelStart:i]
	if label == "" {
		return -1
	}
	if quote != 0 {
		if i >= len(s) || s[i] != quote {
			return -1
		}
		i++
	}

	for {
		nl := strings.IndexByte(s[i:], '\n')
		if nl < 0 {
			return -1
		}
		lineStart := i + nl + 1
		j := lineStart
		for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
			j++
		}
		if strings.HasPrefix(s[j:], label) {
			after := j + len(label)
			if after >= len(s) || !isLabelChar(s[after]) {
				return after
			}
		}
		i = lineStart
	}
}

func isLabelChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// splitTopLevelArgs splits the argument list on top-level commas, trimming each
// segment. The second return value reports whether the list ended with a
// top-level trailing comma, so the caller can preserve it.
func splitTopLevelArgs(inner string) ([]string, bool) {
	var args []string
	depth := 0
	start := 0
	inString := byte(0)
	for j := 0; j < len(inner); j++ {
		c := inner[j]
		if inString != 0 {
			switch c {
			case '\\':
				j++
			case inString:
				inString = 0
			}
			continue
		}
		if c == '<' && strings.HasPrefix(inner[j:], "<<<") {
			if end := heredocEnd(inner, j); end > j {
				j = end - 1
				continue
			}
		}
		switch c {
		case '\'', '"':
			inString = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(inner[start:j]))
				start = j + 1
			}
		}
	}
	trailingComma := strings.TrimSpace(inner[start:]) == "" && len(args) > 0
	if last := strings.TrimSpace(inner[start:]); last != "" {
		args = append(args, last)
	}
	return args, trailingComma
}
