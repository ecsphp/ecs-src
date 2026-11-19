package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ClassFileNameSniff.php
//
// SquizClassFileName reports a class, interface, trait or enum whose name is
// not the file name without its extension.
type SquizClassFileName struct{}

func (SquizClassFileName) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Classes\ClassFileNameSniff`
}

func (SquizClassFileName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ClassFileNameSniff.php"
}

func (SquizClassFileName) Fix(*tokens.Stream) bool { return false }

func (r SquizClassFileName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	path := s.Path()
	if path == "" {
		return nil
	}
	name := filepath.Base(path)
	dot := strings.LastIndexByte(name, '.')
	noExt, ext := "", name
	if dot >= 0 {
		noExt, ext = name[:dot], name[dot+1:]
	}
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if squizReportOO(f, i) == "" {
			continue
		}
		oo := squizReportDeclarationName(f, sc, i)
		if oo == "" || oo == noExt {
			continue
		}
		errs = append(errs, squizReportError(r.Name(), "NoMatch", f.toks[i].line,
			fmt.Sprintf(`Filename doesn't match %s name; expected file name "%s"`, f.toks[i].orig, oo+"."+ext)))
	}
	return errs
}
