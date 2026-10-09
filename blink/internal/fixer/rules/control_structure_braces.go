package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ControlStructureBracesFixer.php
//
// ControlStructureBraces wraps the body of each control structure in braces
// ("if (foo()) echo 'x';" -> "if (foo()) { echo 'x'; }"). It is a no-op on
// already-braced code. Alternative syntax ("if (...): ... endif;") is left to
// NoAlternativeSyntaxFixer, which runs first, so those forms are skipped here.
type ControlStructureBraces struct{}

func (ControlStructureBraces) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\ControlStructureBracesFixer`
}

func (ControlStructureBraces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ControlStructureBracesFixer.php"
}

var csbControlKeywords = map[string]bool{
	"declare": true, "do": true, "else": true, "elseif": true, "finally": true,
	"for": true, "foreach": true, "if": true, "while": true, "try": true,
	"catch": true, "switch": true,
}

var csbAltTerminators = map[string]bool{
	"endif": true, "endwhile": true, "endfor": true,
	"endforeach": true, "endswitch": true, "enddeclare": true,
}

func csbIsControl(t token.Token) bool {
	return t.Kind == token.Keyword && csbControlKeywords[strings.ToLower(t.Value)]
}

// csbIsConstantName reports whether the token at index is a class or global
// constant name - a keyword-spelled identifier declared with `const`, optionally
// preceded by a type (e.g. "const IF", "const string IF"). Scanning back over the
// type tokens, a `const` keyword marks it as a name rather than a control word.
func csbIsConstantName(s *tokens.Stream, index int) bool {
	i := prevSignificantIndex(s, index)
	for i != -1 {
		t := s.At(i)
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "const") {
			return true
		}
		// allow the optional type between `const` and the name (int, ?Foo, A|B, \Ns\C)
		if t.Kind == token.Ident || t.Kind == token.Keyword ||
			(t.Kind == token.Punct && (t.Value == "?" || t.Value == "|" || t.Value == `\`)) {
			i = prevSignificantIndex(s, i)
			continue
		}
		return false
	}
	return false
}

func csbIsKeyword(t token.Token, word string) bool {
	return t.Kind == token.Keyword && strings.EqualFold(t.Value, word)
}

func csbIsPunct(t token.Token, v string) bool {
	return t.Kind == token.Punct && t.Value == v
}

func (f ControlStructureBraces) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		tok := s.At(index)
		if !csbIsControl(tok) {
			continue
		}

		// a keyword-spelled constant name (e.g. "const string IF = ...") is an
		// identifier, not a control structure - the lexer still tags it as a
		// keyword, so skip it here rather than wrap it in braces
		if csbIsConstantName(s, index) {
			continue
		}

		// "else if" - the "if" is handled on its own, skip the "else"
		if csbIsKeyword(tok, "else") {
			ni := nextMeaningfulIndex(s, index)
			if ni != -1 && csbIsKeyword(s.At(ni), "if") {
				continue
			}
		}

		parenEnd := csbFindParenthesisEnd(s, index)
		nextAfter := nextMeaningfulIndex(s, parenEnd)
		if nextAfter == -1 {
			continue
		}
		ta := s.At(nextAfter)

		if csbIsPunct(ta, ";") || csbIsPunct(ta, "{") || csbIsPunct(ta, ":") || ta.Kind == token.CloseTag {
			continue
		}

		// nested control using alternative syntax is normalized elsewhere; skip
		if csbIsControl(ta) && csbIsAltSyntax(s, nextAfter) {
			continue
		}

		statementEnd := csbFindStatementEnd(s, parenEnd)
		if statementEnd == -1 {
			continue
		}

		closing := []token.Token{
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: "}"},
		}
		if !csbIsPunct(s.At(statementEnd), ";") && !csbIsPunct(s.At(statementEnd), "}") {
			closing = append([]token.Token{{Kind: token.Punct, Value: ";"}}, closing...)
		}
		csbInsert(s, statementEnd+1, closing)
		csbInsert(s, parenEnd+1, []token.Token{
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: "{"},
		})
		changed = true
	}
	return changed
}

// csbIsAltSyntax reports whether the control at controlIndex opens with the
// alternative ":" syntax after its parenthesis.
func csbIsAltSyntax(s *tokens.Stream, controlIndex int) bool {
	pe := csbFindParenthesisEnd(s, controlIndex)
	after := nextMeaningfulIndex(s, pe)
	return after != -1 && csbIsPunct(s.At(after), ":")
}

// csbInsert inserts toks at pos, preserving their order.
func csbInsert(s *tokens.Stream, pos int, toks []token.Token) {
	s.InsertSliceAt(pos, toks)
}

// csbFindParenthesisEnd returns the ")" index of the control's condition, or the
// control token index itself when it has no parenthesis (do/else/try/finally).
func csbFindParenthesisEnd(s *tokens.Stream, controlIndex int) int {
	ni := nextMeaningfulIndex(s, controlIndex)
	if ni == -1 || !csbIsPunct(s.At(ni), "(") {
		return controlIndex
	}
	end := s.MatchForward(ni)
	if end == -1 {
		return controlIndex
	}
	return end
}

var csbContinuation = map[string][]string{
	"if":  {"else", "elseif"},
	"do":  {"while"},
	"try": {"catch", "finally"},
}

var csbFinalContinuation = map[string][]string{
	"if":  {"else"},
	"do":  {},
	"try": {"finally"},
}

func csbInList(word string, list []string) bool {
	for _, w := range list {
		if strings.EqualFold(word, w) {
			return true
		}
	}
	return false
}

// csbFindStatementEnd returns the index of the last token of the statement that
// follows the parenthesis, or -1 when it cannot be determined safely.
func csbFindStatementEnd(s *tokens.Stream, parenEnd int) int {
	nextIndex := nextMeaningfulIndex(s, parenEnd)
	if nextIndex == -1 {
		return -1
	}
	next := s.At(nextIndex)

	if csbIsPunct(next, "{") {
		return s.MatchForward(nextIndex)
	}

	if csbIsControl(next) {
		pe := csbFindParenthesisEnd(s, nextIndex)
		endIndex := csbFindStatementEnd(s, pe)
		if endIndex == -1 {
			return -1
		}

		opening := strings.ToLower(next.Value)
		if opening == "if" || opening == "try" || opening == "do" {
			for {
				ni := nextMeaningfulIndex(s, endIndex)
				if ni != -1 && s.At(ni).Kind == token.Keyword && csbInList(s.At(ni).Value, csbContinuation[opening]) {
					pe := csbFindParenthesisEnd(s, ni)
					endIndex = csbFindStatementEnd(s, pe)
					if endIndex == -1 {
						return -1
					}
					if csbInList(s.At(ni).Value, csbFinalContinuation[opening]) {
						return endIndex
					}
				} else {
					break
				}
			}
		}
		return endIndex
	}

	index := parenEnd
	for {
		index++
		if index >= s.Len() {
			return -1
		}
		tok := s.At(index)
		if csbIsPunct(tok, "{") {
			be := s.MatchForward(index)
			if be == -1 {
				return -1
			}
			index = be
			continue
		}
		if csbIsPunct(tok, ";") {
			return index
		}
		if tok.Kind == token.CloseTag {
			return prevSignificantIndex(s, index)
		}
		if tok.Kind == token.Keyword && csbAltTerminators[strings.ToLower(tok.Value)] {
			return -1
		}
	}
}
