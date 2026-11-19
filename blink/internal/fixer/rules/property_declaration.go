package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Classes/PropertyDeclarationSniff.php
//
// PropertyDeclaration puts one space after a property type and orders the
// modifiers: final and abstract first, then the visibility, the set-visibility,
// static and readonly.
type PropertyDeclaration struct{}

func (PropertyDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Classes\PropertyDeclarationSniff`
}

func (PropertyDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Classes/PropertyDeclarationSniff.php"
}

func (PropertyDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.toks[i].kind == token.Variable && phpcsIsMemberVar(f, i) {
				propertyDeclaration(f, i)
			}
		}
	})
}

// phpcsIsMemberVar is AbstractVariableSniff's member var test plus the class
// (not enum) check of File::getMemberProperties.
func phpcsIsMemberVar(f *slvFile, i int) bool {
	ooOpen := -1
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		owner := f.braceOwner(open)
		if owner < 0 {
			continue
		}
		kw := strings.ToLower(f.toks[owner].orig)
		if kw == "function" {
			return false
		}
		if kw == "class" || kw == "interface" || kw == "trait" {
			ooOpen = open
			break
		}
		if kw == "enum" {
			return false
		}
	}
	if ooOpen < 0 {
		return false
	}
	for p := f.enclosing(i); p > ooOpen; p = f.enclosing(p) {
		if !f.is(p, "(") {
			continue
		}
		owner := f.prevEffective(p - 1)
		if f.isKeyword(owner, "function") || f.isKeyword(owner, "use") {
			return false
		}
		if owner >= 0 && f.toks[owner].kind == token.Ident {
			fn := f.prevEffective(owner - 1)
			if f.is(fn, "&") {
				fn = f.prevEffective(fn - 1)
			}
			if f.isKeyword(fn, "function") {
				return false
			}
		}
	}
	return true
}

func (f *slvFile) isScopeKeyword(i int) bool {
	return f.isKeyword(i, "public") || f.isKeyword(i, "protected") || f.isKeyword(i, "private")
}

// isSetScope tells a set-visibility keyword ("private(set)").
func (f *slvFile) isSetScope(i int) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Keyword && strings.HasSuffix(strings.ToLower(f.toks[i].orig), "(set)")
}

// isScopeModifier is Tokens::SCOPE_MODIFIERS.
func (f *slvFile) isScopeModifier(i int) bool { return f.isScopeKeyword(i) || f.isSetScope(i) }

func (f *slvFile) prevMatching(match func(int) bool, start int) int {
	for i := start; i >= 0; i-- {
		if match(i) {
			return i
		}
	}
	return -1
}

type phpcsMemberProps struct {
	scopeSpecified, setScope, static, readonly, final, abstract bool
	typeEnd                                                     int
}

func phpcsMemberProperties(f *slvFile, ptr int) phpcsMemberProps {
	p := phpcsMemberProps{typeEnd: -1}
	start := f.prevMatching(func(i int) bool {
		return f.is(i, ";") || f.is(i, "{") || f.is(i, "}") || f.toks[i].kind == token.AttributeEnd
	}, ptr-1)
	i := start + 1
mods:
	for ; i < ptr; i++ {
		switch {
		case f.isEmpty(i):
		case f.isSetScope(i):
			p.setScope = true
		case f.isScopeKeyword(i):
			p.scopeSpecified = true
		case f.isKeyword(i, "static"):
			p.static = true
		case f.isKeyword(i, "readonly"):
			p.readonly = true
		case f.isKeyword(i, "final"):
			p.final = true
		case f.isKeyword(i, "abstract"):
			p.abstract = true
		case f.isKeyword(i, "var"):
		default:
			break mods
		}
	}
	for ; i < ptr; i++ {
		t := f.toks[i]
		if t.kind == token.Variable {
			break
		}
		if t.kind == token.Ident || f.isKeyword(i, "array") || f.isKeyword(i, "callable") ||
			t.kind == token.Punct && (t.orig == "|" || t.orig == "&" || t.orig == "(" || t.orig == ")") {
			p.typeEnd = i
		}
	}
	return p
}

func propertyDeclaration(f *slvFile, ptr int) {
	prev := f.prevMatching(func(i int) bool {
		return f.toks[i].kind == token.Variable || f.isScopeModifier(i) || f.is(i, ";") || f.is(i, "{") ||
			f.isKeyword(i, "var") || f.isKeyword(i, "static") || f.isKeyword(i, "readonly") ||
			f.isKeyword(i, "final") || f.isKeyword(i, "abstract")
	}, ptr-1)
	if prev >= 0 && f.toks[prev].kind == token.Variable {
		return
	}
	info := phpcsMemberProperties(f, ptr)

	if te := info.typeEnd; te >= 0 {
		switch {
		case !f.isWS(te + 1):
			f.addContent(te, " ")
		case f.toks[te+1].content != " ":
			next := f.nextNot(f.isWS, te+1, len(f.toks))
			if f.nextNot(f.isEmpty, te+1, len(f.toks)) == next {
				if f.toks[next].line != f.toks[te].line {
					f.beginChangeset()
					for x := te + 1; x < next; x++ {
						f.replaceToken(x, "")
					}
					f.addContent(te, " ")
					f.endChangeset()
				} else {
					f.replaceToken(te+1, " ")
				}
			}
		}
	}

	hasVisibility := info.scopeSpecified || info.setScope
	lastVisibility := f.prevMatching(f.isScopeModifier, ptr-1)
	firstVisibility := lastVisibility

	// moveBefore drops the modifier at mod (and the whitespace after it) and
	// puts it before the modifier at to.
	moveBefore := func(mod, to int) {
		f.beginChangeset()
		for i := mod + 1; f.isWS(i); i++ {
			f.replaceToken(i, "")
		}
		content := f.toks[mod].orig
		f.replaceToken(mod, "")
		f.addContentBefore(to, content+" ")
		f.endChangeset()
	}
	moveAfter := func(mod, to int) {
		f.beginChangeset()
		for i := mod + 1; f.isWS(i); i++ {
			f.replaceToken(i, "")
		}
		content := f.toks[mod].orig
		f.replaceToken(mod, "")
		f.addContent(to, " "+content)
		f.endChangeset()
	}

	if info.scopeSpecified && info.setScope {
		scopePtr := f.prevMatching(f.isScopeKeyword, ptr-1)
		setScopePtr := f.prevMatching(f.isSetScope, ptr-1)
		if scopePtr > setScopePtr {
			moveBefore(scopePtr, setScopePtr)
		}
		firstVisibility = min(scopePtr, setScopePtr)
	}
	if hasVisibility && info.final {
		if finalPtr := f.prevMatching(func(i int) bool { return f.isKeyword(i, "final") }, ptr-1); finalPtr > firstVisibility {
			moveBefore(finalPtr, firstVisibility)
		}
	}
	if hasVisibility && info.abstract {
		if abstractPtr := f.prevMatching(func(i int) bool { return f.isKeyword(i, "abstract") }, ptr-1); abstractPtr > firstVisibility {
			moveBefore(abstractPtr, firstVisibility)
		}
	}
	if hasVisibility && info.static {
		if staticPtr := f.prevMatching(func(i int) bool { return f.isKeyword(i, "static") }, ptr-1); lastVisibility > staticPtr {
			moveAfter(staticPtr, lastVisibility)
		}
	}
	if hasVisibility && info.readonly {
		if readonlyPtr := f.prevMatching(func(i int) bool { return f.isKeyword(i, "readonly") }, ptr-1); lastVisibility > readonlyPtr {
			moveAfter(readonlyPtr, lastVisibility)
		}
	}
}
