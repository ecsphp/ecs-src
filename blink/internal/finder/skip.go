package finder

import "strings"

// PathMatcher matches file paths against ECS skip paths like ECS's
// FileInfoMatcher: a path is skipped when it equals a pattern, starts or ends
// with it, or matches it with fnmatch (where "*" crosses "/").
type PathMatcher struct {
	patterns []skipPattern
}

type skipPattern struct {
	raw        string // as configured, for the exact-equality check
	normalized string // after FnMatchPathNormalizer; "" matches nothing more
	glob       string // normalized with "\" as "/", for fnmatch
}

// NewPathMatcher prepares patterns for matching.
func NewPathMatcher(patterns []string) *PathMatcher {
	m := &PathMatcher{patterns: make([]skipPattern, 0, len(patterns))}
	for _, raw := range patterns {
		p := skipPattern{raw: raw, normalized: normalizeForFnmatch(raw)}
		if p.normalized != "" {
			p.glob = slashed(p.normalized)
		}
		m.patterns = append(m.patterns, p)
	}
	return m
}

// Match reports whether path is skipped by any of the matcher's patterns.
func (m *PathMatcher) Match(path string) bool {
	if m == nil {
		return false
	}
	glob := slashed(path)
	for _, p := range m.patterns {
		if path == p.raw {
			return true
		}
		if p.normalized == "" {
			continue
		}
		if strings.HasPrefix(path, p.normalized) || strings.HasSuffix(path, p.normalized) {
			return true
		}
		if fnmatch(p.glob, glob) || fnmatch("*/"+p.glob, glob) {
			return true
		}
	}
	return false
}

// normalizeForFnmatch mirrors ECS's FnMatchPathNormalizer.
func normalizeForFnmatch(path string) string {
	if strings.HasPrefix(path, "*") || strings.HasSuffix(path, "*") {
		return "*" + strings.Trim(path, "*") + "*"
	}
	return path
}

func slashed(path string) string {
	return strings.ReplaceAll(path, `\`, "/")
}

// fnmatch mirrors PHP's fnmatch() with no flags: "*" and "?" also match "/".
// Backslash escapes are not handled: ECS turns "\" into "/" in skip paths.
func fnmatch(pattern, name string) bool {
	px, nx := 0, 0
	starPx, starNx := -1, -1
	for px < len(pattern) || nx < len(name) {
		if px < len(pattern) {
			switch c := pattern[px]; c {
			case '*':
				starPx, starNx = px, nx
				px++
				continue
			case '?':
				if nx < len(name) {
					px++
					nx++
					continue
				}
			case '[':
				if nx < len(name) {
					if matched, width, ok := matchBracket(pattern[px:], name[nx]); ok {
						if matched {
							px += width
							nx++
							continue
						}
					} else if name[nx] == '[' {
						px++
						nx++
						continue
					}
				}
			default:
				if nx < len(name) && name[nx] == c {
					px++
					nx++
					continue
				}
			}
		}
		// backtrack: let the last "*" swallow one more byte
		if starPx >= 0 && starNx < len(name) {
			starNx++
			px, nx = starPx+1, starNx
			continue
		}
		return false
	}
	return true
}

// matchBracket matches ch against the bracket expression that opens p. It
// returns whether ch matched, the expression's width, and false for ok when
// the expression is unterminated (the "[" is then a literal).
func matchBracket(p string, ch byte) (matched bool, width int, ok bool) {
	i := 1
	negate := false
	if i < len(p) && (p[i] == '!' || p[i] == '^') {
		negate = true
		i++
	}
	first := true
	for {
		if i >= len(p) {
			return false, 0, false
		}
		c := p[i]
		if c == ']' && !first {
			i++
			break
		}
		first = false
		if c == '[' && i+1 < len(p) && p[i+1] == ':' {
			if end := strings.Index(p[i+2:], ":]"); end >= 0 {
				if inCharClass(p[i+2:i+2+end], ch) {
					matched = true
				}
				i += 2 + end + 2
				continue
			}
		}
		i++
		if i+1 < len(p) && p[i] == '-' && p[i+1] != ']' {
			if c <= ch && ch <= p[i+1] {
				matched = true
			}
			i += 2
			continue
		}
		if ch == c {
			matched = true
		}
	}
	return matched != negate, i, true
}

// inCharClass reports whether ch is in the POSIX class name, in the C locale.
func inCharClass(name string, ch byte) bool {
	upper := ch >= 'A' && ch <= 'Z'
	lower := ch >= 'a' && ch <= 'z'
	digit := ch >= '0' && ch <= '9'
	graph := ch > ' ' && ch < 0x7f
	switch name {
	case "alpha":
		return upper || lower
	case "digit":
		return digit
	case "alnum":
		return upper || lower || digit
	case "upper":
		return upper
	case "lower":
		return lower
	case "xdigit":
		return digit || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
	case "space":
		return ch == ' ' || (ch >= '\t' && ch <= '\r')
	case "blank":
		return ch == ' ' || ch == '\t'
	case "punct":
		return graph && !upper && !lower && !digit
	case "graph":
		return graph
	case "print":
		return graph || ch == ' '
	case "cntrl":
		return ch < ' ' || ch == 0x7f
	}
	return false
}
