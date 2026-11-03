// Package reporter reproduces ECS's console output: numbered file-diff blocks
// with "Applied checkers" listings, wrapped diffs, and single-line
// success/warning status lines.
package reporter

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"blink/internal/fixer"
	"blink/internal/runner"
)

type formatter struct {
	w     io.Writer
	width int
	color bool
}

// Report writes results to w and returns the process exit code (0 when clean or
// after a fix that leaves no errors, 1 otherwise; "ecs check --blink" turns 1
// into 2, as ECS does).
func Report(w io.Writer, results []runner.FileResult, isFixer bool) int {
	f := newFormatter(w)

	var diffs []runner.FileResult
	var errs []fileError
	for _, r := range results {
		if r.Changed() {
			diffs = append(diffs, r)
		}
		for _, e := range r.Errors {
			errs = append(errs, fileError{path: r.Path, err: e})
		}
	}

	// Both check and --fix list the per-file diff, matching ECS.
	f.reportFileDiffs(diffs)

	if len(diffs) == 0 && len(errs) == 0 {
		f.success("No errors found. Great job - your code is shiny in style!")
		return 0
	}

	f.newLine(2)

	n := len(diffs)
	if isFixer {
		f.printErrors(errs)
		if len(errs) == 0 {
			f.success(fmt.Sprintf(
				"%d error%s successfully fixed and no other errors found!",
				n, plural(n),
			))
			return 0
		}
		f.errorCountStatus(len(errs))
		return 1
	}

	if len(errs) > 0 {
		f.newLine(1)
		f.printErrors(errs)
		f.errorCountStatus(len(errs))
	}
	if n == 0 {
		return 1
	}

	verb := "errors are"
	if n == 1 {
		verb = "error is"
	}
	goodNews := ""
	if len(errs) > 0 {
		goodNews = "Good news is that "
	}
	f.warning(fmt.Sprintf(
		"%s%d %s fixable! Just add \"--fix\" to console command and rerun to apply.",
		goodNews, n, verb,
	))
	return 1
}

// fileError is a reported error with the file it was found in.
type fileError struct {
	path string
	err  fixer.CodingStandardError
}

// errorCountStatus prints ECS's "Found N errors that need to be fixed manually."
func (f formatter) errorCountStatus(n int) {
	need := "s"
	if n != 1 {
		need = ""
	}
	f.errorLine(fmt.Sprintf("Found %d error%s that need%s to be fixed manually.", n, plural(n), need))
}

// printErrors mirrors EasyCodingStandardStyle::printErrors, sorted by file and
// line.
func (f formatter) printErrors(errs []fileError) {
	slices.SortStableFunc(errs, func(a, b fileError) int {
		return cmp.Or(cmp.Compare(a.path, b.path), cmp.Compare(a.err.Line, b.err.Line))
	})
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	width := errorTableWidth()
	separator := " " + strings.Repeat("-", max(width, 0))
	for _, e := range errs {
		f.writeln(separator)
		f.writeln(fmt.Sprintf(" %s:%d", relativePath(cwd, e.path), e.err.Line))
		f.writeln(separator)
		message := fmt.Sprintf("%s\n\n Reported by: \"%s\"", e.err.Message, e.err.Checker)
		f.writeln(" " + phpWordwrap(strings.ReplaceAll(message, "\r", ""), width))
		f.writeln(separator)
		f.newLine(1)
	}
}

// errorTableWidth mirrors EasyCodingStandardStyle::getTerminalWidth.
func errorTableWidth() int {
	const bulgarianConstant = 8
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil {
		return n - bulgarianConstant
	}
	return 120 - bulgarianConstant
}

// phpWordwrap mirrors PHP's wordwrap($text, $width, "\n", false).
func phpWordwrap(text string, width int) string {
	b := []byte(text)
	lastStart, lastSpace := 0, 0
	for current := 0; current < len(b); current++ {
		switch {
		case text[current] == '\n':
			lastStart, lastSpace = current+1, current+1
		case text[current] == ' ':
			if current-lastStart >= width {
				b[current] = '\n'
				lastStart = current + 1
			}
			lastSpace = current
		case current-lastStart >= width && lastStart != lastSpace:
			b[lastSpace] = '\n'
			lastStart = lastSpace + 1
		}
	}
	return string(b)
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

// success / warning render ECS's single-line green/yellow status line.
func (f formatter) success(msg string)   { f.statusLine("[OK]", msg, 32) }
func (f formatter) errorLine(msg string) { f.statusLine("[ERROR]", msg, 31) }
func (f formatter) warning(msg string)   { f.statusLine("[WARNING]", msg, 33) }

func (f formatter) statusLine(tag, msg string, color int) {
	f.newLine(1)
	f.writeln(f.fg(" "+tag+" "+msg, color))
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
