<?php

declare(strict_types=1);

use PhpCsFixer\Fixer\Whitespace\BlankLineBetweenImportGroupsFixer;
use Symplify\CodingStandard\Fixer\Spacing\NoBlankLineBetweenImportsFixer;
use Symplify\EasyCodingStandard\Config\ECSConfig;

return static function (ECSConfig $ecsConfig): void {
    $ecsConfig->rule(BlankLineBetweenImportGroupsFixer::class);
    $ecsConfig->rule(NoBlankLineBetweenImportsFixer::class);
};
