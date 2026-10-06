package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NewWithParenthesesFixer.php
//
// NewWithParentheses adds parentheses to a parameterless "new": "new Foo" ->
// "new Foo()". Anonymous classes and dynamic "new $var" are left alone.
type NewWithParentheses struct {
	// SkipNamedClass and SkipAnonymousClass disable the named_class/anonymous_class options.
	SkipNamedClass     bool
	SkipAnonymousClass bool
}

func (f NewWithParentheses) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["named_class"].(bool); ok {
		f.SkipNamedClass = !v
	}
	if v, ok := config["anonymous_class"].(bool); ok {
		f.SkipAnonymousClass = !v
	}
	return f
}

func (NewWithParentheses) Name() string {
	return `PhpCsFixer\Fixer\Operator\NewWithParenthesesFixer`
}

func (NewWithParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NewWithParenthesesFixer.php"
}

func (f NewWithParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "new" {
			continue
		}
		j := skipWhitespace(s, i+1)
		if j >= s.Len() {
			continue
		}
		// a static class name (Ident, \-qualified, self/parent/static) or a
		// dynamic class reference (new $var, new $this->prop, new $a::$b)
		nt := s.At(j)
		if nt.Kind == token.Keyword && strings.ToLower(nt.Value) == "class" {
			// anonymous class: "new class extends X" -> "new class() extends X"
			// (ECS's psr12 config sets anonymous_class => true)
			if f.SkipAnonymousClass || nextSignificantValue(s, j) == "(" {
				continue // disabled, or already "new class(...)"
			}
			s.InsertAt(j+1, token.Token{Kind: token.Punct, Value: "("})
			s.InsertAt(j+2, token.Token{Kind: token.Punct, Value: ")"})
			changed = true
			continue
		}
		if f.SkipNamedClass {
			continue
		}
		if nt.Kind == token.Variable {
			k := consumeNewVarRef(s, j)
			if nextSignificantValue(s, k-1) == "(" {
				continue // already has parentheses
			}
			s.InsertAt(k, token.Token{Kind: token.Punct, Value: "("})
			s.InsertAt(k+1, token.Token{Kind: token.Punct, Value: ")"})
			changed = true
			continue
		}
		isName := nt.Kind == token.Ident ||
			(nt.Kind == token.Punct && nt.Value == `\`) ||
			(nt.Kind == token.Keyword && isStaticRef(nt.Value))
		if !isName {
			continue // new (expr), ...
		}
		// consume the class reference (Ident / \ / self-parent-static)
		k := j
		for k < s.Len() {
			c := s.At(k)
			if c.Kind == token.Ident || (c.Kind == token.Punct && c.Value == `\`) ||
				(c.Kind == token.Keyword && isStaticRef(c.Value)) {
				k++
				continue
			}
			break
		}
		// a dynamic class name continues through "::member" / "[subscript]"
		// ("new static::$builder(...)"), so the "(" belongs to that call, not a
		// bare "new static" needing "()"
		k = consumeNewStaticTail(s, k)
		if nextSignificantValue(s, k-1) == "(" {
			continue // already has parentheses
		}
		s.InsertAt(k, token.Token{Kind: token.Punct, Value: "("})
		s.InsertAt(k+1, token.Token{Kind: token.Punct, Value: ")"})
		changed = true
	}
	return changed
}

// consumeNewStaticTail advances past the "::member" and "[subscript]" tail of a
// dynamic class name ("static::$builder", "self::$map[0]"), starting at k just
// after the leading name. It stops before a "(", leaving an existing call intact.
func consumeNewStaticTail(s *tokens.Stream, k int) int {
	for k < s.Len() {
		c := s.At(k)
		if c.Kind == token.Punct && c.Value == "::" {
			m := skipWhitespace(s, k+1)
			if m < s.Len() && (s.At(m).Kind == token.Ident || s.At(m).Kind == token.Variable) {
				k = m + 1
				continue
			}
			break
		}
		if c.Kind == token.Punct && c.Value == "[" {
			cl := s.MatchForward(k)
			if cl < 0 {
				break
			}
			k = cl + 1
			continue
		}
		break
	}
	return k
}

// consumeNewVarRef returns the index just past a dynamic class reference that
// starts with a variable at `start`: "$var", "$this->prop", "$a::$b", "$a[0]".
// It stops before a "(" so an existing constructor/method call is left intact.
func consumeNewVarRef(s *tokens.Stream, start int) int {
	k := start + 1 // past the initial variable
	for k < s.Len() {
		c := s.At(k)
		if c.Kind == token.Punct && (c.Value == "->" || c.Value == "?->" || c.Value == "::") {
			m := skipWhitespace(s, k+1)
			if m < s.Len() && (s.At(m).Kind == token.Ident || s.At(m).Kind == token.Variable) {
				k = m + 1
				continue
			}
			break
		}
		if c.Kind == token.Punct && c.Value == "[" {
			cl := s.MatchForward(k)
			if cl < 0 {
				break
			}
			k = cl + 1
			continue
		}
		break
	}
	return k
}

func isStaticRef(v string) bool {
	switch strings.ToLower(v) {
	case "self", "parent", "static":
		return true
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/SingleLineEmptyBodyFixer.php
//
// SingleLineEmptyBody collapses an empty class, function or closure body to "{}"
// on the declaration line ("function f()\n{\n}" -> "function f() {}").
type SingleLineEmptyBody struct{}

func (SingleLineEmptyBody) Name() string {
	return `PhpCsFixer\Fixer\Basic\SingleLineEmptyBodyFixer`
}

func (SingleLineEmptyBody) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/SingleLineEmptyBodyFixer.php"
}

func (SingleLineEmptyBody) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		switch kind, _ := classifyBrace(s, i); kind {
		case braceClassLike, braceFunctionDecl, braceClosure:
		default:
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		empty := true
		for k := i + 1; k < closeIdx; k++ {
			if s.At(k).Kind != token.Whitespace {
				empty = false
				break
			}
		}
		if !empty || closeIdx == i+1 {
			continue
		}
		// remove the whitespace between { and }
		for k := closeIdx - 1; k > i; k-- {
			s.RemoveAt(k)
		}
		// pull the brace onto the declaration line
		if i > 0 && s.At(i-1).Kind == token.Whitespace && hasNewline(s.At(i-1).Value) {
			s.SetValue(i-1, " ")
		}
		changed = true
	}
	return changed
}
