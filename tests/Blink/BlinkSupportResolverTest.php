<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer;
use PHPUnit\Framework\TestCase;
use stdClass;
use Symplify\EasyCodingStandard\Blink\BlinkSupportResolver;
use Symplify\EasyCodingStandard\Blink\EcsGoBinaryLocator;

final class BlinkSupportResolverTest extends TestCase
{
    private BlinkSupportResolver $blinkSupportResolver;

    protected function setUp(): void
    {
        putenv('ECS_BLINK_BIN=' . __DIR__ . '/Source/bin/list-checkers-stub');

        $this->blinkSupportResolver = new BlinkSupportResolver(new EcsGoBinaryLocator());
    }

    protected function tearDown(): void
    {
        putenv('ECS_BLINK_BIN');
    }

    public function testEveryRulePortedIsSupported(): void
    {
        $configData = [
            'rules' => [
                [
                    'class' => LowercaseKeywordsFixer::class,
                    'config' => new stdClass(),
                ],
                [
                    'class' => ArraySyntaxFixer::class,
                    'config' => [
                        'syntax' => 'short',
                    ],
                ],
            ],
        ];

        $this->assertTrue($this->blinkSupportResolver->isFullySupported($configData));
    }

    public function testCustomCheckerIsNotSupported(): void
    {
        $configData = [
            'rules' => [
                [
                    'class' => LowercaseKeywordsFixer::class,
                    'config' => new stdClass(),
                ],
                [
                    'class' => 'App\Custom\MyCustomSniff',
                    'config' => new stdClass(),
                ],
            ],
        ];

        $this->assertFalse($this->blinkSupportResolver->isFullySupported($configData));
    }

    public function testEmptyRulesAreNotSupported(): void
    {
        $this->assertFalse($this->blinkSupportResolver->isFullySupported([
            'rules' => [],
        ]));
    }
}
