<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\StringNotation\HeredocToNowdocFixer;
use PhpCsFixer\Fixer\StringNotation\StringImplicitBackslashesFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withRules([StringImplicitBackslashesFixer::class, HeredocToNowdocFixer::class]);
