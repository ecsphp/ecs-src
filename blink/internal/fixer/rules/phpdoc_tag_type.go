package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagTypeFixer.php
//
// PhpdocTagType writes the configured tags either as annotations ("@tag") or
// inline ("{@tag}"). An inline tag surrounded by text stays inline.
type PhpdocTagType struct {
	// lowercased tag name to "annotation" or "inline"; nil means the default
	tags map[string]string
}

// phpdocTagTypeDefault is the PHP fixer's default `tags`, keys lowercased by
// its normalizer.
var phpdocTagTypeDefault = func() map[string]string {
	m := map[string]string{}
	for _, tag := range []string{
		"api", "author", "copyright", "deprecated", "example", "global", "inheritdoc",
		"internal", "license", "method", "package", "param", "property", "return",
		"see", "since", "throws", "todo", "uses", "var", "version",
	} {
		m[tag] = "annotation"
	}
	return m
}()

func (PhpdocTagType) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTagTypeFixer`
}

func (PhpdocTagType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagTypeFixer.php"
}

// WithConfig keeps the default for a `tags` value PHP rejects. An empty PHP
// array is dumped as the JSON list `[]`, and turns the fixer off.
func (f PhpdocTagType) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := config["tags"].([]any); ok && len(list) == 0 {
		f.tags = map[string]string{}
		return f
	}
	raw, ok := config["tags"].(map[string]any)
	if !ok {
		return f
	}
	tags := map[string]string{}
	for tag, v := range raw {
		typ, _ := v.(string)
		if typ != "annotation" && typ != "inline" {
			return f
		}
		tags[strings.ToLower(tag)] = typ
	}
	f.tags = tags
	return f
}

func (f PhpdocTagType) Fix(s *tokens.Stream) bool {
	tags := f.tags
	if tags == nil {
		tags = phpdocTagTypeDefault
	}
	if len(tags) == 0 {
		return false
	}
	names := make([]string, 0, len(tags))
	for tag := range tags {
		names = append(names, tag)
	}
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		parts := phpdocTagTypeSplit(t.Value, names)
		for k := 1; k < len(parts)-1; k += 2 {
			m := phpdocTagTypeTagRe.FindStringSubmatch(parts[k])
			if m == nil {
				continue
			}
			tag, tagName := m[1], m[2]
			if tag == "" {
				tag, tagName = m[3], m[4]
			}
			typ, ok := tags[strings.ToLower(tagName)]
			if !ok {
				continue
			}
			if typ == "inline" {
				parts[k] = "{" + tag + "}"
				continue
			}
			if !phpdocTagIsSurroundedByText(parts, k) {
				parts[k] = tag
			}
		}
		if content := strings.Join(parts, ""); content != t.Value {
			s.SetValue(i, content)
			changed = true
		}
	}
	return changed
}

// phpdocTagTypeTagRe is the PHP fixer's TAG_REGEX: an annotation "@name rest"
// or an inline tag "{@name rest}" spanning the whole part.
var phpdocTagTypeTagRe = regexp.MustCompile(`^(?:(@(.+?)(?:[` + pcreSpaceClass + `].+)?)|\{(@(.+?)(?:[` + pcreSpaceClass + `].+)?)\})$`)

// phpdocTagTypeSplit ports Preg::split('/({?@(?:TAGS).*?(?:(?=\s\*\/)|(?=\n)}?))/i',
// ..., PREG_SPLIT_DELIM_CAPTURE): text and captured tags alternate, starting
// and ending with text. A tag runs from "@name" (or "{@name") to the end of its
// line or to the whitespace before the closing "*/".
func phpdocTagTypeSplit(content string, names []string) []string {
	var parts []string
	last := 0
	for i := 0; i < len(content); i++ {
		at := i
		if content[i] == '{' && i+1 < len(content) && content[i+1] == '@' {
			at = i + 1
		} else if content[i] != '@' {
			continue
		}
		nameEnd := -1
		for _, name := range names {
			if end := at + 1 + len(name); end <= len(content) && strings.EqualFold(content[at+1:end], name) {
				nameEnd = end
				break
			}
		}
		if nameEnd < 0 {
			continue
		}
		end := -1
		for k := nameEnd; k < len(content); k++ {
			if r, n := pcreRuneAt(content, k); r == '\n' || (pcreIsSpace(r) && strings.HasPrefix(content[k+n:], "*/")) {
				end = k
				break
			}
		}
		if end < 0 {
			// no line end or closer follows: no later tag can match either
			break
		}
		parts = append(parts, content[last:i], content[i:end])
		last = end
		i = end - 1
	}
	return append(parts, content[last:])
}

var (
	phpdocTagTypeNewline = `(?:\r\n|[` + pcreVSpaceClass + `])`
	phpdocTagTypeHSpace  = `[` + pcreHSpaceClass + `]`
	// cleanComment's /(\R)(\h*\*)?\h*/
	phpdocTagTypeCleanRe = regexp.MustCompile(`(` + phpdocTagTypeNewline + `)(?:` + phpdocTagTypeHSpace + `*\*)?` + phpdocTagTypeHSpace + `*`)
	// /(^|\R)\h*[^@\s]\N*/
	phpdocTagTypeTextBeforeRe = regexp.MustCompile(`(?:^|` + phpdocTagTypeNewline + `)` + phpdocTagTypeHSpace + `*[^@` + pcreSpaceClass + `]`)
	// /^.*?\R\s*[^@\s]/
	phpdocTagTypeTextAfterRe = regexp.MustCompile(`^.*?` + phpdocTagTypeNewline + `[` + pcreSpaceClass + `]*[^@` + pcreSpaceClass + `]`)
)

// phpdocTagIsSurroundedByText ports tagIsSurroundedByText: text before the
// tag on any line of the preceding part, or text on a line after the tag.
func phpdocTagIsSurroundedByText(parts []string, k int) bool {
	return phpdocTagTypeTextBeforeRe.MatchString(phpdocTagTypeCleanComment(parts[k-1])) ||
		phpdocTagTypeTextAfterRe.MatchString(phpdocTagTypeCleanComment(parts[k+1]))
}

// phpdocTagTypeCleanComment ports cleanComment: drop the docblock opener and
// closer, then the leading " * " of every line.
func phpdocTagTypeCleanComment(comment string) string {
	comment = strings.TrimPrefix(comment, "/**")
	switch {
	case strings.HasSuffix(comment, "*/"):
		comment = strings.TrimSuffix(comment, "*/")
	case strings.HasSuffix(comment, "*/\n"):
		comment = strings.TrimSuffix(comment, "*/\n") + "\n"
	}
	return phpdocTagTypeCleanRe.ReplaceAllString(comment, "$1")
}
