package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceAfterCastSniff.php
//
// SpaceAfterCast (registered with Spacing 1, the sniff's default) puts exactly Spacing spaces after a cast ("(int)$a" -> "(int) $a").
// A comment after the cast is left alone; with IgnoreNewlines a cast followed
// by a line break is too.
type SpaceAfterCast struct {
	Spacing        int
	IgnoreNewlines bool
}

func (SpaceAfterCast) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Formatting\SpaceAfterCastSniff`
}

func (SpaceAfterCast) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceAfterCastSniff.php"
}

func (f SpaceAfterCast) WithConfig(config map[string]any) fixer.Fixer {
	f.Spacing = configInt(config, "spacing", f.Spacing)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

func (f SpaceAfterCast) Fix(s *tokens.Stream) bool {
	changed := false
	padding := strings.Repeat(" ", f.Spacing)
	for i := 0; i < s.Len(); i++ {
		closeIdx, _, ok := phpCastAt(s, i)
		if !ok {
			continue
		}
		i = closeIdx
		next := closeIdx + 1
		for next < s.Len() && isPhpcsEmpty(s.At(next)) {
			next++
		}
		if next >= s.Len() {
			continue
		}
		ws := closeIdx + 1
		hasWS := s.At(ws).Kind == token.Whitespace
		newline := hasWS && strings.Contains(s.At(ws).Value, "\n")
		if f.IgnoreNewlines && newline {
			continue
		}
		// a comment between the cast and its operand cannot be fixed
		if next != ws && (!hasWS || next != ws+1) {
			continue
		}
		switch {
		case !hasWS:
			if f.Spacing > 0 {
				s.InsertAt(ws, token.Token{Kind: token.Whitespace, Value: padding})
				changed = true
			}
		case s.At(ws).Value != padding:
			if f.Spacing > 0 {
				s.SetValue(ws, padding)
			} else {
				s.RemoveAt(ws)
			}
			changed = true
		}
	}
	return changed
}

func isPhpcsEmpty(t token.Token) bool {
	return t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment
}
