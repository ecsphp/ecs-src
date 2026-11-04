package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/config"
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// appendComment appends a comment to every file, reports an error on line 1,
// and is a candidate only on files holding the word it is given.
type appendComment struct {
	name, word string
	ran        *int
}

func (r appendComment) Name() string    { return r.name }
func (appendComment) SourceURL() string { return "" }
func (r appendComment) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord(r.word)
}

func (r appendComment) Fix(s *tokens.Stream) bool {
	*r.ran++
	s.InsertAt(s.Len(), token.Token{Kind: token.Comment, Value: "// " + r.name})
	return true
}

func (r appendComment) Errors(*tokens.Stream) []fixer.CodingStandardError {
	return []fixer.CodingStandardError{{Line: 1, Message: r.name, Checker: r.name}}
}

// The runner calls neither Fix nor Errors on a file IsCandidate rules out,
// and still clears the empty tokens a rule before left.
func TestRunSkipsFilesRuledOutByIsCandidate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.php"), []byte("<?php\nFoo();\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ranFoo, ranBar int
	sawEmpty := false
	cfg := config.Configure().WithPaths(dir).WithRules(
		insertEmptySilently{},
		appendComment{name: "test\\Bar", word: "bar", ran: &ranBar},
		emptyTokenSpy{&sawEmpty},
		appendComment{name: "test\\Foo", word: "foo", ran: &ranFoo},
	)
	results, err := Run(context.Background(), cfg, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ranBar != 0 || ranFoo != 1 {
		t.Fatalf("Fix ran %d times for the ruled-out rule and %d for the candidate, want 0 and 1", ranBar, ranFoo)
	}
	if len(results) != 1 || len(results[0].AppliedRules) != 1 || results[0].AppliedRules[0] != "test\\Foo" {
		t.Fatalf("got %+v, want only test\\Foo applied", results)
	}
	if errs := results[0].Errors; len(errs) != 1 || errs[0].Checker != "test\\Foo" {
		t.Errorf("errors %+v, want only test\\Foo's", errs)
	}
	if sawEmpty {
		t.Error("an empty token survived a skipped rule")
	}
}
