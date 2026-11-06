package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagNoNamedArgumentsFixer.php
//
// PhpdocTagNoNamedArguments ensures a @no-named-arguments tag in the PHPDoc of
// every class/enum/interface/trait.
type PhpdocTagNoNamedArguments struct {
	description  string
	fixAttribute *bool
	fixInternal  *bool
}

var phpdocNoNamedArgsLine = regexp.MustCompile(`@no-named-arguments.*\r?\n`)

func (PhpdocTagNoNamedArguments) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTagNoNamedArgumentsFixer`
}

func (PhpdocTagNoNamedArguments) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagNoNamedArgumentsFixer.php"
}

func (f PhpdocTagNoNamedArguments) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["description"].(string); ok {
		f.description = v
	}
	if v, ok := config["fix_attribute"].(bool); ok {
		f.fixAttribute = &v
	}
	if v, ok := config["fix_internal"].(bool); ok {
		f.fixInternal = &v
	}
	return f
}

func (f PhpdocTagNoNamedArguments) Fix(s *tokens.Stream) bool {
	fixAttribute := f.fixAttribute == nil || *f.fixAttribute
	fixInternal := f.fixInternal == nil || *f.fixInternal

	preventingAnnotations := []string{"no-named-arguments"}
	if !fixInternal {
		preventingAnnotations = []string{"internal", "no-named-arguments"}
	}

	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		if !cdIsClassy(s, index) {
			continue
		}
		if p := sigPrev(s, index); p >= 0 && kwIs(s.At(p), "new") {
			continue
		}
		if !fixAttribute && puIsPreventedByAttribute(s, index, []string{"attribute"}) {
			continue
		}

		lenBefore := s.Len()
		if puEnsureClassDocBlockAnnotation(s, index, "no-named-arguments", preventingAnnotations, nil) {
			changed = true
		}

		classIdx := index + (s.Len() - lenBefore)
		docBlockIndex := puAnnotationDocBlockIndex(s, classIdx)
		if docBlockIndex < 0 || s.At(docBlockIndex).Kind != token.DocComment {
			continue
		}

		content := s.At(docBlockIndex).Value
		replacement := strings.TrimRight("@no-named-arguments "+f.description, " \t\n\r\x00\x0b") + "\n"
		newContent := phpdocNoNamedArgsLine.ReplaceAllStringFunc(content, func(string) string {
			return replacement
		})
		if newContent != content {
			s.SetValue(docBlockIndex, newContent)
			changed = true
		}
	}
	return changed
}
