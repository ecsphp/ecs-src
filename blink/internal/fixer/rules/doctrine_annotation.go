package rules

import (
	"strings"

	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationSpacesFixer.php
//
// DoctrineAnnotationSpaces normalises whitespace in a Doctrine annotation:
// no space before the "(", none just inside "(" / ")", none around an argument
// "=" and none before a ",", while an array-assignment "=" (inside "{ }") gets a
// single space on each side. String contents are left untouched.
//
// Only single-line annotations whose name starts uppercase or is namespaced
// (e.g. @ORM\Column, @Route, @Assert\NotBlank) are handled - that keeps plain
// phpdoc tags (@param, @return, ...) and multi-line annotations untouched, which
// is a safe under-fire rather than a risky guess.
type DoctrineAnnotationSpaces struct{}

func (DoctrineAnnotationSpaces) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationSpacesFixer`
}

func (DoctrineAnnotationSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationSpacesFixer.php"
}

func (DoctrineAnnotationSpaces) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i := range d.inner {
			fixed, ok := fixDoctrineSpacesLine(d.inner[i].content)
			if ok && fixed != d.inner[i].content {
				d.inner[i].content = fixed
				changed = true
			}
		}
		return changed
	})
}

// fixDoctrineSpacesLine rewrites a single doc line that holds one complete
// Doctrine annotation. It returns ok=false when the line is not such an
// annotation (leaving it untouched).
func fixDoctrineSpacesLine(content string) (string, bool) {
	lead := content[:len(content)-len(strings.TrimLeft(content, " \t"))]
	rest := content[len(lead):]
	if !strings.HasPrefix(rest, "@") {
		return content, false
	}
	// annotation name: @Name or @Ns\Name
	j := 1
	for j < len(rest) && isAnnotationNameByte(rest[j]) {
		j++
	}
	name := rest[1:j]
	if name == "" || !isDoctrineAnnotationName(name) {
		return content, false
	}
	// optional spaces, then the opening "("
	k := j
	for k < len(rest) && (rest[k] == ' ' || rest[k] == '\t') {
		k++
	}
	if k >= len(rest) || rest[k] != '(' {
		return content, false // no argument list - nothing to space
	}
	closeIdx := matchAnnotationParen(rest, k)
	if closeIdx < 0 {
		return content, false // unbalanced - multi-line annotation, skip
	}
	// only trailing whitespace may follow the closing ")"
	if strings.TrimRight(rest[closeIdx+1:], " \t") != "" {
		return content, false
	}
	inner := normalizeDoctrineInner(rest[k+1 : closeIdx])
	rebuilt := lead + "@" + name + "(" + inner + ")" + rest[closeIdx+1:]
	return rebuilt, true
}

func isAnnotationNameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '\\'
}

// isDoctrineAnnotationName reports whether name looks like a Doctrine annotation
// (starts uppercase or is namespaced) rather than a lowercase phpdoc tag.
func isDoctrineAnnotationName(name string) bool {
	if strings.Contains(name, "\\") {
		return true
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// matchAnnotationParen returns the index of the ")" matching the "(" at open,
// honouring double-quoted strings, or -1 when unbalanced within the string.
func matchAnnotationParen(s string, open int) int {
	depth := 0
	inStr := false
	for i := open; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// doctrineToken is a non-space unit of annotation content plus the original
// whitespace that followed it.
type doctrineToken struct {
	text    string
	isPunct bool
	gap     string // whitespace following this token
}

// normalizeDoctrineInner applies the spacing rules to the content between the
// outermost parentheses.
func normalizeDoctrineInner(inner string) string {
	toks := tokenizeDoctrine(inner)
	if len(toks) == 0 {
		return ""
	}
	// brace depth in front of each token (used to tell an array "=" apart from
	// an argument "=")
	depth := make([]int, len(toks))
	d := 0
	for i, t := range toks {
		depth[i] = d
		if t.isPunct && t.text == "{" {
			d++
		} else if t.isPunct && t.text == "}" {
			if d > 0 {
				d--
			}
		}
	}
	var b strings.Builder
	for i, t := range toks {
		b.WriteString(t.text)
		if i == len(toks)-1 {
			break
		}
		next := toks[i+1]
		b.WriteString(gapBetween(t, next, depth[i], depth[i+1]))
	}
	return b.String()
}

// gapBetween decides the whitespace between two adjacent tokens.
func gapBetween(a, next doctrineToken, aDepth, nextDepth int) string {
	// no space just inside parentheses, nor before a comma
	if a.isPunct && a.text == "(" {
		return ""
	}
	if next.isPunct && next.text == ")" {
		return ""
	}
	if next.isPunct && next.text == "," {
		return ""
	}
	// one space after a comma (add when missing; existing spacing is kept)
	if a.isPunct && a.text == "," {
		if a.gap == "" {
			return " "
		}
		return a.gap
	}
	// an "=" is an array assignment when inside "{ }" (one space each side),
	// otherwise an argument assignment (no space)
	if a.isPunct && a.text == "=" {
		if aDepth > 0 {
			return " "
		}
		return ""
	}
	if next.isPunct && next.text == "=" {
		if nextDepth > 0 {
			return " "
		}
		return ""
	}
	return a.gap
}

// tokenizeDoctrine splits annotation content into string-aware tokens, trimming
// the leading and trailing whitespace.
func tokenizeDoctrine(inner string) []doctrineToken {
	var toks []doctrineToken
	i := 0
	n := len(inner)
	for i < n {
		c := inner[i]
		if c == ' ' || c == '\t' {
			// whitespace belongs to the previous token's gap
			start := i
			for i < n && (inner[i] == ' ' || inner[i] == '\t') {
				i++
			}
			if len(toks) > 0 {
				toks[len(toks)-1].gap = inner[start:i]
			}
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < n {
				if inner[i] == '\\' {
					i += 2
					continue
				}
				if inner[i] == '"' {
					i++
					break
				}
				i++
			}
			toks = append(toks, doctrineToken{text: inner[start:i]})
			continue
		}
		if strings.IndexByte("(){},=", c) >= 0 {
			toks = append(toks, doctrineToken{text: string(c), isPunct: true})
			i++
			continue
		}
		// a plain run: identifier, number, ::, etc.
		start := i
		for i < n {
			ch := inner[i]
			if ch == ' ' || ch == '\t' || ch == '"' || strings.IndexByte("(){},=", ch) >= 0 {
				break
			}
			i++
		}
		toks = append(toks, doctrineToken{text: inner[start:i]})
	}
	return toks
}
