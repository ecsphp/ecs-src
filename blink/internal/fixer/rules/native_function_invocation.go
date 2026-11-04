package rules

import (
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
// What counts as a global call is FunctionsAnalyzer::isGlobalFunctionCall
// (funcScan): attribute classes, functions declared in the namespace and
// non-global "use function" imports are not.
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
	sc := sharedSymScan(s)
	fs := newFuncScan(sc)
	edits := newStreamEdits()
	if !f.namespaced {
		f.fixFunctionCalls(fs, edits, 0, s.Len()-1, false)
	} else {
		// every decision is made on the unchanged stream, so the namespaces can
		// be walked in any order (PHP-CS-Fixer walks them back to front)
		for _, ns := range sc.nsDecls {
			f.fixFunctionCalls(fs, edits, ns.scopeStart, ns.scopeEnd, ns.isGlobal())
		}
	}
	return edits.apply(s, true)
}

// fixFunctionCalls ports NativeFunctionInvocationFixer::fixFunctionCalls: every
// global function call in [start, end) gains a leading "\" when it is included,
// or (strict) loses it when it is not, or when tryToRemove marks the global
// namespace of scope "namespaced".
func (f NativeFunctionInvocation) fixFunctionCalls(fs *funcScan, edits *streamEdits, start, end int, tryToRemove bool) {
	s := fs.s
	for i := start; i < end; i++ {
		if !fs.globalFunction(i, true) {
			continue
		}
		prev := fs.prevM(i)
		qualified := fs.isPunct(prev, `\`)
		if tryToRemove || !f.included(s.At(i).Value) {
			if !f.notStrict && qualified {
				edits.removes[prev] = true
			}
			continue
		}
		if !qualified {
			edits.inserts[i] = []token.Token{{Kind: token.Punct, Value: `\`}}
		}
	}
}
