package rules

import (
	"regexp"
	"sort"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoEmptyPhpdocFixer.php
//
// NoEmptyPhpdoc removes a docblock whose body is empty.
type NoEmptyPhpdoc struct{}

func (NoEmptyPhpdoc) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\NoEmptyPhpdocFixer`
}

func (NoEmptyPhpdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoEmptyPhpdocFixer.php"
}

func (NoEmptyPhpdoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		if !docblockIsEmpty(t.Value) {
			continue
		}
		s.RemoveAt(i)
		changed = true
		// Drop one newline of the following whitespace, like removing a blank line.
		if i < s.Len() && s.At(i).Kind == token.Whitespace && strings.HasPrefix(s.At(i).Value, "\n") {
			if v := s.At(i).Value[1:]; v == "" {
				s.RemoveAt(i)
			} else {
				s.SetValue(i, v)
			}
		}
		i--
	}
	return changed
}

// docblockIsEmpty reports whether a docblock has no textual content, ignoring
// the delimiters, "*" line markers and whitespace.
func docblockIsEmpty(v string) bool {
	if !strings.HasPrefix(v, "/**") || !strings.HasSuffix(v, "*/") || len(v) < 5 {
		return false
	}
	inner := strings.Map(func(r rune) rune {
		switch r {
		case '*', ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, v[3:len(v)-2])
	return inner == ""
}

var phpdocTypeKeywords = map[string]bool{
	"array": true, "bool": true, "callable": true, "false": true, "float": true,
	"int": true, "iterable": true, "mixed": true, "null": true, "object": true,
	"parent": true, "self": true, "static": true, "string": true, "true": true,
	"void": true, "never": true, "$this": true, "scalar": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesFixer.php
//
// PhpdocTypes lowercases known phpdoc type keywords to their canonical form.
// Options `groups` (default ["simple","alias","meta"]) and `exclude` (default [])
// select the type set; a nil set keeps blink's built-in keyword list.
type PhpdocTypes struct {
	typeSet map[string]bool
}

func (PhpdocTypes) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTypesFixer`
}

func (PhpdocTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesFixer.php"
}

// phpdocGenericTagRe matches type-carrying generic tags (@implements X<Y>, ...).
var phpdocGenericTagRe = regexp.MustCompile(`(?i)^(@(?:implements|extends|use|template-extends|template-implements)\s+)(\S+)(.*)$`)

// phpdocTypesPossible are the type groups from PhpdocTypesFixer, used to build the
// keyword set when the fixer is configured.
var phpdocTypesPossible = map[string][]string{
	"alias":  {"boolean", "double", "integer"},
	"meta":   {"$this", "false", "mixed", "parent", "resource", "scalar", "self", "static", "true", "void"},
	"simple": {"array", "bool", "callable", "float", "int", "iterable", "null", "object", "string"},
}

func (f PhpdocTypes) WithConfig(config map[string]any) fixer.Fixer {
	groups, ok := phpdocBatchStringList(config["groups"])
	if !ok {
		groups = []string{"simple", "alias", "meta"}
	}
	set := map[string]bool{}
	for _, g := range groups {
		for _, t := range phpdocTypesPossible[g] {
			set[t] = true
		}
	}
	if exclude, ok := phpdocBatchStringList(config["exclude"]); ok {
		for _, e := range exclude {
			delete(set, e)
		}
	}
	f.typeSet = set
	return f
}

func (f PhpdocTypes) Fix(s *tokens.Stream) bool {
	keywords := f.typeSet
	if keywords == nil {
		keywords = phpdocTypeKeywords
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := phpdocTypeTagRe.FindStringSubmatch(trimmed)
			if m == nil {
				m = phpdocGenericTagRe.FindStringSubmatch(trimmed)
			}
			if m == nil {
				continue
			}
			newType := normalizePhpdocTypeCase(m[2], keywords)
			if newType == m[2] {
				continue
			}
			lead := l.content[:len(l.content)-len(trimmed)]
			d.inner[i].content = lead + m[1] + newType + m[3]
			changed = true
		}
		return changed
	})
}

var phpdocTypeWordRe = regexp.MustCompile(`[$A-Za-z_\\][A-Za-z0-9_\\]*`)

