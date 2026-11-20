package rules

import (
	"path/filepath"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Files/FileExtensionSniff.php
//
// FileExtension reports a class, interface, trait or enum in a ".php" file and
// a ".inc" file holding none of them.
type FileExtension struct{}

func (FileExtension) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Files\FileExtensionSniff`
}

func (FileExtension) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Files/FileExtensionSniff.php"
}

func (FileExtension) Fix(*tokens.Stream) bool { return false }

func (r FileExtension) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	open := -1
	for i, t := range f.toks {
		if t.kind == token.OpenTag && !strings.HasPrefix(t.orig, "<?=") {
			open = i
			break
		}
	}
	if open < 0 {
		return nil
	}
	name := filepath.Base(s.Path())
	ext := name
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		ext = name[dot:]
	}
	for i := open; i < len(f.toks); i++ {
		if kw := squizReportOO(f, i); kw != "" {
			if ext != ".php" {
				return nil
			}
			kw = f.toks[i].orig
			return []fixer.CodingStandardError{squizReportError(r.Name(), "ClassFound", f.toks[open].line,
				strings.ToUpper(kw[:1])+kw[1:]+` found in ".php" file; use ".inc" extension instead`)}
		}
	}
	if ext != ".inc" {
		return nil
	}
	return []fixer.CodingStandardError{squizReportError(r.Name(), "NoClass", f.toks[open].line,
		`No interface or class found in ".inc" file; use ".php" extension instead`)}
}
