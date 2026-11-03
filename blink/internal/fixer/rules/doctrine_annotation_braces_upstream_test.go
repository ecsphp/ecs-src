package rules

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var doctrineLeadingNewlines = regexp.MustCompile(`^\n+`)

// doctrineAnnotationTestCases mirrors AbstractDoctrineAnnotationFixerTestCase::
// createTestCases: each comment case as a class, a property and a method
// docblock, plus a no-fix case with the docblock before a non-class statement.
func doctrineAnnotationTestCases(name string, comments [][]string, cfg map[string]any) []upstreamCase {
	var cases, noFix []upstreamCase
	seen := map[string]bool{}
	wrap := func(php, comment string, indent bool) string {
		comment = strings.Trim(comment, " \t\n\r\x00\x0B")
		if indent {
			comment = strings.ReplaceAll(comment, "\n", "\n    ")
		}
		return strings.Replace(php, "%s", doctrineLeadingNewlines.ReplaceAllString(comment, ""), 1)
	}
	templates := []struct {
		php    string
		indent bool
	}{
		{"<?php\n\n%s\nclass FooClass\n{\n}", false},
		{"<?php\n\nclass FooClass\n{\n    %s\n    private $foo;\n}", true},
		{"<?php\n\nclass FooClass\n{\n    %s\n    public function foo()\n    {\n    }\n}", true},
	}
	for i, c := range comments {
		for _, tpl := range templates {
			uc := upstreamCase{cfg: cfg, expected: wrap(tpl.php, c[0], tpl.indent)}
			if len(c) > 1 {
				uc.input = wrap(tpl.php, c[1], tpl.indent)
			}
			uc.name = name + " #" + strconv.Itoa(i)
			cases = append(cases, uc)
		}
		if !seen[c[0]] {
			seen[c[0]] = true
			noFix = append(noFix, upstreamCase{name: name + " (wrong element)", cfg: cfg, expected: wrap("<?php\n\n%s\n$foo = bar();", c[0], false)})
		}
	}
	return append(cases, noFix...)
}

