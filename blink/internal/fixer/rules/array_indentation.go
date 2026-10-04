package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/ArrayIndentationFixer.php
//
// ArrayIndentation indents each element of a multi-line array exactly one level
// past the array's own indentation, and re-indents multi-line elements by the
// same delta. It only rewrites whitespace (never adds or removes tokens), so it
// preserves how many items share a line and is a no-op on ECS-formatted code.
// This is a faithful port of PHP-CS-Fixer's scope-stack algorithm.
type ArrayIndentation struct{}

func (ArrayIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\ArrayIndentationFixer`
}

func (ArrayIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/ArrayIndentationFixer.php"
}

const arrayIndentUnit = "    "

type aiScope struct {
	kind          string // "array" or "expression"
	endIndex      int
	initialIndent string
	newIndent     string // expression only
}

func (ArrayIndentation) Fix(s *tokens.Stream) bool {
	changed := false
	lastIndent := ""
	var scopes []aiScope
	prevLineInitialIndent := ""
	prevLineNewIndent := ""

	for index := 0; index < s.Len(); index++ {
		cur := len(scopes) - 1
		t := s.At(index)

		if t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}

		if end, ok := aiArrayOpen(s, index); ok {
			scopes = append(scopes, aiScope{kind: "array", endIndex: end, initialIndent: lastIndent})
			continue
		}

		if t.Kind == token.Whitespace && hasNewline(t.Value) {
			lastIndent = aiExtractIndent(t.Value)
		}

		if cur < 0 {
			continue
		}
		sc := &scopes[cur]

		if t.Kind == token.Whitespace {
			if !hasNewline(t.Value) {
				continue
			}
			var content string
			if sc.kind == "array" {
				indent := false
				for k := index + 1; k < sc.endIndex; k++ {
					kt := s.At(k)
					nonTrivia := kt.Kind != token.Whitespace && kt.Kind != token.Comment && kt.Kind != token.DocComment
					if nonTrivia || (kt.Kind == token.Whitespace && hasNewline(kt.Value)) {
						indent = true
						break
					}
				}
				extra := ""
				if indent {
					extra = arrayIndentUnit
				}
				content = aiReindentArray(t.Value, sc.initialIndent+extra)
				prevLineInitialIndent = aiExtractIndent(t.Value)
				prevLineNewIndent = aiExtractIndent(content)
			} else {
				content = aiReindentExpression(t.Value, sc.initialIndent, sc.newIndent)
			}
			if content != t.Value {
				s.SetValue(index, content)
				changed = true
			}
			lastIndent = aiExtractIndent(content)
			continue
		}

		if index == sc.endIndex {
			for len(scopes) > 0 && index == scopes[len(scopes)-1].endIndex {
				scopes = scopes[:len(scopes)-1]
			}
			continue
		}

		if t.Kind == token.Punct && t.Value == "," {
			continue
		}

		if sc.kind != "expression" {
			end := aiExpressionEnd(s, index, sc.endIndex)
			if end != index {
				scopes = append(scopes, aiScope{
					kind: "expression", endIndex: end,
					initialIndent: prevLineInitialIndent, newIndent: prevLineNewIndent,
				})
			}
		}
	}
	return changed
}

// aiArrayOpen reports whether the token at index opens an array literal - "[" as
// a literal (not access), or "(" right after "array"/"list" - and returns its
// matching close index.
func aiArrayOpen(s *tokens.Stream, index int) (int, bool) {
	t := s.At(index)
	if t.Kind != token.Punct {
		return 0, false
	}
	switch t.Value {
	case "[":
		if !isArrayLiteralOpen(s, index) {
			return 0, false
		}
		end := s.MatchForward(index)
		if end < 0 {
			return 0, false
		}
		return end, true
	case "(":
		p := sigPrev(s, index)
		if p >= 0 && s.At(p).Kind == token.Keyword {
			lv := strings.ToLower(s.At(p).Value)
			if lv == "array" || lv == "list" {
				end := s.MatchForward(index)
				if end >= 0 {
					return end, true
				}
			}
		}
	}
	return 0, false
}

// isArrayLiteralOpen reports whether "[" at open starts an array literal rather
// than an array access ($a[0], foo()[0]).
func isArrayLiteralOpen(s *tokens.Stream, open int) bool {
	p := sigPrev(s, open)
	if p < 0 {
		return true
	}
	t := s.At(p)
	switch t.Kind {
	case token.Variable, token.Ident, token.String, token.Number:
		return false
	case token.Punct:
		switch t.Value {
		case ")":
			// a cast before "[" ("(object) [...]") opens an array literal, not an
			// access on a call result
			if o := s.MatchBackward(p); o >= 0 && isCastParen(s, o) {
				return true
			}
			return false
		case "]", "}":
			return false
		}
	}
	return true
}

// aiExpressionEnd finds the end of the array element expression starting at index,
// up to the next top-level comma (or the array's end), skipping nested blocks.
func aiExpressionEnd(s *tokens.Stream, index, parentEnd int) int {
	end := -1
	for k := index + 1; k < parentEnd; k++ {
		kt := s.At(k)
		if kt.Kind == token.Punct && (kt.Value == "(" || kt.Value == "{" || kt.Value == "[") {
			if be := s.MatchForward(k); be >= 0 {
				k = be
			}
			continue
		}
		if kt.Kind == token.Punct && kt.Value == "," {
			end = sigPrev(s, k)
			break
		}
	}
	if end >= 0 {
		return end
	}
	return sigPrev(s, parentEnd)
}

// aiExtractIndent returns the horizontal whitespace after the last newline.
func aiExtractIndent(ws string) string {
	if i := strings.LastIndexByte(ws, '\n'); i >= 0 {
		return ws[i+1:]
	}
	return ws
}

// aiReindentArray keeps the newline run of a whitespace token and sets the line's
// indentation to target.
func aiReindentArray(ws, target string) string {
	if i := strings.LastIndexByte(ws, '\n'); i >= 0 {
		return ws[:i+1] + target
	}
	return ws
}

// aiReindentExpression shifts a continuation line: if its indent begins with the
// expression's original indent, that prefix is swapped for the new indent.
func aiReindentExpression(ws, initialIndent, newIndent string) string {
	i := strings.LastIndexByte(ws, '\n')
	if i < 0 {
		return ws
	}
	line := ws[i+1:]
	if !strings.HasPrefix(line, initialIndent) {
		return ws
	}
	return ws[:i+1] + newIndent + line[len(initialIndent):]
}