// normalizePhpdocTypeCase lowercases keyword bases anywhere in a phpdoc type -
// unions, array suffixes, nullables and generics (Foo<Scalar> -> Foo<scalar>).
// A word that is namespaced ("\Foo") or a class-constant/member reference
// ("Ref::STATIC") is left as-is.
func normalizePhpdocTypeCase(typ string, keywords map[string]bool) string {
	var b strings.Builder
	prev := 0
	for _, loc := range phpdocTypeWordRe.FindAllStringIndex(typ, -1) {
		st, en := loc[0], loc[1]
		b.WriteString(typ[prev:st])
		w := typ[st:en]
		qualified := strings.Contains(w, `\`) || (st > 0 && (typ[st-1] == ':' || typ[st-1] == '\\'))
		if !qualified {
			if low := strings.ToLower(w); low != w && keywords[low] {
				w = low
			}
		}
		b.WriteString(w)
		prev = en
	}
	b.WriteString(typ[prev:])
	return b.String()
}

// phpdocBatchStringList reads a config value as a list of strings.
func phpdocBatchStringList(v any) ([]string, bool) {
	switch list := v.(type) {
	case []string:
		return append([]string{}, list...), true
	case []any:
		out := make([]string, 0, len(list))
		for _, e := range list {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out, true
	}
	return nil, false
}

var aliasTagRe = regexp.MustCompile(`^@(type|link)\b`)

var aliasTagMap = map[string]string{"type": "var", "link": "see"}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoAliasTagFixer.php
//
// PhpdocNoAliasTag rewrites alias tags: @type -> @var, @link -> @see. Option
// `replacements` overrides the tag mapping; a nil map keeps blink's default.
type PhpdocNoAliasTag struct {
	replacements map[string]string
}

func (PhpdocNoAliasTag) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoAliasTagFixer`
}

func (PhpdocNoAliasTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoAliasTagFixer.php"
}

func (f PhpdocNoAliasTag) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["replacements"].(map[string]any); ok {
		m := make(map[string]string, len(raw))
		for k, v := range raw {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
		f.replacements = m
	}
	return f
}

func (f PhpdocNoAliasTag) Fix(s *tokens.Stream) bool {
	if f.replacements == nil {
		return applyToDocblocks(s, func(d *docblock) bool {
			changed := false
			for i, l := range d.inner {
				trimmed := strings.TrimLeft(l.content, " ")
				m := aliasTagRe.FindStringSubmatch(trimmed)
				if m == nil {
					continue
				}
				lead := l.content[:len(l.content)-len(trimmed)]
				d.inner[i].content = lead + "@" + aliasTagMap[m[1]] + trimmed[len(m[0]):]
				changed = true
			}
			return changed
		})
	}
	if len(f.replacements) == 0 {
		return false
	}
	re := phpdocNoAliasTagRe(f.replacements)
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			lead := l.content[:len(l.content)-len(trimmed)]
			d.inner[i].content = lead + "@" + f.replacements[m[1]] + trimmed[len(m[0]):]
			changed = true
		}
		return changed
	})
}

// phpdocNoAliasTagRe builds a "^@(tag|...)\b" matcher for the configured tags,
// longest first so a longer tag (e.g. property-read) wins over a prefix.
func phpdocNoAliasTagRe(replacements map[string]string) *regexp.Regexp {
	keys := make([]string, 0, len(replacements))
	for k := range replacements {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for i, k := range keys {
		keys[i] = regexp.QuoteMeta(k)
	}
	return regexp.MustCompile(`^@(` + strings.Join(keys, "|") + `)\b`)
}

var noPackageRe = regexp.MustCompile(`^@(?:package|subpackage)\b`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoPackageFixer.php
//
// PhpdocNoPackage removes @package and @subpackage tag lines.
type PhpdocNoPackage struct{}

func (PhpdocNoPackage) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoPackageFixer`
}

func (PhpdocNoPackage) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoPackageFixer.php"
}

func (PhpdocNoPackage) Fix(s *tokens.Stream) bool {
	return removeDocLines(s, noPackageRe)
}

var noAccessRe = regexp.MustCompile(`^@access\b`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoAccessFixer.php
//
// PhpdocNoAccess removes @access tag lines.
type PhpdocNoAccess struct{}

func (PhpdocNoAccess) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoAccessFixer`
}

func (PhpdocNoAccess) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoAccessFixer.php"
}

