package rules

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DeprecatedFunctionsSniff.php
//
// DeprecatedFunctions is ForbiddenFunctions over the functions PHP flags
// deprecated (php_deprecated_functions.txt, from bin/php-builtin-functions.sh).
// ECS names its errors plain "Deprecated": it takes an error code that is
// a class name for the checker, and PHP 8.4 has a Deprecated attribute class.
type DeprecatedFunctions struct{ ForbiddenFunctions }

//go:embed php_deprecated_functions.txt
var deprecatedFunctionsList string

func (DeprecatedFunctions) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\DeprecatedFunctionsSniff`
}

func (DeprecatedFunctions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DeprecatedFunctionsSniff.php"
}

func (d DeprecatedFunctions) WithConfig(config map[string]any) fixer.Fixer {
	return DeprecatedFunctions{d.withDefaults().WithConfig(config).(ForbiddenFunctions)}
}

func (d DeprecatedFunctions) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return d.withDefaults().errors(s, func(function string, _ *string) (string, string) {
		return fmt.Sprintf("Function %s() has been deprecated", function), "Deprecated"
	})
}

func (d DeprecatedFunctions) withDefaults() ForbiddenFunctions {
	f := d.ForbiddenFunctions
	if f.forbidden == nil {
		f.forbidden = map[string]*string{}
		for name := range strings.FieldsSeq(deprecatedFunctionsList) {
			f.forbidden[name] = nil
		}
	}
	return f
}
