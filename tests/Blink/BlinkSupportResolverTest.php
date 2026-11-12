<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer;
use PHPUnit\Framework\TestCase;
use stdClass;
use Symplify\EasyCodingStandard\Blink\BlinkSupportResolver;
use Symplify\EasyCodingStandard\Blink\EcsGoBinaryLocator;
use Symplify\EasyCodingStandard\DependencyInjection\SimpleParameterProvider;
use Symplify\EasyCodingStandard\FixerRunner\WhitespacesFixerConfigFactory;
use Symplify\EasyCodingStandard\ValueObject\Option;

final class BlinkSupportResolverTest extends TestCase
{
    private BlinkSupportResolver $blinkSupportResolver;

    protected function setUp(): void
    {
        putenv('ECS_BLINK_BIN=' . __DIR__ . '/Source/bin/list-checkers-stub');

        SimpleParameterProvider::setParameter(Option::INDENTATION, Option::INDENTATION_SPACES);
        SimpleParameterProvider::setParameter(Option::LINE_ENDING, "\n");

        $this->blinkSupportResolver = new BlinkSupportResolver(
            new EcsGoBinaryLocator(),
            new WhitespacesFixerConfigFactory()
        );
    }

    protected function tearDown(): void
    {
        putenv('ECS_BLINK_BIN');

        SimpleParameterProvider::setParameter(Option::INDENTATION, Option::INDENTATION_SPACES);
        SimpleParameterProvider::setParameter(Option::LINE_ENDING, PHP_EOL);
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

    public function testTabIndentationIsNotSupported(): void
    {
        SimpleParameterProvider::setParameter(Option::INDENTATION, Option::INDENTATION_TAB);

        $this->assertFalse($this->blinkSupportResolver->isFullySupported($this->createPortedConfigData()));
    }

    public function testWindowsLineEndingIsNotSupported(): void
    {
        SimpleParameterProvider::setParameter(Option::LINE_ENDING, "\r\n");

        $this->assertFalse($this->blinkSupportResolver->isFullySupported($this->createPortedConfigData()));
    }

    /**
     * @return array{rules: array<array{class: string, config: object}>}
     */
    private function createPortedConfigData(): array
    {
        return [
            'rules' => [
                [
                    'class' => LowercaseKeywordsFixer::class,
                    'config' => new stdClass(),
                ],
            ],
        ];
    }
}
