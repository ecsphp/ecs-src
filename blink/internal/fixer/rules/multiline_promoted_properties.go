package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/MultilinePromotedPropertiesFixer.php
//
// MultilinePromotedProperties puts each constructor parameter on its own line
// when the constructor has a promoted property and at least
// minimum_number_of_parameters parameters; below that threshold it collapses a
// promoted constructor's parameters back onto one line (never when a comment,
// doc comment or attribute is present). keep_blank_lines leaves author blank
// lines between parameters alone.
type MultilinePromotedProperties struct {
	configured                bool
	minimumNumberOfParameters int
	keepBlankLines            bool
}

func (MultilinePromotedProperties) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\MultilinePromotedPropertiesFixer`
}

func (MultilinePromotedProperties) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/MultilinePromotedPropertiesFixer.php"
}

func (f MultilinePromotedProperties) WithConfig(config map[string]any) fixer.Fixer {
	f.configured = true
	f.minimumNumberOfParameters = configInt(config, "minimum_number_of_parameters", 1)
	if v, ok := config["keep_blank_lines"].(bool); ok {
		f.keepBlankLines = v
	}
	return f
}

// resolved fills the zero-value defaults: a bare MultilinePromotedProperties{}
// means minimum_number_of_parameters=1, keep_blank_lines=false.
func (f MultilinePromotedProperties) resolved() MultilinePromotedProperties {
	if f.configured {
		return f
	}
	f.minimumNumberOfParameters = 1
	return f
}

func (f MultilinePromotedProperties) Fix(s *tokens.Stream) bool {
	f = f.resolved()
	changed := false
	// a promoted property only ever lives in a constructor; iterating constructors
	// mirrors upstream's getClassyElements walk, whose shouldBe* checks only hold
	// when a promotion token is present.
	for i := 0; i < s.Len(); i++ {
		open := constructorParamOpen(s, i)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		if f.shouldBeMultiline(s, open, closeIdx) {
			if f.makeMultiline(s, open, closeIdx) {
				changed = true
			}
		} else if f.shouldBeSingleline(s, open, closeIdx) {
			if f.makeSingleline(s, open, closeIdx) {
				changed = true
			}
		}
	}
	return changed
}

func (f MultilinePromotedProperties) shouldBeMultiline(s *tokens.Stream, open, closeIdx int) bool {
	promoted := false
	parameterCount := 0
	for j := open + 1; j < closeIdx; j++ {
		if s.At(j).Kind == token.Variable {
			parameterCount++
		}
		if isPromotedVisibility(s, j) {
			promoted = true
		}
	}
	return promoted && parameterCount >= f.minimumNumberOfParameters
}

func (f MultilinePromotedProperties) shouldBeSingleline(s *tokens.Stream, open, closeIdx int) bool {
	promoted := false
	parameterCount := 0
	isMultiline := false
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Variable {
			parameterCount++
		}
		if isPromotedVisibility(s, j) {
			promoted = true
		}
		if t.Kind == token.Comment || t.Kind == token.DocComment || t.Kind == token.AttributeStart {
			return false
		}
		if t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			isMultiline = true
		}
	}
	return promoted && isMultiline && parameterCount < f.minimumNumberOfParameters
}

func (f MultilinePromotedProperties) makeSingleline(s *tokens.Stream, open, closeIdx int) bool {
	changed := false
	for index := closeIdx - 1; index > open; index-- {
		t := s.At(index)
		if t.Kind != token.Whitespace || !strings.Contains(t.Value, "\n") {
			continue
		}
		if index == open+1 || index == closeIdx-1 {
			if t.Value != "" {
				s.SetValue(index, "")
				changed = true
			}
		} else if t.Value != " " {
			s.SetValue(index, " ")
			changed = true
		}
	}
	return changed
}

func (f MultilinePromotedProperties) makeMultiline(s *tokens.Stream, open, closeIdx int) bool {
	indent := detectIndent(s, open)

	// break before the closing parenthesis; closeIdx stays as the upstream local
	// does (it may now address the inserted whitespace, which getPrevMeaningful skips)
	changed := mppEnsureWhitespace(s, closeIdx-1, 1, "\n"+indent)

	index := sigPrev(s, closeIdx)
	if index < 0 || !isPunctVal(s, index, ",") {
		index = closeIdx
	}

	for index > open {
		index = sigPrev(s, index)
		if index < 0 {
			break
		}
		if start := mppBlockStart(s, index); start >= 0 {
			index = start
			continue
		}
		if !isPunctVal(s, index, "(") && !isPunctVal(s, index, ",") {
			continue
		}
		if f.fixParameter(s, index+1, indent) {
			changed = true
		}
	}
	return changed
}

func (f MultilinePromotedProperties) fixParameter(s *tokens.Stream, index int, indent string) bool {
	if f.keepBlankLines && index < s.Len() && s.At(index).Kind == token.Whitespace && strings.Contains(s.At(index).Value, "\n") {
		return false
	}
	return mppEnsureWhitespace(s, index, 0, "\n"+indent+"    ")
}

// mppBlockStart returns the matching opener when index is a block closer (")",
// "]", "}" or an attribute's "]"), so the backward walk skips a default value,
// attribute or other nested block; -1 otherwise.
func mppBlockStart(s *tokens.Stream, index int) int {
	t := s.At(index)
	if t.Kind == token.AttributeEnd {
		return s.AttributeStartOf(index)
	}
	if t.Kind == token.Punct {
		switch t.Value {
		case ")", "]", "}":
			return s.MatchBackward(index)
		}
	}
	return -1
}

// mppEnsureWhitespace mirrors Tokens::ensureWhitespaceAtIndex: it rewrites the
// whitespace token at index, or inserts one at index+indexOffset when the token
// there is not whitespace. An empty whitespace clears the slot.
func mppEnsureWhitespace(s *tokens.Stream, index, indexOffset int, whitespace string) bool {
	if index >= 0 && index < s.Len() && s.At(index).Kind == token.Whitespace {
		if s.At(index).Value == whitespace {
			return false
		}
		s.SetValue(index, whitespace)
		return true
	}
	if whitespace == "" {
		return false
	}
	s.InsertAt(index+indexOffset, token.Token{Kind: token.Whitespace, Value: whitespace})
	return true
}
