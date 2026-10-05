package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// fnEnsureSingleSpaceAfter forces exactly one single-line space between the token
// at i and the next one. A newline between them is left intact.
func fnEnsureSingleSpaceAfter(s *tokens.Stream, i int) bool {
	if i+1 >= s.Len() {
		return false
	}
	n := s.At(i + 1)
	if n.Kind == token.Whitespace {
		if hasNewline(n.Value) || n.Value == " " {
			return false
		}
		s.SetValue(i+1, " ")
		return true
	}
	s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}

// fnEnsureSingleSpaceBefore forces exactly one single-line space between the token
// at i and the one before it. A newline between them is left intact.
func fnEnsureSingleSpaceBefore(s *tokens.Stream, i int) bool {
	if i-1 < 0 {
		return false
	}
	p := s.At(i - 1)
	if p.Kind == token.Whitespace {
		if hasNewline(p.Value) || p.Value == " " {
			return false
		}
		s.SetValue(i-1, " ")
		return true
	}
	s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}

// fnGlueAfter removes a single-line whitespace token immediately after i, gluing
// the token at i to the following significant token. A newline is left intact.
// fnIsArrowFn reports whether the "fn" keyword at i opens an arrow function - its
// parameter "(" follows directly or after a by-reference "&".
func fnIsArrowFn(s *tokens.Stream, i int) bool {
	n := nextSignificantIndex(s, i)
	if n < 0 || s.At(n).Kind != token.Punct {
		return false
	}
	if s.At(n).Value == "(" {
		return true
	}
	if s.At(n).Value == "&" {
		nn := nextSignificantIndex(s, n)
		return nn >= 0 && s.At(nn).Kind == token.Punct && s.At(nn).Value == "("
	}
	return false
}

