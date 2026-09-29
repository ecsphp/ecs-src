package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SingleClassElementPerStatementFixer.php
//
// SingleClassElementPerStatement splits a multi-element property or constant
// declaration into one per line: "public $a, $b;" -> "public $a;\npublic $b;".
//
// elements mirrors the "elements" option (subset of const/property); a nil value
// (the zero value) means both, matching the default.
type SingleClassElementPerStatement struct {
	elements []string
}

func (f SingleClassElementPerStatement) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["elements"]; ok {
		f.elements = singleElementConfigStrings(raw)
	}
	return f
}

// singleElementConfigStrings converts a config list ([]any or []string) to []string.
func singleElementConfigStrings(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if sv, ok := e.(string); ok {
				out = append(out, sv)
			}
		}
		return out
	}
	return nil
}

func (f SingleClassElementPerStatement) singleElementEnabled(kind string) bool {
	if f.elements == nil {
		return true // default: both const and property
	}
	return slices.Contains(f.elements, kind)
}

func (SingleClassElementPerStatement) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\SingleClassElementPerStatementFixer`
}

func (SingleClassElementPerStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SingleClassElementPerStatementFixer.php"
}

func (f SingleClassElementPerStatement) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		if kind, _ := classifyBrace(s, i); kind != braceClassLike {
			continue
		}
		for _, m := range slices.Backward(classMemberStarts(s, i)) {
			if f.splitClassElement(s, m) {
				changed = true
			}
		}
	}
	return changed
}

func (f SingleClassElementPerStatement) splitClassElement(s *tokens.Stream, m int) bool {
	// step over leading modifiers
	k := m
	for k < s.Len() && s.At(k).Kind == token.Keyword && memberModifiers[strings.ToLower(s.At(k).Value)] {
		k = skipWhitespace(s, k+1)
	}
	if k >= s.Len() {
		return false
	}

	isConst := false
	if s.At(k).Kind == token.Keyword {
		switch strings.ToLower(s.At(k).Value) {
		case "const":
			isConst = true
		default:
			return false // function / use / case / unknown
		}
	}

	if isConst {
		if !f.singleElementEnabled("const") {
			return false
		}
	} else if !f.singleElementEnabled("property") {
		return false
	}

	semi := memberEndSemi(s, m)
	if semi < 0 {
		return false
	}

	firstElem := -1
	if isConst {
		firstElem = skipWhitespace(s, k+1)
	} else {
		depth := 0
		for x := m; x < semi; x++ {
			t := s.At(x)
			if t.Kind == token.Punct {
				switch t.Value {
				case "(", "[":
					depth++
				case ")", "]":
					depth--
				}
			}
			if depth == 0 && t.Kind == token.Variable {
				firstElem = x
				break
			}
		}
	}
	if firstElem < 0 || firstElem >= semi {
		return false
	}

	var commas []int
	depth := 0
	for x := firstElem; x < semi; x++ {
		t := s.At(x)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case ",":
			if depth == 0 {
				commas = append(commas, x)
			}
		}
	}
	if len(commas) == 0 {
		return false
	}

	prefix := make([]token.Token, 0, firstElem-m)
	for x := m; x < firstElem; x++ {
		prefix = append(prefix, s.At(x))
	}
	indent := lineIndent(s, m)

	bounds := append([]int{firstElem - 1}, commas...)
	bounds = append(bounds, semi)
	var repl []token.Token
	for b := 0; b+1 < len(bounds); b++ {
		var part []token.Token
		for x := bounds[b] + 1; x < bounds[b+1]; x++ {
			part = append(part, s.At(x))
		}
		part = trimWhitespace(part)
		if b > 0 {
			repl = append(repl, token.Token{Kind: token.Whitespace, Value: "\n" + indent})
		}
		repl = append(repl, prefix...)
		repl = append(repl, part...)
		repl = append(repl, token.Token{Kind: token.Punct, Value: ";"})
	}
	s.ReplaceRange(m, semi, repl)
	return true
}

// memberEndSemi returns the ";" that ends the member starting at m, skipping
// nested groups. Returns -1 if a "}" is reached first (e.g. a method body).
func memberEndSemi(s *tokens.Stream, m int) int {
	for k := m; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			if mm := s.MatchForward(k); mm > 0 {
				k = mm
			}
		case ";":
			return k
		case "}":
			return -1
		}
	}
	return -1
}
