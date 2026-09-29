package rules

import "blink/internal/tokens"

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocTagRenameFixer.php
//
// GeneralPhpdocTagRename renames PHPDoc tags according to a configured
// `replacements` map. The ECS set configures it with no replacements, so it is
// a no-op; it is kept as a registered no-op to match the set.
type GeneralPhpdocTagRename struct{}

func (GeneralPhpdocTagRename) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocTagRenameFixer`
}

func (GeneralPhpdocTagRename) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocTagRenameFixer.php"
}

func (GeneralPhpdocTagRename) Fix(*tokens.Stream) bool {
	return false
}
