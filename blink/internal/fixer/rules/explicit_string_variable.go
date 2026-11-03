package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/ExplicitStringVariableFixer.php
//
// ExplicitStringVariable wraps simple variable interpolation in double-quoted
// strings and heredocs in the explicit "{$...}" form: "$a" -> "{$a}",
// "$a->b" -> "{$a->b}", "$a[key]" -> "{$a['key']}". Already-explicit ("{$a}",
// "${a}"), single-quoted strings, nowdocs and backtick literals are left
// untouched.
type ExplicitStringVariable struct{}

func (ExplicitStringVariable) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\ExplicitStringVariableFixer`
}

func (ExplicitStringVariable) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/ExplicitStringVariableFixer.php"
}

func (ExplicitStringVariable) Fix(s *tokens.Stream) bool {
	changed := false
	// like PHP-CS-Fixer, skip every literal between two "`" tokens, nested
	// ones included; going backwards keeps the edits from shifting a later literal
	backticks := 0
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.String && t.Part == token.Head && backticks%2 == 0 {
			if chunks := literalChunks(s, i); chunks != nil && explicitStringVariables(s, chunks) {
				changed = true
			}
		}
		if t.IsShellExec() && t.Part != token.Whole {
			backticks++
		}
	}
	return changed
}

// explicitStringVariables wraps the simple interpolations of one literal (its
// chunk indices) in "{...}" as PHP-CS-Fixer does: each run of variables that
// follows a string part is wrapped, a bareword offset quoted.
func explicitStringVariables(s *tokens.Stream, chunks []int) bool {
	type group struct{ first, last int }
	var groups []group
	collecting := false
	afterExpr := false
	brackets := 0 // PHP counts the "[" and "]" of the whole run
	for k, c := range chunks[:len(chunks)-1] {
		v := s.At(c).Value
		e := scanChunk(v, k == 0, afterExpr)
		afterExpr = e.opensExpr
		switch {
		case e.hasStringPart:
			collecting, brackets = e.endsWithStringPart, 0
		case strings.Contains(v, "${"):
			collecting = false
		}
		if e.opensExpr || !collecting {
			continue
		}
		// a run of simple interpolations up to the next chunk
		for j := c + 1; j < chunks[k+1]; j++ {
			t := s.At(j)
			switch {
			case t.Kind == token.Variable && brackets != 1:
				groups = append(groups, group{j, j})
			case len(groups) > 0:
				groups[len(groups)-1].last = j
				if t.Kind == token.Punct && (t.Value == "[" || t.Value == "]") {
					brackets++
				}
			}
		}
	}
	for g := range slices.Backward(groups) {
		first, last := groups[g].first, groups[g].last
		for j := first + 1; j < last; j++ {
			if t := s.At(j); t.Kind == token.Ident && s.At(j+1).Value == "]" {
				s.Set(j, token.Token{Kind: token.String, Value: "'" + t.Value + "'", Pos: t.Pos})
			}
		}
		if n := s.At(last + 1); n.Kind == token.String && n.Part != token.Whole {
			s.SetValue(last+1, "}"+n.Value)
		} else {
			s.InsertAt(last+1, token.Token{Kind: token.String, Value: "}", Part: token.Mid})
		}
		if p := s.At(first - 1); p.Kind == token.String && p.Part != token.Whole {
			s.SetValue(first-1, p.Value+"{")
		} else {
			s.InsertAt(first, token.Token{Kind: token.String, Value: "{", Part: token.Mid})
		}
	}
	return len(groups) > 0
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isNameByte(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}
