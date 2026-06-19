<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Console\Style;

use Entropy\Console\Output\OutputColorizer;
use Entropy\Console\Output\OutputPrinter;
use Entropy\Console\Output\ProgressBar;
use Iterator;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use ReflectionMethod;
use Symplify\EasyCodingStandard\Console\Style\EasyCodingStandardStyle;

/**
 * Guards the symfony/console → entropy/entropy migration: the Symfony console
 * tags still emitted by the reporters must be translated into the smaller tag
 * vocabulary understood by Entropy's OutputColorizer.
 */
final class EasyCodingStandardStyleTest extends TestCase
{
    private ReflectionMethod $normalizeTagsReflectionMethod;

    private EasyCodingStandardStyle $easyCodingStandardStyle;

    protected function setUp(): void
    {
        $this->easyCodingStandardStyle = new EasyCodingStandardStyle(
            new OutputPrinter(new OutputColorizer()),
            new ProgressBar()
        );

        $this->normalizeTagsReflectionMethod = new ReflectionMethod($this->easyCodingStandardStyle, 'normalizeTags');
    }

    #[DataProvider('provideData')]
    public function testNormalizeTags(string $input, string $expected): void
    {
        $normalized = $this->normalizeTagsReflectionMethod->invoke($this->easyCodingStandardStyle, $input);

        $this->assertSame($expected, $normalized);
    }

    public static function provideData(): Iterator
    {
        yield 'comment tag becomes yellow' => ['<comment>hello</comment>', '<fg=yellow>hello</>'];

        yield 'info tag becomes green' => ['<info>done</info>', '<fg=green>done</>'];

        yield 'bold options are stripped, text kept' => ['<options=bold>title</>', 'title'];

        yield 'explicit color closing tag is normalized' => ['<fg=red>err</fg=red>', '<fg=red>err</>'];

        yield 'plain text is left untouched' => ['no tags here', 'no tags here'];
    }
}
