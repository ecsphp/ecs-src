package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitExpectationFixer.php
//
// PhpUnitExpectation rewrites `$this->setExpectedException*(...)` calls in a
// PHPUnit test class to the dedicated `expectException()`,
// `expectExceptionMessage*()` and `expectExceptionCode()` calls. The default
// target is the newest PHPUnit, so `setExpectedExceptionRegExp` and
// `expectExceptionMessageRegExp` both map to `expectExceptionMessageMatches`.
type PhpUnitExpectation struct{}

// phpUnitExpectationMethodMap is the newest-target method map: the input method
// name to the message-method it produces.
var phpUnitExpectationMethodMap = map[string]string{
	"setExpectedException":         "expectExceptionMessage",
	"setExpectedExceptionRegExp":   "expectExceptionMessageMatches",
	"expectExceptionMessageRegExp": "expectExceptionMessageMatches",
}

func (PhpUnitExpectation) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitExpectationFixer`
}

func (PhpUnitExpectation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitExpectationFixer.php"
}

func (PhpUnitExpectation) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if phpUnitExpectationClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

func phpUnitExpectationClass(s *tokens.Stream, startIndex, endIndex int) bool {
	changed := false
	for index := startIndex; index < endIndex; {
		thisIdx, nameIdx, ok := phpUnitExpectationNextCall(s, index, endIndex)
		if !ok {
			break
		}
		index = nameIdx + 1
		newName, inMap := phpUnitExpectationMethodMap[s.At(nameIdx).Value]
		if !inMap {
			continue
		}
		if phpUnitExpectationRewrite(s, thisIdx, nameIdx, newName) {
			changed = true
			endIndex = s.MatchForward(startIndex) // inserts grow the class body
		}
	}
	return changed
}

// phpUnitExpectationNextCall mirrors Tokens::findSequence([$this, ->, T_STRING]):
// the next `$this->name` (or `$this?->name`) at or after from, within the class.
func phpUnitExpectationNextCall(s *tokens.Stream, from, endIndex int) (thisIdx, nameIdx int, ok bool) {
	for i := from; i < endIndex && i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Variable || t.Value != "$this" {
			continue
		}
		op := aliasNextMeaningful(s, i)
		if !isPunctVal(s, op, "->") && !isPunctVal(s, op, "?->") {
			continue
		}
		name := aliasNextMeaningful(s, op)
		if name < 0 || s.At(name).Kind != token.Ident {
			continue
		}
		return i, name, true
	}
	return 0, 0, false
}

func phpUnitExpectationRewrite(s *tokens.Stream, thisIdx, nameIdx int, newName string) bool {
	content := s.At(nameIdx).Value
	openIdx := nextPunctOfKind(s, nameIdx, "(")
	if openIdx < 0 {
		return false
	}
	closeIdx := s.MatchForward(openIdx)
	if closeIdx < 0 {
		return false
	}

	// drop a trailing comma and the whitespace after it
	if commaIdx := aliasPrevMeaningful(s, closeIdx); isPunctVal(s, commaIdx, ",") {
		if w := commaIdx + 1; w < s.Len() && s.At(w).Kind == token.Whitespace {
			clearTokenAt(s, w)
		}
		clearTokenAt(s, commaIdx)
	}

	args := aliasArguments(s, openIdx, closeIdx)
	replacements := []string{"expectException", newName, "expectExceptionCode"}
	indent := "\n" + detectIndent(s, thisIdx)
	multiline := false

	for cnt := len(args) - 1; cnt >= 1 && cnt < len(replacements); cnt-- {
		argStart := args[cnt][0]
		argBefore := aliasPrevMeaningful(s, argStart)
		repl := replacements[cnt]

		if repl == "expectExceptionMessage" {
			paramIndicator := aliasNextMeaningful(s, argBefore)
			afterParam := aliasNextMeaningful(s, paramIndicator)
			if phpUnitExpectationIsNull(s, paramIndicator) && isPunctVal(s, afterParam, ")") {
				if w := argBefore + 1; w < s.Len() && s.At(w).Kind == token.Whitespace {
					clearTokenAndMergeWS(s, w)
				}
				clearTokenAndMergeWS(s, argBefore)
				clearTokenAndMergeWS(s, paramIndicator)
				continue
			}
		}

		startTok := s.At(argStart)
		multiline = multiline || (startTok.Kind == token.Whitespace && phpUnitExpectationIsMultilineWS(startTok.Value))

		overrideStart := []token.Token{
			{Kind: token.Whitespace, Value: indent},
			{Kind: token.Variable, Value: "$this"},
			{Kind: token.Punct, Value: "->"},
			{Kind: token.Ident, Value: repl},
			{Kind: token.Punct, Value: "("},
		}
		overrideBefore := []token.Token{
			{Kind: token.Punct, Value: ")"},
			{Kind: token.Punct, Value: ";"},
		}
		if multiline {
			overrideStart = append(overrideStart, token.Token{Kind: token.Whitespace, Value: indent + "    "})
			overrideBefore = append([]token.Token{{Kind: token.Whitespace, Value: indent}}, overrideBefore...)
		}

		if startTok.Kind == token.Whitespace {
			s.ReplaceRange(argStart, argStart, overrideStart)
		} else {
			s.InsertSliceAt(argStart, overrideStart)
		}
		s.ReplaceRange(argBefore, argBefore, overrideBefore)
	}

	methodName := "expectException"
	if content == "expectExceptionMessageRegExp" {
		methodName = newName
	}
	s.Set(nameIdx, token.Token{Kind: token.Ident, Value: methodName, Pos: s.At(nameIdx).Pos})
	return true
}

// phpUnitExpectationIsNull reports whether the token at i is the native
// constant `null` in any case.
func phpUnitExpectationIsNull(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Ident && strings.EqualFold(s.At(i).Value, "null")
}

// phpUnitExpectationIsMultilineWS mirrors Token::isWhitespace(" \t") being
// false: the whitespace holds a character other than a space or tab.
func phpUnitExpectationIsMultilineWS(v string) bool {
	return strings.ContainsFunc(v, func(r rune) bool { return r != ' ' && r != '\t' })
}
