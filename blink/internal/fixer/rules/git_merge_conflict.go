package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/VersionControl/GitMergeConflictSniff.php
//
// GitMergeConflict reports git merge conflict boundaries that start a line, in
// code, comments, docblocks, heredocs and inline HTML. An opener in code starts
// an unterminated heredoc for PHP's tokenizer, as in PHP_CodeSniffer, so later
// boundaries are found in its lines. It fixes nothing.
type GitMergeConflict struct{}

func (GitMergeConflict) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\VersionControl\GitMergeConflictSniff`
}

func (GitMergeConflict) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/VersionControl/GitMergeConflictSniff.php"
}

func (GitMergeConflict) Fix(*tokens.Stream) bool { return false }

func (r GitMergeConflict) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	hasOpenTag := false
	for _, t := range f.toks {
		if t.kind == token.OpenTag {
			hasOpenTag = true
			break
		}
	}
	if !hasOpenTag {
		return nil
	}
	var errs []fixer.CodingStandardError
	add := func(i int, code, kind string) {
		errs = append(errs, fixer.CodingStandardError{
			Line:    f.toks[i].line,
			Message: "Merge conflict boundary found; type: " + kind,
			Checker: r.Name() + "." + code,
		})
	}
	for i := 0; i < len(f.toks); i++ {
		t := f.toks[i]
		if t.column != 1 {
			continue
		}
		switch {
		case f.is(i, "<<"):
			// T_SL T_SL T_STRING: "<<<<<<< HEAD" opens a heredoc PHP's tokenizer never closes
			if i+2 < len(f.toks) && f.is(i+1, "<<") && f.toks[i+2].heredoc &&
				strings.TrimSpace(f.toks[i+2].content) == "<<< HEAD" {
				add(i, "OpenerFound", "opener")
				i += 2
			}
		case f.is(i, ">>"):
			if i+4 < len(f.toks) && f.is(i+1, ">>") && f.is(i+2, ">>") && f.is(i+3, ">") &&
				f.toks[i+4].kind == token.Whitespace && f.toks[i+4].content == " " {
				add(i, "CloserFound", "closer")
				i += 4
			}
		case t.kind == token.Comment, t.kind == token.InlineHTML, t.heredoc && gitMergeConflictHeredocBody(f, i),
			t.kind == token.DocComment && t.doc == slvDocString:
			switch {
			case strings.HasPrefix(t.content, "<<<<<<< HEAD"):
				add(i, "OpenerFound", "opener")
			case strings.HasPrefix(t.content, ">>>>>>> "):
				add(i, "CloserFound", "closer")
			case t.kind == token.DocComment:
				if t.content == "=======" && i+1 < len(f.toks) && f.toks[i+1].doc == slvDocWhitespace {
					add(i, "DelimiterFound", "delimiter")
				}
			case t.content == "=======\n":
				add(i, "DelimiterFound", "delimiter")
			}
		}
	}
	return errs
}

// gitMergeConflictHeredocBody reports whether the heredoc line at i is body
// text (T_HEREDOC, T_NOWDOC, T_ENCAPSED_AND_WHITESPACE), not its start marker
// or the end marker closing it.
func gitMergeConflictHeredocBody(f *slvFile, i int) bool {
	start := i
	for start > 0 && f.toks[start-1].heredoc {
		start--
	}
	if i == start {
		return false
	}
	if i+1 < len(f.toks) && f.toks[i+1].heredoc {
		return true
	}
	label := strings.Trim(strings.TrimSpace(strings.TrimPrefix(f.toks[start].orig, "<<<")), `'"`)
	return label == "" || !strings.HasPrefix(strings.TrimLeft(f.toks[i].orig, " \t"), label)
}
