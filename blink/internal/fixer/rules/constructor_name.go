package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/ConstructorNameSniff.php
//
// ConstructorName reports PHP4 style constructors and PHP4 style calls to the
// parent constructor.
type ConstructorName struct{}

func (ConstructorName) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\NamingConventions\ConstructorNameSniff`
}

func (ConstructorName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/ConstructorNameSniff.php"
}

func (ConstructorName) Fix(*tokens.Stream) bool { return false }

func (c ConstructorName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, c.Name())
	f := a.f
	currentClass := ""
	var functionList []string
	a.run(phpcsSet("class", "anon_class"), phpcsSet("function"), func(i, scope int) {
		if !a.deepest(i, scope) {
			return
		}
		className := "[Anonymous Class]"
		if a.code(scope) != "anon_class" {
			className = strings.ToLower(a.declarationName(scope))
		}
		if className != currentClass {
			functionList = constructorNameFunctions(a, scope)
			currentClass = className
		}
		methodName := strings.ToLower(a.declarationName(i))
		if methodName == "" {
			return
		}
		if methodName == className {
			if !slices.Contains(functionList, "__construct") {
				a.addError(i, "OldStyle", `PHP4 style constructors are not allowed; use "__construct()" instead`)
			}
		} else if methodName != "__construct" {
			return
		}
		if a.sc.opener[i] < 0 || a.sc.closer[i] < 0 {
			return
		}
		parent, ok := constructorNameExtended(a, scope)
		if !ok {
			return
		}
		parentLc := strings.ToLower(parent)
		end := a.sc.closer[i]
		for start := a.sc.opener[i]; ; {
			colon := start + 1
			for colon < end && !f.is(colon, "::") {
				colon++
			}
			if colon >= end {
				break
			}
			next := f.nextEffective(colon+1, len(f.toks))
			if next < 0 {
				break
			}
			start = next
			if !a.isTString(next) || strings.ToLower(f.toks[next].orig) != parentLc {
				continue
			}
			prev := f.prevEffective(colon - 1)
			prevLc := strings.ToLower(f.toks[max(prev, 0)].orig)
			if prev >= 0 && (prevLc == "parent" || prevLc == "self" || prevLc == "static" || (a.isTString(prev) && prevLc == parentLc)) {
				a.addError(next, "OldStyleCall", `PHP4 style calls to parent constructors are not allowed; use "parent::__construct()" instead`)
			}
		}
	}, nil)
	return a.errors
}

// constructorNameFunctions is loadFunctionNamesInScope: the lowercase names of
// the functions declared directly in the class body.
func constructorNameFunctions(a *abstractScopeFile, scope int) []string {
	var names []string
	for i := a.sc.opener[scope] + 1; i < a.sc.closer[scope]; i++ {
		if a.code(i) != "function" {
			continue
		}
		name := a.declarationName(i)
		if name == "" {
			continue
		}
		names = append(names, strings.ToLower(name))
		if a.sc.closer[i] >= 0 {
			i = a.sc.closer[i]
		}
	}
	return names
}

// constructorNameExtended is File::findExtendedClassName.
func constructorNameExtended(a *abstractScopeFile, scope int) (string, bool) {
	f := a.f
	opener := a.sc.opener[scope]
	if opener < 0 {
		return "", false
	}
	extends := scope
	for extends < opener && !f.isKeyword(extends, "extends") {
		extends++
	}
	if extends >= opener {
		return "", false
	}
	var name strings.Builder
	for j := extends + 1; j <= opener; j++ {
		if k := f.toks[j].kind; k != token.Whitespace && k != token.Ident {
			break
		}
		name.WriteString(f.toks[j].orig)
	}
	trimmed := strings.Trim(name.String(), " \t\n\r\x00\x0b")
	return trimmed, trimmed != ""
}
