package rules

import (
	"strings"

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
type FunctionDeclaration struct{}

func (FunctionDeclaration) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\FunctionDeclarationFixer`
}

func (FunctionDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FunctionDeclarationFixer.php"
}

func (FunctionDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.Value) {
		case "function":
			if fixFunctionDeclaration(s, i) {
				changed = true
			}
		case "fn":
			// arrow function: one space between "fn" and "(" (closure_fn_spacing)
			if n := nextSignificantIndex(s, i); n >= 0 &&
				s.At(n).Kind == token.Punct && s.At(n).Value == "(" &&
				fnEnsureSingleSpaceAfter(s, i) {
				changed = true
			}
		}
	}
	return changed
}

// fixFunctionDeclaration normalizes the single declaration whose "function"
// keyword is at fi. Mutations happen only at indices >= fi, so fi and the outer
// loop stay valid.
func fixFunctionDeclaration(s *tokens.Stream, fi int) bool {
	startParen := findFunctionParamOpen(s, fi)
	if startParen < 0 {
		return false
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

	changed := false

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

	// one space after the "function" keyword (before name, "&" or "(")
	if fnEnsureSingleSpaceAfter(s, fi) {
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
