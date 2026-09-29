package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBeforeStatementFixer.php
//
// BlankLineBeforeStatement puts an empty line before each configured statement.
// It mirrors the default set: break, continue, declare, return, throw, try. A
// statement that opens its block (preceded by "{" or ":") keeps no blank line;
// an attached comment above the statement is hoisted with it.
type BlankLineBeforeStatement struct{}

func (BlankLineBeforeStatement) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\BlankLineBeforeStatementFixer`
}

func (BlankLineBeforeStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBeforeStatementFixer.php"
}

var blankLineBeforeStatementKeywords = map[string]bool{
	"break": true, "continue": true, "declare": true,
	"return": true, "throw": true, "try": true,
}

func (BlankLineBeforeStatement) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i > 0; i-- {
		t := s.At(i)
		if t.Kind != token.Keyword || !blankLineBeforeStatementKeywords[strings.ToLower(t.Value)] {
			continue
		}
		insertIdx := blbsInsertIndex(s, i)
		prevNW := prevNonWhitespaceIdx(s, insertIdx)
		if prevNW >= 0 && blbsShouldAdd(s, prevNW) {
			if blbsInsert(s, insertIdx) {
				changed = true
			}
		}
		i = prevNW
		if i < 1 {
			break
		}
	}
	return changed
}

// prevNonWhitespaceIdx returns the previous token that is not whitespace
// (comments included, unlike prevSignificantIndex).
func prevNonWhitespaceIdx(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

func countNewlines(v string) int { return strings.Count(v, "\n") }

func isCommentTok(t token.Token) bool {
	return t.Kind == token.Comment || t.Kind == token.DocComment
}

// blbsInsertIndex hoists the insertion point above a comment that sits directly
// on the line before the statement, matching getInsertBlankLineIndex.
func blbsInsertIndex(s *tokens.Stream, index int) int {
	for index > 0 {
		if s.At(index-1).Kind == token.Whitespace && countNewlines(s.At(index-1).Value) > 1 {
			break
		}
		prevIndex := prevNonWhitespaceIdx(s, index)
		if prevIndex < 0 || !isCommentTok(s.At(prevIndex)) {
			break
		}
		if prevIndex-1 < 0 || s.At(prevIndex-1).Kind != token.Whitespace {
			break
		}
		if countNewlines(s.At(prevIndex-1).Value) != 1 {
			break
		}
		index = prevIndex
	}
	return index
}

// blbsShouldAdd reports whether the statement follows another statement or a
// block close ("}"), so an empty line belongs before it.
func blbsShouldAdd(s *tokens.Stream, prevNonWhitespace int) bool {
	prev := s.At(prevNonWhitespace)
	if isCommentTok(prev) {
		for j := prevNonWhitespace - 1; j >= 0; j-- {
			if strings.Contains(s.At(j).Value, "\n") {
				return false
			}
			if s.At(j).Kind == token.Whitespace || isCommentTok(s.At(j)) {
				continue
			}
			return s.At(j).Kind == token.Punct && (s.At(j).Value == ";" || s.At(j).Value == "}")
		}
		return false
	}
	return prev.Kind == token.Punct && (prev.Value == ";" || prev.Value == "}")
}

// blbsInsert ensures exactly one empty line before index. Returns whether it
// changed anything (keeps the fixer idempotent).
func blbsInsert(s *tokens.Stream, index int) bool {
	prevIndex := index - 1
	if prevIndex >= 0 && s.At(prevIndex).Kind == token.Whitespace {
		v := s.At(prevIndex).Value
		switch countNewlines(v) {
		case 0:
			s.SetValue(prevIndex, strings.TrimRight(v, " \t")+"\n\n")
			return true
		case 1:
			s.SetValue(prevIndex, "\n"+v)
			return true
		default:
			return false
		}
	}
	s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: "\n\n"})
	return true
}
