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

// braceDepthAt returns the number of "{" still open before index idx.
func braceDepthAt(s *tokens.Stream, idx int) int {
	d := 0
	for j := range idx {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "{":
			d++
		case "}":
			if d > 0 {
				d--
			}
		}
	}
	return d
}

// lineIndent returns the leading indentation of the line containing token idx.
func lineIndent(s *tokens.Stream, idx int) string {
	for j := idx; j >= 0; j-- {
		if s.At(j).Kind == token.Whitespace {
			if v := s.At(j).Value; strings.Contains(v, "\n") {
				return v[strings.LastIndexByte(v, '\n')+1:]
			}
		}
	}
	return ""
}
