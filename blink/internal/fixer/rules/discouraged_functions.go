package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DiscouragedFunctionsSniff.php
//
// DiscouragedFunctions is ForbiddenFunctions over error_log, print_r and
// var_dump as warnings, which ECS drops for this sniff: it reports errors only
// when configured with error true.
type DiscouragedFunctions struct{ ForbiddenFunctions }

func (DiscouragedFunctions) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DiscouragedFunctionsSniff`
}

func (DiscouragedFunctions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DiscouragedFunctionsSniff.php"
}

func (d DiscouragedFunctions) WithConfig(config map[string]any) fixer.Fixer {
	return DiscouragedFunctions{d.withDefaults().WithConfig(config).(ForbiddenFunctions)}
}

func (d DiscouragedFunctions) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := d.withDefaults()
	return f.errors(s, func(function string, alternative *string) (string, string) {
		message, checker := f.message(function, alternative)
		return message, d.Name() + checker
	})
}

func (d DiscouragedFunctions) withDefaults() ForbiddenFunctions {
	f := d.ForbiddenFunctions
	if f.forbidden == nil {
		f.forbidden = map[string]*string{"error_log": nil, "print_r": nil, "var_dump": nil}
		f.warning = true
	}
	return f
}
