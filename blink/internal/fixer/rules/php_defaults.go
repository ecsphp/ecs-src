package rules

import (
	"encoding/json"

	"blink/internal/fixer"
)

// phpDefaultsJSON holds PHP-CS-Fixer's default for each option whose blink zero
// value behaves differently. ECS's dump-config drops options left at their
// default, so the registry (All, ByName) applies these. A struct literal such as
// YodaStyle{} keeps its zero values. JSON gives WithConfig the types of a dumped config.
const phpDefaultsJSON = `{
	"PhpCsFixer\\Fixer\\ArrayNotation\\NoWhitespaceBeforeCommaInArrayFixer": {
		"after_heredoc": false
	},
	"PhpCsFixer\\Fixer\\Basic\\BracesPositionFixer": {
		"anonymous_functions_opening_brace": "same_line"
	},
	"PhpCsFixer\\Fixer\\Basic\\NoTrailingCommaInSinglelineFixer": {
		"elements": ["arguments", "array", "array_destructuring", "group_import"]
	},
	"PhpCsFixer\\Fixer\\ControlStructure\\YodaStyleFixer": {
		"equal": true, "identical": true, "less_and_greater": null
	},
	"PhpCsFixer\\Fixer\\FunctionNotation\\FunctionDeclarationFixer": {
		"trailing_comma_single_line": false
	},
	"PhpCsFixer\\Fixer\\FunctionNotation\\MethodArgumentSpaceFixer": {
		"after_heredoc": false
	},
	"PhpCsFixer\\Fixer\\FunctionNotation\\NullableTypeDeclarationForDefaultNullValueFixer": {
		"use_nullable_type_declaration": true
	},
	"PhpCsFixer\\Fixer\\Import\\OrderedImportsFixer": {
		"imports_order": null
	},
	"PhpCsFixer\\Fixer\\LanguageConstruct\\FunctionToConstantFixer": {
		"functions": ["get_called_class", "get_class", "get_class_this", "php_sapi_name", "phpversion", "pi"]
	},
	"PhpCsFixer\\Fixer\\LanguageConstruct\\SingleSpaceAroundConstructFixer": {
		"constructs_contain_a_single_space": ["yield_from"],
		"constructs_followed_by_a_single_space": [
			"abstract", "as", "attribute", "break", "case", "catch", "class", "clone", "comment",
			"const", "const_import", "continue", "do", "echo", "else", "elseif", "enum", "extends",
			"final", "finally", "for", "foreach", "function", "function_import", "global", "goto",
			"if", "implements", "include", "include_once", "instanceof", "insteadof", "interface",
			"match", "named_argument", "namespace", "new", "open_tag_with_echo", "php_doc",
			"php_open", "print", "private", "private_set", "protected", "protected_set", "public",
			"public_set", "readonly", "require", "require_once", "return", "static", "switch",
			"throw", "trait", "try", "type_colon", "use", "use_lambda", "use_trait", "var", "while",
			"yield", "yield_from"
		],
		"constructs_preceded_by_a_single_space": ["as", "use_lambda"]
	},
	"PhpCsFixer\\Fixer\\Operator\\ConcatSpaceFixer": {
		"spacing": "none"
	},
	"PhpCsFixer\\Fixer\\Operator\\UnaryOperatorSpacesFixer": {
		"only_dec_inc": false
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\AlignMultilineCommentFixer": {
		"comment_type": "phpdocs_only"
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\GeneralPhpdocAnnotationRemoveFixer": {
		"annotations": [], "case_sensitive": true
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\NoSuperfluousPhpdocTagsFixer": {
		"allow_mixed": false, "allow_unused_params": false
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\PhpdocInlineTagNormalizerFixer": {
		"tags": ["example", "id", "internal", "inheritdoc", "inheritdocs", "link", "source", "toc", "tutorial"]
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\PhpdocLineSpanFixer": {
		"case": "multi", "class": "multi", "const": "multi", "function": "multi",
		"method": "multi", "other": null, "property": "multi", "trait_import": null
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\PhpdocNoAliasTagFixer": {
		"replacements": {"property-read": "property", "property-write": "property", "type": "var", "link": "see"}
	},
	"PhpCsFixer\\Fixer\\Phpdoc\\PhpdocTypesFixer": {
		"groups": ["alias", "meta", "simple"]
	},
	"PhpCsFixer\\Fixer\\Whitespace\\NoSpacesAroundOffsetFixer": {
		"positions": ["inside", "outside"]
	},
	"PhpCsFixer\\Fixer\\Whitespace\\TypesSpacesFixer": {
		"space": "none", "space_multiple_catch": null
	}
}`

var phpDefaults = func() map[string]map[string]any {
	var m map[string]map[string]any
	if err := json.Unmarshal([]byte(phpDefaultsJSON), &m); err != nil {
		panic("rules: invalid phpDefaultsJSON: " + err.Error())
	}
	return m
}()

func withPHPDefaults(f fixer.Fixer) fixer.Fixer {
	if config, ok := phpDefaults[f.Name()]; ok {
		return f.(fixer.ConfigurableFixer).WithConfig(config)
	}
	return f
}
