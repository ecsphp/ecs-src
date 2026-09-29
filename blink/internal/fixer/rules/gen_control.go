package rules

import (
	"strconv"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchContinueToBreakFixer.php
//
// SwitchContinueToBreak rewrites a "continue" that targets a switch into
// "break". A "continue" targeting an enclosing loop is left untouched.
type SwitchContinueToBreak struct{}

func (SwitchContinueToBreak) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SwitchContinueToBreakFixer`
}

func (SwitchContinueToBreak) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchContinueToBreakFixer.php"
}

func (SwitchContinueToBreak) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "continue" {
			continue
		}
		level, ok := continueLevel(s, i)
		if !ok {
			continue
		}
		targets := enclosingLoopsAndSwitches(s, i)
		if level <= len(targets) && targets[level-1] == "switch" {
			s.SetValue(i, "break")
			changed = true
		}
	}
	return changed
}

// continueLevel returns the loop level a "continue" at i targets: 1 for a bare
// "continue;" (or "continue 1;"/"continue 0;"), n for "continue n;". ok is
// false when i is not a plain continue statement (e.g. an unparsable level, or a
// level not immediately followed by ";").
func continueLevel(s *tokens.Stream, i int) (int, bool) {
	j := meaningfulAfter(s, i)
	if j < 0 {
		return 0, false
	}
	tj := s.At(j)
	if tj.Kind == token.Punct && tj.Value == ";" {
		return 1, true // bare "continue" == "continue 1"
	}
	if tj.Kind != token.Number {
		return 0, false
	}
	if k := meaningfulAfter(s, j); k < 0 || s.At(k).Kind != token.Punct || s.At(k).Value != ";" {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.ReplaceAll(tj.Value, "_", ""), 0, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	if n == 0 {
		n = 1 // "continue 0" behaves like "continue 1"
	}
	return int(n), true
}

// enclosingLoopsAndSwitches lists the loop and switch constructs enclosing token
// i, innermost first (each entry is "loop" or "switch"). "if"/"try"/"catch" and
// free blocks are transparent to continue and skipped. The walk stops at a
// function, closure or class boundary a continue cannot cross.
func enclosingLoopsAndSwitches(s *tokens.Stream, i int) []string {
	var out []string
	depth := 0
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			kind, kw := classifyBrace(s, j)
			switch kind {
			case braceFunctionDecl, braceClosure, braceClassLike:
				return out
			case braceControl:
				switch strings.ToLower(s.At(kw).Value) {
				case "switch":
					out = append(out, "switch")
				case "for", "foreach", "while", "do":
					out = append(out, "loop")
				}
			}
		}
	}
	return out
}

// meaningfulAfter returns the first index after i that is not whitespace or a
// comment, or -1 if there is none.
func meaningfulAfter(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		switch s.At(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}
