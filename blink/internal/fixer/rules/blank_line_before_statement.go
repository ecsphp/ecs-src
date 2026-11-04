package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBeforeStatementFixer.php
//
// BlankLineBeforeStatement puts an empty line before each configured statement.
// A statement that opens its block (preceded by "{" or ":") keeps no blank line;
// an attached comment above the statement is hoisted with it.
type BlankLineBeforeStatement struct {
	// Statements is the configured statements option; nil keeps the default.
	Statements []string
}

func (BlankLineBeforeStatement) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\BlankLineBeforeStatementFixer`
}

func (BlankLineBeforeStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/BlankLineBeforeStatementFixer.php"
}

// blankLineBeforeStatementDefaults is the upstream default of the statements option.
var blankLineBeforeStatementDefaults = []string{"break", "continue", "declare", "return", "throw", "try"}

func (f BlankLineBeforeStatement) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["statements"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.Statements = append([]string{}, list...)
		}
	}
	return f
}

// blbsStatement names the statements option entry the token at i belongs to
// (TOKEN_MAP of the upstream fixer), or "" when it is none of them.
func blbsStatement(s *tokens.Stream, i int) string {
	t := s.At(i)
	switch t.Kind {
	case token.DocComment:
		return "phpdoc"
	case token.Ident:
		// exit and die are both T_EXIT; blink lexes them as identifiers
		if lower := strings.ToLower(t.Value); lower == "exit" || lower == "die" {
			if !isMemberAccessPrev(s, i) {
				return "exit"
			}
		}
		return ""
	case token.Keyword:
	default:
		return ""
	}
	lower := strings.ToLower(t.Value)
	switch lower {
	case "break", "case", "continue", "declare", "default", "do", "for", "foreach",
		"goto", "if", "include", "include_once", "require", "require_once", "return",
		"switch", "throw", "try", "while":
		return lower
	case "yield":
		return "yield"
	}
	// the lexer emits "yield from" (whitespace and comments included) as one
	// token, like PHP's T_YIELD_FROM
	if strings.HasPrefix(lower, "yield") && strings.HasSuffix(lower, "from") {
		return "yield_from"
	}
	return ""
}

func (f BlankLineBeforeStatement) Fix(s *tokens.Stream) bool {
	statements := f.Statements
	if statements == nil {
		statements = blankLineBeforeStatementDefaults
	}
	enabled := buildStringSet(statements)
	if len(enabled) == 0 {
		return false
	}
	changed := false
	for i := s.Len() - 1; i > 0; i-- {
		stmt := blbsStatement(s, i)
		if stmt == "" || !enabled[stmt] {
			continue
		}
		if stmt == "while" && isWhilePartOfDoWhile(s, i) {
			continue
		}
		if stmt == "case" && isEnumCase(s, i, true) {
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

// isWhilePartOfDoWhile mirrors TokensAnalyzer::isWhilePartOfDoWhile: the
// "while" at i follows a "}" that closes a "do" block.
func isWhilePartOfDoWhile(s *tokens.Stream, i int) bool {
	end := sigPrev(s, i)
	if end < 0 || s.At(end).Kind != token.Punct || s.At(end).Value != "}" {
		return false
	}
	start := s.MatchBackward(end)
	if start < 0 {
		return false
	}
	before := sigPrev(s, start)
	return before >= 0 && s.At(before).Kind == token.Keyword && strings.EqualFold(s.At(before).Value, "do")
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
