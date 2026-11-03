package rules

import (
	"blink/internal/fixer"
	"blink/internal/tokens"
)

var memberModifiers = map[string]bool{
	"public": true, "private": true, "protected": true, "static": true,
	"abstract": true, "final": true, "readonly": true, "var": true,
	"public(set)": true, "protected(set)": true, "private(set)": true,
}

var visibilityModifiers = map[string]bool{
	"public": true, "private": true, "protected": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/VisibilityRequiredFixer.php
//
// VisibilityRequired is PHP-CS-Fixer's deprecated proxy of ModifierKeywords:
// it runs ModifierKeywords with the same "elements" option.
type VisibilityRequired struct {
	proxy ModifierKeywords
}

func (f VisibilityRequired) WithConfig(config map[string]any) fixer.Fixer {
	f.proxy = f.proxy.WithConfig(config).(ModifierKeywords)
	return f
}

func (VisibilityRequired) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\VisibilityRequiredFixer`
}

func (VisibilityRequired) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/VisibilityRequiredFixer.php"
}

func (f VisibilityRequired) Fix(s *tokens.Stream) bool { return f.proxy.Fix(s) }
