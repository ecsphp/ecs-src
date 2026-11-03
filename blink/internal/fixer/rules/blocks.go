package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// braceKind is the construct an opening "{" belongs to. It is the small scope
// model the structural fixers reason over.
type braceKind int

const (
	braceOther        braceKind = iota // free block, trait adaptation, match arm, ...
	braceClassLike                     // class / interface / trait / enum body
	braceFunctionDecl                  // named function or method body
	braceClosure                       // anonymous function body
	braceControl                       // if / for / while / switch / try / ...
)

var controlKeywords = map[string]bool{
	"if": true, "elseif": true, "else": true, "for": true, "foreach": true,
	"while": true, "do": true, "switch": true, "try": true, "catch": true,
	"finally": true, "match": true,
}

// classifyBrace determines what construct the "{" at brace opens, and returns
// the index of the governing keyword (for indentation). It walks the header
// backwards, jumping over (...) groups, until it meets a construct keyword or a
// statement boundary.
func classifyBrace(s *tokens.Stream, brace int) (braceKind, int) {
	// a "{" right after "->", "?->" or "::" is dynamic access ("$o->{$p}"), not a block
	if p := prevMeaningfulIndex(s, brace); p >= 0 {
		if pt := s.At(p); pt.Kind == token.Punct && (pt.Value == "->" || pt.Value == "?->" || pt.Value == "::") {
			return braceOther, -1
		}
	}
	for j := brace - 1; j >= 0; j-- {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case ")":
				open := s.MatchBackward(j)
				if open < 0 {
					return braceOther, -1
				}
				j = open // loop's j-- steps before the "("
				continue
			case ";", "{", "}":
				return braceOther, -1 // statement boundary before any keyword
			}
			continue
		}
		if t.Kind == token.Keyword {
			lw := strings.ToLower(t.Value)
			switch {
			case classLikeKeywords[lw]:
				return braceClassLike, j
			case lw == "function":
				return functionBraceKind(s, j), j
			case controlKeywords[lw]:
				return braceControl, j
			}
		}
	}
	return braceOther, -1
}

// functionBraceKind tells a named function/method (has a name) from a closure
// (a "(" directly follows "function", optionally after a "&").
func functionBraceKind(s *tokens.Stream, fn int) braceKind {
	k := skipWhitespace(s, fn+1)
	if k < s.Len() && s.At(k).Kind == token.Punct && s.At(k).Value == "&" {
		k = skipWhitespace(s, k+1)
	}
	if k < s.Len() && s.At(k).Kind == token.Punct && s.At(k).Value == "(" {
		return braceClosure
	}
	return braceFunctionDecl
}

// classMemberStarts returns the index of the first significant token of each
// direct member declaration in the class body opened at open. Nested bodies
// (method bodies, defaults, argument lists) are skipped.
func classMemberStarts(s *tokens.Stream, open int) []int {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var starts []int
	expect := true
	for k := open + 1; k < closeIdx; k++ {
		t := s.At(k)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if t.Kind == token.AttributeStart {
			// an attribute decorates the member that follows it; skip it as trivia so
			// the member start is the keyword, as when attributes were comment tokens
			if end := s.AttributeEnd(k); end > k {
				k = end
			}
			continue
		}
		if expect {
			starts = append(starts, k)
			expect = false
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "{", "(", "[":
				if m := s.MatchForward(k); m > 0 {
					wasBody := t.Value == "{"
					k = m
					if wasBody {
						expect = true // a method body closed; next token starts a member
					}
				}
			case ";":
				expect = true
			}
		}
	}
	return starts
}

// lineIndent is PHP-CS-Fixer's IndentationTrait::getLineIndentation: the
// spaces and tabs after the last line break of the nearest whitespace or
// inline-HTML token before idx that has one.
func lineIndent(s *tokens.Stream, idx int) string {
	for j := idx - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Whitespace && t.Kind != token.InlineHTML {
			continue
		}
		if _, rest := splitTagWhitespace(s, j); rest == "" {
			continue // PHP has no such token: the tag before it owns all of it
		}
		// the tag's part counts too: PHP adds it back (computeNewLineContent)
		v := t.Value
		nl := strings.LastIndexAny(v, "\r\n")
		if nl < 0 {
			continue
		}
		line := v[nl+1:]
		return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	}
	return ""
}

// splitTagWhitespace splits token j where PHP's lexer differs from blink's:
// PHP gives "<?php" the space, tab or line break after it, and "?>" the line
// break after it. tag is that part, rest the remainder.
func splitTagWhitespace(s *tokens.Stream, j int) (tag, rest string) {
	v := s.At(j).Value
	if j == 0 {
		return "", v
	}
	prev := s.At(j - 1)
	switch {
	case prev.Kind == token.OpenTag && strings.EqualFold(prev.Value, "<?php") && s.At(j).Kind == token.Whitespace:
		if v != "" && (v[0] == ' ' || v[0] == '\t') {
			return v[:1], v[1:]
		}
	case prev.Kind == token.CloseTag && s.At(j).Kind == token.InlineHTML:
	default:
		return "", v
	}
	n := 0
	switch {
	case strings.HasPrefix(v, "\r\n"):
		n = 2
	case strings.HasPrefix(v, "\n"), strings.HasPrefix(v, "\r"):
		n = 1
	}
	return v[:n], v[n:]
}
