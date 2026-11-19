package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/ClassCommentSniff.php
//
// SquizClassComment reports a class without a "/**" doc comment right above it.
// Its TagNotAllowed warning is dropped by ECS, so it is not ported.
type SquizClassComment struct{}

func (SquizClassComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\ClassCommentSniff`
}

func (SquizClassComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/ClassCommentSniff.php"
}

func (SquizClassComment) Fix(*tokens.Stream) bool { return false }

func (r SquizClassComment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if squizReportOO(f, i) != "class" {
			continue
		}
		previous := -1
		end := i - 1
		for ; end >= 0; end-- {
			if f.toks[end].kind == token.Whitespace || f.isKeyword(end, "abstract") || f.isKeyword(end, "final") || f.isKeyword(end, "readonly") {
				continue
			}
			if previous < 0 {
				previous = end
			}
			if f.toks[end].kind == token.AttributeEnd && f.toks[end].closer >= 0 {
				end = f.toks[end].closer
				continue
			}
			break
		}
		end = max(end, 0)
		isDoc := f.toks[end].doc == slvDocClose
		isComment := f.toks[end].kind == token.Comment && !isPhpcsAnnotationComment(f.toks[end].orig)
		switch {
		case !isDoc && !isComment:
			errs = append(errs, squizReportError(r.Name(), "Missing", f.toks[i].line,
				"Missing doc comment for class "+squizReportDeclarationName(f, sc, i)))
		case isComment:
			errs = append(errs, squizReportError(r.Name(), "WrongStyle", f.toks[i].line,
				`You must use "/**" style comments for a class comment`))
		case f.toks[previous].line != f.toks[i].line-1:
			errs = append(errs, squizReportError(r.Name(), "SpacingAfter", f.toks[end].line,
				"There must be no blank lines after the class comment"))
		}
	}
	return errs
}
