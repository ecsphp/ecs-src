package rules

import (
	"regexp"
	"testing"

	"blink/internal/fixer"
)

var sourceURLPattern = regexp.MustCompile(
	`^https://github\.com/(PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src|symplify/coding-standard/blob/main/src|ecsphp/ecs-src/blob/main/packages/coding-standard/src)/Fixer/[A-Za-z]+/[A-Za-z]+Fixer\.php$`,
)

// TestEveryFixerHasSourceURL enforces the Fixer contract offline: every rule
// must expose a well-formed PHP-CS-Fixer source link consistent with its name.
func TestEveryFixerHasSourceURL(t *testing.T) {
	for _, f := range All() {
		url := f.SourceURL()
		if url == "" {
			t.Errorf("%s: empty SourceURL", f.Name())
			continue
		}
		if !sourceURLPattern.MatchString(url) {
			t.Errorf("%s: SourceURL %q does not match the PHP-CS-Fixer source pattern", f.Name(), url)
		}
		if want := fixer.SourceURLFor(f.Name()); url != want {
			t.Errorf("%s: SourceURL %q is inconsistent with the name, want %q", f.Name(), url, want)
		}
	}
}
