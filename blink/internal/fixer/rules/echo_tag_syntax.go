package rules

import (
	"cmp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/EchoTagSyntaxFixer.php
//
// EchoTagSyntax turns "<?=" into "<?php echo" (format "long", the default), or
// "<?php echo"/"<?php print" into "<?=" (format "short").
type EchoTagSyntax struct {
	format                      string // "long" (default) or "short"
	longFunction                string // "echo" (default) or "print"
	shortenSimpleStatementsOnly bool   // default true
}

func (EchoTagSyntax) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\EchoTagSyntaxFixer`
}

func (EchoTagSyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/EchoTagSyntaxFixer.php"
}

func (f EchoTagSyntax) WithConfig(config map[string]any) fixer.Fixer {
	f.format, f.longFunction, f.shortenSimpleStatementsOnly = "long", "echo", true
	if v, ok := config["format"].(string); ok && (v == "long" || v == "short") {
		f.format = v
	}
	if v, ok := config["long_function"].(string); ok && (v == "echo" || v == "print") {
		f.longFunction = v
	}
	if v, ok := config["shorten_simple_statements_only"].(bool); ok {
		f.shortenSimpleStatementsOnly = v
	}
	return f
}

func (f EchoTagSyntax) Fix(s *tokens.Stream) bool {
	if f.format == "short" {
		return echoTagLongToShort(s, f.shortenSimpleStatementsOnly)
	}
	return echoTagShortToLong(s, cmp.Or(f.longFunction, "echo"))
}

func echoTagShortToLong(s *tokens.Stream, longFunction string) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.OpenTag || t.Value != "<?=" {
			continue
		}
		// PHP-CS-Fixer writes a "<?php " open tag; in blink the space is a token
		replace := []token.Token{
			{Kind: token.OpenTag, Value: "<?php"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Keyword, Value: longFunction},
		}
		if i+1 >= s.Len() || s.At(i+1).Kind != token.Whitespace {
			replace = append(replace, token.Token{Kind: token.Whitespace, Value: " "})
		}
		s.ReplaceRange(i, i, replace)
		i += len(replace) - 1
		changed = true
	}
	return changed
}

func echoTagLongToShort(s *tokens.Stream, simpleOnly bool) bool {
	changed := false
	count := s.Len()
	for index := 0; index < count; index++ {
		if t := s.At(index); t.Kind != token.OpenTag || t.Value == "<?=" {
			continue
		}
		echoPrint := sigNext(s, index)
		if echoPrint < 0 {
			break
		}
		if et := s.At(echoPrint); !kwIs(et, "echo") && !kwIs(et, "print") {
			index = echoPrint
			continue
		}
		if simpleOnly && echoTagIsComplexCode(s, echoPrint+1) {
			index = echoPrint
			continue
		}
		s.Set(index, token.Token{Kind: token.OpenTag, Value: "<?=", Pos: s.At(index).Pos})
		// PHP-CS-Fixer's "<?php " tag owns one whitespace character and
		// ensureWhitespaceAtIndex clears the rest: the whitespace after it goes.
		if index+1 < s.Len() && s.At(index+1).Kind == token.Whitespace {
			clearTokenAt(s, index+1)
		}
		echoTagFixWhitespaceBefore(s, echoPrint)
		clearTokenAndMergeWS(s, echoPrint)
		changed = true
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// echoTagIsComplexCode reports whether the tag content from index holds code
// after a semicolon, before the closing tag.
func echoTagIsComplexCode(s *tokens.Stream, index int) bool {
	semicolonFound := false
	for ; index < s.Len(); index++ {
		t := s.At(index)
		if t.Kind == token.CloseTag {
			return false
		}
		if t.Value == ";" {
			semicolonFound = true
		} else if semicolonFound && t.Kind != token.Whitespace {
			return true
		}
	}
	return false
}

// echoTagFixWhitespaceBefore clears the whitespace before the echo/print,
// unless a line comment precedes it (removing the newline would comment out
// the code).
func echoTagFixWhitespaceBefore(s *tokens.Stream, echoPrint int) {
	ws := nonEmptySibling(s, echoPrint, -1)
	if ws < 0 || s.At(ws).Kind != token.Whitespace {
		return
	}
	prev := nonEmptySibling(s, ws, -1)
	if prev >= 0 && s.At(prev).Kind == token.Comment && !strings.HasSuffix(s.At(prev).Value, "*/") {
		return
	}
	clearTokenAt(s, ws)
}
