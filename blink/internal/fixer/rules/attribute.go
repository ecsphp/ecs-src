package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
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

// The helpers below port PHP-CS-Fixer's AttributeAnalyzer. They model an
// attribute group ("#[...]") and the individual attributes inside it as token
// ranges, so a fixer can reorder or remove whole attributes while keeping the
// surrounding whitespace, commas and comments byte-for-byte.

// attrItem is one attribute inside a group: its token range [start,end], the
// index of the first token of its name and the name as written (a leading
// backslash is kept).
type attrItem struct {
	start, end int
	nameStart  int
	name       string
}

// attrGroup is one "#[...]" group in a run. [start,end] spans the group together
// with its trailing whitespace; start reaches back to a leading comment when the
// group follows another. opening and closing are the AttributeStart/AttributeEnd.
// key is the sort key the ordered-attributes fixer fills in.
type attrGroup struct {
	start, end int
	opening    int
	closing    int
	attrs      []attrItem
	key        string
}

// nextNonWhitespaceIndex ports Tokens::getNextNonWhitespace: it skips whitespace
// only, so a comment stops the scan.
func nextNonWhitespaceIndex(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

// attrNameTokenStart ports getNextTokenOfKind(from, [T_STRING, T_NS_SEPARATOR]):
// the first name token (identifier or "\") after from.
func attrNameTokenStart(s *tokens.Stream, from int) int {
	for j := from + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
			return j
		}
	}
	return -1
}

// attrPartialName joins token values over [a,b], mirroring generatePartialCode.
func attrPartialName(s *tokens.Stream, a, b int) string {
	if a < 0 || b < a {
		return ""
	}
	var sb strings.Builder
	for j := a; j <= b; j++ {
		sb.WriteString(s.At(j).Value)
	}
	return sb.String()
}

// collectAttributesIn ports AttributeAnalyzer::collectAttributes: the attributes
// of one group, bracketed by the AttributeStart at openIdx and AttributeEnd at
// closing.
func collectAttributesIn(s *tokens.Stream, openIdx, closing int) []attrItem {
	var elements []attrItem
	index := openIdx
	for {
		attrStart := index + 1
		nameStart := attrNameTokenStart(s, index)
		index = attrNextParenCommaOrClose(s, attrStart)
		if index < 0 {
			break
		}
		name := attrPartialName(s, nameStart, sigPrev(s, index))
		if isPunctVal(s, index, "(") {
			end := s.MatchForward(index)
			if end < 0 {
				break
			}
			index = attrNextCommaOrClose(s, end)
			if index < 0 {
				break
			}
		}
		elements = append(elements, attrItem{start: attrStart, end: index - 1, nameStart: nameStart, name: name})
		nextIndex := index
		if nextIndex < closing {
			nextIndex = sigNext(s, index)
		}
		if nextIndex < 0 || nextIndex >= closing {
			break
		}
	}
	// Trim a trailing newline off the last element when there is no trailing
	// comma, so the "\n]" stays as structure rather than part of the attribute.
	for index--; index >= 0 && s.At(index).Kind == token.Whitespace; index-- {
		if strings.ContainsAny(s.At(index).Value, "\r\n") {
			if len(elements) > 0 {
				elements[len(elements)-1].end = index - 1
			}
			break
		}
	}
	return elements
}

// collectOneAttrGroup ports AttributeAnalyzer::collectOne for the group whose
// AttributeStart is at idx.
func collectOneAttrGroup(s *tokens.Stream, idx int) attrGroup {
	start := idx
	if prev := sigPrev(s, idx); prev >= 0 && s.At(prev).Kind == token.AttributeEnd {
		if ns := nextNonWhitespaceIndex(s, prev); ns >= 0 {
			start = ns
		}
	}
	closing := s.AttributeEnd(idx)
	end := closing
	if ns := nextNonWhitespaceIndex(s, closing); ns >= 0 {
		end = ns - 1
	}
	return attrGroup{start: start, end: end, opening: idx, closing: closing, attrs: collectAttributesIn(s, idx, closing)}
}

// collectAttrGroupRun ports AttributeAnalyzer::collect: the run of consecutive
// "#[...]" groups beginning at idx.
func collectAttrGroupRun(s *tokens.Stream, idx int) []attrGroup {
	var groups []attrGroup
	for opening := idx; ; {
		g := collectOneAttrGroup(s, opening)
		groups = append(groups, g)
		nx := sigNext(s, g.end)
		if nx < 0 || s.At(nx).Kind != token.AttributeStart {
			break
		}
		opening = nx
	}
	return groups
}
