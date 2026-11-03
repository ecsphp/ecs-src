package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/BlankLineAfterOpeningTagFixer.php
//
// BlankLineAfterOpeningTag ensures one blank line after the opening "<?php" tag:
// "<?php\n$x" becomes "<?php\n\n$x", and code on the tag line moves below
// it ("<?php $x;\n" -> "<?php\n\n$x;\n"). Like PHP-CS-Fixer it only runs on
// monolithic PHP without "<?=", and skips files with no newline at all.
type BlankLineAfterOpeningTag struct{}

func (BlankLineAfterOpeningTag) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\BlankLineAfterOpeningTagFixer`
}

func (BlankLineAfterOpeningTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/BlankLineAfterOpeningTagFixer.php"
}

func (BlankLineAfterOpeningTag) Fix(s *tokens.Stream) bool {
	if !nasIsMonolithic(s) {
		return false
	}
	newlineFound := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.OpenTag && t.Value == "<?=" {
			return false
		}
		if t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			newlineFound = true
		}
	}
	if !newlineFound {
		return false
	}
	open := 0
	if s.At(0).Kind == token.InlineHTML {
		open = 1
	}
	if open >= s.Len() || s.At(open).Kind != token.OpenTag || !strings.EqualFold(s.At(open).Value, "<?php") {
		return false
	}
	ws := ""
	hasWS := open+1 < s.Len() && s.At(open+1).Kind == token.Whitespace
	if hasWS {
		ws = s.At(open + 1).Value
	}
	// PHP's open tag token swallows one space, tab or newline after "<?php";
	// without a newline in it, the fixer rewrites the tag to "<?php\n".
	tagTail, rest := openTagSplit(ws)
	if !strings.Contains(tagTail, "\n") {
		tagTail = "\n"
	}
	// then the token after the tag must hold a newline, if there is one: in
	// PHP, "<?php\n" alone is a single token and the fixer stops there
	next := open + 1
	if hasWS {
		next++
	}
	if !strings.Contains(rest, "\n") && (rest != "" || next < s.Len()) {
		rest = "\n" + rest
	}
	want := tagTail + rest
	if hasWS {
		if want == ws {
			return false
		}
		s.SetValue(open+1, want)
		return true
	}
	s.InsertAt(open+1, token.Token{Kind: token.Whitespace, Value: want})
	return true
}

// openTagSplit splits the whitespace after "<?php" into the part PHP's
// T_OPEN_TAG includes (one space, tab or newline) and the rest.
func openTagSplit(ws string) (tagTail, rest string) {
	switch {
	case strings.HasPrefix(ws, "\r\n"):
		return ws[:2], ws[2:]
	case ws != "" && strings.ContainsRune(" \t\n\r", rune(ws[0])):
		return ws[:1], ws[1:]
	}
	return "", ws
}
