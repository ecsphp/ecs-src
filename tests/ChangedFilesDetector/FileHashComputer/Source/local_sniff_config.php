<?php

declare(strict_types=1);

use Symplify\EasyCodingStandard\Config\ECSConfig;
use Symplify\EasyCodingStandard\Tests\ChangedFilesDetector\FileHashComputer\Source\LocalSniff;

return ECSConfig::configure()
    ->withRules([LocalSniff::class]);