func fnGlueAfter(s *tokens.Stream, i int) bool {
	if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
		s.RemoveAt(i + 1)
		return true
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FunctionDeclarationFixer.php
//
// FunctionDeclaration normalizes the spacing of a function, method or closure
// declaration (DEFAULT config, a @PSR12/@PhpCsFixer rule): one space after the
// "function" keyword, no space between a function name and its "(", one space
// between "function" and "(" for a closure, and one space around a closure's
// "use". A leading by-reference "&" is glued to the name ("function &foo()"), and
// a static closure keeps one space after "static". Only "function" declarations
// are touched, never a call; arrow "fn" and the "{"/"=>" and inner-parenthesis
// spacing (owned by other fixers) are left alone.
//
// The fields mirror the fixer options; their zero values reproduce today's
// behavior (one space for closures and arrow fns, trailing commas left in place):
//   - closureFunctionSpacingNone: "closure_function_spacing" == "none"
//   - closureFnSpacingNone: "closure_fn_spacing" == "none"
//   - removeTrailingCommaSingleLine: "trailing_comma_single_line" == false
type FunctionDeclaration struct {
	closureFunctionSpacingNone    bool
	closureFnSpacingNone          bool
	removeTrailingCommaSingleLine bool
}

func (f FunctionDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["closure_function_spacing"].(string); ok {
		f.closureFunctionSpacingNone = v == "none"
	}
	if v, ok := config["closure_fn_spacing"].(string); ok {
		f.closureFnSpacingNone = v == "none"
	}
	if v, ok := config["trailing_comma_single_line"].(bool); ok {
		f.removeTrailingCommaSingleLine = !v
	}
	return f
}

// fnRemoveTrailingCommaSingleLine drops a trailing comma before the ")" that the
// "(" at open matches, but only when the parenthesized list is on a single line.
func fnRemoveTrailingCommaSingleLine(s *tokens.Stream, open int) bool {
	closeParen := s.MatchForward(open)
	if closeParen < 0 {
		return false
	}
	for k := open; k <= closeParen; k++ {
		if s.At(k).Kind == token.Whitespace && hasNewline(s.At(k).Value) {
			return false // multiline signature: leave as-is
		}
	}
	p := prevSignificantIndex(s, closeParen)
	if p <= open || (s.At(p).Kind != token.Punct || s.At(p).Value != ",") {
		return false
	}
	// drop any single-line whitespace between the comma and ")", then the comma
	if p+1 < closeParen && s.At(p+1).Kind == token.Whitespace {
		s.RemoveAt(p + 1)
	}
	s.RemoveAt(p)
	return true
}

func (FunctionDeclaration) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\FunctionDeclarationFixer`
}

func (FunctionDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FunctionDeclarationFixer.php"
}

func (f FunctionDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.Value) {
		case "function":
			if f.fixFunctionDeclaration(s, i) {
				changed = true
			}
		case "fn":
			// arrow function: spacing after "fn" (closure_fn_spacing). The "("
			// follows directly, or after a by-reference "&" ("fn &() => ...").
			if fnIsArrowFn(s, i) {
				if f.closureFnSpacingNone {
					if fnGlueAfter(s, i) {
						changed = true
					}
				} else if fnEnsureSingleSpaceAfter(s, i) {
					changed = true
				}
			}
		}
	}
	return changed
}

// fixFunctionDeclaration normalizes the single declaration whose "function"
// keyword is at fi. Mutations happen only at indices >= fi, so fi and the outer
// loop stay valid.
func (f FunctionDeclaration) fixFunctionDeclaration(s *tokens.Stream, fi int) bool {
	startParen := findFunctionParamOpen(s, fi)
	if startParen < 0 {
		return false
	}

	changed := false
	if f.removeTrailingCommaSingleLine {
		if fnRemoveTrailingCommaSingleLine(s, startParen) {
			changed = true
		}
	}

	// classify: optional leading "&", optional name directly before "("
	a := nextSignificantIndex(s, fi)
	ampIndex := -1
	if a >= 0 && s.At(a).Kind == token.Punct && s.At(a).Value == "&" {
		ampIndex = a
		a = nextSignificantIndex(s, a)
	}
	nameIndex := -1
	if a >= 0 && a < startParen {
		if s.At(a).Kind == token.Ident && nextSignificantIndex(s, a) == startParen {
			nameIndex = a
		} else {
			return false // unexpected token between "function" and "("
		}
	}
	named := nameIndex >= 0

	// closure "use": one space on each side (processed first, rightmost region)
	if !named {
		if closeParen := s.MatchForward(startParen); closeParen >= 0 {
			u := nextSignificantIndex(s, closeParen)
			if u >= 0 && s.At(u).Kind == token.Keyword && strings.ToLower(s.At(u).Value) == "use" {
				if fnEnsureSingleSpaceAfter(s, u) {
					changed = true
				}
				if fnEnsureSingleSpaceBefore(s, u) {
					changed = true
				}
			}
		}
	}

	// named declaration: glue name (and any "&") to the "("
	if named {
		if fnGlueAfter(s, nameIndex) {
			changed = true
		}
		if ampIndex >= 0 && fnGlueAfter(s, ampIndex) {
			changed = true
		}
	}

	// spacing after the "function" keyword (before name, "&" or "("). A closure
	// with closure_function_spacing "none" glues "function" to "("; otherwise one
	// space, and a named declaration always keeps one space before its name.
	if !named && f.closureFunctionSpacingNone {
		if fnGlueAfter(s, fi) {
			changed = true
		}
	} else if fnEnsureSingleSpaceAfter(s, fi) {
		changed = true
	}

	// static closure: one space between "static" and "function"
	if !named {
		if p := prevSignificantIndex(s, fi); p >= 0 &&
			s.At(p).Kind == token.Keyword && strings.ToLower(s.At(p).Value) == "static" {
			if fi-1 >= 0 && s.At(fi-1).Kind == token.Whitespace &&
				!hasNewline(s.At(fi-1).Value) && s.At(fi-1).Value != " " {
				s.SetValue(fi-1, " ")
				changed = true
			}
		}
	}

	return changed
}

// findFunctionParamOpen returns the index of the "(" that opens the parameter
// list of the "function" declaration at fi, or -1 when this "function" is not a
// declaration (e.g. a "use function" import) or has no parameter list. Only
// whitespace, a name and a leading "&" may appear before the "(".
func findFunctionParamOpen(s *tokens.Stream, fi int) int {
	for j := fi + 1; j < s.Len(); j++ {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Ident:
			continue
		case token.Punct:
			switch t.Value {
			case "(":
				return j
			case "&":
				continue
			default:
				return -1
			}
		default:
			return -1
		}
	}
	return -1
}
