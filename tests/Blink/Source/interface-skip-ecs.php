<?php

declare(strict_types=1);

use PhpCsFixer\AbstractFixer;
use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\CastNotation\LowercaseCastFixer;
use PhpCsFixer\Fixer\ConfigurableFixerInterface;
use PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer;
use PhpCsFixer\Fixer\WhitespacesAwareFixerInterface;
use PhpCsFixer\Fixer\Whitespace\NoTrailingWhitespaceFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withRules([
        ArraySyntaxFixer::class,
        LowercaseCastFixer::class,
        FullyQualifiedStrictTypesFixer::class,
        NoTrailingWhitespaceFixer::class,
    ])
    ->withSkip([
        // an interface on some paths, an interface everywhere, a parent class on some paths
        ConfigurableFixerInterface::class => [__DIR__ . '/legacy'],
        WhitespacesAwareFixerInterface::class,
        AbstractFixer::class => [__DIR__ . '/generated/*'],
        NoTrailingWhitespaceFixer::class => [__DIR__ . '/old'],
    ]);
