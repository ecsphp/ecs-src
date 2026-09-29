// Package diff produces a unified text diff in the shape PHP-CS-Fixer emits
// (an "--- Original / +++ New" header and "@@ @@" hunks), so the reporter can
// wrap it exactly like ECS.
package diff

import "strings"

const context = 3

type opKind int

const (
	equal opKind = iota
	del
	add
)

type op struct {
	kind opKind
	text string
}

// Unified returns a unified diff of before vs after, or "" if they are equal.
func Unified(before, after string) string {
	if before == after {
		return ""
	}
	a := splitLines(before)
	b := splitLines(after)
	ops := diffLines(a, b)

	var out strings.Builder
	out.WriteString("--- Original\n+++ New\n")
	for _, h := range hunks(ops) {
		out.WriteString("@@ @@\n")
		for _, o := range h {
			switch o.kind {
			case equal:
				out.WriteString(" " + o.text + "\n")
			case del:
				out.WriteString("-" + o.text + "\n")
			case add:
				out.WriteString("+" + o.text + "\n")
			}
		}
	}
	return out.String()
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	// a trailing newline yields a final empty element; drop it for diffing
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffLines is a straightforward LCS diff, adequate for small source files.
func diffLines(a, b []string) []op {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{equal, a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, op{del, a[i]})
			i++
		default:
			ops = append(ops, op{add, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{del, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{add, b[j]})
	}
	return ops
}

// hunks groups ops into change regions, each padded with up to `context` equal
// lines, dropping long stretches of unchanged code between them.
func hunks(ops []op) [][]op {
	changed := make([]bool, len(ops))
	any := false
	for i, o := range ops {
		if o.kind != equal {
			changed[i] = true
			any = true
		}
	}
	if !any {
		return nil
	}

	keep := make([]bool, len(ops))
	for i, c := range changed {
		if !c {
			continue
		}
		lo := max(i-context, 0)
		hi := i + context
		if hi >= len(ops) {
			hi = len(ops) - 1
		}
		for k := lo; k <= hi; k++ {
			keep[k] = true
		}
	}

	var result [][]op
	var cur []op
	for i := range ops {
		if keep[i] {
			cur = append(cur, ops[i])
		} else if len(cur) > 0 {
			result = append(result, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		result = append(result, cur)
	}
	return result
}
