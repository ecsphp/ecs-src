package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/FullyQualifiedExceptionsSniff.php
//
// SlevomatFullyQualifiedExceptions references exceptions, errors and
// Throwable via their fully qualified name.
type SlevomatFullyQualifiedExceptions struct {
	special []string
	ignored []string
}

func (SlevomatFullyQualifiedExceptions) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\FullyQualifiedExceptionsSniff`
}

func (SlevomatFullyQualifiedExceptions) SourceURL() string {
	return fixer.SourceURLFor(SlevomatFullyQualifiedExceptions{}.Name())
}

func (r SlevomatFullyQualifiedExceptions) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := slvStringList(config["specialExceptionNames"]); ok {
		r.special = v
	}
	if v, ok := slvStringList(config["ignoredNames"]); ok {
		r.ignored = v
	}
	return r
}

// slvStringList is SniffSettingsHelper::normalizeArray.
func slvStringList(v any) ([]string, bool) {
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	var out []string
	for _, item := range list {
		s, _ := item.(string)
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out, true
}

// slvFirstOpenTag returns the first T_OPEN_TAG, which "<?=" is not, or -1.
func (f *slvFile) slvFirstOpenTag() int {
	for i, t := range f.toks {
		if t.kind == token.OpenTag && !strings.HasPrefix(t.orig, "<?=") {
			return i
		}
	}
	return -1
}

func (r SlevomatFullyQualifiedExceptions) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		openTag := f.slvFirstOpenTag()
		if openTag < 0 {
			return
		}
		uses := f.useStatements()
		names := newSlvNames(f)
		namespaces := f.refNamespacePointers()
		for _, ref := range names.referencedNames(openTag) {
			pointer, name := ref.start, ref.name
			if fqn, ok := uses.forPointer(pointer)[slvUseID(ref.typ, name)]; ok {
				if slices.Contains(r.ignored, fqn) ||
					!strings.HasSuffix(fqn, "Exception") && fqn != "Throwable" &&
						(!strings.HasSuffix(fqn, "Error") || slvHasNamespace(fqn)) && !slices.Contains(r.special, fqn) {
					continue
				}
			} else {
				canonical := name
				if !strings.HasPrefix(name, `\`) {
					for _, ns := range slices.Backward(namespaces) {
						if ns < pointer {
							canonical = f.refNamespaceName(ns) + `\` + name
							break
						}
					}
				}
				if slices.Contains(r.ignored, canonical) ||
					!strings.HasSuffix(name, "Exception") && name != "Throwable" &&
						(!strings.HasSuffix(canonical, "Error") || slvHasNamespace(canonical)) && !slices.Contains(r.special, canonical) {
					continue
				}
			}
			if strings.HasPrefix(name, `\`) {
				continue
			}
			fqn := uses.resolveName(f, name, slvRefClass, pointer)
			f.beginChangeset()
			f.change(ref.start, ref.end, fqn)
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/FullyQualifiedGlobalConstantsSniff.php
//
// SlevomatFullyQualifiedGlobalConstants references global constants via their
// fully qualified name.
type SlevomatFullyQualifiedGlobalConstants struct {
	slvGlobalReference
}

func (SlevomatFullyQualifiedGlobalConstants) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\FullyQualifiedGlobalConstantsSniff`
}

func (SlevomatFullyQualifiedGlobalConstants) SourceURL() string {
	return fixer.SourceURLFor(SlevomatFullyQualifiedGlobalConstants{}.Name())
}

func (r SlevomatFullyQualifiedGlobalConstants) WithConfig(config map[string]any) fixer.Fixer {
	r.slvGlobalReference = r.configure(config)
	return r
}

