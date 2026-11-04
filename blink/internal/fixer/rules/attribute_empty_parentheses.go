package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/AttributeEmptyParenthesesFixer.php
//
// AttributeEmptyParentheses normalizes attributes declared without arguments:
// by default it removes the empty parentheses ("#[Foo()]" -> "#[Foo]"), and with
// use_parentheses it adds them ("#[Foo]" -> "#[Foo()]").
type AttributeEmptyParentheses struct {
	// useParentheses adds empty parentheses instead of removing them; the zero
	// value (false) is PHP-CS-Fixer's default (remove).
	useParentheses bool
}

func (AttributeEmptyParentheses) Name() string {
	return `PhpCsFixer\Fixer\AttributeNotation\AttributeEmptyParenthesesFixer`
}

func (AttributeEmptyParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/AttributeEmptyParenthesesFixer.php"
}

func (f AttributeEmptyParentheses) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["use_parentheses"].(bool); ok {
		f.useParentheses = v
	}
	return f
}

func (f AttributeEmptyParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	index := 0
	for {
		start := attrNextStart(s, index)
		if start < 0 {
			break
		}
		nextIndex := start
		for {
			parenIdx := attrNextParenCommaOrClose(s, nextIndex)
			if parenIdx < 0 {
				break
			}
			if f.useParentheses {
				if attrEnsureParentheses(s, parenIdx) {
					changed = true
				}
			} else {
				if attrEnsureNoParentheses(s, parenIdx) {
					changed = true
				}
			}
			nextIndex = attrNextParenCommaOrClose(s, nextIndex)
			if nextIndex < 0 {
				break
			}
			if isPunctVal(s, nextIndex, "(") {
				end := s.MatchForward(nextIndex)
				if end < 0 {
					break
				}
				nextIndex = attrNextCommaOrClose(s, end)
				if nextIndex < 0 {
					break
				}
			}
			if s.At(nextIndex).Kind != token.AttributeEnd {
				nextIndex = aliasNextMeaningful(s, nextIndex)
				if nextIndex < 0 {
					break
				}
			}
			if s.At(nextIndex).Kind == token.AttributeEnd {
				break
			}
		}
		index = start
	}
	return changed
}

func attrNextStart(s *tokens.Stream, after int) int {
	for j := after + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.AttributeStart {
			return j
		}
	}
	return -1
}

func attrNextParenCommaOrClose(s *tokens.Stream, from int) int {
	for j := from + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.AttributeEnd {
			return j
		}
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == ",") {
			return j
		}
	}
	return -1
}

func attrNextCommaOrClose(s *tokens.Stream, from int) int {
	for j := from + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.AttributeEnd {
			return j
		}
		if t.Kind == token.Punct && t.Value == "," {
			return j
		}
	}
	return -1
}

func attrEnsureParentheses(s *tokens.Stream, index int) bool {
	if isPunctVal(s, index, "(") {
		return false
	}
	at := aliasPrevMeaningful(s, index) + 1
	s.InsertSliceAt(at, []token.Token{{Kind: token.Punct, Value: "("}, {Kind: token.Punct, Value: ")"}})
	return true
}

func attrEnsureNoParentheses(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, "(") {
		return false
	}
	closing := aliasNextMeaningful(s, index)
	if !isPunctVal(s, closing, ")") {
		return false // the attribute has arguments
	}
	clearTokenAndMergeWS(s, closing)
	clearTokenAndMergeWS(s, index)
	return true
}
