<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer;
use PhpCsFixer\Fixer\CastNotation\LowercaseCastFixer;
use PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withConfiguredRule(ArraySyntaxFixer::class, [
        'syntax' => 'long',
    ])
    ->withRules([LowercaseCastFixer::class, FullyQualifiedStrictTypesFixer::class])
    ->withSkip([
        __DIR__ . '/legacy/*',
        LowercaseKeywordsFixer::class,
    ]);
