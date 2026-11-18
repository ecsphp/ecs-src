package reporter

import (
	"bytes"
	"cmp"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/runner"
)

// ReportJSON writes results as ECS's JsonOutputFormatter does and returns the
// same exit code as Report. Files are keyed by absolute path, in ECS's order:
// files with errors first (sorted by relative path and line), then the
// remaining files with a diff (sorted by relative path).
func ReportJSON(w io.Writer, results []runner.FileResult, isFixer bool) int {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	var errs []fileError
	var diffs []runner.FileResult
	for _, r := range results {
		if r.Changed() {
			diffs = append(diffs, r)
		}
		for _, e := range r.Errors {
			errs = append(errs, fileError{path: r.Path, err: e})
		}
	}
	slices.SortStableFunc(errs, func(a, b fileError) int {
		return cmp.Or(cmp.Compare(relativePath(cwd, a.path), relativePath(cwd, b.path)), cmp.Compare(a.err.Line, b.err.Line))
	})
	slices.SortStableFunc(diffs, func(a, b runner.FileResult) int {
		return cmp.Compare(relativePath(cwd, a.Path), relativePath(cwd, b.Path))
	})

	files := jsonFiles{byPath: map[string]*jsonFile{}}
	for _, e := range errs {
		path := absolutePath(e.path)
		files.get(path).Errors = append(files.get(path).Errors, jsonError{
			Line:        e.err.Line,
			FilePath:    path,
			Message:     e.err.Message,
			SourceClass: e.err.Checker,
		})
	}
	for _, r := range diffs {
		path := absolutePath(r.Path)
		files.get(path).Diffs = append(files.get(path).Diffs, jsonDiff{
			Diff:            r.Diff,
			AppliedCheckers: r.AppliedRules,
		})
	}

	report := jsonReport{
		Totals: jsonTotals{Errors: len(errs), Diffs: len(diffs)},
		Files:  files,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	if err := enc.Encode(report); err != nil {
		panic(err)
	}
	_, _ = w.Write(buf.Bytes())

	switch {
	case len(errs) == 0 && len(diffs) == 0:
		return 0
	case isFixer && len(errs) == 0:
		return 0
	default:
		return 1
	}
}

// absolutePath mirrors the realpath() ECS uses for JSON file keys.
func absolutePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs
}

type jsonReport struct {
	Totals jsonTotals `json:"totals"`
	Files  jsonFiles  `json:"files"`
}

type jsonTotals struct {
	Errors int `json:"errors"`
	Diffs  int `json:"diffs"`
}

type jsonFile struct {
	Errors []jsonError `json:"errors,omitempty"`
	Diffs  []jsonDiff  `json:"diffs,omitempty"`
}

type jsonError struct {
	Line        int    `json:"line"`
	FilePath    string `json:"file_path"`
	Message     string `json:"message"`
	SourceClass string `json:"source_class"`
}

type jsonDiff struct {
	Diff            string   `json:"diff"`
	AppliedCheckers []string `json:"applied_checkers"`
}

// jsonFiles is a JSON object that keeps insertion order, as PHP arrays do; with
// no files it encodes as [], like PHP's empty array.
type jsonFiles struct {
	order  []string
	byPath map[string]*jsonFile
}

func (f *jsonFiles) get(path string) *jsonFile {
	file, ok := f.byPath[path]
	if !ok {
		file = &jsonFile{}
		f.byPath[path] = file
		f.order = append(f.order, path)
	}
	return file
}

func (f jsonFiles) MarshalJSON() ([]byte, error) {
	if len(f.order) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, path := range f.order {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := marshalNoEscape(path)
		if err != nil {
			return nil, err
		}
		value, err := marshalNoEscape(f.byPath[path])
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshalNoEscape is json.Marshal without HTML escaping, matching PHP's
// json_encode, which leaves <, > and & as they are.
func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
