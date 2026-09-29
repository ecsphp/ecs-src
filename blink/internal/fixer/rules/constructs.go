package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// constructKeywords are the keywords PSR-12 requires to be followed by a single
// space (single_space_around_construct).
var constructKeywords = map[string]bool{
	"abstract": true, "as": true, "case": true, "catch": true, "class": true,
	"do": true, "else": true, "elseif": true, "final": true, "finally": true,
	"for": true, "foreach": true, "function": true, "if": true, "insteadof": true,
	"interface": true, "namespace": true, "new": true, "private": true,
	"protected": true, "public": true, "readonly": true, "static": true,
	"switch": true, "trait": true, "try": true, "use": true, "while": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/SingleSpaceAroundConstructFixer.php
//
// SingleSpaceAroundConstruct ensures a single space after a language construct
// keyword ("if(" -> "if (", "else{" -> "else {").
type SingleSpaceAroundConstruct struct{}

func (SingleSpaceAroundConstruct) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAroundConstructFixer`
}

func (SingleSpaceAroundConstruct) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/SingleSpaceAroundConstructFixer.php"
}

func (SingleSpaceAroundConstruct) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !constructKeywords[strings.ToLower(t.Value)] || i+1 >= s.Len() {
			i++
			continue
		}
		next := s.At(i + 1)
		// "static" and "class" directly before "(" are a class reference, not a
		// construct: "new static(...)", "new class(...)". They take no space.
		kw := strings.ToLower(t.Value)
		classRef := kw == "static" || kw == "class"
		if next.Kind == token.Whitespace {
			if classRef && !hasNewline(next.Value) && i+2 < s.Len() &&
				s.At(i+2).Kind == token.Punct && s.At(i+2).Value == "(" {
				s.RemoveAt(i + 1) // "new static (" -> "new static("
				changed = true
				i++
				continue
			}
			if !hasNewline(next.Value) && next.Value != " " {
				s.SetValue(i+1, " ")
				changed = true
			}
			i++
			continue
		}
		switch next.Value {
		case "::", "->", "?->", ";", ",", ")", ":":
			// member access, statement end or label - not a construct body
		case "(":
			if !classRef {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
				i++
			}
		default:
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
			i++
		}
		i++
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoSpacesAfterFunctionNameFixer.php
//
// NoSpacesAfterFunctionName removes whitespace between a function name and its
// opening parenthesis ("foo ()" -> "foo()").
type NoSpacesAfterFunctionName struct{}

func (NoSpacesAfterFunctionName) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NoSpacesAfterFunctionNameFixer`
}

func (NoSpacesAfterFunctionName) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoSpacesAfterFunctionNameFixer.php"
}

func (NoSpacesAfterFunctionName) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident {
			continue
		}
		if i+2 < s.Len() &&
			s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			s.At(i+2).Kind == token.Punct && s.At(i+2).Value == "(" {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SpacesInsideParenthesesFixer.php
//
// NoSpacesInsideParenthesis removes single-line whitespace just inside
// parentheses ("( $a )" -> "($a)"). Newlines are kept for multi-line calls.
type NoSpacesInsideParenthesis struct{}

func (NoSpacesInsideParenthesis) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\SpacesInsideParenthesesFixer`
}

func (NoSpacesInsideParenthesis) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SpacesInsideParenthesesFixer.php"
}

func (NoSpacesInsideParenthesis) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == "(" &&
			i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
			s.RemoveAt(i + 1)
			changed = true
		}
		if s.At(i).Kind == token.Punct && s.At(i).Value == ")" &&
			i >= 1 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/UnaryOperatorSpacesFixer.php
//
// UnaryOperatorSpaces removes whitespace between an operand and ++ or --
// ("$i ++" -> "$i++").
type UnaryOperatorSpaces struct{}

func (UnaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\UnaryOperatorSpacesFixer`
}

func (UnaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/UnaryOperatorSpacesFixer.php"
}

func (UnaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "++" && t.Value != "--") {
			continue
		}
		if i >= 2 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) && isOperand(s.At(i-2)) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
		if i+2 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) && isOperand(s.At(i+2)) {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}
