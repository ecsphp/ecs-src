package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/SingleLineThrowFixer.php
//
// SingleLineThrow collapses a throw statement onto a single line: newlines
// between "throw" and the end of the thrown expression are removed or replaced
// with a single space, and line comments are rewritten as block comments.
type SingleLineThrow struct{}

func (SingleLineThrow) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\SingleLineThrowFixer`
}

func (SingleLineThrow) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/SingleLineThrowFixer.php"
}

func (SingleLineThrow) Fix(s *tokens.Stream) bool {
	changed := false
	for index := 0; index < s.Len(); index++ {
		if !kwIs(s.At(index), "throw") {
			continue
		}
		end := aliasNextMeaningful(s, index)
		for end >= 0 && !throwIsStop(s, end) {
			t := s.At(end)
			switch {
			case t.Kind == token.AttributeStart:
				end = s.AttributeEnd(end)
			case t.Kind == token.Punct && (t.Value == "(" || t.Value == "["):
				end = s.MatchForward(end)
			case t.Kind == token.Punct && t.Value == "{":
				if !throwIsDynamicBrace(s, end) {
					goto trim // a plain "{" is a brace block: stop before it
				}
				end = s.MatchForward(end)
			case t.Kind == token.Punct && t.Value == "}":
				goto trim // a block close: stop before it
			}
			if end < 0 {
				break
			}
			end = aliasNextMeaningful(s, end)
		}
	trim:
		if end < 0 {
			continue
		}
		if throwTrimNewLines(s, index, aliasPrevMeaningful(s, end)) {
			changed = true
		}
	}
	return changed
}

// throwIsStop reports whether the token at i ends the throw-expression scan.
func throwIsStop(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind == token.CloseTag {
		return true
	}
	return t.Kind == token.Punct && (t.Value == ")" || t.Value == "]" || t.Value == "," || t.Value == ";")
}

// throwIsDynamicBrace reports whether the "{" at i is a dynamic variable or
// property brace ("${" or "->{"), which detectBlockType treats as a block to
// skip rather than a plain brace block.
func throwIsDynamicBrace(s *tokens.Stream, i int) bool {
	p := aliasPrevMeaningful(s, i)
	if p < 0 {
		return false
	}
	return (s.At(p).Kind == token.Variable && s.At(p).Value == "$") ||
		isPunctVal(s, p, "->") || isPunctVal(s, p, "?->")
}

// throwTrimNewLines mirrors SingleLineThrowFixer::trimNewLines over the tokens
// in [startIndex, endIndex).
func throwTrimNewLines(s *tokens.Stream, startIndex, endIndex int) bool {
	changed := false
	for index := startIndex; index >= 0 && index < endIndex; index++ {
		t := s.At(index)
		content := t.Value

		if t.Kind == token.Comment {
			switch {
			case strings.HasPrefix(content, "//"):
				content = "/*" + content[2:] + " */"
				clearTokenAt(s, index+1)
			case strings.HasPrefix(content, "#") && !strings.HasPrefix(content, "#["):
				content = "/*" + content[1:] + " */"
				clearTokenAt(s, index+1)
			case hasNewline(content):
				content = throwReplaceNewlines(content)
			default:
				continue
			}
			s.SetValue(index, content)
			changed = true
			continue
		}

		if t.Kind != token.Whitespace || !hasNewline(content) {
			continue
		}

		prevIndex := nonEmptySibling(s, index, -1)
		if prevIndex >= 0 && throwIsPrevToClear(s, prevIndex) {
			clearTokenAt(s, index)
			changed = true
			continue
		}

		nextIndex := nonEmptySibling(s, index, 1)
		if nextIndex >= 0 && throwIsNextToClear(s, nextIndex) &&
			(prevIndex < 0 || !kwIs(s.At(prevIndex), "function")) {
			clearTokenAt(s, index)
			changed = true
			continue
		}

		s.SetValue(index, " ")
		changed = true
	}
	return changed
}

// throwReplaceNewlines replaces each line break with a single space.
func throwReplaceNewlines(v string) string {
	v = strings.ReplaceAll(v, "\r\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	return strings.ReplaceAll(v, "\n", " ")
}

// throwIsPrevToClear reports whether whitespace right after the token at i is
// dropped (REMOVE_WHITESPACE_AFTER/AROUND, or an object operator).
func throwIsPrevToClear(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind == token.Punct {
		switch t.Value {
		case "[", "(", "::", "->", "?->":
			return true
		}
	}
	return false
}

// throwIsNextToClear reports whether whitespace right before the token at i is
// dropped (REMOVE_WHITESPACE_AROUND/BEFORE, or an object operator).
func throwIsNextToClear(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind == token.Punct {
		switch t.Value {
		case "(", "::", ")", "]", ",", ";", "->", "?->":
			return true
		}
	}
	return false
}
