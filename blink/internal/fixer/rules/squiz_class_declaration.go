package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ClassDeclarationSniff.php
//
// SquizClassDeclaration runs PSR2's opening checks, puts the keyword of an
// unmodified declaration at column 1, and wants the closing brace alone on its
// line, at column 1, followed by one blank line.
type SquizClassDeclaration struct {
	Indent int
}

func (SquizClassDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Classes\ClassDeclarationSniff`
}

func (SquizClassDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ClassDeclarationSniff.php"
}

func (r SquizClassDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r SquizClassDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open := phpcsClassyOpener(f, i); open >= 0 {
				pearClassDeclaration(f, i, open)
				d := classDecl{keyword: i, open: open, parenOpen: -1, indent: r.Indent}
				d.processOpen(f)
				squizSpaceBeforeKeyword(f, i)
				squizClassClose(f, open)
			}
		}
	})
}

func squizSpaceBeforeKeyword(f *slvFile, stackPtr int) {
	if stackPtr < 1 || !f.isWS(stackPtr-1) || f.toks[stackPtr-1].content == f.eol {
		return
	}
	if f.isKeyword(stackPtr-2, "abstract") || f.isKeyword(stackPtr-2, "final") || f.isKeyword(stackPtr-2, "readonly") {
		return
	}
	f.replaceToken(stackPtr-1, "")
}

func squizClassClose(f *slvFile, open int) {
	closeBrace := f.toks[open].closer
	if closeBrace < 0 {
		return
	}
	n := len(f.toks)
	line := func(i int) int { return f.toks[i].line }

	nextContent := closeBrace + 1
	for ; nextContent < n; nextContent++ {
		if line(nextContent) == line(closeBrace) &&
			(f.isWS(nextContent) || f.toks[nextContent].kind == token.Comment || phpcsIsAnnotation(f, nextContent)) {
			continue
		}
		if !f.isWS(nextContent) {
			break
		}
	}

	difference := 1
	if nextContent < n {
		difference = line(nextContent) - line(closeBrace) - 1
	}

	lastContent := f.prevNonWhitespace(closeBrace - 1)

	if difference == -1 || line(lastContent) == line(closeBrace) {
		if difference == -1 {
			f.addContentBefore(nextContent, f.eol)
		}
		if line(lastContent) == line(closeBrace) {
			f.addContentBefore(closeBrace, f.eol)
		}
	} else if f.isWS(closeBrace - 1) {
		if f.toks[closeBrace-1].content != f.eol && line(closeBrace-1) == line(closeBrace) {
			f.replaceToken(closeBrace-1, "")
		}
	}

	if difference == -1 || difference == 1 {
		return
	}

	next := nextContent
	for ; next < n; next++ {
		if f.isWS(next) {
			continue
		}
		if f.toks[next].doc == slvDocOpen {
			for next < n && f.toks[next].doc != slvDocClose {
				next++
			}
			continue
		}
		if f.toks[next].kind == token.AttributeStart && f.toks[next].closer >= 0 {
			next = f.toks[next].closer
			continue
		}
		break
	}
	if next < n && f.isKeyword(next, "function") {
		return
	}

	if difference == 0 {
		f.addContentBefore(f.firstOnLine(nextContent), f.eol)
		return
	}
	f.beginChangeset()
	for i := closeBrace + 1; i < nextContent; i++ {
		if line(i) <= line(closeBrace)+1 {
			continue
		}
		if line(i) == line(nextContent) {
			break
		}
		f.replaceToken(i, "")
	}
	f.endChangeset()
}
