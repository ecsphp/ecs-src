package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidFunctionNameSniff.php
//
// PEARValidFunctionName reports function and method names that break the PEAR
// naming rules: camel caps, an underscore prefix only on private methods, and
// a capitalised package prefix on functions.
type PEARValidFunctionName struct{}

func (PEARValidFunctionName) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\NamingConventions\ValidFunctionNameSniff`
}

func (PEARValidFunctionName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/NamingConventions/ValidFunctionNameSniff.php"
}

func (PEARValidFunctionName) Fix(*tokens.Stream) bool { return false }

func (p PEARValidFunctionName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, p.Name())
	a.run(abstractScopeOO, phpcsSet("function"), func(i, scope int) {
		pearValidFunctionNameMethod(a, i, scope)
	}, func(i int) {
		pearValidFunctionNameFunction(a, i)
	})
	return a.errors
}

// pearValidFunctionNameMethod is the sniff's processTokenWithinScope.
func pearValidFunctionNameMethod(a *abstractScopeFile, i, scope int) {
	if !a.deepest(i, scope) {
		return
	}
	methodName := a.declarationName(i)
	if methodName == "" {
		return
	}
	className := a.className(scope)
	errorData := className + "::" + methodName
	methodLc, classLc := strings.ToLower(methodName), strings.ToLower(className)
	if abstractScopeMagicPrefixed(methodName) {
		if abstractScopeMagicMethods[methodLc[2:]] {
			return
		}
		a.addError(i, "MethodDoubleUnderscore", `Method name "%s" is invalid; only PHP magic methods should be prefixed with a double underscore`, errorData)
	}
	if methodLc == classLc || methodLc == "_"+classLc {
		return
	}
	props := a.methodProps(i)
	isPublic := props.scope != "private"
	if !isPublic && methodName[0] != '_' {
		a.addError(i, "PrivateNoUnderscore", `Private method name "%s" must be prefixed with an underscore`, errorData)
	}
	if isPublic && props.scopeSpecified && methodName[0] == '_' {
		a.addError(i, "PublicUnderscore", `%s method name "%s" must not be prefixed with an underscore`, abstractScopeUcfirst(props.scope), errorData)
	}
	if abstractScopeIsCamelCaps(strings.TrimLeft(methodName, "_"), false) {
		return
	}
	if props.scopeSpecified {
		a.addError(i, "ScopeNotCamelCaps", `%s method name "%s" is not in camel caps format`, abstractScopeUcfirst(props.scope), errorData)
	} else {
		a.addError(i, "NotCamelCaps", `Method name "%s" is not in camel caps format`, errorData)
	}
}

// pearValidFunctionNameFunction is the sniff's processTokenOutsideScope.
func pearValidFunctionNameFunction(a *abstractScopeFile, i int) {
	name := a.declarationName(i)
	if strings.TrimLeft(name, "_") == "" {
		return
	}
	if abstractScopeMagicPrefixed(name) {
		if strings.ToLower(name[2:]) == "autoload" {
			return
		}
		a.addError(i, "FunctionDoubleUnderscore", `Function name "%s" is invalid; only PHP magic methods should be prefixed with a double underscore`, name)
	}
	packagePart, camelCapsPart := "", name
	if pos := strings.LastIndexByte(name, '_'); pos >= 0 {
		packagePart = strings.TrimLeft(name[:pos], "_")
		camelCapsPart = name[pos+1:]
	}
	upper := func(b byte) byte { return strings.ToUpper(string(b))[0] }
	if packagePart != "" {
		if name[0] == '_' {
			a.addError(i, "FunctionUnderscore", `Function name "%s" is invalid; only private methods should be prefixed with an underscore`, name)
		}
		if name[0] != upper(name[0]) {
			a.addError(i, "FunctionNoCapital", `Function name "%s" is prefixed with a package name but does not begin with a capital letter`, name)
		}
	}
	if strings.Trim(camelCapsPart, " \t\n\r\x00\x0b") == "" {
		a.addError(i, "FunctionInvalid", `Function name "%s" is not valid; name appears incomplete`, name)
		return
	}
	valid := true
	newPackagePart, newCamelCapsPart := packagePart, camelCapsPart
	if !abstractScopeIsCamelCaps(camelCapsPart, false) {
		valid = false
		newCamelCapsPart = strings.ToLower(camelCapsPart[:1]) + camelCapsPart[1:]
	}
	if packagePart != "" {
		var bits []string
		for bit := range strings.SplitSeq(packagePart, "_") {
			if bit != "" && bit != "0" {
				bits = append(bits, bit)
			}
		}
		for _, bit := range bits {
			if bit[0] != upper(bit[0]) {
				newPackagePart = ""
				for _, b := range bits {
					newPackagePart += string(upper(b[0])) + b[1:] + "_"
				}
				valid = false
				break
			}
		}
	}
	if valid {
		return
	}
	newName := newCamelCapsPart
	if newPackagePart != "" {
		newName = strings.TrimRight(newPackagePart, "_") + "_" + newCamelCapsPart
	}
	a.addError(i, "FunctionNameInvalid", `Function name "%s" is invalid; consider "%s" instead`, name, newName)
}