func (r SlevomatFullyQualifiedGlobalConstants) Fix(s *tokens.Stream) bool {
	return r.fix(s, slvRefConstant, true, nil)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/FullyQualifiedGlobalFunctionsSniff.php
//
// SlevomatFullyQualifiedGlobalFunctions references global functions via their
// fully qualified name.
type SlevomatFullyQualifiedGlobalFunctions struct {
	slvGlobalReference
	includeSpecial bool
}

func (SlevomatFullyQualifiedGlobalFunctions) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\FullyQualifiedGlobalFunctionsSniff`
}

func (SlevomatFullyQualifiedGlobalFunctions) SourceURL() string {
	return fixer.SourceURLFor(SlevomatFullyQualifiedGlobalFunctions{}.Name())
}

func (r SlevomatFullyQualifiedGlobalFunctions) WithConfig(config map[string]any) fixer.Fixer {
	r.slvGlobalReference = r.configure(config)
	if v, ok := config["includeSpecialFunctions"].(bool); ok {
		r.includeSpecial = v
	}
	return r
}

// slvSpecialFunctions is FunctionHelper::SPECIAL_FUNCTIONS.
var slvSpecialFunctions = []string{
	"array_key_exists", "array_slice", "assert", "boolval", "call_user_func", "call_user_func_array",
	"chr", "constant", "count", "define", "defined", "dirname", "doubleval", "extension_loaded",
	"floatval", "func_get_args", "func_num_args", "function_exists", "get_called_class", "get_class",
	"gettype", "in_array", "ini_get", "intval", "is_array", "is_bool", "is_callable", "is_double",
	"is_float", "is_int", "is_integer", "is_long", "is_null", "is_object", "is_real", "is_resource",
	"is_scalar", "is_string", "ord", "sizeof", "sprintf", "strlen", "strval",
}

func (r SlevomatFullyQualifiedGlobalFunctions) Fix(s *tokens.Stream) bool {
	var extra []string
	if r.includeSpecial {
		extra = slvSpecialFunctions
	}
	return r.fix(s, slvRefFunction, false, extra)
}

// slvGlobalReference is AbstractFullyQualifiedGlobalReference.
type slvGlobalReference struct {
	include []string
	exclude []string
}

func (g slvGlobalReference) configure(config map[string]any) slvGlobalReference {
	if v, ok := slvStringList(config["include"]); ok {
		g.include = v
	}
	if v, ok := slvStringList(config["exclude"]); ok {
		g.exclude = v
	}
	return g
}

func (g slvGlobalReference) fix(s *tokens.Stream, typ slvRefType, caseSensitive bool, extraInclude []string) bool {
	normalize := func(names []string) map[string]bool {
		out := map[string]bool{}
		for _, v := range names {
			if !caseSensitive {
				v = strings.ToLower(v)
			}
			out[v] = true
		}
		return out
	}
	include := normalize(append(append([]string(nil), g.include...), extraInclude...))
	exclude := normalize(g.exclude)
	return slvRun(s, func(f *slvFile) {
		openTag := f.slvFirstOpenTag()
		if openTag < 0 {
			return
		}
		for i := openTag; i < len(f.toks); i++ {
			if f.is(i, "{") && f.is(f.prevEffective(i-1), `\`) {
				// T_OPEN_USE_GROUP
				return
			}
		}
		namespaces := f.refNamespacePointers()
		if len(namespaces) == 0 {
			return
		}
		uses := f.useStatements()
		names := newSlvNames(f)
		for _, ref := range names.referencedNames(openTag) {
			name, pointer := ref.name, ref.start
			if ref.typ != typ || strings.HasPrefix(name, `\`) || slvHasNamespace(name) {
				continue
			}
			canonical := name
			if !caseSensitive {
				canonical = strings.ToLower(name)
			}
			if _, ok := uses.forPointer(pointer)[slvUseID(typ, canonical)]; ok {
				if slvHasNamespace(uses.resolveName(f, name, typ, pointer)) {
					continue
				}
			}
			if len(include) > 0 && !include[canonical] || exclude[canonical] {
				continue
			}
			f.beginChangeset()
			f.addBefore(pointer, `\`)
			f.endChangeset()
		}
	})
}
