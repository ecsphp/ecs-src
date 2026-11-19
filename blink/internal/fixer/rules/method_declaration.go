package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/MethodDeclarationSniff.php
//
// MethodDeclaration orders method modifiers: abstract and final before the
// visibility, static after it: "public final static function" -> "final public static function".
type MethodDeclaration struct{}

func (MethodDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Methods\MethodDeclarationSniff`
}

func (MethodDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/MethodDeclarationSniff.php"
}

func (MethodDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.isKeyword(i, "function") && !f.isClosure(i) && phpcsInOOScope(f, i) {
				methodDeclaration(f, i)
			}
		}
	})
}

// phpcsInOOScope reports whether the deepest PHP_CodeSniffer condition of i is
// a class, interface, trait or enum body.
func phpcsInOOScope(f *slvFile, i int) bool {
	open := f.enclosingScope(i)
	if open < 0 {
		return false
	}
	owner := f.braceOwner(open)
	if owner < 0 {
		return false
	}
	switch strings.ToLower(f.toks[owner].orig) {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

func phpcsIsMethodModifier(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "public", "protected", "private", "abstract", "static", "final":
		return true
	}
	return false
}

func methodDeclaration(f *slvFile, ptr int) {
	name := f.nextEffective(ptr+1, len(f.toks))
	if f.is(name, "&") {
		name = f.nextEffective(name+1, len(f.toks))
	}
	if name < 0 || f.toks[name].kind == token.Punct {
		return
	}

	prev := ptr - 1
	for prev >= 0 && (f.isEmpty(prev) || phpcsIsMethodModifier(f, prev)) {
		prev--
	}
	visibility, static, abstract, final := 0, 0, 0, 0
	for p := ptr - 1; p > prev; p-- {
		if !phpcsIsMethodModifier(f, p) {
			continue
		}
		switch strings.ToLower(f.toks[p].orig) {
		case "static":
			static = p
		case "abstract":
			abstract = p
		case "final":
			final = p
		default:
			visibility = p
		}
	}

	var order []int
	fixes := map[int]string{}
	set := func(i int, v string) {
		if _, ok := fixes[i]; !ok {
			order = append(order, i)
		}
		fixes[i] = v
	}
	prefix := func(mod int, word string) {
		set(mod, "")
		set(mod+1, "")
		if v, ok := fixes[visibility]; ok {
			set(visibility, word+" "+v)
		} else {
			set(visibility, word+" "+f.toks[visibility].orig)
		}
	}
	if visibility != 0 && final > visibility {
		prefix(final, "final")
	}
	if visibility != 0 && abstract > visibility {
		prefix(abstract, "abstract")
	}
	if static != 0 && static < visibility {
		set(static, "")
		set(static+1, "")
		if v, ok := fixes[visibility]; ok {
			set(visibility, v+" static")
		} else {
			set(visibility, f.toks[visibility].orig+" static")
		}
	}
	if len(order) == 0 {
		return
	}
	f.beginChangeset()
	for _, i := range order {
		f.replaceToken(i, fixes[i])
	}
	f.endChangeset()
}
