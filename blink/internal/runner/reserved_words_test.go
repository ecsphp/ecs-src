package runner

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/config"
)

// The fixture names enum cases and constants after every PHP reserved
// word. PHP ECS (psr12 + common) leaves it as it is.
func TestDefaultSetKeepsReservedWordNames(t *testing.T) {
	res, err := fixFile(config.Configure(), nil, "testdata/reserved_word_names.php", false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed() || len(res.Errors) > 0 {
		t.Errorf("changed by %v, errors %v:\n%s", res.AppliedRules, res.Errors, res.Diff)
	}
}
