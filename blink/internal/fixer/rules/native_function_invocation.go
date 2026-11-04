package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NativeFunctionInvocationFixer.php
//
// NativeFunctionInvocation adds a leading "\" before a global function call that
// is in the configured include set ("count(" -> "\count(") so the engine resolves
// it directly. With strict (the default) it also removes a leading "\" from a
// global call that is not in the set ("\json_encode(" -> "json_encode(").
//
// Defaults match PHP-CS-Fixer: include @compiler_optimized, scope all, strict on.
// Options: exclude (names never touched), include (names plus the sets @all,
// @internal, @compiler_optimized), scope ("all" or "namespaced") and strict.
type NativeFunctionInvocation struct {
	exclude      map[string]bool
	includeNames map[string]bool
	includeAll   bool
	hasInclude   bool
	inclInternal bool
	inclOptim    bool
	namespaced   bool
	notStrict    bool
}

func (NativeFunctionInvocation) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NativeFunctionInvocationFixer`
}

func (NativeFunctionInvocation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NativeFunctionInvocationFixer.php"
}

func (f NativeFunctionInvocation) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := singleSpaceStringList(config["exclude"]); ok {
		f.exclude = lowerSet(list)
	}
	if list, ok := singleSpaceStringList(config["include"]); ok {
		f.hasInclude = true
		f.includeNames = map[string]bool{}
		f.includeAll, f.inclInternal, f.inclOptim = false, false, false
		for _, name := range list {
			switch name {
			case "@all":
				f.includeAll = true
			case "@internal":
				f.inclInternal = true
			case "@compiler_optimized":
				f.inclOptim = true
			default:
				f.includeNames[strings.ToLower(name)] = true
			}
		}
	}
	if scope, ok := config["scope"].(string); ok {
		f.namespaced = scope == "namespaced"
	}
	if strict, ok := config["strict"].(bool); ok {
		f.notStrict = !strict
	}
	return f
}

func lowerSet(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, name := range names {
		out[strings.ToLower(name)] = true
	}
	return out
}

// included reports whether a function name gets a leading "\".
func (f NativeFunctionInvocation) included(name string) bool {
	lower := strings.ToLower(name)
	if f.exclude[lower] {
		return false
	}
	if f.includeAll {
		return true
	}
	if f.includeNames[lower] {
		return true
	}
	if f.inclInternal {
		return nativeInternal[lower]
	}
	// default (no include option) and @compiler_optimized both use the optimized set
	if f.inclOptim || !f.hasInclude {
		return nativeCompilerOptimized[lower]
	}
	return false
}

func (f NativeFunctionInvocation) Fix(s *tokens.Stream) bool {
	if !f.namespaced {
		return f.fixRange(s, 0, false)
	}
	changed := false
	// walk regions back to front so index-shifting edits stay local to a region
	regions := nativeNamespaceRegions(s)
	for _, region := range slices.Backward(regions) {
		if f.fixRange(s, region.start, region.global) {
			changed = true
		}
	}
	return changed
}

// fixRange adds or removes the leading "\" of every global function call at or
// after start. tryToRemove forces removal mode (global-namespace scope), where a
// call never gains a "\" and strict strips any it has.
func (f NativeFunctionInvocation) fixRange(s *tokens.Stream, start int, tryToRemove bool) bool {
	end := nativeRegionEnd(s, start)
	changed := false
	for i := start; i < end && i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident {
			continue
		}
		ok, backslash := f.globalCall(s, i)
		if !ok {
			continue
		}
		if tryToRemove || !f.included(s.At(i).Value) {
			if !f.notStrict && backslash >= 0 {
				s.RemoveAt(backslash)
				i--
				end--
				changed = true
			}
			continue
		}
		if backslash >= 0 {
			continue // already fully qualified
		}
		s.InsertAt(i, token.Token{Kind: token.Punct, Value: `\`})
		i++
		end++
		changed = true
	}
	return changed
}

// globalCall reports whether the name token at i is a global function call and,
// if so, the index of its leading "\" (or -1). A method/static call, a
// declaration, an instantiation or a namespace-qualified name is not global.
func (f NativeFunctionInvocation) globalCall(s *tokens.Stream, i int) (bool, int) {
	n := sigNext(s, i)
	if n < 0 || s.At(n).Kind != token.Punct || s.At(n).Value != "(" {
		return false, -1
	}
	// "name(...)" is a first-class callable, not a call the engine resolves
	if d := sigNext(s, n); d >= 0 && s.At(d).Kind == token.Punct && s.At(d).Value == "..." {
		if c := sigNext(s, d); c >= 0 && s.At(c).Kind == token.Punct && s.At(c).Value == ")" {
			return false, -1
		}
	}
	p := sigPrev(s, i)
	backslash := -1
	if p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == `\` {
		backslash = p
		p = sigPrev(s, p)
	}
	if p >= 0 {
		pt := s.At(p)
		// a name before the "\" makes it a namespace-qualified call, not global
		if backslash >= 0 && pt.Kind == token.Ident {
			return false, -1
		}
		switch {
		case pt.Kind == token.Punct && (pt.Value == "->" || pt.Value == "?->" || pt.Value == "::"):
			return false, -1
		case pt.Kind == token.Keyword && isNativeCallDisqualifier(pt.Value):
			return false, -1
		case pt.Kind == token.Punct && pt.Value == "&":
			if pp := sigPrev(s, p); pp >= 0 && s.At(pp).Kind == token.Keyword && strings.EqualFold(s.At(pp).Value, "function") {
				return false, -1
			}
		}
	}
	return true, backslash
}

func isNativeCallDisqualifier(keyword string) bool {
	switch strings.ToLower(keyword) {
	case "new", "function", "const":
		return true
	}
	return false
}

// nativeRegion is a namespace scope: calls from start up to the next region are
// fixed, and global marks the global namespace (where strict strips every "\").
type nativeRegion struct {
	start  int
	global bool
}

// nativeNamespaceRegions splits the stream into namespace scopes. A file with no
// namespace declaration is one global namespace spanning the whole file, matching
// PHP-CS-Fixer's getNamespaceDeclarations.
func nativeNamespaceRegions(s *tokens.Stream) []nativeRegion {
	var regions []nativeRegion
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || !strings.EqualFold(s.At(i).Value, "namespace") {
			continue
		}
		n := sigNext(s, i)
		global := n < 0 || (s.At(n).Kind == token.Punct && (s.At(n).Value == ";" || s.At(n).Value == "{"))
		regions = append(regions, nativeRegion{start: i, global: global})
	}
	if len(regions) == 0 {
		return []nativeRegion{{start: 0, global: true}}
	}
	return regions
}

// nativeRegionEnd returns the first namespace-declaration index after start, or
// the stream length, bounding a region's fix range.
func nativeRegionEnd(s *tokens.Stream, start int) int {
	for i := start + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "namespace") {
			return i
		}
	}
	return s.Len()
}
