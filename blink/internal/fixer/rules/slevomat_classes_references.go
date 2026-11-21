package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clgScopeCloser is scopeCloser, except that a backed enum's ":" does not end
// the search for its "{".
func (f *slvFile) clgScopeCloser(i int) int {
	if !f.isKeyword(i, "enum") {
		return f.scopeCloser(i)
	}
	for j := i + 1; j < len(f.toks); j++ {
		switch {
		case f.is(j, "{"):
			return f.toks[j].closer
		case f.is(j, ";") || f.is(j, "(") || f.is(j, "["):
			return -1
		}
	}
	return -1
}

// clgIsAnonClass tells a "class" PHP_CodeSniffer retokenizes to T_ANON_CLASS:
// one with a scope followed by "(", "{", "extends" or "implements".
func (f *slvFile) clgIsAnonClass(i int) bool {
	if !f.isKeyword(i, "class") || f.scopeCloser(i) < 0 {
		return false
	}
	n := f.nextEffective(i+1, len(f.toks))
	return f.is(n, "(") || f.is(n, "{") || f.isKeyword(n, "extends") || f.isKeyword(n, "implements")
}

// clgClassPointers is ClassHelper::getAllClassPointers: every class, interface,
// trait, enum and anonymous class with a scope.
func (f *slvFile) clgClassPointers() []int {
	var out []int
	for i := range f.toks {
		if f.isOOScopeOwner(i) && f.clgScopeCloser(i) >= 0 {
			out = append(out, i)
		}
	}
	return out
}

// clgScopeOpener is the scope_opener of an OO keyword with a scope.
func (f *slvFile) clgScopeOpener(i int) int {
	return f.toks[f.clgScopeCloser(i)].closer
}

// clgClassPointer is ClassHelper::getClassPointer, or -1.
func (f *slvFile) clgClassPointer(pointers []int, i int) int {
	for _, c := range slices.Backward(pointers) {
		if f.clgScopeOpener(c) < i && f.clgScopeCloser(c) > i {
			return c
		}
	}
	return -1
}

// clgClassFullyQualifiedName is ClassHelper::getFullyQualifiedName.
func (f *slvFile) clgClassFullyQualifiedName(c int) string {
	if f.clgIsAnonClass(c) {
		return "class@anonymous"
	}
	name := ""
	for j := c + 1; j < f.clgScopeOpener(c); j++ {
		if f.toks[j].kind == token.Ident {
			name = f.toks[j].content
			break
		}
	}
	name = `\` + name
	if ns, ok := f.refCurrentNamespace(c); ok {
		name = `\` + ns + name
	}
	return name
}

// clgOOCondition is the innermost Tokens::OO_SCOPE_TOKENS condition of i, or -1.
func (f *slvFile) clgOOCondition(i int) int {
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		if owner := f.braceOwner(open); f.isOOScopeOwner(owner) {
			return owner
		}
	}
	return -1
}

// clgInnermostCondition is the last condition of i, or -1.
func (f *slvFile) clgInnermostCondition(i int) int {
	open := f.enclosingScope(i)
	if open < 0 {
		return -1
	}
	return f.braceOwner(open)
}

// clgStaticDoubleColon returns the "::" after a "static", or -1.
func (f *slvFile) clgStaticDoubleColon(i int) int {
	if !f.isKeyword(i, "static") {
		return -1
	}
	n := f.nextEffective(i+1, len(f.toks))
	if !f.is(n, "::") {
		return -1
	}
	return n
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/RequireSelfReferenceSniff.php
//
// SlevomatRequireSelfReference replaces a reference to the enclosing class by
// its name with "self".
type SlevomatRequireSelfReference struct{}

func (SlevomatRequireSelfReference) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\RequireSelfReferenceSniff`
}

func (SlevomatRequireSelfReference) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireSelfReference{}.Name())
}

