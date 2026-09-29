package rules

import "blink/internal/tokens"

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationArrayAssignmentFixer.php
//
// DoctrineAnnotationArrayAssignment makes array assignments inside a Doctrine
// annotation use the configured operator ("=" by default), so `{bar : "baz"}`
// becomes `{bar = "baz"}`.
type DoctrineAnnotationArrayAssignment struct{}

func (DoctrineAnnotationArrayAssignment) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationArrayAssignmentFixer`
}

func (DoctrineAnnotationArrayAssignment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationArrayAssignmentFixer.php"
}

func (DoctrineAnnotationArrayAssignment) Fix(s *tokens.Stream) bool {
	return applyToDoctrineAnnotations(s, func(toks []daToken) bool {
		changed := false
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
				if len(scopes) > 0 && scopes[len(scopes)-1] == "array" && toks[i].content != "=" {
					toks[i].content = "="
					changed = true
				}
			}
		}
		return changed
	})
}