func (PhpdocNoAccess) Fix(s *tokens.Stream) bool {
	return removeDocLines(s, noAccessRe)
}

// removeDocLines drops every inner line whose tag matches re (checked at the
// start of the content, after any leading spaces).
func removeDocLines(s *tokens.Stream, re *regexp.Regexp) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		kept := d.inner[:0:0]
		removed := false
		for _, l := range d.inner {
			if re.MatchString(strings.TrimLeft(l.content, " ")) {
				removed = true
				continue
			}
			kept = append(kept, l)
		}
		if removed {
			d.inner = kept
		}
		return removed
	})
}

var singleLineVarRe = regexp.MustCompile(`^(@(?:var|type|param))\s+([^\s$]+)\s*(\$\S+)?\s*(.*)$`)

var phpdocWsRe = regexp.MustCompile(`\s+`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSingleLineVarSpacingFixer.php
//
// PhpdocSingleLineVarSpacing normalizes spacing in single-line @var/@type/@param docblocks.
type PhpdocSingleLineVarSpacing struct{}

func (PhpdocSingleLineVarSpacing) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocSingleLineVarSpacingFixer`
}

func (PhpdocSingleLineVarSpacing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSingleLineVarSpacingFixer.php"
}

func (PhpdocSingleLineVarSpacing) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		if !d.single || len(d.inner) == 0 {
			return false
		}
		content := d.inner[0].content
		m := singleLineVarRe.FindStringSubmatch(content)
		if m == nil {
			return false
		}
		out := []string{m[1], m[2]}
		if m[3] != "" {
			out = append(out, m[3])
		}
		normalized := strings.Join(out, " ")
		if rest := strings.TrimSpace(phpdocWsRe.ReplaceAllString(m[4], " ")); rest != "" {
			normalized += " " + rest
		}
		if normalized == content {
			return false
		}
		d.inner[0].content = normalized
		return true
	})
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTrimConsecutiveBlankLineSeparationFixer.php
//
// PhpdocTrimConsecutiveBlankLineSeparation collapses runs of blank inner lines into one.
type PhpdocTrimConsecutiveBlankLineSeparation struct{}

func (PhpdocTrimConsecutiveBlankLineSeparation) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTrimConsecutiveBlankLineSeparationFixer`
}

func (PhpdocTrimConsecutiveBlankLineSeparation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTrimConsecutiveBlankLineSeparationFixer.php"
}

func (PhpdocTrimConsecutiveBlankLineSeparation) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		kept := d.inner[:0:0]
		prevBlank := false
		changed := false
		for _, l := range d.inner {
			blank := strings.TrimSpace(l.content) == ""
			if blank && prevBlank {
				changed = true
				continue
			}
			kept = append(kept, l)
			prevBlank = blank
		}
		if changed {
			d.inner = kept
		}
		return changed
	})
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoBlankLinesAfterPhpdocFixer.php
//
// NoBlankLinesAfterPhpdoc removes blank lines between a docblock and the code it documents.
type NoBlankLinesAfterPhpdoc struct{}

func (NoBlankLinesAfterPhpdoc) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\NoBlankLinesAfterPhpdocFixer`
}

func (NoBlankLinesAfterPhpdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/NoBlankLinesAfterPhpdocFixer.php"
}

func (NoBlankLinesAfterPhpdoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		j := i + 1
		if j >= s.Len() || s.At(j).Kind != token.Whitespace {
			continue
		}
		v := s.At(j).Value
		if strings.Count(v, "\n") < 2 {
			continue
		}
		// a file-level docblock before "declare" keeps its blank line - ECS only
		// trims the blank after a docblock attached to a structural element
		if k := nextSignificantIndex(s, i); k >= 0 &&
			s.At(k).Kind == token.Keyword && strings.EqualFold(s.At(k).Value, "declare") {
			continue
		}
		// Keep the last newline plus the following statement's indentation.
		nv := v[strings.LastIndexByte(v, '\n'):]
		if nv != v {
			s.SetValue(j, nv)
			changed = true
		}
	}
	return changed
}
