package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/LowercaseClassKeywordsSniff.php
//
// LowercaseClassKeywords lowercases the class-like declaration keywords:
// "Abstract Class Foo Extends Bar" -> "abstract class Foo extends Bar".
type LowercaseClassKeywords struct{}

func (LowercaseClassKeywords) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Classes\LowercaseClassKeywordsSniff`
}

func (LowercaseClassKeywords) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/LowercaseClassKeywordsSniff.php"
}

var classKeywords = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true,
	"extends": true, "implements": true, "abstract": true, "final": true,
	"readonly": true, "var": true, "const": true,
}

func (LowercaseClassKeywords) Fix(s *tokens.Stream) bool {
	return lowercaseKeywordsIn(s, classKeywords)
}
