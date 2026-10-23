// Package reporter reproduces ECS's console output: numbered file-diff blocks
// with "Applied checkers" listings, wrapped diffs, and Symfony-style
// success/warning status blocks.
package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"blink/internal/runner"
)

type formatter struct {
	w     io.Writer
	width int
	color bool
}

// Report writes results to w and returns the process exit code (0 when clean or
// after a successful fix, 1 when a check finds fixable issues).
func Report(w io.Writer, results []runner.FileResult, isFixer bool) int {
	f := newFormatter(w)

	// Both check and --fix list the per-file diff, matching ECS.
	f.reportFileDiffs(results)

	if len(results) == 0 {
		f.success("No errors found. Great job - your code is shiny in style!")
		return 0
	}

	f.newLine(2)

	n := len(results)
	if isFixer {
		f.success(fmt.Sprintf(
			"%d error%s successfully fixed and no other errors found!",
			n, plural(n),
		))
		return 0
	}

	verb := "errors are"
	if n == 1 {
		verb = "error is"
	}
	f.warning(fmt.Sprintf(
		"%d %s fixable! Just add \"--fix\" to console command and rerun to apply.",
		n, verb,
	))
	return 1
}

// Footer prints an ECS-style run summary: files scanned, time and memory,
// greyed and preceded by a blank line.
func Footer(w io.Writer, files int, d time.Duration, memBytes uint64) {
	f := newFormatter(w)
	line := fmt.Sprintf(" // %d files · %s · %d MB", files, formatElapsed(d), memBytes/(1024*1024))
	_, _ = fmt.Fprintf(w, "\n%s\n", f.grey(line))
}

// formatElapsed shows seconds past 5s, milliseconds below.
func formatElapsed(d time.Duration) string {
	if d > 5*time.Second {
		return fmt.Sprintf("%.2f s", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Truncate(time.Millisecond).Milliseconds())
}

func newFormatter(w io.Writer) formatter {
	width := 80
	color := false
	if file, ok := w.(*os.File); ok {
		if ws, ok2 := winsize(file); ok2 {
			color = true
			if ws > 0 {
				width = ws
			}
		}
	}
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 {
			width = n
		}
	}
	return formatter{w: w, width: width, color: color}
}

// relativePath resolves an absolute file path against cwd, matching ECS's
// StaticRelativeFilePathHelper output. An empty cwd yields the path unchanged.
func relativePath(cwd, path string) string {
	if cwd == "" {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

func (f formatter) reportFileDiffs(results []runner.FileResult) {
	if len(results) == 0 {
		return
	}
	f.newLine(1)

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	for i, r := range results {
		f.newLine(2)
		f.writeln(f.bold(fmt.Sprintf("%d) %s", i+1, relativePath(cwd, r.Path))))
		f.newLine(1)
		f.writeln(f.formatDiff(r.Diff))
		f.newLine(1)
		f.writeln(f.underscore("Applied checkers:"))
		f.newLine(1)
		f.listing(r.AppliedRules)
	}
}

// formatDiff wraps a unified diff in the begin/end frame and colors +, - and @
// lines, matching ECS's ColorConsoleDiffFormatter.
func (f formatter) formatDiff(diff string) string {
	var colored []string
	for line := range strings.SplitSeq(strings.TrimRight(diff, "\n"), "\n") {
		switch {
		case line == "--- Original", line == "+++ New":
			continue
		case strings.HasPrefix(line, "+"):
			line = f.fg(line, 32) // green
		case strings.HasPrefix(line, "-"):
			line = f.fg(line, 31) // red
		case strings.HasPrefix(line, "@"):
			line = f.fg(line, 36) // cyan
		case line == " ":
			line = ""
		}
		colored = append(colored, line)
	}

	begin := f.comment("    ---------- begin diff ----------")
	end := f.comment("    ----------- end diff -----------")
	return begin + "\n" + strings.Join(colored, "\n") + "\n" + end + "\n"
}

func (f formatter) listing(items []string) {
	for _, it := range items {
		f.writeln(" * " + it)
	}
	f.newLine(1)
}

// success / warning render Symfony-style padded, colored blocks.
func (f formatter) success(msg string) { f.block("[OK]", msg, 30, 42) }
func (f formatter) warning(msg string) { f.block("[WARNING]", msg, 30, 43) }

func (f formatter) block(tag, msg string, fg, bg int) {
	f.newLine(1)
	pad := func(s string) string {
		if len([]rune(s)) < f.width {
			s += strings.Repeat(" ", f.width-len([]rune(s)))
		}
		return f.bg(s, fg, bg)
	}
	f.writeln(pad(""))
	f.writeln(pad(" " + tag + " " + msg))
	f.writeln(pad(""))
	f.newLine(1)
}

func (f formatter) newLine(n int) {
	for range n {
		_, _ = io.WriteString(f.w, "\n")
	}
}

func (f formatter) writeln(s string) { _, _ = fmt.Fprintln(f.w, s) }

// ANSI helpers, no-ops when color is disabled.
func (f formatter) fg(s string, code int) string {
	if !f.color {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[39m", code, s)
}

func (f formatter) bg(s string, fg, bg int) string {
	if !f.color {
		return s
	}
	return fmt.Sprintf("\x1b[%d;%dm%s\x1b[39;49m", fg, bg, s)
}

func (f formatter) comment(s string) string { return f.fg(s, 33) } // yellow
func (f formatter) grey(s string) string    { return f.fg(s, 90) } // bright black, matches PHP
func (f formatter) bold(s string) string {
	if !f.color {
		return s
	}
	return "\x1b[1m" + s + "\x1b[22m"
}
func (f formatter) underscore(s string) string {
	if !f.color {
		return s
	}
	return "\x1b[4m" + s + "\x1b[24m"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
