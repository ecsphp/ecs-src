package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoMultipleStatementsPerLineFixer.php
//
// NoMultipleStatementsPerLine puts each statement on its own line: "$a=1; $b=2;"
// becomes two lines. Semicolons inside "(...)"/"[...]" (e.g. a for header) and a
// trailing ";" before "}" or a newline are left alone.
type NoMultipleStatementsPerLine struct{}

func (NoMultipleStatementsPerLine) Name() string {
	return `PhpCsFixer\Fixer\Basic\NoMultipleStatementsPerLineFixer`
}

func (NoMultipleStatementsPerLine) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoMultipleStatementsPerLineFixer.php"
}

func (NoMultipleStatementsPerLine) Fix(s *tokens.Stream) bool {
	changed := false
	depth := 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			depth++
			continue
		case ")", "]":
			depth--
			continue
		case ";":
		default:
			continue
		}
		if depth != 0 || i+1 >= s.Len() {
			continue
		}

		// what follows the ";" on the same line?
		nextIdx := i + 1
		inlineWS := false
		if s.At(nextIdx).Kind == token.Whitespace {
			if hasNewline(s.At(nextIdx).Value) {
				continue // already end of line
			}
			inlineWS = true
			nextIdx++
		}
		if nextIdx >= s.Len() {
			continue
		}
		nt := s.At(nextIdx)
		if nt.Kind == token.Punct && (nt.Value == "}" || nt.Value == ";") {
			continue
		}
		if nt.Kind == token.CloseTag {
			continue
		}
		// a trailing comment after ";" is not a second statement
		if nt.Kind == token.Comment || nt.Kind == token.DocComment {
			continue
		}

		indent := lineIndent(s, i)
		if inlineWS {
			s.SetValue(i+1, "\n"+indent)
		} else {
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: "\n" + indent})
		}
		changed = true
	}
	return changed
}
