package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitAssertNewNamesFixer.php
//
// PhpUnitAssertNewNames renames deprecated PHPUnit assertions to their new
// names (assertFileNotExists -> assertFileDoesNotExist, ...) inside test
// classes. Risky: assumes the methods are PHPUnit's own.
type PhpUnitAssertNewNames struct{}

var phpUnitAssertNewNamesMap = map[string]string{
	"assertnotisreadable":          "assertIsNotReadable",
	"assertnotiswritable":          "assertIsNotWritable",
	"assertdirectorynotexists":     "assertDirectoryDoesNotExist",
	"assertfilenotexists":          "assertFileDoesNotExist",
	"assertdirectorynotisreadable": "assertDirectoryIsNotReadable",
	"assertdirectorynotiswritable": "assertDirectoryIsNotWriteable",
	"assertfilenotisreadable":      "assertFileIsNotReadable",
	"assertfilenotiswritable":      "assertFileIsNotWriteable",
	"assertregexp":                 "assertMatchesRegularExpression",
	"assertnotregexp":              "assertDoesNotMatchRegularExpression",
}

func (PhpUnitAssertNewNames) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitAssertNewNamesFixer`
}

func (PhpUnitAssertNewNames) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitAssertNewNamesFixer.php"
}

func (PhpUnitAssertNewNames) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		for _, call := range puPreviousAssertCalls(s, cls[0], cls[1]) {
			if replacement, ok := phpUnitAssertNewNamesMap[call.lowered]; ok {
				s.SetValue(call.index, replacement)
				changed = true
			}
		}
	}
	return changed
}
