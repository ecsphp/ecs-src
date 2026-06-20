<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\Operator\LongToShorthandOperatorFixer;
use PhpCsFixer\Fixer\Operator\NoUselessConcatOperatorFixer;
use PhpCsFixer\Fixer\Operator\NoUselessNullsafeOperatorFixer;
use PhpCsFixer\Fixer\Operator\StandardizeNotEqualsFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withRules([
        NoUselessConcatOperatorFixer::class,
        NoUselessNullsafeOperatorFixer::class,
        StandardizeNotEqualsFixer::class,
        LongToShorthandOperatorFixer::class,
    ]);
