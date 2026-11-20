package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/ClassCommentSniff.php
//
// PEARClassComment reports a missing or malformed class, interface, trait or
// enum docblock, checking its tags like PEAR FileComment. It fixes nothing.
type PEARClassComment struct{}

func (PEARClassComment) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Commenting\ClassCommentSniff`
}

func (PEARClassComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/ClassCommentSniff.php"
}

func (PEARClassComment) Fix(*tokens.Stream) bool { return false }

func (r PEARClassComment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	m := newSiModel(newSlvFile(lexer.Lex(s.Render())))
	f, toks := m.f, m.f.toks
	e := &pearDocErrors{f: f, checker: r.Name()}
	for i := range toks {
		if !pearOODeclaration(m, i) {
			continue
		}
		kind := strings.ToLower(toks[i].orig)
		end := i - 1
		for ; end >= 0; end-- {
			if toks[end].kind == token.Whitespace || m.code[end] == "abstract" || m.code[end] == "final" || m.code[end] == "readonly" {
				continue
			}
			if toks[end].kind == token.AttributeEnd && toks[end].closer >= 0 {
				end = toks[end].closer
				continue
			}
			break
		}
		isDocEnd := end >= 0 && toks[end].doc == slvDocClose
		if !isDocEnd && (end < 0 || !pearIsPlainComment(f, end)) {
			name := ""
			if n := f.nextEffective(i+1, len(toks)); n >= 0 {
				name = toks[n].orig
			}
			e.add(i, "Missing", "Missing doc comment for "+kind+" "+name)
			continue
		}
		if !isDocEnd {
			e.add(i, "WrongStyle", `You must use "/**" style comments for a `+kind+" comment")
			continue
		}
		open := end
		for open >= 0 && toks[open].doc != slvDocOpen {
			open--
		}
		e.processTags(open, end, "class")
	}
	return e.errs
}
