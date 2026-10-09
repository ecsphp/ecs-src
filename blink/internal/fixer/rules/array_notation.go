package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// convertLongArray rewrites a long "name(...)" construct to "[...]", where name
// is "array" or "list". Method/constant uses (after -> ?-> ::) and a method
// declared with that name (after "function") are skipped.
func convertLongArray(s *tokens.Stream, name string) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword && t.Kind != token.Ident {
			continue
		}
		if strings.ToLower(t.Value) != name {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok {
			switch prev.Value {
			case "->", "?->", "::", "function":
				continue
			}
		}
		j := skipWhitespace(s, i+1)
		if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(j)
		if closeIdx < 0 {
			continue
		}
		s.SetValue(closeIdx, "]")
		s.SetValue(j, "[")
		s.ReplaceRange(i, j-1, nil)
		changed = true
	}
	return changed
}

// arrayNotationStringList reads a string-list option (a []string or []any of strings).
func arrayNotationStringList(v any) ([]string, bool) {
	switch list := v.(type) {
	case []string:
		return list, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, str)
		}
		return out, true
	}
	return nil, false
}

func arrayNotationContains(list []string, want string) bool {
	return slices.Contains(list, want)
}

func arrayNotationIsHeredoc(t token.Token) bool {
	return t.Kind == token.String && strings.HasPrefix(t.Value, "<<<")
}

// arrayNotationEnclosing returns the index of the innermost unclosed opener before i, or -1.
func arrayNotationEnclosing(s *tokens.Stream, i int) int {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			if depth == 0 {
				return j
			}
			depth--
		}
	}
	return -1
}

func arrayNotationIsAs(t token.Token) bool {
	return (t.Kind == token.Keyword || t.Kind == token.Ident) && strings.EqualFold(t.Value, "as")
}

// arrayNotationInForeachHead reports whether the token at i sits inside the
// "(... as ...)" head of a foreach, after the "as" keyword.
func arrayNotationInForeachHead(s *tokens.Stream, i int) bool {
	depth := 0
	seenAs := false
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case ")", "]", "}":
				depth++
			case "(", "[", "{":
				if depth > 0 {
					depth--
					continue
				}
				if t.Value != "(" || !seenAs {
					return false
				}
				p := sigPrev(s, j)
				return p >= 0 && strings.EqualFold(s.At(p).Value, "foreach")
			}
			continue
		}
		if depth == 0 && arrayNotationIsAs(t) {
			seenAs = true
		}
	}
	return false
}

// arrayNotationIsDestructuring reports whether the short "[" at open is an
// array destructuring target rather than an array literal.
func arrayNotationIsDestructuring(s *tokens.Stream, open int) bool {
	if closeIdx := s.MatchForward(open); closeIdx >= 0 {
		if n := sigNext(s, closeIdx); n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "=" {
			return true
		}
	}
	if p := sigPrev(s, open); p >= 0 {
		if arrayNotationIsAs(s.At(p)) {
			return true
		}
		if s.At(p).Kind == token.Punct && s.At(p).Value == "=>" && arrayNotationInForeachHead(s, open) {
			return true
		}
	}
	enc := arrayNotationEnclosing(s, open)
	if enc < 0 {
		return false
	}
	switch s.At(enc).Value {
	case "[":
		return !isOffsetOpen(s, enc) && arrayNotationIsDestructuring(s, enc)
	case "(":
		q := sigPrev(s, enc)
		return q >= 0 && strings.EqualFold(s.At(q).Value, "list")
	}
	return false
}

