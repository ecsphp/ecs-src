package rules

import (
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitFqcnAnnotationFixer.php
//
// PhpUnitFqcnAnnotation adds a leading `\` to the class names of the
// @expectedException, @covers, @coversDefaultClass and @uses annotations in the
// docblocks of a PHPUnit test class, from the last docblock before its body.
type PhpUnitFqcnAnnotation struct{}

func (PhpUnitFqcnAnnotation) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitFqcnAnnotationFixer`
}

func (PhpUnitFqcnAnnotation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitFqcnAnnotationFixer.php"
}

func (PhpUnitFqcnAnnotation) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		start := cls[0]
		for j := start - 1; j >= 0; j-- {
			if s.At(j).Kind == token.DocComment {
				start = j
				break
			}
		}
		for i := start; i < cls[1]; i++ {
			if s.At(i).Kind != token.DocComment {
				continue
			}
			if fixed := phpUnitFqcnAnnotationReplace(s.At(i).Value); fixed != s.At(i).Value {
				s.SetValue(i, fixed)
				changed = true
			}
		}
	}
	return changed
}

// phpUnitFqcnAnnotationReplace applies upstream's
//
//	~^(\s*\*\s*@(?:expectedException|covers|coversDefaultClass|uses)\h+)(?!(?:self|static)::)(\w.*)$~m
//
// replaced by '$1\\$2', matched by hand because RE2 has no lookahead. Like
// Preg::replace it runs in UTF-8 mode, falling back to byte mode for invalid
// UTF-8, which only changes what \h accepts.
func phpUnitFqcnAnnotationReplace(content string) string {
	utf := utf8.ValidString(content)
	var b strings.Builder
	last := 0
	for p := 0; p <= len(content); {
		if insert, ok := phpUnitFqcnAnnotationMatchAt(content, p, utf); ok {
			b.WriteString(content[last:insert])
			b.WriteByte('\\')
			last = insert
			p = insert
		}
		nl := strings.IndexByte(content[p:], '\n')
		if nl < 0 {
			break
		}
		p += nl + 1
	}
	if last == 0 {
		return content
	}
	b.WriteString(content[last:])
	return b.String()
}

var phpUnitFqcnAnnotationTags = []string{"expectedException", "covers", "coversDefaultClass", "uses"}

// phpUnitFqcnAnnotationMatchAt returns where $2 starts when the pattern matches
// at line start p.
func phpUnitFqcnAnnotationMatchAt(content string, p int, utf bool) (int, bool) {
	i := p
	for i < len(content) && phpAlignIsWS(content[i]) {
		i++
	}
	if i >= len(content) || content[i] != '*' {
		return 0, false
	}
	i++
	for i < len(content) && phpAlignIsWS(content[i]) {
		i++
	}
	if i >= len(content) || content[i] != '@' {
		return 0, false
	}
	i++
	for _, tag := range phpUnitFqcnAnnotationTags {
		if !strings.HasPrefix(content[i:], tag) {
			continue
		}
		j := i + len(tag)
		k := j
		for k < len(content) {
			n := pcreHorizontalSpaceLen(content[k:], utf)
			if n == 0 {
				break
			}
			k += n
		}
		if k == j || k >= len(content) || !isIdentByte(content[k]) {
			continue
		}
		if strings.HasPrefix(content[k:], "self::") || strings.HasPrefix(content[k:], "static::") {
			continue
		}
		return k, true
	}
	return 0, false
}

// pcreHorizontalSpaceLen returns the byte length of the PCRE \h character at
// the start of v, or 0. In UTF-8 mode \h is the Unicode horizontal spaces; in
// byte mode it is tab, space and the 0xA0 byte.
func pcreHorizontalSpaceLen(v string, utf bool) int {
	if v == "" {
		return 0
	}
	if !utf {
		if c := v[0]; c == ' ' || c == '\t' || c == 0xA0 {
			return 1
		}
		return 0
	}
	r, n := utf8.DecodeRuneInString(v)
	switch {
	case r == ' ', r == '\t', r == 0xA0, r == 0x1680, r == 0x180E,
		r >= 0x2000 && r <= 0x200A, r == 0x202F, r == 0x205F, r == 0x3000:
		return n
	}
	return 0
}
