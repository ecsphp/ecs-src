package rules

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAnnotationWithoutDotFixer.php
//
// PhpdocAnnotationWithoutDot drops a trailing "." (or fullwidth "。") from a
// @throws/@return/@param/@internal/@deprecated/@var/@type description when the
// description is a single sentence, and lowercases its first letter.
type PhpdocAnnotationWithoutDot struct{}

var annotationWithoutDotTags = map[string]bool{
	"throws": true, "return": true, "param": true, "internal": true,
	"deprecated": true, "var": true, "type": true,
}

func (PhpdocAnnotationWithoutDot) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocAnnotationWithoutDotFixer`
}

func (PhpdocAnnotationWithoutDot) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAnnotationWithoutDotFixer.php"
}

func (PhpdocAnnotationWithoutDot) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		lines := splitDocLines(t.Value)
		anns := docAnnotations(lines)
		if len(anns) == 0 {
			continue
		}
		docChanged := false
		for _, a := range anns {
			if !annotationWithoutDotTags[a.name] {
				continue
			}
			if a.end+1 < len(lines) {
				lt := strings.TrimLeft(lines[a.end+1], " \t\n\r\x00\x0b")
				if lt == "" || !strings.HasPrefix(lt, "*") {
					continue // malformed PHPDoc, missing asterisk
				}
			}
			content := strings.Join(lines[a.start:a.end+1], "")
			utf := docUTF(content)
			if !annDotEndsWithDot(content, utf) || annDotHasMidDot(content, utf) {
				continue
			}
			lines[a.end] = annDotRemoveEndDot(lines[a.end], utf)
			lines[a.start] = annDotLowerFirst(lines[a.start], a.name)
			docChanged = true
		}
		if docChanged {
			s.SetValue(i, strings.Join(lines, ""))
			changed = true
		}
	}
	return changed
}

// annDotIsDotByte reports whether a byte is in the byte-mode [.。] class: a "."
// or one of the UTF-8 bytes of "。" (U+3002).
func annDotIsDotByte(b byte) bool {
	return b == '.' || b == 0xe3 || b == 0x80 || b == 0x82
}

// annDotEndsWithDot ports Preg::match('/[.。]\h*$/u', $content).
func annDotEndsWithDot(content string, utf bool) bool {
	end := len(content)
	if end > 0 && content[end-1] == '\n' {
		end--
	}
	end = annDotSkipHSpaceBackward(content, end, utf)
	if end == 0 {
		return false
	}
	if utf {
		r, _ := utf8.DecodeLastRuneInString(content[:end])
		return r == '.' || r == '。'
	}
	return annDotIsDotByte(content[end-1])
}

// annDotHasMidDot ports Preg::match('/[.。](?!\h*$)/u', $content): a dot that is
// not at the end (ignoring trailing horizontal whitespace and one final newline).
func annDotHasMidDot(content string, utf bool) bool {
	for i := 0; i < len(content); {
		isDot := false
		var next int
		if utf {
			r, sz := utf8.DecodeRuneInString(content[i:])
			if r == '.' || r == '。' {
				isDot, next = true, i+sz
			} else {
				i += sz
				continue
			}
		} else {
			if annDotIsDotByte(content[i]) {
				isDot, next = true, i+1
			} else {
				i++
				continue
			}
		}
		if isDot && !annDotHSpaceStarThenEnd(content[next:], utf) {
			return true
		}
		i = next
	}
	return false
}

// annDotHSpaceStarThenEnd reports whether rest is \h*$ (horizontal whitespace
// then end, allowing one trailing newline).
func annDotHSpaceStarThenEnd(rest string, utf bool) bool {
	p := annDotSkipHSpaceForward(rest, 0, utf)
	switch rest[p:] {
	case "", "\n":
		return true
	}
	return false
}

// annDotRemoveEndDot ports Preg::replace('/(?<![.。])[.。]\h*(\H+)$/u', '\1', $line).
func annDotRemoveEndDot(line string, utf bool) string {
	for i := 0; i < len(line); {
		ds := i
		var de int
		isDot := false
		if utf {
			r, sz := utf8.DecodeRuneInString(line[i:])
			if r == '.' || r == '。' {
				isDot, de = true, i+sz
			} else {
				i += sz
				continue
			}
		} else {
			if annDotIsDotByte(line[i]) {
				isDot, de = true, i+1
			} else {
				i++
				continue
			}
		}
		if isDot && !annDotPrecededByDot(line, ds, utf) {
			p := annDotSkipHSpaceForward(line, de, utf)
			if rest := line[p:]; rest != "" && !annDotContainsHSpace(rest, utf) {
				return line[:ds] + line[p:]
			}
		}
		i = de
	}
	return line
}

// annDotPrecededByDot reports whether a [.。] char ends right before ds.
func annDotPrecededByDot(line string, ds int, utf bool) bool {
	if ds == 0 {
		return false
	}
	if utf {
		r, _ := utf8.DecodeLastRuneInString(line[:ds])
		return r == '.' || r == '。'
	}
	return annDotIsDotByte(line[ds-1])
}

// annDotContainsHSpace reports whether s holds any \h (so it is not \H+).
func annDotContainsHSpace(s string, utf bool) bool {
	for i := 0; i < len(s); {
		if utf {
			r, sz := utf8.DecodeRuneInString(s[i:])
			if pcreIsHSpace(r) {
				return true
			}
			i += sz
		} else {
			if isDocH(s[i]) {
				return true
			}
			i++
		}
	}
	return false
}

func annDotSkipHSpaceForward(s string, p int, utf bool) int {
	for p < len(s) {
		if utf {
			r, sz := utf8.DecodeRuneInString(s[p:])
			if pcreIsHSpace(r) {
				p += sz
				continue
			}
		} else if isDocH(s[p]) {
			p++
			continue
		}
		break
	}
	return p
}

func annDotSkipHSpaceBackward(s string, end int, utf bool) int {
	for end > 0 {
		if utf {
			r, sz := utf8.DecodeLastRuneInString(s[:end])
			if pcreIsHSpace(r) {
				end -= sz
				continue
			}
		} else if isDocH(s[end-1]) {
			end--
			continue
		}
		break
	}
	return end
}

// annDotLowerFirst ports the first-letter lowercasing: it matches the tag (and,
// for typed tags, the type and optional variable) then lowercases a leading
// uppercase letter of the description when it precedes a lowercase letter or a
// space separator.
func annDotLowerFirst(line, tagName string) string {
	i := annDotSkipPcreS(line, 0)
	if i >= len(line) || line[i] != '*' {
		return line
	}
	i = annDotSkipPcreS(line, i+1)
	if i >= len(line) || line[i] != '@' {
		return line
	}
	i++
	j := i
	for j < len(line) && annDotIsWordByte(line[j]) {
		j++
	}
	if j == i {
		return line
	}
	i = j
	if k := annDotSkipPcreS(line, i); k != i {
		i = k
	} else {
		return line
	}
	if phpdocTypedTags[tagName] {
		typesStr := annDotTypesStr(line, i)
		if typesStr == "" || !strings.HasPrefix(line[i:], typesStr) {
			return line
		}
		i += len(typesStr)
		if k := annDotSkipPcreS(line, i); k != i {
			i = k
		} else {
			return line
		}
		if i < len(line) && line[i] == '$' {
			m := i + 1
			for m < len(line) && annDotIsWordByte(line[m]) {
				m++
			}
			if m > i+1 {
				if n := annDotSkipPcreS(line, m); n != m {
					i = n
				}
			}
		}
	}
	if i >= len(line) {
		return line
	}
	r1, sz1 := utf8.DecodeRuneInString(line[i:])
	if !unicode.IsUpper(r1) {
		return line
	}
	r2, _ := utf8.DecodeRuneInString(line[i+sz1:])
	if !unicode.IsLower(r2) && !unicode.Is(unicode.Zs, r2) {
		return line
	}
	return line[:i] + string(unicode.ToLower(r1)) + line[i+sz1:]
}

// annDotTypesStr ports preg_quote(implode('|', $annotation->getTypes())): the
// type's members joined with "|".
func annDotTypesStr(line string, i int) string {
	ts, ok := parsePHPTypes(line[i:], 0)
	if !ok {
		return ""
	}
	members, _ := phpTypeMembers(line[i : i+ts.end])
	return strings.Join(members, "|")
}

// annDotSkipPcreS skips PCRE \s ([ \t\n\r\f\v]) from p.
func annDotSkipPcreS(s string, p int) int {
	for p < len(s) {
		switch s[p] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			p++
		default:
			return p
		}
	}
	return p
}

// annDotIsWordByte reports PCRE \w ([0-9A-Za-z_]).
func annDotIsWordByte(b byte) bool {
	return b == '_' || ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}
