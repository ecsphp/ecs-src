package finder

import (
	"os"
	"path/filepath"
	"strings"
)

// PathMatcher matches file paths against ECS skip paths like ECS's
// FileInfoMatcher: a path is skipped when it equals a pattern, starts or ends
// with it, matches it with fnmatch (where "*" crosses "/"), or lies under it
// after realpath.
type PathMatcher struct {
	patterns []skipPattern
}

type skipPattern struct {
	raw        string // as configured, for the exact-equality check
	normalized string // after FnMatchPathNormalizer; "" matches nothing more
	glob       string // normalized with "\" as "/", for fnmatch
	real       string // realpath of normalized, "/"-separated; "" if it does not resolve
	realIsFile bool
	realIsDir  bool
}

// NewPathMatcher prepares patterns for matching.
func NewPathMatcher(patterns []string) *PathMatcher {
	m := &PathMatcher{patterns: make([]skipPattern, 0, len(patterns))}
	for _, raw := range patterns {
		p := skipPattern{raw: raw, normalized: normalizeForFnmatch(raw)}
		if p.normalized != "" {
			p.glob = slashed(p.normalized)
			if real, ok := realpath(p.normalized); ok {
				p.real = slashed(real)
				if info, err := os.Stat(p.real); err == nil {
					p.realIsFile = info.Mode().IsRegular()
					p.realIsDir = info.IsDir()
				}
			}
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
	var real string
	resolved, realOK := false, false
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
		if p.real == "" {
			continue
		}
		if !resolved {
			resolved = true
			if r, ok := realpath(path); ok {
				real, realOK = slashed(r), true
			}
		}
		if !realOK {
			continue
		}
		if p.realIsFile {
			if real == p.real {
				return true
			}
			continue
		}
		prefix := p.real
		if p.realIsDir {
			// a directory covers what is below it, not a sibling it prefixes
			prefix = strings.TrimRight(prefix, "/") + "/"
		}
		if strings.HasPrefix(real, prefix) {
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
	if strings.Contains(path, "..") {
		real, ok := realpath(path)
		if !ok {
			return ""
		}
		return real
	}
	return path
}

// realpath mirrors PHP's realpath(). It avoids filepath.Abs, which would clean
// ".." before resolving the symlink in front of it.
func realpath(path string) (string, bool) {
	abs := path
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", false
		}
		abs = wd + string(filepath.Separator) + path
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	return real, true
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
