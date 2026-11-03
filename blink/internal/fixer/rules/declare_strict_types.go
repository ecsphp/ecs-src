package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/DeclareStrictTypesFixer.php
//
// DeclareStrictTypes forces "declare(strict_types=1);" right after the opening
// tag of a single-tag PHP file (risky).
type DeclareStrictTypes struct {
	// Strategy is "enforce" (the default when empty), "add_when_missing" or "remove".
	Strategy string
}

func (f DeclareStrictTypes) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["strategy"].(string); ok {
		f.Strategy = v
	}
	// the deprecated preserve_existing_declaration=true turns "enforce" into "add_when_missing"
	if v, ok := config["preserve_existing_declaration"].(bool); ok && v && f.strategy() == "enforce" {
		f.Strategy = "add_when_missing"
	}
	return f
}

func (DeclareStrictTypes) Name() string {
	return `PhpCsFixer\Fixer\Strict\DeclareStrictTypesFixer`
}

func (DeclareStrictTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/DeclareStrictTypesFixer.php"
}

func (f DeclareStrictTypes) strategy() string {
	if f.Strategy == "" {
		return "enforce"
	}
	return f.Strategy
}

func (f DeclareStrictTypes) Fix(s *tokens.Stream) bool {
	if !dstIsCandidate(s) {
		return false
	}
	before := s.Render()
	openTagIndex := 0
	if s.At(0).Kind == token.InlineHTML {
		openTagIndex = 1
	}
	declareIndex, open, closeIdx, seq := dstDeclaration(s)
	if f.strategy() == "remove" {
		if seq != nil {
			dstRemove(s, declareIndex, open, closeIdx, seq)
			compactEmptyTokens(s)
		}
	} else if seq == nil {
		dstInsert(s, openTagIndex)
	} else {
		for _, index := range seq {
			t := s.At(index)
			if t.Kind == token.Ident {
				s.SetValue(index, strings.ToLower(t.Value))
				continue
			}
			if t.Kind == token.Number && f.strategy() == "enforce" {
				s.SetValue(index, "1")
				break
			}
		}
	}
	return s.Render() != before
}

var dstShebangRe = regexp.MustCompile(`\A#![^\n]+\n?\z`)

// dstIsCandidate mirrors isCandidate: Tokens::isMonolithicPhp and no "<?=".
func dstIsCandidate(s *tokens.Stream) bool {
	openTags, inlineHTML := 0, 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.OpenTag:
			if t.Value == "<?=" {
				return false
			}
			openTags++
		case token.InlineHTML:
			// PHP's T_CLOSE_TAG swallows the one newline that follows "?>"
			v := t.Value
			if i > 0 && s.At(i-1).Kind == token.CloseTag {
				if strings.HasPrefix(v, "\r\n") {
					v = v[2:]
				} else if strings.HasPrefix(v, "\n") {
					v = v[1:]
				}
			}
			if v != "" {
				inlineHTML++
			}
		}
	}
	if openTags != 1 {
		return false
	}
	return inlineHTML == 0 || (inlineHTML == 1 && s.At(0).Kind == token.InlineHTML && dstShebangRe.MatchString(s.At(0).Value))
}

// dstIsLNumber reports whether a Number token is an integer literal (T_LNUMBER).
func dstIsLNumber(t token.Token) bool {
	if t.Kind != token.Number {
		return false
	}
	v := strings.ToLower(t.Value)
	if strings.HasPrefix(v, "0x") {
		return true
	}
	return !strings.ContainsAny(v, ".e")
}

// dstDeclaration mirrors getStrictTypesDeclaration: the first declare whose
// parentheses hold "strict_types = <int>" (case-insensitive).
func dstDeclaration(s *tokens.Stream) (declareIndex, open, closeIdx int, seq []int) {
	pattern := []taSeqItem{
		taSeqTString("strict_types", true),
		taSeqPunct("="),
		func(s *tokens.Stream, i int) bool { return dstIsLNumber(s.At(i)) },
	}
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "declare") {
			continue
		}
		open := sigNext(s, i)
		if !isPunctVal(s, open, "(") {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		if seq := taFindSequence(s, pattern, open, closeIdx); seq != nil {
			return i, open, closeIdx, seq
		}
	}
	return -1, -1, -1, nil
}

