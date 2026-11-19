package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Namespaces/FullyQualifiedClassNameInAnnotationSniff.php

// SlevomatFullyQualifiedClassNameInAnnotation writes the class and constant
// names in annotations fully qualified.
type SlevomatFullyQualifiedClassNameInAnnotation struct {
	ignoredAnnotationNames []string
}

func (SlevomatFullyQualifiedClassNameInAnnotation) Name() string {
	return `SlevomatCodingStandard\Sniffs\Namespaces\FullyQualifiedClassNameInAnnotationSniff`
}

func (SlevomatFullyQualifiedClassNameInAnnotation) SourceURL() string {
	return fixer.SourceURLFor(SlevomatFullyQualifiedClassNameInAnnotation{}.Name())
}

func (r SlevomatFullyQualifiedClassNameInAnnotation) WithConfig(config map[string]any) fixer.Fixer {
	r.ignoredAnnotationNames = nil
	if names, ok := config["ignoredAnnotationNames"].([]any); ok {
		for _, n := range names {
			if s, ok := n.(string); ok && strings.TrimSpace(s) != "" {
				r.ignoredAnnotationNames = append(r.ignoredAnnotationNames, strings.TrimSpace(s))
			}
		}
	}
	return r
}

func (r SlevomatFullyQualifiedClassNameInAnnotation) Fix(s *tokens.Stream) bool {
	aborted := false
	src := s.Render()
	changed := slvRun(s, func(f *slvFile) {
		if aborted {
			return
		}
		d := newSlvDocs(f)
		f.eachToken(f.isDocOpen, func(open int) int {
			if !aborted && !r.process(d, open) {
				aborted = true
			}
			return 0
		})
	})
	if aborted && changed {
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(src))
		return false
	}
	return changed
}

// process fixes one doc comment; it is false where upstream throws.
func (r SlevomatFullyQualifiedClassNameInAnnotation) process(d *slvDocs, open int) bool {
	for _, a := range d.annotations(open, "") {
		identifiers := pdNodesByType(a.node, "IdentifierTypeNode")
		if slices.Contains(r.ignoredAnnotationNames, a.name) {
			continue
		}
		for _, n := range identifiers {
			name := n.name
			lower := strings.ToLower(name)
			if slvTypeHintIsSimple(lower) || slvTypeHintIsSimpleUnofficial(lower) || !slvTypeHintIsTypeName(name) ||
				d.isTypeDefinedInAnnotation(open, name) {
				continue
			}
			fq := d.useStatements().resolveClassName(d.f, name, open)
			if fq == name {
				continue
			}
			if !d.fix(open, n, fq) {
				return false
			}
		}
		for _, n := range pdNodesByType(a.node, "ConstFetchNode") {
			isClassConstant := n.className != ""
			name := n.name
			if isClassConstant {
				name = n.className
			}
			if name == "self" {
				continue
			}
			var fq string
			if isClassConstant {
				fq = d.useStatements().resolveClassName(d.f, name, open)
			} else {
				fq = d.resolveConstantName(name, open)
			}
			if fq == name {
				continue
			}
			replacement := fq
			if isClassConstant {
				replacement = fq + "::" + n.name
			}
			if !d.fix(open, n, replacement) {
				return false
			}
		}
	}
	return true
}

var slvTypeHintSimpleList = []string{"int", "integer", "false", "float", "string", "bool", "boolean",
	"callable", "self", "array", "iterable", "void", "never"}

// slvTypeHintIsSimple is TypeHintHelper::isSimpleTypeHint.
func slvTypeHintIsSimple(t string) bool { return slices.Contains(slvTypeHintSimpleList, t) }

var slvTypeHintUnofficialList = []string{"null", "mixed", "scalar", "numeric", "true", "object",
	"resource", "static", "$this", "array-key", "list", "non-empty-array", "non-empty-list", "empty",
	"positive-int", "non-positive-int", "negative-int", "non-negative-int", "literal-int", "int-mask",
	"min", "max", "callable-array", "callable-string"}

// slvTypeHintIsSimpleUnofficial is TypeHintHelper::isSimpleUnofficialTypeHints.
func slvTypeHintIsSimpleUnofficial(t string) bool {
	return slices.Contains(slvTypeHintUnofficialList, t) || strings.HasSuffix(strings.ToLower(t), "-string")
}

// slvTypeHintIsTypeName is TypeHelper::isTypeName: backslash separated PHP names,
// optionally fully qualified.
func slvTypeHintIsTypeName(name string) bool {
	name = strings.TrimPrefix(name, `\`)
	for part := range strings.SplitSeq(name, `\`) {
		if part == "" || pdIsDigit(part[0]) {
			return false
		}
		for i := range len(part) {
			if c := part[i]; !pdIsLetter(c) && !pdIsDigit(c) && c != '_' && c < 0x7f {
				return false
			}
		}
	}
	return true
}
