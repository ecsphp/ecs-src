package rules

import "github.com/ecsphp/ecs-src/blink/internal/fixer"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/FunctionCallSignatureSniff.php
//
// PSR2FunctionCallSignature is PEAR's FunctionCallSignature with one argument
// per line, where a call is multi-line once an argument or the closing
// parenthesis starts a new line.
type PSR2FunctionCallSignature struct {
	PEARFunctionCallSignature
}

func newPSR2FunctionCallSignature() PSR2FunctionCallSignature {
	return PSR2FunctionCallSignature{PEARFunctionCallSignature{Indent: 4, psr2MultiLine: true}}
}

func (PSR2FunctionCallSignature) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Methods\FunctionCallSignatureSniff`
}

func (PSR2FunctionCallSignature) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Methods/FunctionCallSignatureSniff.php"
}

func (c PSR2FunctionCallSignature) WithConfig(config map[string]any) fixer.Fixer {
	if pear, ok := c.PEARFunctionCallSignature.WithConfig(config).(PEARFunctionCallSignature); ok {
		c.PEARFunctionCallSignature = pear
	}
	return c
}
