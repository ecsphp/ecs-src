package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php
//
// BracesPosition places opening braces per PSR-12: classes/interfaces/traits/
// enums and named functions/methods get their "{" on the next line, aligned
// with the declaration; control structures keep it on the same line after a
// single space. Closures and free blocks are left alone.
type BracesPosition struct{}

func (BracesPosition) Name() string {
	return `PhpCsFixer\Fixer\Basic\BracesPositionFixer`
}

func (BracesPosition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php"
}

func (BracesPosition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		kind, kw := classifyBrace(s, i)

		// an empty class/function body already collapsed to "{}" stays on its
		// line - ECS keeps "class A {}" as-is (single_line_empty_body owns the
		// collapse); only reflow a body with content
		if (kind == braceClassLike || kind == braceFunctionDecl) && s.MatchForward(i) == i+1 {
			continue
		}

		nextLine := false
		switch kind {
		case braceClassLike:
			// an anonymous class ("new class ... {") keeps its opening brace on
			// the same line (PHP-CS-Fixer anonymous_classes_opening_brace default)
			if kw >= 0 {
				if prev, ok := prevSignificant(s, kw); ok && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
					nextLine = false
					break
				}
			}
			nextLine = true
		case braceFunctionDecl:
			// PSR-12 keeps "){" on one line when the signature is multiline
			nextLine = !funcSignatureMultiline(s, i)
		case braceControl:
			nextLine = false
		default:
			continue
		}

		want := " "
		if nextLine {
			// depth-based so it stays correct regardless of the header's own
			// (possibly wrong) indentation; statement_indentation fixes the rest
			want = "\n" + strings.Repeat("    ", braceDepthAt(s, i))
		}
		if i > 0 && s.At(i-1).Kind == token.Whitespace {
			if s.At(i-1).Value != want {
				s.SetValue(i-1, want)
				changed = true
			}
		} else {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: want})
			i++
			changed = true
		}
	}
	return changed
}

// funcSignatureMultiline reports whether the parameter list of the function
// whose body opens at brace spans more than one line.
func funcSignatureMultiline(s *tokens.Stream, brace int) bool {
	closeParen := -1
	for j := brace - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		if t.Value == ")" {
			closeParen = j
			break
		}
		if t.Value == "{" || t.Value == "}" || t.Value == ";" {
			return false
		}
	}
	if closeParen < 0 {
		return false
	}
	open := s.MatchBackward(closeParen)
	if open < 0 {
		return false
	}
	for k := open; k <= closeParen; k++ {
		if s.At(k).Kind == token.Whitespace && hasNewline(s.At(k).Value) {
			return true
		}
	}
	return false
}
