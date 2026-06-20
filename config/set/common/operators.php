<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\Operator\AssignNullCoalescingToCoalesceEqualFixer;
use PhpCsFixer\Fixer\Operator\LongToShorthandOperatorFixer;
use PhpCsFixer\Fixer\Operator\NoUselessConcatOperatorFixer;
use PhpCsFixer\Fixer\Operator\NoUselessNullsafeOperatorFixer;
use PhpCsFixer\Fixer\Operator\ObjectOperatorWithoutWhitespaceFixer;
use PhpCsFixer\Fixer\Operator\StandardizeNotEqualsFixer;
use PhpCsFixer\Fixer\Operator\TernaryToNullCoalescingFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withRules([
        NoUselessConcatOperatorFixer::class,
        NoUselessNullsafeOperatorFixer::class,
        StandardizeNotEqualsFixer::class,
        LongToShorthandOperatorFixer::class,
        ObjectOperatorWithoutWhitespaceFixer::class,
        TernaryToNullCoalescingFixer::class,
        AssignNullCoalescingToCoalesceEqualFixer::class,
    ]);
