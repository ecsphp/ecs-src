package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/InlineCommentSniff.php
//
// SquizInlineComment rewrites "#" comments to "//", puts exactly one space
// after "//", drops a blank "//" comment and the blank lines after a comment
// on its own line.
type SquizInlineComment struct{}

func (SquizInlineComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\InlineCommentSniff`
}

func (SquizInlineComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/InlineCommentSniff.php"
}

func (f SquizInlineComment) Fix(s *tokens.Stream) bool {
	return phpcsFixLoop(s, squizInlineCommentPass)
}

func squizInlineCommentPass(s *tokens.Stream) bool {
	eol := phpcsEolChar(s)
	lines := phpcsLines(s, eol)
	endLine := func(i int) int {
		return lines[i] + strings.Count(strings.TrimSuffix(s.At(i).Value, eol), eol)
	}
	var edits []phpcsDocEdit
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if !isPhpcsComment(t) {
			continue
		}
		if strings.HasPrefix(t.Value, "#") {
			at, v := i, "// "+strings.TrimLeft(t.Value, "# \t")
			edits = append(edits, phpcsDocEdit{at, func(s *tokens.Stream) { s.SetValue(at, v) }})
		}
		prev := prevNonWhitespaceIdx(s, i)
		if prev >= 0 && endLine(prev) == lines[i] && isPunctVal(s, prev, "}") {
			continue
		}
		if !strings.HasPrefix(t.Value, "//") {
			continue
		}
		block := []int{i}
		for n := i + 1; n < s.Len(); n++ {
			last := block[len(block)-1]
			if lines[n] > lines[last]+1 {
				break
			}
			if !isPhpcsComment(s.At(n)) {
				continue
			}
			if lines[n] != lines[last]+1 || !strings.HasPrefix(s.At(n).Value, "//") || prevNonWhitespaceIdx(s, n) != last {
				break
			}
			block = append(block, n)
		}
		last := block[len(block)-1]
		blank := true
		for _, c := range block {
			v := phpRtrim(s.At(c).Value)
			if phpTrim(v[2:]) == "" {
				continue
			}
			if spacing := v[2:]; strings.HasPrefix(strings.TrimLeft(spacing, " "), "\t") || !strings.HasPrefix(spacing, " ") || strings.HasPrefix(spacing, "  ") {
				at, nv := c, "// "+strings.TrimLeft(s.At(c).Value, "/\t ")
				edits = append(edits, phpcsDocEdit{at, func(s *tokens.Stream) { s.SetValue(at, nv) }})
			}
			blank = false
		}
		i = last
		if blank {
			at := block[0]
			edits = append(edits, phpcsDocEdit{at, func(s *tokens.Stream) { removeLineComment(s, at) }})
			continue
		}
		if prev >= 0 && endLine(prev) >= lines[block[0]] {
			continue
		}
		next := nextNonWhitespaceIndex(s, last)
		if next < 0 || s.At(next).Kind == token.DocComment || lines[next] <= lines[last]+1 {
			continue
		}
		if w := last + 1; w < next {
			edits = append(edits, phpcsDocEdit{w, func(s *tokens.Stream) {
				v := s.At(w).Value
				s.SetValue(w, v[:phpcsCommentNewline(s, w)]+phpcsAfterLastBreak(v))
			}})
		}
	}
	return applyPhpcsEdits(s, edits)
}

// removeLineComment drops the comment at i with the line break PHP_CodeSniffer
// counts as part of it, merging the whitespace around it.
func removeLineComment(s *tokens.Stream, i int) {
	if n := i + 1; n < s.Len() && s.At(n).Kind == token.Whitespace {
		if k := phpcsCommentNewline(s, n); k > 0 {
			s.SetValue(n, s.At(n).Value[k:])
		}
	}
	s.RemoveAt(i)
	if i > 0 && i < s.Len() && s.At(i-1).Kind == token.Whitespace && s.At(i).Kind == token.Whitespace {
		s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
		s.RemoveAt(i)
	} else if i < s.Len() && s.At(i).Kind == token.Whitespace && s.At(i).Value == "" {
		s.RemoveAt(i)
	}
}
