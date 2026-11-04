package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SemicolonAfterInstructionFixer.php
//
// SemicolonAfterInstruction terminates the instruction before a closing tag
// with a semicolon ("<?php echo 1 ?>" -> "<?php echo 1; ?>"), unless it already
// ends with ";", "{", "}", ":" or is an empty "<?php" block.
type SemicolonAfterInstruction struct{}

func (SemicolonAfterInstruction) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\SemicolonAfterInstructionFixer`
}

func (SemicolonAfterInstruction) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SemicolonAfterInstructionFixer.php"
}

func (SemicolonAfterInstruction) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index > 1; index-- {
		if s.At(index).Kind != token.CloseTag {
			continue
		}
		prev := sigPrev(s, index)
		if prev < 0 || semicolonAfterInstructionTerminated(s, prev) {
			continue
		}
		s.InsertAt(prev+1, token.Token{Kind: token.Punct, Value: ";"})
		changed = true
	}
	return changed
}

// semicolonAfterInstructionTerminated reports whether the token at prev is one
// of ";", "{", "}", ":" or a T_OPEN_TAG. The braces of "$a->{...}", "${...}",
// "Foo::{...}" and group imports are custom tokens in PHP-CS-Fixer, so they do
// not count.
func semicolonAfterInstructionTerminated(s *tokens.Stream, prev int) bool {
	t := s.At(prev)
	switch t.Kind {
	case token.OpenTag:
		return t.Value != "<?="
	case token.Punct:
		switch t.Value {
		case ";", ":":
			return true
		case "{":
			return !semicolonAfterInstructionSpecialBrace(s, prev)
		case "}":
			open := s.MatchBackward(prev)
			return open < 0 || !semicolonAfterInstructionSpecialBrace(s, open)
		}
	}
	return false
}

// semicolonAfterInstructionSpecialBrace reports whether the "{" at open is a
// dynamic property/variable/class-constant brace or a group import brace.
func semicolonAfterInstructionSpecialBrace(s *tokens.Stream, open int) bool {
	p := sigPrev(s, open)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Punct:
		return t.Value == "->" || t.Value == "?->" || t.Value == "::" || t.Value == `\`
	case token.Variable:
		return t.Value == "$"
	}
	return false
}