// dstInsert mirrors insertSequence. PHP's T_OPEN_TAG holds the one space or
// newline after "<?php"; blink keeps it in the next Whitespace token instead.
func dstInsert(s *tokens.Stream, openTagIndex int) {
	wsIndex := openTagIndex + 1
	rest, hasWS := "", false
	if wsIndex < s.Len() && s.At(wsIndex).Kind == token.Whitespace {
		hasWS = true
		v := s.At(wsIndex).Value
		rest = v[1:]
		if strings.HasPrefix(v, "\r\n") {
			rest = v[2:]
		}
	}
	seq := []token.Token{
		{Kind: token.Whitespace, Value: " "},
		{Kind: token.Keyword, Value: "declare"},
		{Kind: token.Punct, Value: "("},
		{Kind: token.Ident, Value: "strict_types"},
		{Kind: token.Punct, Value: "="},
		{Kind: token.Number, Value: "1"},
		{Kind: token.Punct, Value: ")"},
		{Kind: token.Punct, Value: ";"},
	}
	end := openTagIndex // last token replaced
	if hasWS {
		end = wsIndex
	}
	more := end+1 < s.Len() || rest != ""
	if more {
		seq = append(seq, token.Token{Kind: token.Whitespace, Value: "\n" + strings.TrimLeft(rest, " \t")})
	}
	s.ReplaceRange(openTagIndex+1, end, seq)
}

// dstRemove mirrors removeStrictTypesDeclaration.
func dstRemove(s *tokens.Stream, declareIndex, open, closeIdx int, seq []int) {
	sequenceStart, sequenceEnd := seq[0], seq[len(seq)-1]
	inSeq := map[int]bool{}
	for _, i := range seq {
		inSeq[i] = true
	}
	hasOther := false
	for i := sigNext(s, open); i >= 0 && i < closeIdx; i = sigNext(s, i) {
		if taIsTString(s, i) && !inSeq[i] {
			hasOther = true
			break
		}
	}
	if !hasOther {
		semicolon := sigNext(s, closeIdx)
		if semicolon < 0 {
			return
		}
		for i := semicolon; i >= declareIndex; i-- {
			if !isEmptyToken(s, i) {
				clearTokenAndMergeWS(s, i)
			}
		}
		return
	}
	for i := sequenceEnd; i >= sequenceStart; i-- {
		if !isEmptyToken(s, i) {
			clearTokenAndMergeWS(s, i)
		}
	}
	if prev := sigPrev(s, sequenceStart); prev >= 0 && isPunctVal(s, prev, ",") {
		clearTokenAndMergeWS(s, prev)
		if next := sigNext(s, sequenceEnd); next >= 0 {
			dstRemoveLeadingBlanks(s, next)
		}
		return
	}
	if next := sigNext(s, sequenceEnd); next >= 0 && isPunctVal(s, next, ",") {
		clearTokenAndMergeWS(s, next)
		if after := sigNext(s, next); after >= 0 {
			dstRemoveLeadingBlanks(s, after)
		}
	}
}

// dstRemoveLeadingBlanks mirrors Tokens::removeLeadingWhitespace($index, " \t").
func dstRemoveLeadingBlanks(s *tokens.Stream, index int) {
	ws := nonEmptySibling(s, index, -1)
	if ws < 0 || s.At(ws).Kind != token.Whitespace {
		return
	}
	content, keep := s.At(ws).Value, ""
	if ws > 0 {
		if p := s.At(ws - 1); p.Kind == token.Comment && !strings.HasPrefix(p.Value, "/*") {
			// a single-line comment owns the newline that ends it
			nl := 0
			if strings.HasPrefix(content, "\r\n") {
				nl = 2
			} else if strings.HasPrefix(content, "\n") || strings.HasPrefix(content, "\r") {
				nl = 1
			}
			if nl == 0 {
				return
			}
			keep, content = content[:nl], content[nl:]
			if content == "" {
				return
			}
		}
	}
	if strings.Trim(content, " \t") != "" {
		return
	}
	if keep == "" {
		clearTokenAt(s, ws)
	} else {
		s.SetValue(ws, keep)
	}
}
