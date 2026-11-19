package rules

import (
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LineLengthSniff.php
//
// GenericLineLength reports lines longer than AbsoluteLineLimit. Lines over
// LineLimit only warn upstream, which ECS drops. It fixes nothing.
type GenericLineLength struct {
	LineLimit         int
	AbsoluteLineLimit int
	IgnoreComments    bool
}

func (GenericLineLength) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\LineLengthSniff`
}

func (GenericLineLength) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/LineLengthSniff.php"
}

func (l GenericLineLength) WithConfig(config map[string]any) fixer.Fixer {
	l.LineLimit = configInt(config, "lineLimit", l.LineLimit)
	l.AbsoluteLineLimit = configInt(config, "absoluteLineLimit", l.AbsoluteLineLimit)
	if v, ok := config["ignoreComments"].(bool); ok {
		l.IgnoreComments = v
	}
	return l
}

func (GenericLineLength) Fix(*tokens.Stream) bool { return false }

func (l GenericLineLength) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	var errs []fixer.CodingStandardError
	for i := 1; i < len(f.toks); i++ {
		if f.toks[i].column == 1 {
			errs = l.check(f, i, errs)
		}
	}
	return l.check(f, len(f.toks), errs)
}

func (l GenericLineLength) check(f *slvFile, i int, errs []fixer.CodingStandardError) []fixer.CodingStandardError {
	i--
	if i < 0 {
		return errs
	}
	t := f.toks[i]
	if t.column == 1 && t.length == 0 {
		return errs
	}
	if t.column != 1 && t.content == f.eol && i > 0 {
		i--
		t = f.toks[i]
	}
	comment := t.kind == token.Comment || t.kind == token.DocComment
	onlyComment := false
	if comment {
		p := i - 1
		for p >= 0 && f.isEmpty(p) {
			p--
		}
		onlyComment = t.line != f.toks[max(p, 0)].line
	}
	if onlyComment && t.kind == token.Comment && isPhpcsAnnotationComment(t.orig) {
		return errs
	}
	length := t.column + t.length - 1
	if l.IgnoreComments && comment {
		if onlyComment {
			return errs
		}
		length -= t.length
	}
	if onlyComment && length > l.LineLimit {
		// an unbreakable comment (a long URL) is left alone
		indent := t.column - 1 + len(t.content) - len(strings.TrimLeft(t.content, "/#\t "))
		nonBreaking := t.length
		if space := strings.LastIndexByte(t.content, ' '); space >= 0 {
			nonBreaking -= space + 1
		}
		if nonBreaking+indent > l.LineLimit {
			return errs
		}
	}
	if l.AbsoluteLineLimit > 0 && length > l.AbsoluteLineLimit {
		errs = append(errs, fixer.CodingStandardError{
			Line:    t.line,
			Message: "Line exceeds maximum limit of " + strconv.Itoa(l.AbsoluteLineLimit) + " characters; contains " + strconv.Itoa(length) + " characters",
			Checker: l.Name() + ".MaxExceeded",
		})
	}
	return errs
}
