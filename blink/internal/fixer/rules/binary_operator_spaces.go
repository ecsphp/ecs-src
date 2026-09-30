package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php
//
// binaryOperators are unambiguously binary operators (never unary), safe to
// space on a flat token stream. Ambiguous ones (+ - & * used as unary/reference/
// splat) are intentionally excluded.
var binaryOperators = map[string]bool{
	"==": true, "===": true, "!=": true, "!==": true, "<>": true,
	"<=": true, ">=": true, "<=>": true, "<": true, ">": true,
	"&&": true, "||": true, "??": true, "=>": true,
}

// BinaryOperatorSpaces normalizes spacing around binary operators (assignment
// "=", arrow "=>", comparison and logical operators). The zero value applies a
// single space to the default operator set. Whitespace spanning a newline is left
// alone to preserve alignment. Reference assignment ("=& $x") and declare()
// headers are skipped.
//
// Options: Default is the mode for operators without an override and Operators
// maps an operator to its own mode. Modes are "single", "no_space",
// "at_least_single" and "skip". The align_* modes and null are unsupported in a
// token model and map to "skip" (operator left untouched).
type BinaryOperatorSpaces struct {
	Default   string
	Operators map[string]string
}

func (BinaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\BinaryOperatorSpacesFixer`
}

func (BinaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php"
}

func (f BinaryOperatorSpaces) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["default"]; ok {
		f.Default = binaryOperatorSpacesMode(v)
	}
	if raw, ok := config["operators"].(map[string]any); ok {
		f.Operators = make(map[string]string, len(raw))
		for op, v := range raw {
			f.Operators[op] = binaryOperatorSpacesMode(v)
		}
	}
	return f
}

func binaryOperatorSpacesMode(v any) string {
	str, _ := v.(string)
	switch str {
	case "single_space":
		return "single"
	case "no_space":
		return "no_space"
	case "at_least_single_space":
		return "at_least_single"
	}
	return "skip"
}

func (f BinaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		mode := f.modeAt(s, i)
		if mode == "" || mode == "skip" {
			i++
			continue
		}
		var ch bool
		i, ch = binaryOperatorSpacesApply(s, i, mode)
		if ch {
			changed = true
		}
		i++
	}
	// Attributes are lexed as one opaque token, so the loop above never sees the
	// "=>" inside them; normalize their arrow alignment directly when "=>" is set
	// to a single space (php-cs-fixer collapses it the same way).
	if f.arrowSingleSpace() {
		for j := 0; j < s.Len(); j++ {
			t := s.At(j)
			if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "#[") {
				continue
			}
			if nv, ch := collapseAttributeArrowSpacing(t.Value); ch {
				s.SetValue(j, nv)
				changed = true
			}
		}
	}
	return changed
}

// arrowSingleSpace reports whether "=>" resolves to a single space under this
// configuration.
func (f BinaryOperatorSpaces) arrowSingleSpace() bool {
	if m, ok := f.Operators["=>"]; ok {
		return m == "single"
	}
	if f.Default == "" {
		return true
	}
	return f.Default == "single"
}

// collapseAttributeArrowSpacing collapses aligned whitespace around each "=>" in
// an attribute token's text to a single space, ignoring "=>" inside strings and
// leaving newlines (multiline alignment) alone.
func collapseAttributeArrowSpacing(value string) (string, bool) {
	var b []byte
	changed := false
	var quote byte
	for i := 0; i < len(value); i++ {
		c := value[i]
		if quote != 0 {
			b = append(b, c)
			if c == '\\' && i+1 < len(value) {
				i++
				b = append(b, value[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			b = append(b, c)
			continue
		}
		if c == '=' && i+1 < len(value) && value[i+1] == '>' {
			// collapse a same-line run of spaces already written before "=>"
			end := len(b)
			start := end
			for start > 0 && (b[start-1] == ' ' || b[start-1] == '\t') {
				start--
			}
			if start > 0 && start < end {
				b = append(b[:start], ' ')
				changed = true
			}
			b = append(b, '=', '>')
			i++
			// collapse a same-line run of spaces after "=>"
			j := i + 1
			for j < len(value) && (value[j] == ' ' || value[j] == '\t') {
				j++
			}
			if j > i+1 && j < len(value) && value[j] != '\n' && value[j] != '\r' {
				b = append(b, ' ')
				i = j - 1
				changed = true
			}
			continue
		}
		b = append(b, c)
	}
	return string(b), changed
}

// modeAt returns the spacing mode for the token at i, or "" when it is not a
// handled operator.
func (f BinaryOperatorSpaces) modeAt(s *tokens.Stream, i int) string {
	t := s.At(i)
	if t.Kind != token.Punct {
		return ""
	}
	legacy := binaryOperatorSpacesDefaultTarget(s, i)
	if m, ok := f.Operators[t.Value]; ok {
		if legacy || binaryOperatorSpacesExtraTarget(s, i) {
			return m
		}
		return ""
	}
	if !legacy {
		return ""
	}
	if f.Default == "" {
		return "single"
	}
	return f.Default
}

func binaryOperatorSpacesDefaultTarget(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct {
		return false
	}
	if t.Value == "=" {
		// leave reference assignment ("=& $x") and declare(...) headers alone
		return nextSignificantValue(s, i) != "&" && !insideDeclareArgs(s, i)
	}
	return binaryOperators[t.Value]
}

// binaryOperatorSpacesExtraTarget covers operators outside the default set that
// may only be spaced when explicitly configured through the operators option.
func binaryOperatorSpacesExtraTarget(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Value {
	case "+=", "-=", "*=", "/=", ".=", "%=", "**=", "??=", "&=", "|=", "^=", "<<=", ">>=",
		"**", "/", "%", "^", "<<", ">>":
		return true
	case "+", "-", "*":
		prev, ok := prevSignificant(s, i)
		return ok && (isOperand(prev) || prev.Kind == token.Number || prev.Kind == token.String)
	case "&", "|":
		prev, ok := prevSignificant(s, i)
		if !ok || isTypeUnionOperator(s, i) {
			return false
		}
		return prev.Kind == token.Variable || prev.Kind == token.Number || prev.Kind == token.String ||
			prev.Value == ")" || prev.Value == "]"
	}
	return false
}

// binaryOperatorSpacesApply spaces the operator at i and returns its new index.
func binaryOperatorSpacesApply(s *tokens.Stream, i int, mode string) (int, bool) {
	changed := false
	if i > 0 {
		prev := s.At(i - 1)
		if prev.Kind == token.Whitespace {
			switch {
			case hasNewline(prev.Value):
			case mode == "no_space":
				s.RemoveAt(i - 1)
				i--
				changed = true
			case mode == "single" && prev.Value != " ":
				s.SetValue(i-1, " ")
				changed = true
			}
		} else if mode != "no_space" {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
			i++
			changed = true
		}
	}
	if i+1 < s.Len() {
		next := s.At(i + 1)
		if next.Kind == token.Whitespace {
			switch {
			case hasNewline(next.Value):
			case mode == "no_space":
				s.RemoveAt(i + 1)
				changed = true
			case mode == "single" && next.Value != " ":
				s.SetValue(i+1, " ")
				changed = true
			}
		} else if mode != "no_space" {
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
	}
	return i, changed
}
