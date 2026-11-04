package rules

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocTagRenameFixer.php
//
// GeneralPhpdocTagRename renames PHPDoc tags according to the `replacements`
// map (default empty: a no-op).
type GeneralPhpdocTagRename struct {
	// in configuration order: PHP joins the keys into one regex alternation,
	// so the first key that matches wins
	replacements    []phpdocTagReplacement
	noFixAnnotation bool
	noFixInline     bool
	caseSensitive   bool
}

type phpdocTagReplacement struct {
	from *regexp.Regexp
	to   string
}

func (GeneralPhpdocTagRename) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocTagRenameFixer`
}

func (GeneralPhpdocTagRename) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocTagRenameFixer.php"
}

// WithConfig takes the `replacements` keys in sorted order, as a map has
// none; WithRawConfig keeps the dumped order.
func (f GeneralPhpdocTagRename) WithConfig(config map[string]any) fixer.Fixer {
	var keys []string
	if raw, ok := config["replacements"].(map[string]any); ok {
		for from := range raw {
			keys = append(keys, from)
		}
		sort.Strings(keys)
	}
	return f.configure(config, keys)
}

func (f GeneralPhpdocTagRename) WithRawConfig(raw []byte) fixer.Fixer {
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return f
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return f
	}
	keys, err := jsonObjectKeys(fields["replacements"])
	if err != nil {
		return f
	}
	return f.configure(config, keys)
}

// jsonObjectKeys lists the keys of a JSON object in document order (none for
// anything that is not an object).
func jsonObjectKeys(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, err
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// configure takes the `replacements` keys in the given order. A replacement
// map PHP's normalizer rejects (a target that is not one word or holds "*/",
// two targets for one tag, or a chain a -> b -> c) leaves the fixer a no-op.
func (f GeneralPhpdocTagRename) configure(config map[string]any, keys []string) fixer.Fixer {
	if v, ok := config["fix_annotation"].(bool); ok {
		f.noFixAnnotation = !v
	}
	if v, ok := config["fix_inline"].(bool); ok {
		f.noFixInline = !v
	}
	if v, ok := config["case_sensitive"].(bool); ok {
		f.caseSensitive = v
	}
	raw, ok := config["replacements"].(map[string]any)
	if !ok {
		return f
	}
	f.replacements = nil
	// a PHP array: a key assigned again keeps its first position
	var order []string
	normalized := map[string]string{}
	for _, from := range keys {
		to, _ := raw[from].(string)
		// #^\S+$# (a final newline passes) and no "*/"
		if word := strings.TrimSuffix(to, "\n"); word == "" || strings.IndexFunc(word, pcreIsSpace) >= 0 || strings.Contains(to, "*/") {
			return f
		}
		const phpTrim = " \t\n\r\x00\x0B"
		from, to = strings.Trim(from, phpTrim), strings.Trim(to, phpTrim)
		if !f.caseSensitive {
			from = asciiLower(from)
			if prev, dup := normalized[from]; dup && prev != to {
				return f
			}
		}
		if _, dup := normalized[from]; !dup {
			order = append(order, from)
		}
		normalized[from] = to
	}
	for _, from := range order {
		if next, chained := normalized[normalized[from]]; chained && next != normalized[from] {
			return f
		}
	}
	prefix := "^(?:"
	if !f.caseSensitive {
		prefix = "(?i)^(?:"
	}
	for _, key := range order {
		// PHP splices the keys into its regex unquoted
		re, err := regexp.Compile(prefix + key + ")")
		if err != nil {
			re = regexp.MustCompile(prefix + regexp.QuoteMeta(key) + ")")
		}
		f.replacements = append(f.replacements, phpdocTagReplacement{from: re, to: normalized[key]})
	}
	return f
}

func (f GeneralPhpdocTagRename) Fix(s *tokens.Stream) bool {
	if len(f.replacements) == 0 {
		return false
	}
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		if content := f.rename(t.Value); content != t.Value {
			s.SetValue(i, content)
			changed = true
		}
	}
	return changed
}

// rename ports the PHP fixer's Preg::replaceCallback over one docblock. Its
// regex depends on the options:
//
//	annotation+inline: /(["'])[^\1]*\1(*SKIP)(*FAIL)|\b(?<=@)(?P<tag>TAGS)\b/
//	annotation only:   /(["'])[^\1]*\1(*SKIP)(*FAIL)|(?<!\{@)(?<=@)(?P<tag>TAGS)(?!\})/
//	inline only:       /(?<={@)(?P<tag>TAGS)(?=[ \t}])/
//
// The quote branch skips from a quote to the last same quote before any \x01
// ([^\1] is a class excluding \x01, not a backreference).
func (f GeneralPhpdocTagRename) rename(content string) string {
	var b strings.Builder
	last := 0
	for i := 0; i < len(content); {
		if !f.noFixAnnotation && (content[i] == '"' || content[i] == '\'') {
			region := content[i+1:]
			if stop := strings.IndexByte(region, '\x01'); stop >= 0 {
				region = region[:stop]
			}
			if closing := strings.LastIndexByte(region, content[i]); closing >= 0 {
				i += closing + 2
				continue
			}
		}
		if end, to, ok := f.matchAt(content, i); ok {
			b.WriteString(content[last:i])
			b.WriteString(to)
			last, i = end, end
			continue
		}
		i++
	}
	if last == 0 {
		return content
	}
	b.WriteString(content[last:])
	return b.String()
}

// matchAt reports whether a configured tag starts at i under the active
// regex, returning the end of the tag and its replacement.
func (f GeneralPhpdocTagRename) matchAt(content string, i int) (int, string, bool) {
	if i == 0 || content[i-1] != '@' {
		return 0, "", false
	}
	inlineOpen := i >= 2 && content[i-2] == '{'
	switch {
	case f.noFixAnnotation:
		if !inlineOpen {
			return 0, "", false
		}
	case f.noFixInline:
		if inlineOpen {
			return 0, "", false
		}
	case !pcreWordBoundary(content, i):
		// \b before the tag
		return 0, "", false
	}
	for _, r := range f.replacements {
		loc := r.from.FindStringIndex(content[i:])
		if loc == nil || loc[1] == 0 {
			continue
		}
		end := i + loc[1]
		next := byte(0)
		if end < len(content) {
			next = content[end]
		}
		switch {
		case f.noFixAnnotation:
			if end == len(content) || (next != ' ' && next != '\t' && next != '}') {
				continue
			}
		case f.noFixInline:
			if end < len(content) && next == '}' {
				continue
			}
		case !pcreWordBoundary(content, end):
			// no \b after the tag
			continue
		}
		return end, r.to, true
	}
	return 0, "", false
}

// asciiLower is PHP 8's strtolower, which only lowercases ASCII letters.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
