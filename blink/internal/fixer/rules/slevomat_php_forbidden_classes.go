package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/ForbiddenClassesSniff.php
//
// SlevomatForbiddenClasses replaces forbidden classes, parents, interfaces
// and traits with their configured alternatives.
type SlevomatForbiddenClasses struct {
	classes, extends, interfaces, traits map[string]*string
}

func (SlevomatForbiddenClasses) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\ForbiddenClassesSniff`
}

func (SlevomatForbiddenClasses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatForbiddenClasses{}.Name())
}

func (r SlevomatForbiddenClasses) WithConfig(config map[string]any) fixer.Fixer {
	r.classes = slvfcOption(config["forbiddenClasses"], r.classes)
	r.extends = slvfcOption(config["forbiddenExtends"], r.extends)
	r.interfaces = slvfcOption(config["forbiddenInterfaces"], r.interfaces)
	r.traits = slvfcOption(config["forbiddenTraits"], r.traits)
	return r
}

// slvfcOption is normalizeInputOption: fully qualified names mapped to their
// fully qualified alternative, nil when there is none.
func slvfcOption(v any, def map[string]*string) map[string]*string {
	m, ok := v.(map[string]any)
	if !ok {
		return def
	}
	out := map[string]*string{}
	for name, alt := range m {
		key := slvfcNormalize(name)
		if key == nil {
			continue
		}
		s, _ := alt.(string)
		out[*key] = slvfcNormalize(s)
	}
	return out
}

func slvfcNormalize(name string) *string {
	if name == "" || strings.EqualFold(name, "null") {
		return nil
	}
	if !strings.HasPrefix(name, `\`) {
		name = `\` + name
	}
	return &name
}

func (r SlevomatForbiddenClasses) Fix(s *tokens.Stream) bool {
	if len(r.classes)+len(r.extends)+len(r.interfaces)+len(r.traits) == 0 {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		uses := f.useStatements()
		n := len(f.toks)
		f.eachToken(func(i int) bool {
			switch {
			case len(r.classes) > 0 && (f.isKeyword(i, "new") || f.is(i, "::")):
				return true
			case len(r.extends) > 0 && f.isKeyword(i, "extends"),
				len(r.interfaces) > 0 && f.isKeyword(i, "implements"),
				len(r.traits) > 0 && f.isKeyword(i, "use"):
				return true
			}
			return false
		}, func(i int) int {
			switch {
			case f.isKeyword(i, "implements") || f.isKeyword(i, "use") && f.forbiddenIsTraitUse(i):
				end := i
				for end < n && !f.is(end, ";") && !f.is(end, "{") {
					end++
				}
				refs := f.forbiddenReferences(uses, i, end)
				if f.isKeyword(i, "implements") {
					f.forbiddenCheck(refs, r.interfaces, true)
				} else {
					// upstream does not fix a trait use with aliases
					f.forbiddenCheck(refs, r.traits, end >= n || !f.is(end, "{"))
				}
			case f.isKeyword(i, "new") || f.isKeyword(i, "extends"):
				end := i + 1
				for end < n && (f.isName(end) || f.isEmpty(end)) {
					end++
				}
				refs := f.forbiddenReferences(uses, i, end)
				if f.isKeyword(i, "new") {
					f.forbiddenCheck(refs, r.classes, true)
				} else {
					f.forbiddenCheck(refs, r.extends, true)
				}
			case f.is(i, "::") && !f.forbiddenInTraitUseBlock(i):
				start := i - 1
				for start >= 0 && (f.isName(start) || f.isEmpty(start)) {
					start--
				}
				f.forbiddenCheck(f.forbiddenReferences(uses, start, i), r.classes, true)
			}
			return 0
		})
	})
}

type slvfcReference struct {
	name    string
	pointer int
}

// forbiddenReferences is getAllReferences: the names after start (exclusive)
// while the search pointer is before end. Like upstream, the search for the
// next name is not bounded by end.
func (f *slvFile) forbiddenReferences(uses *slvUseStatements, start, end int) []slvfcReference {
	var refs []slvfcReference
	for start++; start < end; {
		ref := -1
		for j := start; j < len(f.toks); j++ {
			if f.isName(j) || f.isClassKeyword(j) {
				ref = j
				break
			}
		}
		if ref < 0 {
			break
		}
		if !f.isClassKeyword(ref) {
			content := f.toks[ref].content
			switch strings.ToLower(content) {
			case "", "self", "parent", "static":
			default:
				refs = append(refs, slvfcReference{uses.resolveClassName(f, content, start), ref})
			}
		}
		start = ref + 1
	}
	return refs
}

func (f *slvFile) forbiddenCheck(refs []slvfcReference, forbidden map[string]*string, fixable bool) {
	for _, ref := range refs {
		alt, ok := forbidden[ref.name]
		if !ok || alt == nil || !fixable {
			continue
		}
		f.beginChangeset()
		f.change(ref.pointer, ref.pointer, *alt)
		f.endChangeset()
	}
}

// forbiddenInTraitUseBlock tells a "::" whose innermost scope is the block of
// a trait use ("use A, B { A::x insteadof B; }").
func (f *slvFile) forbiddenInTraitUseBlock(i int) bool {
	o := f.enclosing(i)
	for o >= 0 && !f.is(o, "{") {
		o = f.enclosing(o)
	}
	return o >= 0 && f.isTraitUseBlock(o)
}

// forbiddenIsTraitUse is UseStatementHelper::isTraitUse.
func (f *slvFile) forbiddenIsTraitUse(use int) bool {
	next := f.nextEffective(use+1, len(f.toks))
	if next < 0 || f.is(next, "(") {
		return false
	}
	switch strings.ToLower(f.toks[next].orig) {
	case "function", "const":
		return false
	}
	prev := -1
	for j := use; j >= 0; j-- {
		if f.toks[j].kind == token.OpenTag && !strings.HasPrefix(f.toks[j].orig, "<?=") ||
			f.isKeyword(j, "declare") || f.isKeyword(j, "namespace") {
			return false
		}
		if f.is(j, "{") || f.is(j, "}") {
			prev = j
			break
		}
	}
	if prev < 0 {
		return false
	}
	cond := f.scopeCondition(prev)
	if cond < 0 {
		return false
	}
	if f.is(prev, "{") && f.isOOScopeOwner(cond) {
		return true
	}
	if f.isKeyword(cond, "use") {
		return true
	}
	if f.isKeyword(cond, "function") {
		return f.forbiddenInClass(use)
	}
	return false
}

// forbiddenInClass is ClassHelper::getClassPointer() !== null.
func (f *slvFile) forbiddenInClass(i int) bool {
	for o := f.enclosing(i); o >= 0; o = f.enclosing(o) {
		if f.is(o, "{") && f.isOOScopeOwner(f.braceOwner(o)) {
			return true
		}
	}
	return false
}
