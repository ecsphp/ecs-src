<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Console\Output;

use Symplify\EasyCodingStandard\Console\Output\CheckstyleOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\ConsoleOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\GitlabOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\JsonOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\JUnitOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\OutputFormatterCollector;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

final class OutputFormatterCollectorTest extends AbstractTestCase
{
    private OutputFormatterCollector $outputFormatterCollector;

    protected function setUp(): void
    {
        parent::setUp();

        $this->outputFormatterCollector = $this->make(OutputFormatterCollector::class);
    }

    public function test(): void
    {
        $this->assertInstanceOf(
            ConsoleOutputFormatter::class,
            $this->outputFormatterCollector->getByName(ConsoleOutputFormatter::getName())
        );
        $this->assertInstanceOf(
            JsonOutputFormatter::class,
            $this->outputFormatterCollector->getByName(JsonOutputFormatter::getName())
        );
        $this->assertInstanceOf(
            JUnitOutputFormatter::class,
            $this->outputFormatterCollector->getByName(JUnitOutputFormatter::getName())
        );
        $this->assertInstanceOf(
            GitlabOutputFormatter::class,
            $this->outputFormatterCollector->getByName(GitlabOutputFormatter::getName())
        );
        $this->assertInstanceOf(
            CheckstyleOutputFormatter::class,
            $this->outputFormatterCollector->getByName(CheckstyleOutputFormatter::getName())
        );
    }
}