func (SlevomatRequireSelfReference) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		openTag := f.slvFirstOpenTag()
		if openTag < 0 {
			return
		}
		names := newSlvNames(f)
		refs := append(names.referencedNames(openTag), names.referencedNamesInAttributes(openTag)...)
		var anonymous []int
		for i := range f.toks {
			if f.clgIsAnonClass(i) {
				anonymous = append(anonymous, i)
			}
		}
		classes := f.clgClassPointers()
		uses := f.useStatements()
		for _, ref := range refs {
			if ref.typ != slvRefClass {
				continue
			}
			anon := -1
			for _, a := range anonymous {
				if a >= ref.start {
					break
				}
				anon = a
			}
			if anon >= 0 && f.scopeCloser(anon) > ref.end {
				continue
			}
			c := f.clgClassPointer(classes, ref.start)
			if c < 0 {
				continue
			}
			if f.clgClassFullyQualifiedName(c) != uses.resolveName(f, ref.name, slvRefClass, ref.start) {
				continue
			}
			f.beginChangeset()
			if f.toks[ref.start].kind == token.AttributeStart {
				f.replace(ref.start, clgReplaceName(f.content(ref.start, ref.end), ref.name))
			} else {
				f.replace(ref.start, "self")
			}
			f.removeBetweenIncluding(ref.start+1, ref.end)
			f.endChangeset()
		}
	})
}

// clgIsWordByte tells a byte PCRE's \w matches without the u modifier.
func clgIsWordByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// clgReplaceName is preg_replace('~(?<=\W)name(?=\W)~', 'self', $content).
func clgReplaceName(content, name string) string {
	var b strings.Builder
	pos := 0
	for k := 0; k+len(name) <= len(content); {
		end := k + len(name)
		if content[k:end] == name && k > 0 && !clgIsWordByte(content[k-1]) && end < len(content) && !clgIsWordByte(content[end]) {
			b.WriteString(content[pos:k])
			b.WriteString("self")
			pos, k = end, end
			continue
		}
		k++
	}
	b.WriteString(content[pos:])
	return b.String()
}

// clgPropertyName is the name in a quoted string matching
// ~^(["'])([a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*)\1$~.
func clgPropertyName(v string) (string, bool) {
	if len(v) < 3 || v[0] != '"' && v[0] != '\'' || v[len(v)-1] != v[0] {
		return "", false
	}
	name := v[1 : len(v)-1]
	for k := 0; k < len(name); k++ {
		c := name[k]
		if c >= 0x7f || c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || k > 0 && c >= '0' && c <= '9' {
			continue
		}
		return "", false
	}
	return name, true
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ClassConstantVisibilitySniff.php
//
// SlevomatClassConstantVisibility adds "public" to a class constant without a
// visibility when fixable is on.
type SlevomatClassConstantVisibility struct {
	fixable bool
}

func (SlevomatClassConstantVisibility) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ClassConstantVisibilitySniff`
}

func (SlevomatClassConstantVisibility) SourceURL() string {
	return fixer.SourceURLFor(SlevomatClassConstantVisibility{}.Name())
}

func (r SlevomatClassConstantVisibility) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["fixable"].(bool); ok {
		r.fixable = v
	}
	return r
}

func (r SlevomatClassConstantVisibility) Fix(s *tokens.Stream) bool {
	if !r.fixable {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if !f.isKeyword(i, "const") {
				continue
			}
			if !f.isOOScopeOwner(f.clgInnermostCondition(i)) {
				continue
			}
			visibility := f.prevEffective(i - 1)
			if f.isKeyword(visibility, "final") {
				visibility = f.prevEffective(visibility - 1)
			}
			if f.isKeyword(visibility, "public") || f.isKeyword(visibility, "protected") || f.isKeyword(visibility, "private") {
				continue
			}
			f.beginChangeset()
			f.addBefore(i, "public ")
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/DisallowStringExpressionPropertyFetchSniff.php
//
// SlevomatDisallowStringExpressionPropertyFetch turns $a->{'b'} into $a->b.
type SlevomatDisallowStringExpressionPropertyFetch struct{}

func (SlevomatDisallowStringExpressionPropertyFetch) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\DisallowStringExpressionPropertyFetchSniff`
}

