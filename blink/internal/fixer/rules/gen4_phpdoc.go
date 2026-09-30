package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocLineSpanFixer.php
//
// PhpdocLineSpan expands a single-line docblock that documents a class member
// into a multi-line one, matching the fixer's default config (const, method and
// property all default to "multi").
//
// When configured, each element kind (const, property, method, class, function,
// case, trait_import, other) can be forced to "single", "multi" or left untouched
// (null). The zero value keeps blink's built-in behavior: only member docblocks
// (preceded by a visibility keyword or "var") are expanded single -> multi.
type PhpdocLineSpan struct {
	configured bool
	settings   map[string]string // element kind -> "single" | "multi" | "" (null)
}

func (PhpdocLineSpan) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocLineSpanFixer`
}

func (PhpdocLineSpan) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocLineSpanFixer.php"
}

var phpdocLineSpanKeys = []string{"class", "trait_import", "const", "property", "method", "case", "function", "other"}

func (f PhpdocLineSpan) WithConfig(config map[string]any) fixer.Fixer {
	f.configured = true
	f.settings = map[string]string{
		"class": "multi", "const": "multi", "property": "multi",
		"method": "multi", "case": "multi", "function": "multi",
		"trait_import": "", "other": "",
	}
	for _, key := range phpdocLineSpanKeys {
		if v, present := config[key]; present {
			f.settings[key] = phpdocLineSpanValue(v)
		}
	}
	return f
}

// phpdocLineSpanValue maps a config value to "single", "multi" or "" (null).
func phpdocLineSpanValue(v any) string {
	if s, ok := v.(string); ok && (s == "single" || s == "multi") {
		return s
	}
	return ""
}

func (f PhpdocLineSpan) Fix(s *tokens.Stream) bool {
	if !f.configured {
		return phpdocLineSpanFixDefault(s)
	}
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		kind := phpdocLineSpanElementKind(s, i)
		if kind == "" || insideFunctionBody(s, i) {
			// docblocks inside a function/method body are never class members
			// (PHP's getClassyElements covers class-body elements only); a local
			// `static $x` is not a property, so it falls through to "other"
			kind = "other"
		}
		setting := f.settings[kind]
		if setting == "" {
			continue
		}
		d, ok := parseDoc(t.Value)
		if !ok {
			continue
		}
		indent, _ := docblockLineIndent(s, i)
		if setting == "multi" {
			if d.single {
				expandDocblock(&d, indent)
				s.SetValue(i, d.render())
				changed = true
			}
		} else if phpdocLineSpanMakeSingle(&d) {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

// phpdocLineSpanFixDefault is blink's built-in behavior: expand a single-line
// member docblock into a multi-line one.
func phpdocLineSpanFixDefault(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(t.Value)
		if !ok || !d.single {
			continue
		}
		if !documentsMember(s, i) {
			continue
		}
		indent, ok := docblockLineIndent(s, i)
		if !ok {
			continue
		}
		expandDocblock(&d, indent)
		s.SetValue(i, d.render())
		changed = true
	}
	return changed
}

// phpdocLineSpanElementKind classifies the code element a docblock documents by
// scanning forward past whitespace, folded attributes and modifier keywords. It
// returns one of the config keys, or "" when nothing recognizable follows. A
// method without a visibility modifier cannot be told from a free function on a
// flat token stream and is reported as "function".
func phpdocLineSpanElementKind(s *tokens.Stream, i int) string {
	j := skipWhitespace(s, i+1)
	sawModifier := false
	for j < s.Len() {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment:
			j++
			continue
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private", "final", "abstract", "static", "readonly":
				sawModifier = true
				j++
				continue
			case "var":
				return "property"
			case "function", "fn":
				if sawModifier {
					return "method"
				}
				return "function"
			case "const":
				return "const"
			case "case":
				return "case"
			case "use":
				return "trait_import"
			case "class", "interface", "trait", "enum":
				return "class"
			default:
				j++
				continue
			}
		case token.Variable:
			// a property always carries a modifier (public/var/static/...); a bare
			// $var after a docblock is an inline @var on a local variable, which the
			// real fixer (class members only) leaves alone
			if sawModifier {
				return "property"
			}
			return ""
		case token.Ident:
			j++
			continue
		case token.Punct:
			if t.Value == "?" || t.Value == "|" || t.Value == "&" || t.Value == `\` {
				j++
				continue
			}
			return ""
		default:
			return ""
		}
	}
	return ""
}

// phpdocLineSpanMakeSingle collapses a multi-line docblock with a single content
// line into a single-line one. It is a no-op for already-single blocks or blocks
// carrying more than one content line (which cannot be represented on one line).
func phpdocLineSpanMakeSingle(d *docblock) bool {
	if d.single {
		return false
	}
	var content string
	found := 0
	for _, l := range d.inner {
		if c := strings.TrimSpace(l.content); c != "" {
			content = c
			found++
		}
	}
	if found != 1 {
		return false
	}
	d.single = true
	d.open = "/**"
	d.inner = []docLine{{content: content}}
	d.close = "*/"
	return true
}

// documentsMember reports whether the token after the docblock at index i is a
// visibility keyword (public/private/protected) or "var", which unambiguously
// introduce a class member (property or method). Only whitespace is skipped, so
// an intervening attribute is a no-op. Bare const/function without visibility is
// intentionally excluded - on flat tokens it cannot be told from a top-level
// const or a free function, which the real fixer (class-scope only) leaves alone.
func documentsMember(s *tokens.Stream, i int) bool {
	j := skipWhitespace(s, i+1)
	if j >= s.Len() {
		return false
	}
	next := s.At(j)
	if next.Kind != token.Keyword {
		return false
	}
	lw := strings.ToLower(next.Value)
	return visibilityModifiers[lw] || lw == "var"
}

// insideFunctionBody reports whether the token at index i sits inside the "{...}"
// body of a function or method. It scans from the start tracking brace nesting
// and marks the first "{" after a "function" keyword as a body brace. PHP's
// PhpdocLineSpanFixer treats class members via getClassyElements (class-body
// scope only), so a docblock inside a method body - e.g. on a local `static $x`
// - is never a property/method/const and must be handled as "other".
func insideFunctionBody(s *tokens.Stream, i int) bool {
	depth := 0
	pendingBody := false
	funcDepths := []int{}
	for j := range i {
		t := s.At(j)
		switch {
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "function"):
			pendingBody = true
		case t.Kind == token.Punct && t.Value == ";":
			pendingBody = false
		case t.Kind == token.Punct && t.Value == "{":
			depth++
			if pendingBody {
				funcDepths = append(funcDepths, depth)
				pendingBody = false
			}
		case t.Kind == token.Punct && t.Value == "}":
			if n := len(funcDepths); n > 0 && funcDepths[n-1] == depth {
				funcDepths = funcDepths[:n-1]
			}
			depth--
		}
	}
	return len(funcDepths) > 0
}

// expandDocblock rewrites a single-line docblock into a multi-line one, keeping
// the content verbatim and aligning the "*" one column past the given indent.
func expandDocblock(d *docblock, indent string) {
	content := ""
	if len(d.inner) > 0 {
		content = d.inner[0].content
	}
	d.single = false
	d.open = "/**"
	d.inner = []docLine{{prefix: indent + " * ", content: content}}
	d.close = indent + " */"
}
