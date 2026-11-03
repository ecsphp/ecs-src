// Command customfixer is an example blink binary with one custom fixer.
package main

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/extend"
)

func main() {
	extend.Register(RenameFunction{}, 0)
	extend.Main()
}

// RenameFunction renames global function calls, configured as
// {"functions": {"sizeof": "count"}}. It ignores `use function` imports.
type RenameFunction struct {
	functions map[string]string // lowercase old name -> new name
}

func (RenameFunction) Name() string { return `Acme\CodingStandard\Fixer\RenameFunctionFixer` }

func (RenameFunction) SourceURL() string {
	return "https://example.com/acme/coding-standard/src/Fixer/RenameFunctionFixer.php"
}

func (RenameFunction) WithConfig(config map[string]any) extend.Fixer {
	functions, _ := config["functions"].(map[string]any)
	f := RenameFunction{functions: make(map[string]string, len(functions))}
	for from, to := range functions {
		if to, ok := to.(string); ok {
			f.functions[strings.ToLower(from)] = to
		}
	}
	return f
}

func (f RenameFunction) Fix(s *extend.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.Kind(i) != extend.Ident {
			continue
		}
		to, ok := f.functions[strings.ToLower(s.At(i).Value)]
		if ok && isGlobalFunctionCall(s, i) {
			s.SetValue(i, to)
			changed = true
		}
	}
	return changed
}

// isGlobalFunctionCall is false for methods, namespaced functions, attributes
// and declarations.
func isGlobalFunctionCall(s *extend.Stream, i int) bool {
	next := extend.NextMeaningful(s, i)
	if next < 0 || !isPunct(s, next, "(") {
		return false
	}
	prev := extend.PrevMeaningful(s, i)
	if prev < 0 {
		return true
	}
	switch {
	case isPunct(s, prev, `\`):
		before := extend.PrevMeaningful(s, prev)
		return before < 0 || s.Kind(before) != extend.Ident
	case isPunct(s, prev, "&"):
		return !isKeyword(s, extend.PrevMeaningful(s, prev), "function")
	case isPunct(s, prev, "->", "?->", "::"), s.Kind(prev) == extend.AttributeStart:
		return false
	}
	return !isKeyword(s, prev, "function", "new", "const")
}

func isPunct(s *extend.Stream, i int, values ...string) bool {
	return i >= 0 && s.Kind(i) == extend.Punct && slices.Contains(values, s.At(i).Value)
}

func isKeyword(s *extend.Stream, i int, words ...string) bool {
	return i >= 0 && s.Kind(i) == extend.Keyword && slices.Contains(words, strings.ToLower(s.At(i).Value))
}