func (SlevomatDisallowStringExpressionPropertyFetch) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowStringExpressionPropertyFetch{}.Name())
}

func (SlevomatDisallowStringExpressionPropertyFetch) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if !f.is(i, "->") {
				continue
			}
			open := f.nextEffective(i+1, len(f.toks))
			if !f.is(open, "{") || f.toks[open].closer < 0 {
				continue
			}
			closeIdx := f.toks[open].closer
			onlyStrings := true
			for j := open + 1; j < closeIdx; j++ {
				if t := f.toks[j]; t.kind != token.String || t.interp || t.heredoc {
					onlyStrings = false
					break
				}
			}
			if !onlyStrings {
				continue
			}
			if f.is(f.nextEffective(closeIdx+1, len(f.toks)), "(") {
				continue
			}
			name, ok := clgPropertyName(f.toks[open+1].content)
			if !ok {
				continue
			}
			f.beginChangeset()
			f.change(open, closeIdx, name)
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/UselessLateStaticBindingSniff.php
//
// SlevomatUselessLateStaticBinding replaces "static::" by "self::" inside a
// final class.
type SlevomatUselessLateStaticBinding struct{}

func (SlevomatUselessLateStaticBinding) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\UselessLateStaticBindingSniff`
}

func (SlevomatUselessLateStaticBinding) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessLateStaticBinding{}.Name())
}

func (SlevomatUselessLateStaticBinding) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.clgStaticDoubleColon(i) < 0 {
				continue
			}
			c := f.clgOOCondition(i)
			if c < 0 || !f.isKeyword(f.prevEffective(c-1), "final") {
				continue
			}
			f.beginChangeset()
			f.replace(i, "self")
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/DisallowLateStaticBindingForConstantsSniff.php
//
// SlevomatDisallowLateStaticBindingForConstants replaces "static::" by
// "self::" in a constant fetch.
type SlevomatDisallowLateStaticBindingForConstants struct{}

func (SlevomatDisallowLateStaticBindingForConstants) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\DisallowLateStaticBindingForConstantsSniff`
}

func (SlevomatDisallowLateStaticBindingForConstants) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowLateStaticBindingForConstants{}.Name())
}

func (SlevomatDisallowLateStaticBindingForConstants) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			colon := f.clgStaticDoubleColon(i)
			if colon < 0 {
				continue
			}
			// PHP_CodeSniffer makes a keyword after "::" a T_STRING
			name := f.nextEffective(colon+1, len(f.toks))
			if name < 0 || f.toks[name].kind != token.Ident && f.toks[name].kind != token.Keyword {
				continue
			}
			if strings.EqualFold(f.toks[name].content, "class") {
				continue
			}
			if f.is(f.nextEffective(name+1, len(f.toks)), "(") {
				continue
			}
			f.beginChangeset()
			f.replace(i, "self")
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/RequireAbstractOrFinalSniff.php
//
// SlevomatRequireAbstractOrFinal makes a class neither abstract nor final
// final.
type SlevomatRequireAbstractOrFinal struct{}

func (SlevomatRequireAbstractOrFinal) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\RequireAbstractOrFinalSniff`
}

func (SlevomatRequireAbstractOrFinal) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireAbstractOrFinal{}.Name())
}

func (SlevomatRequireAbstractOrFinal) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if !f.isKeyword(i, "class") || f.clgIsAnonClass(i) {
				continue
			}
			prev := f.prevEffective(i - 1)
			if f.isKeyword(prev, "readonly") {
				prev = f.prevEffective(prev - 1)
			}
			if f.isKeyword(prev, "abstract") || f.isKeyword(prev, "final") {
				continue
			}
			f.beginChangeset()
			f.addBefore(i, "final ")
			f.endChangeset()
		}
	})
}
