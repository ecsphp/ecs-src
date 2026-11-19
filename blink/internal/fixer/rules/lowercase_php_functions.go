package rules

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpBuiltinFunctionsList is PHP's get_defined_functions()['internal'],
// regenerated with bin/php-builtin-functions.sh.
//
//go:embed php_builtin_functions.txt
var phpBuiltinFunctionsList string

var phpBuiltinFunctions = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	for name := range strings.SplitSeq(phpBuiltinFunctionsList, "\n") {
		if name = strings.TrimSpace(name); name != "" {
			m[name] = true
		}
	}
	return m
})

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/LowercasePHPFunctionsSniff.php
//
// LowercasePHPFunctions lowercases calls to PHP's built-in functions
// ("Array_Keys($a)" -> "array_keys($a)") and their "use function" imports.
// Methods, declarations, instantiations and attribute arguments are left alone.
type LowercasePHPFunctions struct{}

func (LowercasePHPFunctions) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\LowercasePHPFunctionsSniff`
}

func (LowercasePHPFunctions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/LowercasePHPFunctionsSniff.php"
}

func (LowercasePHPFunctions) Fix(s *tokens.Stream) bool {
	changed := false
	inAttribute := false
	for i := range s.Len() {
		t := s.At(i)
		switch t.Kind {
		case token.AttributeStart:
			inAttribute = true
			continue
		case token.AttributeEnd:
			inAttribute = false
			continue
		}
		if t.Kind != token.Ident || inAttribute {
			continue
		}
		lower := strings.ToLower(t.Value)
		if lower == t.Value || !phpBuiltinFunctions()[lower] {
			continue
		}
		start, ok := lowercasePHPFunctionNameStart(s, i)
		if !ok || (i+1 < s.Len() && s.At(i+1).Value == `\`) {
			continue
		}
		next := phpcsNextNonEmpty(s, i+1)
		if next >= s.Len() {
			continue
		}
		prev := lowercasePHPFunctionPrev(s, start-1)
		if s.At(next).Value != "(" {
			// "use function StrLen;" imports a native function
			if start != i || prev < 0 || s.At(prev).Value != "function" {
				continue
			}
			if pp := phpcsPrevNonEmpty(s, prev-1); pp < 0 || !strings.EqualFold(s.At(pp).Value, "use") {
				continue
			}
		} else if prev >= 0 {
			switch strings.ToLower(s.At(prev).Value) {
			case "function", "new", "->", "?->", "::":
				continue
			}
		}
		s.SetValue(i, lower)
		changed = true
	}
	return changed
}

// lowercasePHPFunctionNameStart is where the name at i begins: i itself, or the
// leading "\" of a fully qualified "\strlen". A name inside a qualified or
// relative name ("Foo\strlen", "namespace\strlen", "strlen\Foo") is not a candidate.
func lowercasePHPFunctionNameStart(s *tokens.Stream, i int) (int, bool) {
	if i == 0 || s.At(i-1).Kind != token.Punct || s.At(i-1).Value != `\` {
		return i, true
	}
	if i >= 2 {
		if k := s.At(i - 2).Kind; k == token.Ident || k == token.Keyword {
			return 0, false
		}
	}
	return i - 1, true
}

// lowercasePHPFunctionPrev is the previous token before i skipping comments,
// whitespace and "&" (a by-reference function declaration).
func lowercasePHPFunctionPrev(s *tokens.Stream, i int) int {
	for j := i; j >= 0; j-- {
		t := s.At(j)
		if isPhpcsEmpty(t) || (t.Kind == token.Punct && t.Value == "&") {
			continue
		}
		return j
	}
	return -1
}
