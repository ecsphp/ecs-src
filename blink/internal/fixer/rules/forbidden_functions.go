package rules

import (
	"fmt"
	"maps"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/ForbiddenFunctionsSniff.php
//
// ForbiddenFunctions reports every call to a function of forbiddenFunctions
// (lowercased name => alternative, or nil when there is none; sizeof and
// delete by default). It fixes nothing. With error false the sniff reports
// warnings, which ECS drops unless the sniff is listed in
// reportSniffClassWarnings, so blink reports nothing.
type ForbiddenFunctions struct {
	forbidden map[string]*string
	warning   bool
}

var forbiddenFunctionsDefaults = map[string]*string{
	"sizeof": new("count"),
	"delete": new("unset"),
}

// forbiddenFunctionsKeywordCalls are the function-like names PHP tokenizes as
// a keyword of their own (T_EVAL, T_EXIT, T_PRINT, ...) rather than T_STRING.
// The sniff listens for those tokens too, and reports them without requiring
// a "(" after the name.
var forbiddenFunctionsKeywordCalls = map[string]bool{
	"array": true, "die": true, "echo": true, "empty": true, "eval": true,
	"exit": true, "include": true, "include_once": true, "isset": true,
	"list": true, "print": true, "require": true, "require_once": true,
	"unset": true, "__halt_compiler": true,
}

// forbiddenFunctionsIgnoredKeywords are the keywords of the sniff's $ignore
// list: next to one of them, the name is not a call to a PHP function.
var forbiddenFunctionsIgnoredKeywords = map[string]bool{
	"function": true, "const": true, "public": true, "private": true,
	"protected": true, "as": true, "new": true, "insteadof": true,
	"implements": true,
}

func (ForbiddenFunctions) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\ForbiddenFunctionsSniff`
}

func (ForbiddenFunctions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/ForbiddenFunctionsSniff.php"
}

func (ForbiddenFunctions) Fix(*tokens.Stream) bool { return false }

func (f ForbiddenFunctions) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["forbiddenFunctions"]; ok {
		f.forbidden = map[string]*string{}
		add := func(name string, alternative any) {
			var alt *string
			if alternative != nil {
				alt = new(fmt.Sprint(alternative))
			}
			f.forbidden[strings.ToLower(name)] = alt
		}
		switch functions := raw.(type) {
		case map[string]any:
			for name, alternative := range functions {
				add(name, alternative)
			}
		case []any:
			// a PHP list: the names are its keys and the entries their alternatives
			for i, alternative := range functions {
				add(fmt.Sprint(i), alternative)
			}
		}
	}
	if isError, ok := config["error"].(bool); ok {
		f.warning = !isError
	}
	return f
}

func (f ForbiddenFunctions) forbiddenFunctions() map[string]*string {
	if f.forbidden == nil {
		return maps.Clone(forbiddenFunctionsDefaults)
	}
	return f.forbidden
}

func (f ForbiddenFunctions) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	forbidden := f.forbiddenFunctions()
	if f.warning || len(forbidden) == 0 {
		return nil
	}

	var errs []fixer.CodingStandardError
	inAttribute := false
	line, lineAt := 1, 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.AttributeStart:
			inAttribute = true
			continue
		case token.AttributeEnd:
			inAttribute = false
			continue
		case token.Ident, token.Keyword:
		default:
			continue
		}

		function := strings.ToLower(t.Value)
		alternative, ok := forbidden[function]
		if !ok || inAttribute {
			continue
		}
		start, ok := forbiddenFunctionsNameStart(s, i)
		if !ok {
			continue
		}
		// "\print" is not PHP; a keyword call is never fully qualified
		fullyQualified := start != i
		keywordCall := !fullyQualified && forbiddenFunctionsKeywordCalls[function]

		prev, next := sigPrev(s, start), sigNext(s, i)
		if forbiddenFunctionsIgnored(s, prev) || forbiddenFunctionsIgnored(s, next) {
			continue
		}
		if !keywordCall && !isPunctVal(s, next, "(") {
			continue
		}

		for ; lineAt < start; lineAt++ {
			line += strings.Count(s.At(lineAt).Value, "\n")
		}
		message := fmt.Sprintf("The use of function %s() is forbidden", function)
		code := "Found"
		if alternative != nil {
			message += fmt.Sprintf("; use %s() instead", *alternative)
			code = "FoundWithAlternative"
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    line,
			Message: message,
			Checker: f.Name() + "." + code,
		})
	}
	return errs
}

// forbiddenFunctionsNameStart returns where the name token PHP_CodeSniffer
// sees for the name at i begins: i for an unqualified name, the leading "\"
// for a fully qualified one. A qualified ("Foo\bar") or relative
// ("namespace\bar") name, or a namespace segment, is a token the sniff does
// not listen for (ok false).
func forbiddenFunctionsNameStart(s *tokens.Stream, i int) (int, bool) {
	if i+1 < s.Len() && isPunctVal(s, i+1, `\`) {
		return 0, false
	}
	if i == 0 || !isPunctVal(s, i-1, `\`) {
		return i, true
	}
	if i >= 2 && s.At(i-2).Kind == token.Ident {
		return 0, false
	}
	return i - 1, true
}

// forbiddenFunctionsIgnored reports whether the token at i is in the sniff's
// $ignore list.
func forbiddenFunctionsIgnored(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	if t.Kind == token.Keyword {
		return forbiddenFunctionsIgnoredKeywords[strings.ToLower(t.Value)]
	}
	return isPunctVal(s, i, "::") || isPunctVal(s, i, "->") || isPunctVal(s, i, "?->")
}
