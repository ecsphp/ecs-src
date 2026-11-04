package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitNamespacedFixer.php
//
// PhpUnitNamespaced rewrites the old underscore PHPUnit class names to their
// namespaced form (PHPUnit_Framework_TestCase -> PHPUnit\Framework\TestCase).
// The default target (newest) matches every PHPUnit_* class and uses a
// dedicated map for the classes whose namespaced name differs. Risky: assumes
// PHPUnit's own classes.
type PhpUnitNamespaced struct {
	target string
}

func (PhpUnitNamespaced) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitNamespacedFixer`
}

func (PhpUnitNamespaced) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitNamespacedFixer.php"
}

func (f PhpUnitNamespaced) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["target"].(string); ok {
		f.target = v
	}
	return f
}

// phpUnitNamespacedClassMap is the target=6.0/newest dedicated map (lowercased
// keys), for classes whose namespaced name is not the underscore-to-backslash
// transform of the original.
var phpUnitNamespacedClassMap = map[string]string{
	"phpunit_extensions_phpttestcase":                               `PHPUnit\Runner\PhptTestCase`,
	"phpunit_framework_constraint":                                  `PHPUnit\Framework\Constraint\Constraint`,
	"phpunit_framework_constraint_stringmatches":                    `PHPUnit\Framework\Constraint\StringMatchesFormatDescription`,
	"phpunit_framework_constraint_jsonmatches_errormessageprovider": `PHPUnit\Framework\Constraint\JsonMatchesErrorMessageProvider`,
	"phpunit_framework_constraint_pcrematch":                        `PHPUnit\Framework\Constraint\RegularExpression`,
	"phpunit_framework_constraint_exceptionmessageregexp":           `PHPUnit\Framework\Constraint\ExceptionMessageRegularExpression`,
	"phpunit_framework_constraint_and":                              `PHPUnit\Framework\Constraint\LogicalAnd`,
	"phpunit_framework_constraint_or":                               `PHPUnit\Framework\Constraint\LogicalOr`,
	"phpunit_framework_constraint_not":                              `PHPUnit\Framework\Constraint\LogicalNot`,
	"phpunit_framework_constraint_xor":                              `PHPUnit\Framework\Constraint\LogicalXor`,
	"phpunit_framework_error":                                       `PHPUnit\Framework\Error\Error`,
	"phpunit_framework_testsuite_dataprovider":                      `PHPUnit\Framework\DataProviderTestSuite`,
	"phpunit_framework_mockobject_invocation_static":                `PHPUnit\Framework\MockObject\Invocation\StaticInvocation`,
	"phpunit_framework_mockobject_invocation_object":                `PHPUnit\Framework\MockObject\Invocation\ObjectInvocation`,
	"phpunit_framework_mockobject_stub_return":                      `PHPUnit\Framework\MockObject\Stub\ReturnStub`,
	"phpunit_runner_filter_group_exclude":                           `PHPUnit\Runner\Filter\ExcludeGroupFilterIterator`,
	"phpunit_runner_filter_group_include":                           `PHPUnit\Runner\Filter\IncludeGroupFilterIterator`,
	"phpunit_runner_filter_test":                                    `PHPUnit\Runner\Filter\NameFilterIterator`,
	"phpunit_util_php":                                              `PHPUnit\Util\PHP\AbstractPhpProcess`,
	"phpunit_util_php_default":                                      `PHPUnit\Util\PHP\DefaultPhpProcess`,
	"phpunit_util_php_windows":                                      `PHPUnit\Util\PHP\WindowsPhpProcess`,
	"phpunit_util_regex":                                            `PHPUnit\Util\RegularExpression`,
	"phpunit_util_testdox_resultprinter_xml":                        `PHPUnit\Util\TestDox\XmlResultPrinter`,
	"phpunit_util_testdox_resultprinter_html":                       `PHPUnit\Util\TestDox\HtmlResultPrinter`,
	"phpunit_util_testdox_resultprinter_text":                       `PHPUnit\Util\TestDox\TextResultPrinter`,
	"phpunit_util_testsuiteiterator":                                `PHPUnit\Framework\TestSuiteIterator`,
	"phpunit_util_xml":                                              `PHPUnit\Util\Xml`,
}

func (f PhpUnitNamespaced) Fix(s *tokens.Stream) bool {
	match := f.matcher()
	imported := map[string]bool{}
	changed := false
	currIndex := 0
	for {
		currIndex = nsNextIdent(s, currIndex)
		if currIndex < 0 {
			break
		}
		prev := sigPrev(s, currIndex)
		if prev >= 0 && (kwIs(s.At(prev), "const") || isPunctVal(s, prev, "::")) {
			continue
		}
		original := s.At(currIndex).Value
		allowed := nsIsClassyInvocation(s, currIndex) || nsIsImport(s, currIndex)
		if !allowed || !match(original) {
			currIndex++
			continue
		}
		repl := nsGenerateReplacement(original)
		if imported[original] {
			repl = repl[len(repl)-1:]
		}
		s.ReplaceRange(currIndex, currIndex, repl)
		changed = true

		prev = sigPrev(s, currIndex)
		if prev >= 0 && kwIs(s.At(prev), "use") {
			imported[original] = true
		} else if prev >= 0 && isPunctVal(s, prev, `\`) {
			if pp := sigPrev(s, prev); pp >= 0 && kwIs(s.At(pp), "use") {
				imported[original] = true
			}
		}
	}
	return changed
}

// matcher returns the predicate for the configured target: which PHPUnit_*
// names are rewritten. The classMap is only non-empty for target 6.0/newest.
func (f PhpUnitNamespaced) matcher() func(string) bool {
	switch {
	case phpUnitTargetFulfills(f.target, "6.0"):
		return func(name string) bool {
			l := strings.ToLower(name)
			return strings.HasPrefix(l, "phpunit_") && len(l) > len("phpunit_")
		}
	case phpUnitTargetFulfills(f.target, "5.7"):
		return func(name string) bool {
			switch strings.ToLower(name) {
			case "phpunit_framework_testcase", "phpunit_framework_assert",
				"phpunit_framework_basetestlistener", "phpunit_framework_testlistener":
				return true
			}
			return false
		}
	default:
		return func(name string) bool {
			return strings.EqualFold(name, "phpunit_framework_testcase")
		}
	}
}

// nsGenerateReplacement ports generateReplacement: build the namespaced token
// sequence for an original class name.
func nsGenerateReplacement(original string) []token.Token {
	delimiter := "_"
	str := original
	if mapped, ok := phpUnitNamespacedClassMap[strings.ToLower(original)]; ok {
		delimiter = `\`
		str = mapped
	}
	parts := strings.Split(str, delimiter)
	out := make([]token.Token, 0, len(parts)*2-1)
	for i, p := range parts {
		if i > 0 {
			out = append(out, token.Token{Kind: token.Punct, Value: `\`})
		}
		out = append(out, token.Token{Kind: token.Ident, Value: p})
	}
	return out
}

// nsNextIdent returns the first Ident with index strictly greater than from.
func nsNextIdent(s *tokens.Stream, from int) int {
	for i := from + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Ident {
			return i
		}
	}
	return -1
}

// nsIsImport ports PhpUnitNamespacedFixer::isImport.
func nsIsImport(s *tokens.Stream, index int) bool {
	prev := sigPrev(s, index)
	if prev >= 0 && isPunctVal(s, prev, `\`) {
		prev = sigPrev(s, prev)
	}
	return prev >= 0 && kwIs(s.At(prev), "use")
}

// nsIsClassyInvocation ports the branches of ClassyAnalyzer::isClassyInvocation
// that can apply to a PHPUnit_* legacy class name (never a reserved type, no
// type-position unions for such names). It is only consulted for PHPUnit_*
// tokens, so cases that cannot occur for them are omitted.
func nsIsClassyInvocation(s *tokens.Stream, index int) bool {
	next := sigNext(s, index)
	if next >= 0 {
		if isPunctVal(s, next, `\`) {
			return false
		}
		if isPunctVal(s, next, "::") || isPunctVal(s, next, "...") || s.At(next).Kind == token.Variable {
			return true
		}
	}

	prev := sigPrev(s, index)
	for prev >= 0 {
		t := s.At(prev)
		if t.Kind == token.Ident || isPunctVal(s, prev, `\`) || aliasIsNamespaceOperator(s, prev) {
			prev = sigPrev(s, prev)
			continue
		}
		break
	}
	if prev < 0 {
		return false
	}
	pt := s.At(prev)
	if pt.Kind == token.Keyword {
		switch strings.ToLower(pt.Value) {
		case "extends", "instanceof", "insteadof", "implements", "new":
			return true
		}
	}
	if pt.Kind == token.AttributeStart {
		return true
	}
	// catch (PHPUnit_Exception): next ")", prev "(", before "catch"
	if next >= 0 && isPunctVal(s, next, ")") && isPunctVal(s, prev, "(") {
		if pp := sigPrev(s, prev); pp >= 0 && kwIs(s.At(pp), "catch") {
			return true
		}
	}
	// implements A, PHPUnit_X list
	if isPunctVal(s, prev, ",") {
		p := prev
		for p >= 0 {
			t := s.At(p)
			if isPunctVal(s, p, ",") || t.Kind == token.Ident || isPunctVal(s, p, `\`) || aliasIsNamespaceOperator(s, p) {
				p = sigPrev(s, p)
				continue
			}
			break
		}
		return p >= 0 && kwIs(s.At(p), "implements")
	}
	return false
}
