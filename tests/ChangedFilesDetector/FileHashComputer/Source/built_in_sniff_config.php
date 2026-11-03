<?php

declare(strict_types=1);

use Symplify\EasyCodingStandard\Config\ECSConfig;
use Symplify\EasyCodingStandard\Tests\ChangedFilesDetector\FileHashComputer\Source\BuiltInSniff;

return ECSConfig::configure()
    ->withRules([BuiltInSniff::class]);
