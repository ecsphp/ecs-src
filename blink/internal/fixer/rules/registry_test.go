package rules

import (
	"slices"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

type namedFixer string

func (f namedFixer) Name() string          { return string(f) }
func (namedFixer) SourceURL() string       { return "https://example.com/" }
func (namedFixer) Fix(*tokens.Stream) bool { return false }

func TestRegisteredFixersRunByPriority(t *testing.T) {
	t.Parallel()
	first, last := namedFixer(`Acme\FirstFixer`), namedFixer(`Acme\LastFixer`)
	r := newRegistry(builtinFixers(), []registration{{fixer: last, priority: -1000}, {fixer: first, priority: 1000}})

	if r.sorted[0] != fixer.Fixer(first) || r.sorted[len(r.sorted)-1] != fixer.Fixer(last) {
		t.Errorf("registered fixers out of priority order: first %s, last %s", r.sorted[0].Name(), r.sorted[len(r.sorted)-1].Name())
	}
	if got, ok := r.byName[first.Name()]; !ok || got != fixer.Fixer(first) {
		t.Errorf("%s not resolvable by name", first.Name())
	}
	if slices.Contains(r.builtin, fixer.Fixer(first)) || len(r.builtin) != len(r.sorted)-2 {
		t.Errorf("builtin holds %d of %d fixers, want all but the 2 registered", len(r.builtin), len(r.sorted))
	}
}

func TestRegisteredFixerTiesBreakByName(t *testing.T) {
	t.Parallel()
	b, a := namedFixer(`Acme\BFixer`), namedFixer(`Acme\AFixer`)
	r := newRegistry(nil, []registration{{fixer: b}, {fixer: a}})
	if r.sorted[0] != fixer.Fixer(a) {
		t.Errorf("equal priorities ran %s first, want %s", r.sorted[0].Name(), a.Name())
	}
}

func TestNameTaken(t *testing.T) {
	t.Parallel()
	builtins := builtinFixers()
	registered := []registration{{fixer: namedFixer(`Acme\AFixer`)}}
	for name, want := range map[string]bool{
		builtins[0].Name(): true,
		`Acme\AFixer`:      true,
		`PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`: true,
		`Acme\BFixer`: false,
	} {
		if got := nameTaken(name, builtins, registered); got != want {
			t.Errorf("nameTaken(%s) = %v, want %v", name, got, want)
		}
	}
}

func TestRegisterAfterStartupPanics(t *testing.T) {
	t.Parallel()
	All()
	defer func() {
		if msg, _ := recover().(string); !strings.Contains(msg, "after the fixer registry was built") {
			t.Errorf("recovered %q, want a late-registration panic", msg)
		}
	}()
	Register(namedFixer(`Acme\LateFixer`), 0)
}
