package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"blink/internal/config"
	"blink/internal/fixer/rules"
	"blink/internal/token"
	"blink/internal/tokens"
)

// clearDropComments clears `/* drop */` comments in place, like Tokens::clearAt.
type clearDropComments struct{}

func (clearDropComments) Name() string      { return "test\\ClearDropComments" }
func (clearDropComments) SourceURL() string { return "" }

func (clearDropComments) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind == token.Comment && s.At(i).Value == "/* drop */" {
			s.SetValue(i, "")
			changed = true
		}
	}
	return changed
}

// emptyTokenSpy records whether any rule run after it saw an empty token.
type emptyTokenSpy struct{ sawEmpty *bool }

func (emptyTokenSpy) Name() string      { return "test\\EmptyTokenSpy" }
func (emptyTokenSpy) SourceURL() string { return "" }

func (r emptyTokenSpy) Fix(s *tokens.Stream) bool {
	for i := range s.Len() {
		if s.At(i).Value == "" {
			*r.sawEmpty = true
		}
	}
	return false
}

// Without clearing empty tokens between rules, ConcatSpace sees the empty token
// before "." and adds a second space (`$b  . 'c'`).
func TestRunClearsEmptyTokensBetweenRules(t *testing.T) {
	src := "<?php\n$a = $b /* drop */. 'c';\n"
	want := "<?php\n$a = $b . 'c';\n"

	for _, write := range []bool{false, true} {
		dir := t.TempDir()
		path := filepath.Join(dir, "sample.php")
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		sawEmpty := false
		cfg := config.Configure().WithPaths(dir).WithRules(
			clearDropComments{},
			emptyTokenSpy{&sawEmpty},
			rules.ConcatSpace{},
		)

		results, err := Run(context.Background(), cfg, write, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].After != want {
			t.Fatalf("write=%v: got %+v, want After %q", write, results, want)
		}
		if sawEmpty {
			t.Errorf("write=%v: a rule ran on a stream that still held empty tokens", write)
		}
		b, _ := os.ReadFile(path)
		if write && string(b) != want {
			t.Errorf("fix mode wrote %q, want %q", b, want)
		}
		if !write && string(b) != src {
			t.Error("check mode must not modify the file")
		}
	}
}

// Empty tokens are cleared even after a rule that reports no change.
func TestRunClearsEmptyTokensAfterUnreportedChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.php"), []byte("<?php\n$a = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sawEmpty := false
	cfg := config.Configure().WithPaths(dir).WithRules(
		insertEmptySilently{},
		emptyTokenSpy{&sawEmpty},
	)
	if _, err := Run(context.Background(), cfg, false, nil); err != nil {
		t.Fatal(err)
	}
	if sawEmpty {
		t.Error("empty token survived a rule that reported no change")
	}
}

type insertEmptySilently struct{}

func (insertEmptySilently) Name() string      { return "test\\InsertEmptySilently" }
func (insertEmptySilently) SourceURL() string { return "" }

func (insertEmptySilently) Fix(s *tokens.Stream) bool {
	s.InsertAt(1, token.Token{Kind: token.Whitespace, Value: ""})
	return false
}
