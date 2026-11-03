package rules

import "unicode/utf8"

// PHP-CS-Fixer's Preg adds /u to every pattern, and PHP compiles /u with
// PCRE2_UCP: \s, \h, \v, \w and \b then match Unicode characters, unlike
// RE2's. These stand in for them in ported patterns.

// Character class bodies, for use inside [...].
const (
	pcreHSpaceClass = `\t \x{a0}\x{1680}\x{180e}\x{2000}-\x{200a}\x{202f}\x{205f}\x{3000}`
	pcreVSpaceClass = `\n\v\f\r\x{85}\x{2028}\x{2029}`
	pcreSpaceClass  = pcreHSpaceClass + pcreVSpaceClass
)

// pcreIsHSpace is \h.
func pcreIsHSpace(r rune) bool {
	switch r {
	case '\t', ' ', 0xa0, 0x1680, 0x180e, 0x202f, 0x205f, 0x3000:
		return true
	}
	return 0x2000 <= r && r <= 0x200a
}

// pcreIsVSpace is \v (and the single characters \R matches).
func pcreIsVSpace(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// pcreIsSpace is \s, which under UCP is \h plus \v.
func pcreIsSpace(r rune) bool { return pcreIsHSpace(r) || pcreIsVSpace(r) }

// pcreRuneAt decodes the character at byte offset i; -1 past the end.
func pcreRuneAt(s string, i int) (rune, int) {
	if i >= len(s) {
		return -1, 0
	}
	return utf8.DecodeRuneInString(s[i:])
}

// pcreSkip skips the characters matching is from byte offset i.
func pcreSkip(s string, i int, is func(rune) bool) int {
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if !is(r) {
			break
		}
		i += n
	}
	return i
}
