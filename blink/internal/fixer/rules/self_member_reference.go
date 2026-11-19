package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/SelfMemberReferenceSniff.php
//
// SelfMemberReference rewrites a static reference to the enclosing class by name
// to "self::" ("Foo::bar()" in class Foo -> "self::bar()"), lowercases "Self::"
// and removes whitespace around "::" inside a class.
type SelfMemberReference struct{}

func (SelfMemberReference) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Classes\SelfMemberReferenceSniff`
}

func (SelfMemberReference) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/SelfMemberReferenceSniff.php"
}

func (SelfMemberReference) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.is(i, "::") }, func(i int) int {
			if class := f.innermostClassScope(i); class >= 0 {
				f.selfMemberReference(i, class)
			}
			return 0
		})
	})
}

// innermostClassScope returns the "class" keyword of the named class whose body
// is the innermost class, anonymous class or closure scope holding i, or -1.
func (f *slvFile) innermostClassScope(i int) int {
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		owner := f.braceOwner(open)
		if owner < 0 {
			continue
		}
		switch {
		case f.isKeyword(owner, "class"):
			if f.isAnonClass(owner) {
				return -1
			}
			return owner
		case f.isKeyword(owner, "function") && f.isClosure(owner):
			return -1
		}
	}
	return -1
}

// isAnonClass tells "new class" from a class declaration.
func (f *slvFile) isAnonClass(class int) bool {
	n := f.nextEffective(class+1, len(f.toks))
	return n < 0 || f.toks[n].kind != token.Ident
}

func (f *slvFile) selfMemberReference(i, class int) {
	called := f.prevEffective(i - 1)
	if called < 0 {
		return
	}
	t := f.toks[called]
	lower := strings.ToLower(t.orig)
	switch {
	case t.kind == token.Ident && lower == "self":
		if t.orig != "self" {
			f.replaceToken(called, "self")
			return
		}
	case t.kind == token.Ident && lower != "parent":
		ns := f.namespaceBefore(class)
		if ns == `\` {
			ns = ""
		}
		name := ""
		if n := f.nextEffective(class+1, len(f.toks)); n >= 0 {
			name = f.toks[n].orig
		}
		var inline string
		switch {
		case strings.HasPrefix(t.orig, `\`):
			inline = t.orig
		case strings.HasPrefix(lower, `namespace\`):
			inline = ns + t.orig[len("namespace"):]
		default:
			inline = ns + `\` + t.orig
		}
		if ns+`\`+name == inline {
			f.beginChangeset()
			f.replaceToken(called, "")
			f.replaceToken(i, "self::")
			f.endChangeset()
			return
		}
	}
	if i > 0 && f.toks[i-1].kind == token.Whitespace {
		f.beginChangeset()
		for j := i - 1; j >= 0 && f.toks[j].kind == token.Whitespace; j-- {
			f.replaceToken(j, "")
		}
		f.endChangeset()
	}
	if i+1 < len(f.toks) && f.toks[i+1].kind == token.Whitespace {
		f.beginChangeset()
		for j := i + 1; j < len(f.toks) && f.toks[j].kind == token.Whitespace; j++ {
			f.replaceToken(j, "")
		}
		f.endChangeset()
	}
}

// namespaceBefore is the sniff's getNamespaceName: "\" plus the name of the
// last namespace declaration before i, or "\" for none.
func (f *slvFile) namespaceBefore(i int) string {
	for j := i - 1; j >= 0; j-- {
		if !f.isKeyword(j, "namespace") {
			continue
		}
		if n := f.nextEffective(j+1, len(f.toks)); n >= 0 && f.toks[n].kind == token.Ident && !strings.HasPrefix(f.toks[n].orig, `\`) {
			return `\` + f.toks[n].orig
		}
		return `\`
	}
	return `\`
}
