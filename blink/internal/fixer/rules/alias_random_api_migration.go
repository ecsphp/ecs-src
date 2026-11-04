package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/RandomApiMigrationFixer.php
//
// RandomApiMigration replaces calls to rand/srand/getrandmax with their mt_*
// analogs (the v3 default), rewriting only global calls whose argument count
// matches. When a configured replacement is random_int and the call has no
// argument, "0, getrandmax()" is inserted. Risky.
type RandomApiMigration struct {
	// replacements maps an original function to its replacement; nil means the
	// default v3 mapping.
	replacements map[string]string
}

func (RandomApiMigration) Name() string {
	return `PhpCsFixer\Fixer\Alias\RandomApiMigrationFixer`
}

func (RandomApiMigration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/RandomApiMigrationFixer.php"
}

// randomArgumentCounts is RandomApiMigrationFixer::ARGUMENT_COUNTS: the argument
// counts a call must have for the rewrite to apply, keyed by original name.
var randomArgumentCounts = map[string]map[int]bool{
	"mt_getrandmax": counts(0),
	"getrandmax":    counts(0),
	"mt_rand":       counts(1, 2),
	"rand":          counts(0, 2),
	"srand":         counts(0, 1),
	"mt_srand":      counts(0, 1),
	"random_int":    counts(0, 2),
}

// defaultRandomReplacements is the v3 default (ECS runs PHP-CS-Fixer v3).
var defaultRandomReplacements = map[string]string{
	"getrandmax": "mt_getrandmax",
	"rand":       "mt_rand",
	"srand":      "mt_srand",
}

func (f RandomApiMigration) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["replacements"]; ok {
		if m, ok := raw.(map[string]any); ok {
			reps := make(map[string]string, len(m))
			for k, v := range m {
				if s, ok := v.(string); ok {
					reps[k] = s
				}
			}
			f.replacements = reps
		}
	}
	return f
}

func (f RandomApiMigration) reps() map[string]string {
	if f.replacements != nil {
		return f.replacements
	}
	return defaultRandomReplacements
}

func (f RandomApiMigration) Fix(s *tokens.Stream) bool {
	reps := f.reps()
	identities := make([]string, 0, len(reps))
	for identity := range reps {
		identities = append(identities, identity)
	}
	slices.Sort(identities) // deterministic order

	changed := false
	for _, identity := range identities {
		replacement := reps[identity]
		if identity == replacement {
			continue
		}
		allowed := randomArgumentCounts[identity]
		currIndex := 0
		for {
			fn, open, closeIdx, ok := aliasFind(s, identity, currIndex, s.Len()-1)
			if !ok {
				break
			}
			count := len(aliasArguments(s, open, closeIdx))
			if allowed == nil || !allowed[count] {
				break // mirrors PHP-CS-Fixer's "continue 2"
			}
			currIndex = open
			s.SetValue(fn, replacement)
			if count == 0 && replacement == "random_int" {
				s.InsertSliceAt(open+1, []token.Token{
					{Kind: token.Number, Value: "0"},
					{Kind: token.Punct, Value: ","},
					aliasSpace(),
					{Kind: token.Ident, Value: "getrandmax"},
					{Kind: token.Punct, Value: "("},
					{Kind: token.Punct, Value: ")"},
				})
			}
			changed = true
		}
	}
	return changed
}
