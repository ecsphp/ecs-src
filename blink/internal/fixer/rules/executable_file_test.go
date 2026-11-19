package rules

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// TestExecutableFile ports ExecutableFileUnitTest: an executable file is
// reported with its mode, a plain one is not.
func TestExecutableFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no execute permission bits")
	}
	dir := t.TempDir()
	for mode, want := range map[os.FileMode]string{
		0o644: "",
		0o664: "",
		0o755: "A PHP file should not be executable; found file permissions set to 0755",
		0o775: "A PHP file should not be executable; found file permissions set to 0775",
		0o744: "A PHP file should not be executable; found file permissions set to 0744",
	} {
		path := filepath.Join(dir, mode.String()+".php")
		if err := os.WriteFile(path, []byte("<?php\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		s := tokens.New(lexer.Lex("<?php\n"))
		s.SetPath(path)
		errs := ExecutableFile{}.Errors(s)
		switch {
		case want == "" && len(errs) != 0:
			t.Errorf("%o: want no error, got %+v", mode, errs)
		case want != "" && (len(errs) != 1 || errs[0].Message != want || errs[0].Line != 1 ||
			errs[0].Checker != `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\ExecutableFileSniff.Executable`):
			t.Errorf("%o: want %q, got %+v", mode, want, errs)
		}
	}
	if errs := (ExecutableFile{}).Errors(tokens.New(lexer.Lex("<?php\n"))); len(errs) != 0 {
		t.Errorf("no path: want no error, got %+v", errs)
	}
}
