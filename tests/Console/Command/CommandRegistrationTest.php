<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Console\Command;

use Entropy\Console\CommandRegistry;
use Symplify\EasyCodingStandard\Console\Command\CheckCommand;
use Symplify\EasyCodingStandard\Console\Command\WorkerCommand;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

/**
 * Guards the symfony/console → entropy/entropy migration: the commands must be
 * discovered and wired into Entropy's CommandRegistry through the container.
 */
final class CommandRegistrationTest extends AbstractTestCase
{
    private CommandRegistry $commandRegistry;

    protected function setUp(): void
    {
        parent::setUp();

        $this->commandRegistry = $this->make(CommandRegistry::class);
    }

    public function testCommandsAreRegistered(): void
    {
        $this->assertTrue($this->commandRegistry->has('check'));
        $this->assertTrue($this->commandRegistry->has('worker'));
        $this->assertTrue($this->commandRegistry->has('list-checkers'));
    }

    public function testCheckIsTheDefaultCommand(): void
    {
        $this->assertInstanceOf(CheckCommand::class, $this->commandRegistry->getDefault());
    }

    public function testWorkerCommandIsHidden(): void
    {
        $visibleCommandClasses = array_map(
            static fn (object $command): string => $command::class,
            $this->commandRegistry->getVisible()
        );

        $this->assertNotContains(WorkerCommand::class, $visibleCommandClasses);
    }
}
