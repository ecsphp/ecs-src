package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php
//
// BracesPosition places opening braces per PSR-12: classes/interfaces/traits/
// enums and named functions/methods get their "{" on the next line, aligned
// with the declaration; control structures keep it on the same line after a
// single space. Closures and free blocks are left alone by default. Each brace
// position is configurable; empty option fields keep this default behaviour.
type BracesPosition struct {
	classesOpeningBrace            string
	functionsOpeningBrace          string
	anonymousClassesOpeningBrace   string
	anonymousFunctionsOpeningBrace string
	controlStructuresOpeningBrace  string
}

const (
	bracesNextLine = "next_line_unless_newline_at_signature_end"
	bracesSameLine = "same_line"
)

func (f BracesPosition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["classes_opening_brace"].(string); ok {
		f.classesOpeningBrace = v
	}
	if v, ok := config["functions_opening_brace"].(string); ok {
		f.functionsOpeningBrace = v
	}
	if v, ok := config["anonymous_classes_opening_brace"].(string); ok {
		f.anonymousClassesOpeningBrace = v
	}
	if v, ok := config["anonymous_functions_opening_brace"].(string); ok {
		f.anonymousFunctionsOpeningBrace = v
	}
	if v, ok := config["control_structures_opening_brace"].(string); ok {
		f.controlStructuresOpeningBrace = v
	}
	// allow_single_line_empty_anonymous_classes and
	// allow_single_line_anonymous_functions are accepted but not applied: blink
	// already keeps single-line empty bodies (matching the default true).
	return f
}

func (BracesPosition) Name() string {
	return `PhpCsFixer\Fixer\Basic\BracesPositionFixer`
}

func (BracesPosition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/BracesPositionFixer.php"
}

func (f BracesPosition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		kind, kw := classifyBrace(s, i)

		var category string
		switch kind {
		case braceClassLike:
			category = "classes"
			// an anonymous class ("new class ... {") is configured separately
			if kw >= 0 {
				if prev, ok := prevSignificant(s, kw); ok && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "new") {
					category = "anon_classes"
				}
			}
		case braceFunctionDecl:
			category = "functions"
		case braceClosure:
			category = "anon_functions"
		case braceControl:
			category = "control"
		default:
			continue
		}

		// an empty class/function body already collapsed to "{}" stays on its
		// line - ECS keeps "class A {}" as-is (single_line_empty_body owns the
		// collapse); only reflow a body with content
		if (kind == braceClassLike || kind == braceFunctionDecl) && s.MatchForward(i) == i+1 {
			continue
		}

		nextLine, handle := f.wantNextLine(category, s, i)
		if !handle {
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

// wantNextLine resolves the configured position for a brace category to whether
// the "{" goes on the next line. handle is false for closures left at their
// default (blink does not move anonymous-function braces unless configured).
func (f BracesPosition) wantNextLine(category string, s *tokens.Stream, brace int) (nextLine, handle bool) {
	var pos, def string
	switch category {
	case "classes":
		pos, def = f.classesOpeningBrace, bracesNextLine
	case "functions":
		pos, def = f.functionsOpeningBrace, bracesNextLine
	case "anon_classes":
		pos, def = f.anonymousClassesOpeningBrace, bracesSameLine
	case "control":
		pos, def = f.controlStructuresOpeningBrace, bracesSameLine
	case "anon_functions":
		if f.anonymousFunctionsOpeningBrace == "" {
			return false, false
		}
		pos, def = f.anonymousFunctionsOpeningBrace, bracesSameLine
	default:
		return false, false
	}
	if pos == "" {
		pos = def
	}
	if pos == bracesSameLine {
		return false, true
	}
	// next_line_unless_newline_at_signature_end
	return !funcSignatureMultiline(s, brace), true
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
