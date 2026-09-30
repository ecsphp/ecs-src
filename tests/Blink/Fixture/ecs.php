<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

// Config for the blink end-to-end CI check. The path is supplied on the CLI so
// the check fixes a throwaway copy, not the committed fixture.
return ECSConfig::configure()
    ->withConfiguredRule(ArraySyntaxFixer::class, [
        'syntax' => 'short',
    ]);
