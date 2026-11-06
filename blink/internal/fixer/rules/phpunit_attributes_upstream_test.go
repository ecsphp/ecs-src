package rules

import "testing"

// Generated from PHP-CS-Fixer 3.95 PhpUnitAttributesFixer default output. See
// phpunit_attributes.go.
func TestPhpUnitAttributesUpstream(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitAttributes{}, []aliasUpstreamCase{
		{
			name: "covers_class_on_class_docblock",
			input: `<?php
/**
 * @covers \VendorName\Foo
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\CoversClass(\VendorName\Foo::class)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "covers_trait_guess",
			input: `<?php
/**
 * @covers \VendorName\FooTrait
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\CoversTrait(\VendorName\FooTrait::class)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "covers_function",
			input: `<?php
/**
 * @covers ::foo
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\CoversFunction('foo')]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "covers_method",
			input: `<?php
/**
 * @covers \VendorName\Foo::bar
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\CoversMethod(\VendorName\Foo::class, 'bar')]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "covers_nothing",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @coversNothing
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\CoversNothing]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "data_provider_method",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @dataProvider provideData
     */
    public function testFoo($a)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\DataProvider('provideData')]
    public function testFoo($a)
    {
    }
}
`,
		},
		{
			name: "data_provider_external",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @dataProvider \Vendor\Provider::provide
     */
    public function testFoo($a)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\DataProviderExternal(\Vendor\Provider::class, 'provide')]
    public function testFoo($a)
    {
    }
}
`,
		},
		{
			name: "group_and_test",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @group slow
     * @test
     */
    public function foo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\Group('slow')]
    #[\PHPUnit\Framework\Attributes\Test]
    public function foo()
    {
    }
}
`,
		},
		{
			name: "depends_simple",
			input: `<?php
class FooTest extends TestCase
{
    public function testFirst()
    {
    }

    /**
     * @depends testFirst
     */
    public function testSecond()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    public function testFirst()
    {
    }

    /**
     */
    #[\PHPUnit\Framework\Attributes\Depends('testFirst')]
    public function testSecond()
    {
    }
}
`,
		},
		{
			name: "depends_external",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @depends \Vendor\OtherTest::testThing
     */
    public function testSecond()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\DependsExternal(\Vendor\OtherTest::class, 'testThing')]
    public function testSecond()
    {
    }
}
`,
		},
		{
			name: "depends_clone",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @depends clone testFirst
     */
    public function testSecond()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\DependsUsingDeepClone('testFirst')]
    public function testSecond()
    {
    }
}
`,
		},
		{
			name: "testwith",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @testWith [1, 2]
     *           [3, 4]
     */
    public function testFoo($a, $b)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\TestWithJson('[1, 2]')]
    #[\PHPUnit\Framework\Attributes\TestWithJson('[3, 4]')]
    public function testFoo($a, $b)
    {
    }
}
`,
		},
		{
			name: "backup_globals",
			input: `<?php
/**
 * @backupGlobals enabled
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\BackupGlobals(true)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "backup_static_attributes",
			input: `<?php
/**
 * @backupStaticAttributes disabled
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\BackupStaticProperties(false)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "ticket_and_testdox",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @ticket 42
     * @testdox It works nicely
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     * @testdox It works nicely
     */
    #[\PHPUnit\Framework\Attributes\Ticket('42')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "uses_class",
			input: `<?php
/**
 * @uses \Vendor\Helper
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\UsesClass(\Vendor\Helper::class)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "uses_function",
			input: `<?php
/**
 * @uses ::helperFunc
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\UsesFunction('helperFunc')]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "requires_php",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @requires PHP 8.0
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\RequiresPhp('>= 8.0')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "requires_extension",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @requires extension redis
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\RequiresPhpExtension('redis')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "mixed_with_kept_description",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * Some description.
     *
     * @param int $a
     * @dataProvider provideData
     * @group slow
     */
    public function testFoo($a)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     * Some description.
     *
     * @param int $a
     */
    #[\PHPUnit\Framework\Attributes\DataProvider('provideData')]
    #[\PHPUnit\Framework\Attributes\Group('slow')]
    public function testFoo($a)
    {
    }
}
`,
		},
		{
			name: "keep_annotations",
			cfg:  map[string]any{"keep_annotations": true},
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @dataProvider provideData
     */
    public function testFoo($a)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     * @dataProvider provideData
     */
    #[\PHPUnit\Framework\Attributes\DataProvider('provideData')]
    public function testFoo($a)
    {
    }
}
`,
		},
		{
			name: "noop_non_phpunit_class",
			input: `<?php
class Foo
{
    /**
     * @dataProvider provideData
     */
    public function bar($a)
    {
    }
}
`,
			expected: `<?php
class Foo
{
    /**
     * @dataProvider provideData
     */
    public function bar($a)
    {
    }
}
`,
		},
		{
			name: "noop_already_attributed",
			input: `<?php
class FooTest extends TestCase
{
    #[\PHPUnit\Framework\Attributes\DataProvider('provideData')]
    public function testFoo($a)
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    #[\PHPUnit\Framework\Attributes\DataProvider('provideData')]
    public function testFoo($a)
    {
    }
}
`,
		},
		{
			name: "noop_unknown_annotation",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @see https://example.com
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     * @see https://example.com
     */
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "method_scope_covers_ignored",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @covers \VendorName\Foo
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     * @covers \VendorName\Foo
     */
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "single_line_docblock",
			input: `<?php
class FooTest extends TestCase
{
    /** @group fast */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    
    #[\PHPUnit\Framework\Attributes\Group('fast')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "final_class_docblock",
			input: `<?php
/**
 * @coversNothing
 */
final class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\CoversNothing]
final class FooTest extends TestCase
{
}
`,
		},
		{
			name: "preserve_global_state_disabled",
			input: `<?php
/**
 * @preserveGlobalState disabled
 */
class FooTest extends TestCase
{
}
`,
			expected: `<?php
/**
 */
#[\PHPUnit\Framework\Attributes\PreserveGlobalState(false)]
class FooTest extends TestCase
{
}
`,
		},
		{
			name: "requires_php_range",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @requires PHP >= 7.0 < 8.0
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\RequiresPhp('>= 7.0 < 8.0')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "requires_setting",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @requires setting date.timezone Europe/Prague
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\RequiresSetting('date.timezone', 'Europe/Prague')]
    public function testFoo()
    {
    }
}
`,
		},
		{
			name: "depends_on_class",
			input: `<?php
class FooTest extends TestCase
{
    /**
     * @depends \Vendor\OtherTest::class
     */
    public function testFoo()
    {
    }
}
`,
			expected: `<?php
class FooTest extends TestCase
{
    /**
     */
    #[\PHPUnit\Framework\Attributes\DependsOnClass(\Vendor\OtherTest::class)]
    public function testFoo()
    {
    }
}
`,
		},
	})
}
