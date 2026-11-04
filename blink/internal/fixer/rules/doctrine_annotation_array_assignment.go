package rules

import (
	"blink/internal/fixer"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationArrayAssignmentFixer.php
//
// DoctrineAnnotationArrayAssignment makes array assignments inside a Doctrine
// annotation use the configured operator option ("=" by default, or ":"), so
// `{bar : "baz"}` becomes `{bar = "baz"}`. The ignored_tags option replaces the
// default list of tags that are not Doctrine annotations.
type DoctrineAnnotationArrayAssignment struct {
	// Operator is the operator option; "" keeps the default "=".
	Operator string
	// ignoredTags is the ignored_tags option; nil keeps the default list.
	ignoredTags map[string]bool
}

func (DoctrineAnnotationArrayAssignment) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationArrayAssignmentFixer`
}

func (DoctrineAnnotationArrayAssignment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationArrayAssignmentFixer.php"
}

func (f DoctrineAnnotationArrayAssignment) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["operator"].(string); ok {
		f.Operator = v
	}
	if tags, ok := doctrineIgnoredTagsOption(config); ok {
		f.ignoredTags = tags
	}
	return f
}

func (f DoctrineAnnotationArrayAssignment) Fix(s *tokens.Stream) bool {
	operator := f.Operator
	if operator == "" {
		operator = "="
	}
	ignored := f.ignoredTags
	if ignored == nil {
		ignored = doctrineIgnoredTags
	}
	return applyToDoctrineAnnotations(s, ignored, func(toks []daToken) []daToken {
		var scopes []string
		for i := range toks {
			switch toks[i].typ {
			case daTOpenParen:
				scopes = append(scopes, "annotation")
			case daTOpenCurly:
				scopes = append(scopes, "array")
			case daTCloseParen, daTCloseCurly:
				if len(scopes) > 0 {
					scopes = scopes[:len(scopes)-1]
				}
			case daTEquals, daTColon:
				if len(scopes) > 0 && scopes[len(scopes)-1] == "array" {
					toks[i].content = operator
				}
			}
		}
		return toks
	})
}
