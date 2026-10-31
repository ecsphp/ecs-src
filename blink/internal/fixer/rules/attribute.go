package rules

import (
	"strings"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// Attributes lex natively ("#[" is an AttributeStart token, its contents and the
// closing "]" are ordinary tokens), so most fixers see inside them for free. The
// helpers here cover the few rules that still reason about an attribute as a whole.

// attributeStartsAt reports whether an attribute ("#[") begins at i.
func attributeStartsAt(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.AttributeStart
}

// attributeEndsAt reports whether the attribute-closing "]" is at i.
func attributeEndsAt(s *tokens.Stream, i int) bool {
	return s.AttributeStartOf(i) >= 0
}

// prevMeaningfulSkippingAttributes is prevMeaningfulIndex but also steps back over
// any attributes ("new #[A] #[B] class" -> the "new"), since an attribute decorates
// the token that follows it.
func prevMeaningfulSkippingAttributes(s *tokens.Stream, i int) int {
	p := prevMeaningfulIndex(s, i)
	for p >= 0 && attributeEndsAt(s, p) {
		p = prevMeaningfulIndex(s, s.AttributeStartOf(p))
	}
	return p
}

// attributeText renders the attribute tokens [start..end] back to source text.
func attributeText(s *tokens.Stream, start, end int) string {
	var b strings.Builder
	for j := start; j <= end; j++ {
		b.WriteString(s.At(j).Value)
	}
	return b.String()
}

// rewriteAttributeText feeds the attribute [start..end] to transform as its source
// text ("#[...]"), re-lexes the result and splices it back in place. It bridges the
// text-shaped attribute transforms to the native token stream. Returns whether the
// stream changed.
func rewriteAttributeText(s *tokens.Stream, start, end int, transform func(string) (string, bool)) bool {
	old := attributeText(s, start, end)
	nv, ok := transform(old)
	if !ok || nv == old {
		return false
	}
	s.ReplaceRange(start, end, lexAttributeFragment(nv))
	return true
}

// lexAttributeFragment lexes a standalone "#[...]" fragment into its tokens, from
// the AttributeStart through the end (the opening PHP tag used to drive the lexer
// is dropped).
func lexAttributeFragment(text string) []token.Token {
	toks := lexer.Lex("<?php " + text)
	for i := range toks {
		if toks[i].Kind == token.AttributeStart {
			return toks[i:]
		}
	}
	return toks
}
