package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderNameFixer.php
//
// PhpUnitDataProviderName renames a data provider to match the single test that
// uses it: the test name minus its "test" prefix, wrapped with the configured
// prefix and suffix (default "provide"/"Cases"). Risky.
type PhpUnitDataProviderName struct {
	prefix string
	suffix string
	set    bool
}

func (PhpUnitDataProviderName) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDataProviderNameFixer`
}

func (PhpUnitDataProviderName) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderNameFixer.php"
}

func (f PhpUnitDataProviderName) WithConfig(config map[string]any) fixer.Fixer {
	f.prefix, f.suffix, f.set = "provide", "Cases", true
	if v, ok := config["prefix"].(string); ok {
		f.prefix = v
	}
	if v, ok := config["suffix"].(string); ok {
		f.suffix = v
	}
	return f
}

var puStripTestPrefixRe = regexp.MustCompile(`(?i)^test_*`)

func (f PhpUnitDataProviderName) Fix(s *tokens.Stream) bool {
	prefix, suffix := f.prefix, f.suffix
	if !f.set {
		prefix, suffix = "provide", "Cases"
	}
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if nameProviderClass(s, cls[0], cls[1], prefix, suffix) {
			changed = true
		}
	}
	return changed
}

func nameProviderClass(s *tokens.Stream, start, end int, prefix, suffix string) bool {
	providers := puDataProviders(s, start, end)

	methodsProviders := map[int]map[string]bool{}
	providersMethods := map[string]map[int]bool{}
	for _, dp := range providers {
		for _, u := range dp.usages {
			testFn := puNextFunctionKeyword(s, u.index, end)
			if testFn < 0 {
				continue
			}
			if methodsProviders[testFn] == nil {
				methodsProviders[testFn] = map[string]bool{}
			}
			methodsProviders[testFn][dp.name] = true
			if providersMethods[dp.name] == nil {
				providersMethods[dp.name] = map[int]bool{}
			}
			providersMethods[dp.name][testFn] = true
		}
	}

	changed := false
	for _, dp := range providers {
		if len(providersMethods[dp.name]) > 1 {
			continue
		}
		testFn := puNextFunctionKeyword(s, dp.usages[0].index, end)
		if testFn < 0 || len(methodsProviders[testFn]) > 1 {
			continue
		}
		newName := dataProviderNewName(s, testFn, prefix, suffix)
		if newName == "" || puFindFunctionNamed(s, start, end, newName) >= 0 {
			continue
		}
		s.SetValue(dp.nameIndex, newName)
		for _, u := range dp.usages {
			t := s.At(u.index)
			if t.Kind == token.DocComment {
				s.SetValue(u.index, renameDataProviderDoc(t.Value, dp.name, newName))
			} else {
				q := string(t.Value[0])
				s.SetValue(u.index, q+newName+q)
			}
		}
		changed = true
	}
	return changed
}

// dataProviderNewName mirrors getDataProviderNameForUsageIndex: strip the test
// prefix off the test method name, then wrap with prefix and suffix.
func dataProviderNewName(s *tokens.Stream, testFn int, prefix, suffix string) string {
	nameIdx := sigNext(s, testFn)
	if nameIdx < 0 || s.At(nameIdx).Kind != token.Ident {
		return ""
	}
	name := puStripTestPrefixRe.ReplaceAllString(s.At(nameIdx).Value, "")
	switch {
	case prefix == "":
		name = ptaLcfirst(name)
	case !strings.HasSuffix(prefix, "_"):
		name = ptaUcfirst(name)
	}
	return prefix + name + suffix
}

// renameDataProviderDoc rewrites `@dataProvider <oldName>` references to newName.
func renameDataProviderDoc(content, oldName, newName string) string {
	re := regexp.MustCompile(`(@dataProvider\s+)` + regexp.QuoteMeta(oldName))
	return re.ReplaceAllString(content, "${1}"+newName)
}
