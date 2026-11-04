package rules

import (
	"slices"
	"strings"
	"sync"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ConstantNotation/NativeConstantInvocationFixer.php
//
// NativeConstantInvocation adds a leading "\" before built-in constants
// (PHP_EOL -> \PHP_EOL) and, with "strict", removes it from other constants.
// Names match case-sensitively, except null, false and true. The built-in list
// is in native_names_gen.go.
type NativeConstantInvocation struct {
	// NoBuiltIn disables fixing the built-in constants (fix_built_in: false).
	NoBuiltIn bool
	// Include lists additional constants to fix.
	Include []string
	// Exclude lists constants to ignore; nil means the default [null, false, true].
	Exclude []string
	// Scope is "all" (default) or "namespaced".
	Scope string
	// NotStrict disables removing a leading "\" from constants outside the list.
	NotStrict bool

	// sensitive and insensitive cache lists() for the configured options.
	sensitive, insensitive map[string]bool
}

var nativeConstantDefaultLists = sync.OnceValues(func() (map[string]bool, map[string]bool) {
	return NativeConstantInvocation{}.lists()
})

func (f NativeConstantInvocation) WithConfig(config map[string]any) fixer.Fixer {
	if b, ok := config["fix_built_in"].(bool); ok {
		f.NoBuiltIn = !b
	}
	if list, ok := phpdocScalarStringList(config["include"]); ok {
		f.Include = list
	}
	if list, ok := phpdocScalarStringList(config["exclude"]); ok {
		f.Exclude = list
	}
	if scope, ok := config["scope"].(string); ok {
		f.Scope = scope
	}
	if strict, ok := config["strict"].(bool); ok {
		f.NotStrict = !strict
	}
	f.sensitive, f.insensitive = f.lists()
	return f
}

func (NativeConstantInvocation) Name() string {
	return `PhpCsFixer\Fixer\ConstantNotation\NativeConstantInvocationFixer`
}

func (NativeConstantInvocation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ConstantNotation/NativeConstantInvocationFixer.php"
}

// lists mirrors configurePostNormalisation(): the case-sensitive constants to
// escape and the case-insensitive ones (null, false, true).
func (f NativeConstantInvocation) lists() (sensitive, insensitive map[string]bool) {
	exclude := f.Exclude
	if exclude == nil {
		exclude = []string{"null", "false", "true"}
	}
	candidates := append([]string{}, f.Include...)
	if !f.NoBuiltIn {
		candidates = append(candidates, nativeBuiltInConstantNames...)
	}
	sensitive = map[string]bool{}
	var lowered []string
	for _, c := range candidates {
		if slices.Contains(exclude, c) {
			continue
		}
		switch l := strings.ToLower(c); l {
		case "null", "false", "true":
			lowered = append(lowered, l)
		default:
			sensitive[c] = true
		}
	}
	insensitive = map[string]bool{}
	for _, l := range lowered {
		if !slices.ContainsFunc(exclude, func(e string) bool { return strings.ToLower(e) == l }) {
			insensitive[l] = true
		}
	}
	return sensitive, insensitive
}

func (f NativeConstantInvocation) Fix(s *tokens.Stream) bool {
	hasString := false
	for i := range s.Len() {
		if s.Kind(i) == token.Ident {
			hasString = true
			break
		}
	}
	if !hasString {
		return false
	}
	sc := sharedSymScan(s)
	sensitive, insensitive := f.sensitive, f.insensitive
	if sensitive == nil {
		if f.Include == nil && f.Exclude == nil && !f.NoBuiltIn {
			sensitive, insensitive = nativeConstantDefaultLists()
		} else {
			sensitive, insensitive = f.lists()
		}
	}
	edits := newStreamEdits()
	if f.Scope != "namespaced" {
		f.fixInvocations(sc, sensitive, insensitive, edits, 0, sc.n-1)
	} else {
		for _, d := range slices.Backward(sc.nsDecls) {
			if d.isGlobal() {
				continue
			}
			f.fixInvocations(sc, sensitive, insensitive, edits, d.scopeStart, d.scopeEnd)
		}
	}
	return edits.apply(s, true)
}

// fixInvocations mirrors fixConstantInvocations() over (start, end].
func (f NativeConstantInvocation) fixInvocations(sc *symScan, sensitive, insensitive map[string]bool, edits *streamEdits, start, end int) {
	imported := map[string]bool{}
	for _, u := range sc.uses {
		if u.kind == "const" {
			imported[u.shortName] = true
		}
	}
	for i := end; i > start; i-- {
		if !sc.isConstantInvocation(i) {
			continue
		}
		name := sc.tok(i).Value
		prev := sc.prevM(i)
		if !sensitive[name] && !insensitive[strings.ToLower(name)] {
			if f.NotStrict || !sc.isPunct(prev, `\`) || sc.isTString(sc.prevM(prev)) {
				continue
			}
			edits.removes[prev] = true
			continue
		}
		if imported[name] || sc.isPunct(prev, `\`) {
			continue
		}
		edits.inserts[i] = []token.Token{nsSeparatorToken()}
	}
}
