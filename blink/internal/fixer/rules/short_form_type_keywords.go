package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Keywords/ShortFormTypeKeywordsSniff.php
//
// ShortFormTypeKeywords rewrites the long cast forms to the short ones:
// "(boolean)" -> "(bool)", "(integer)" -> "(int)". Short forms in any case are left alone.
type ShortFormTypeKeywords struct{}

func (ShortFormTypeKeywords) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Keywords\ShortFormTypeKeywordsSniff`
}

func (ShortFormTypeKeywords) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Keywords/ShortFormTypeKeywordsSniff.php"
}

func (ShortFormTypeKeywords) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		closeIdx, typeIdx, ok := phpCastAt(s, i)
		if !ok {
			continue
		}
		switch strings.ToLower(s.At(typeIdx).Value) {
		case "boolean":
			s.SetValue(typeIdx, "bool")
			changed = true
		case "integer":
			s.SetValue(typeIdx, "int")
			changed = true
		}
		i = closeIdx
	}
	return changed
}
