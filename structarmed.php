<?php

declare(strict_types=1);

use Boundwize\StructArmed\Architecture;
use Boundwize\StructArmed\Preset\Preset;
use Boundwize\StructArmed\Preset\Presets\Psr1Preset;

return Architecture::define()
    ->skip([
        __DIR__ . '/tests/Console/Output/Source',
        Psr1Preset::FILES_SHOULD_DECLARE_SYMBOLS_OR_SIDE_EFFECTS => [
            __DIR__ . '/src/Testing/PHPUnit/AbstractCheckerTestCase.php',
        ],
    ])
    ->withPresets(Preset::PSR4(), Preset::PSR1());
