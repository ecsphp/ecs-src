<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PhpCsFixer\Fixer\Import\OrderedImportsFixer;
use PHPUnit\Framework\TestCase;
use ReflectionClass;
use Symplify\EasyCodingStandard\Blink\BlinkCommitRunner;

final class BlinkCommitRunnerTest extends TestCase
{
    private BlinkCommitRunner $blinkCommitRunner;

    protected function setUp(): void
    {
        // createCommitMessage() is pure and touches no dependency
        $this->blinkCommitRunner = new ReflectionClass(BlinkCommitRunner::class)->newInstanceWithoutConstructor();
    }

    public function testCreateCommitMessagePrefixesNamespaceGroup(): void
    {
        $message = $this->blinkCommitRunner->createCommitMessage(
            OrderedImportsFixer::class
        );

        $this->assertSame('[Import] OrderedImportsFixer', $message);
    }

    public function testCreateCommitMessageFallsBackToShortName(): void
    {
        $message = $this->blinkCommitRunner->createCommitMessage('OrderedImportsFixer');

        $this->assertSame('OrderedImportsFixer', $message);
    }
}