// TestDoctrineAnnotationBracesUpstream runs PHP-CS-Fixer's own
// DoctrineAnnotationBracesFixerTest cases.
func TestDoctrineAnnotationBracesUpstream(t *testing.T) {
	t.Parallel()
	var cases []upstreamCase
	cases = append(cases, doctrineAnnotationTestCases("provideFixCases line 60", [][]string{
		{"\n/**\n * @Foo()\n */"},
		{"\n/**\n * @Foo   ()\n */"},
		{"\n/**\n * @Foo\n * (\n * )\n */"},
		{"\n/**\n * Foo.\n *\n * @author John Doe\n *\n * @Foo()\n */", "\n/**\n * Foo.\n *\n * @author John Doe\n *\n * @Foo\n */"},
		{"/** @Foo() */", "/** @Foo */"},
		{"\n/**\n * @Foo(@Bar())\n */", "\n/**\n * @Foo(@Bar)\n */"},
		{"\n/**\n * @Foo(\n *     @Bar()\n * )\n */", "\n/**\n * @Foo(\n *     @Bar\n * )\n */"},
		{"\n/**\n * @Foo(\n *     @Bar(),\n *     \"baz\"\n * )\n */", "\n/**\n * @Foo(\n *     @Bar,\n *     \"baz\"\n * )\n */"},
		{"\n/**\n * @Foo(\n *     @Bar\\Baz()\n * )\n */", "\n/**\n * @Foo(\n *     @Bar\\Baz\n * )\n */"},
		{"\n/**\n * @Foo() @Bar\\Baz()\n */", "\n/**\n * @Foo @Bar\\Baz\n */"},
		{"\n/**\n * @Foo(\"@Bar\")\n */"},
		{"\n/**\n * Description with a single \" character.\n *\n * @Foo(\"string \"\" with inner quote\")\n *\n * @param mixed description with a single \" character.\n */"},
		{"\n/**\n * @Foo(@Bar\n */"},
		{"\n/**\n * @Foo())@Bar)\n */", "\n/**\n * @Foo)@Bar)\n */"},
		{"\n/**\n * See {@link https://help Help} or {@see BarClass} for details.\n */"},
		{"\n/**\n * @var int\n */"},
		{"\n/**\n * // PHPDocumentor 1\n * @abstract\n * @access\n * @code\n * @deprec\n * @encode\n * @exception\n * @final\n * @ingroup\n * @inheritdoc\n * @inheritDoc\n * @magic\n * @name\n * @toc\n * @tutorial\n * @private\n * @static\n * @staticvar\n * @staticVar\n * @throw\n *\n * // PHPDocumentor 2\n * @api\n * @author\n * @category\n * @copyright\n * @deprecated\n * @example\n * @filesource\n * @global\n * @ignore\n * @internal\n * @license\n * @link\n * @method\n * @package\n * @param\n * @property\n * @property-read\n * @property-write\n * @return\n * @see\n * @since\n * @source\n * @subpackage\n * @throws\n * @todo\n * @TODO\n * @usedBy\n * @uses\n * @var\n * @version\n *\n * // PHPUnit\n * @after\n * @afterClass\n * @backupGlobals\n * @backupStaticAttributes\n * @before\n * @beforeClass\n * @codeCoverageIgnore\n * @codeCoverageIgnoreStart\n * @codeCoverageIgnoreEnd\n * @covers\n * @coversDefaultClass\n * @coversNothing\n * @dataProvider\n * @depends\n * @expectedException\n * @expectedExceptionCode\n * @expectedExceptionMessage\n * @expectedExceptionMessageRegExp\n * @group\n * @large\n * @medium\n * @preserveGlobalState\n * @requires\n * @runTestsInSeparateProcesses\n * @runInSeparateProcess\n * @small\n * @test\n * @testdox\n * @ticket\n * @uses\n *\n * // PHPCheckStyle\n * @SuppressWarnings\n *\n * // PHPStorm\n * @noinspection\n *\n * // PEAR\n * @package_version\n *\n * // PlantUML\n * @enduml\n * @startuml\n *\n * // Psalm\n * @psalm\n * @psalm-param\n *\n * // PHPStan\n * @phpstan\n * @phpstan-param\n *\n * // other\n * @fix\n * @FIXME\n * @fixme\n * @fixme: foo\n * @override\n * @todo: foo\n */"},
	}, map[string]any{"syntax": "with_braces"})...)
	cases = append(cases, upstreamCase{name: "provideFixCases line 295", expected: "<?php\n\n/**\n * @see \\User getId()\n */\n"})
	cases = append(cases, doctrineAnnotationTestCases("provideFixCases line 304", [][]string{
		{"\n/**\n * Foo.\n *\n * @author John Doe\n *\n * @Baz\\Bar\n */", "\n/**\n * Foo.\n *\n * @author John Doe\n *\n * @Baz\\Bar ( )\n */"},
		{"/** @Foo */", "/** @Foo   () */"},
		{"\n/**\n * @Foo(\"bar\")\n */"},
		{"\n/**\n * @Foo\n */", "\n/**\n * @Foo\n * (\n * )\n */"},
		{"\n/**\n * @Foo(@Bar)\n */", "\n/**\n * @Foo(@Bar())\n */"},
		{"\n/**\n * @Foo(\n *     @Bar\n * )\n */", "\n/**\n * @Foo(\n *     @Bar()\n * )\n */"},
		{"\n/**\n * @Foo(\n *     @Bar,\n *     \"baz\"\n * )\n */", "\n/**\n * @Foo(\n *     @Bar(),\n *     \"baz\"\n * )\n */"},
		{"\n/**\n * @Foo(\n *     @Bar\\Baz\n * )\n */", "\n/**\n * @Foo(\n *     @Bar\\Baz()\n * )\n */"},
		{"\n/**\n * @Foo @Bar\\Baz\n */", "\n/**\n * @Foo() @Bar\\Baz()\n */"},
		{"\n/**\n * @\\Foo @\\Bar\\Baz\n */", "\n/**\n * @\\Foo() @\\Bar\\Baz()\n */"},
		{"\n/**\n * @Foo(\"@Bar()\")\n */"},
		{"\n/**\n * Description with a single \" character.\n *\n * @Foo(\"string \"\" with inner quote\")\n *\n * @param mixed description with a single \" character.\n */"},
		{"\n/**\n * @Foo(\n */"},
		{"\n/**\n * @Foo)\n */"},
		{"\n/**\n * @Foo(@Bar()\n */"},
		{"\n/**\n * @Foo\n * @Bar\n * @Baz\n */", "\n/**\n * @Foo()\n * @Bar()\n * @Baz()\n */"},
		{"\n/**\n * @FIXME ()\n * @fixme ()\n * @TODO ()\n * @todo ()\n */"},
		{"\n/**\n * // PHPDocumentor 1\n * @abstract()\n * @access()\n * @code()\n * @deprec()\n * @encode()\n * @exception()\n * @final()\n * @ingroup()\n * @inheritdoc()\n * @inheritDoc()\n * @magic()\n * @name()\n * @toc()\n * @tutorial()\n * @private()\n * @static()\n * @staticvar()\n * @staticVar()\n * @throw()\n *\n * // PHPDocumentor 2\n * @api()\n * @author()\n * @category()\n * @copyright()\n * @deprecated()\n * @example()\n * @filesource()\n * @global()\n * @ignore()\n * @internal()\n * @license()\n * @link()\n * @method()\n * @package()\n * @param()\n * @property()\n * @property-read()\n * @property-write()\n * @return()\n * @see()\n * @since()\n * @source()\n * @subpackage()\n * @throws()\n * @todo()\n * @TODO()\n * @usedBy()\n * @uses()\n * @var()\n * @version()\n *\n * // PHPUnit\n * @after()\n * @afterClass()\n * @backupGlobals()\n * @backupStaticAttributes()\n * @before()\n * @beforeClass()\n * @codeCoverageIgnore()\n * @codeCoverageIgnoreStart()\n * @codeCoverageIgnoreEnd()\n * @covers()\n * @coversDefaultClass()\n * @coversNothing()\n * @dataProvider()\n * @depends()\n * @expectedException()\n * @expectedExceptionCode()\n * @expectedExceptionMessage()\n * @expectedExceptionMessageRegExp()\n * @group()\n * @large()\n * @medium()\n * @preserveGlobalState()\n * @requires()\n * @runTestsInSeparateProcesses()\n * @runInSeparateProcess()\n * @small()\n * @test()\n * @testdox()\n * @ticket()\n * @uses()\n *\n * // PHPCheckStyle\n * @SuppressWarnings()\n *\n * // PHPStorm\n * @noinspection()\n *\n * // PEAR\n * @package_version()\n *\n * // PlantUML\n * @enduml()\n * @startuml()\n *\n * // Psalm\n * @psalm()\n * @psalm-param()\n *\n * // PHPStan\n * @phpstan()\n * @psalm-param()\n *\n *\n * // other\n * @fix()\n * @FIXME()\n * @fixme()\n * @fixme: foo()\n * @override()\n * @todo: foo()\n */"},
	}, map[string]any{"syntax": "without_braces"})...)
	cases = append(cases, upstreamCase{name: "provideFixCases line 557", cfg: map[string]any{"syntax": "without_braces"}, expected: "<?php\n\n/**\n * @see \\User getId()\n */\n"})
	cases = append(cases, upstreamCase{name: "provideFix82Cases line 586", expected: "<?php\n\n/**\n * @author John Doe\n *\n * @Baz\\Bar\n */\nreadonly class FooClass{}", input: "<?php\n\n/**\n * @author John Doe\n *\n * @Baz\\Bar ( )\n */\nreadonly class FooClass{}"})
	runUpstreamCases(t, DoctrineAnnotationBraces{}, cases)
}
