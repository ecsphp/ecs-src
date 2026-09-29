package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/FunctionToConstantFixer.php
//
// FunctionToConstant replaces a configured function call with its constant
// equivalent: pi() -> M_PI, phpversion() -> PHP_VERSION, php_sapi_name() ->
// PHP_SAPI, get_called_class() -> static::class, get_class($this) ->
// static::class.
type FunctionToConstant struct {
	// Functions limits the converted functions; nil means the built-in default set.
	Functions []string
}

func (f FunctionToConstant) WithConfig(config map[string]any) fixer.Fixer {
	switch list := config["functions"].(type) {
	case []string:
		f.Functions = append([]string{}, list...)
	case []any:
		f.Functions = []string{}
		for _, v := range list {
			if name, ok := v.(string); ok {
				f.Functions = append(f.Functions, name)
			}
		}
	}
	return f
}

// functionToConstantEnabled reports whether the option key is enabled.
func (f FunctionToConstant) enabled(key string) bool {
	if f.Functions == nil {
		return true
	}
	return slices.Contains(f.Functions, key)
}

func (FunctionToConstant) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\FunctionToConstantFixer`
}

func (FunctionToConstant) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/FunctionToConstantFixer.php"
}

var noArgConstants = map[string]string{
	"pi":            "M_PI",
	"phpversion":    "PHP_VERSION",
	"php_sapi_name": "PHP_SAPI",
}

func (f FunctionToConstant) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident {
			continue
		}
		name := strings.ToLower(s.At(i).Value)
		_, isNoArg := noArgConstants[name]
		if !isNoArg && name != "get_called_class" && name != "get_class" {
			continue
		}
		if name != "get_class" && !f.enabled(name) {
			continue
		}
		// a plain function call: not a method/static call, not a declaration,
		// and optionally prefixed by a single leading "\"
		start := i
		if prev, ok := prevSignificant(s, i); ok {
			switch {
			case prev.Kind == token.Punct && (prev.Value == "->" || prev.Value == "?->" || prev.Value == "::"):
				continue
			case prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "function"):
				continue
			case prev.Kind == token.Punct && prev.Value == `\`:
				pj := prevSignificantIndex(s, i)
				if before, ok := prevSignificant(s, pj); ok && (before.Kind == token.Ident || before.Value == `\`) {
					continue // namespaced call, not the global function
				}
				start = pj
			}
		}
		open := nextSignificantIndex(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		var repl []token.Token
		switch {
		case isNoArg:
			if nextSignificantIndex(s, open) != closeIdx {
				continue // has arguments
			}
			repl = []token.Token{{Kind: token.Ident, Value: noArgConstants[name]}}
		case name == "get_called_class":
			if nextSignificantIndex(s, open) != closeIdx {
				continue
			}
			repl = staticClassTokens()
		case name == "get_class":
			arg := nextSignificantIndex(s, open)
			if arg == closeIdx {
				// get_class() without arguments -> self::class, only when opted in
				if f.Functions == nil || !f.enabled("get_class") {
					continue
				}
				repl = functionToConstantSelfClassTokens()
				break
			}
			if !f.enabled("get_class_this") {
				continue
			}
			if arg < 0 || s.At(arg).Kind != token.Variable || !strings.EqualFold(s.At(arg).Value, "$this") {
				continue
			}
			if nextSignificantIndex(s, arg) != closeIdx {
				continue // more than just $this
			}
			repl = staticClassTokens()
		}
		s.ReplaceRange(start, closeIdx, repl)
		changed = true
		i = start
	}
	return changed
}

func staticClassTokens() []token.Token {
	return []token.Token{
		{Kind: token.Keyword, Value: "static"},
		{Kind: token.Punct, Value: "::"},
		{Kind: token.Ident, Value: "class"},
	}
}

func functionToConstantSelfClassTokens() []token.Token {
	return []token.Token{
		{Kind: token.Keyword, Value: "self"},
		{Kind: token.Punct, Value: "::"},
		{Kind: token.Ident, Value: "class"},
	}
}