// arrayNotationToLong rewrites short "[...]" to "array(...)" (literals) or
// "list(...)" (destructuring targets), depending on destructuring.
func arrayNotationToLong(s *tokens.Stream, destructuring bool) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "[" || isOffsetOpen(s, i) {
			continue
		}
		if arrayNotationIsDestructuring(s, i) != destructuring {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		name := "array"
		if destructuring {
			name = "list"
		}
		s.SetValue(closeIdx, ")")
		s.SetValue(i, "(")
		s.InsertAt(i, token.Token{Kind: token.Keyword, Value: name})
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php
//
// ArraySyntax rewrites long "array(...)" to the short "[...]" form. With
// syntax=long it does the opposite.
type ArraySyntax struct {
	Long bool
}

func (ArraySyntax) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer`
}

func (ArraySyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php"
}

func (f ArraySyntax) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.Long = v == "long"
	}
	return f
}

func (f ArraySyntax) Fix(s *tokens.Stream) bool {
	if f.Long {
		return arrayNotationToLong(s, false)
	}
	return convertLongArray(s, "array")
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php
//
// ListSyntax rewrites "list(...)" to the short "[...]" form. With syntax=long
// it does the opposite.
type ListSyntax struct {
	Long bool
}

func (ListSyntax) Name() string {
	return `PhpCsFixer\Fixer\ListNotation\ListSyntaxFixer`
}

func (ListSyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php"
}

func (f ListSyntax) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.Long = v == "long"
	}
	return f
}

func (f ListSyntax) Fix(s *tokens.Stream) bool {
	if f.Long {
		return arrayNotationToLong(s, true)
	}
	return convertLongArray(s, "list")
}

// enclosingIsArray reports whether the innermost open bracket at index i is "[".
func topBracketIsArray(stack []string) bool {
	return len(stack) > 0 && stack[len(stack)-1] == "["
}

// isDestructuringAssignOpen reports whether the "[" at open is a short-list
// destructuring target - its matching "]" is directly followed by a single "=".
func isDestructuringAssignOpen(s *tokens.Stream, open int) bool {
	c := s.MatchForward(open)
	if c < 0 {
		return false
	}
	n := nextSignificantIndex(s, c)
	return n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "="
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php
//
// NoWhitespaceBeforeCommaInArray removes single-line whitespace before a comma
// inside an array. SkipAfterHeredoc (after_heredoc=false) keeps the whitespace
// between a heredoc end and the comma.
type NoWhitespaceBeforeCommaInArray struct {
	SkipAfterHeredoc bool
}

func (NoWhitespaceBeforeCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NoWhitespaceBeforeCommaInArrayFixer`
}

func (NoWhitespaceBeforeCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php"
}

func (f NoWhitespaceBeforeCommaInArray) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["after_heredoc"].(bool); ok {
		f.SkipAfterHeredoc = !v
	}
	return f
}

func (f NoWhitespaceBeforeCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				stack = append(stack, t.Value)
			case ")", "]", "}":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			case ",":
				if topBracketIsArray(stack) && i > 0 &&
					s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
					if f.SkipAfterHeredoc && i > 1 && arrayNotationIsHeredoc(s.At(i-2)) {
						continue
					}
					// an empty element ("[, , $x]") keeps its space: the token before
					// the whitespace is itself a comma, matching php-cs-fixer
					if i > 1 && s.At(i-2).Kind == token.Punct && s.At(i-2).Value == "," {
						continue
					}
					s.RemoveAt(i - 1)
					i--
					changed = true
				}
			}
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php
//
// WhitespaceAfterCommaInArray ensures a single space after a comma inside an
// array (unless a newline follows, or the comma is a trailing one before "]").
// EnsureSingleSpace also collapses horizontal whitespace after the comma to one space.
type WhitespaceAfterCommaInArray struct {
	EnsureSingleSpace bool
}

func (WhitespaceAfterCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\WhitespaceAfterCommaInArrayFixer`
}

func (WhitespaceAfterCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php"
}

func (f WhitespaceAfterCommaInArray) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ensure_single_space"].(bool); ok {
		f.EnsureSingleSpace = v
	}
	return f
}

func arrayNotationIsHorizontal(t token.Token) bool {
	return t.Kind == token.Whitespace && t.Value != "" && strings.Trim(t.Value, " \t") == ""
}

func (f WhitespaceAfterCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "[":
			// a destructuring target ("[$a,$b] = ...", or nested in one) is not an
			// array literal, so php-cs-fixer does not space its commas
			if (len(stack) > 0 && stack[len(stack)-1] == "d[") || isDestructuringAssignOpen(s, i) {
				stack = append(stack, "d[")
			} else {
				stack = append(stack, "[")
			}
		case "(", "{":
			stack = append(stack, t.Value)
		case ")", "]", "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ",":
			if !topBracketIsArray(stack) || i+1 >= s.Len() {
				continue
			}
			next := s.At(i + 1)
			if next.Kind == token.Whitespace {
				if f.EnsureSingleSpace && next.Value != " " && arrayNotationIsHorizontal(next) {
					afterComment := i+2 < s.Len() && (s.At(i+2).Kind == token.Comment || s.At(i+2).Kind == token.DocComment)
					if !afterComment || (i+3 < s.Len() && arrayNotationIsHorizontal(s.At(i+3))) {
						s.SetValue(i+1, " ")
						changed = true
					}
				}
				continue // already spaced or newline
			}
			if next.Kind == token.Punct && next.Value == "]" {
				continue // trailing comma
			}
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			i++
			changed = true
		}
	}
	return changed
}
