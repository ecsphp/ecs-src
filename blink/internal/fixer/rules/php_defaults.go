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
	"PhpCsFixer\\Fixer\\ControlStructure\\YodaStyleFixer": {
		"equal": true, "identical": true, "less_and_greater": null
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
