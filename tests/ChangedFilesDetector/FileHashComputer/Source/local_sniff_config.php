<?php

declare(strict_types=1);

use App\Ecs\LocalSniff;
use Symplify\EasyCodingStandard\Config\ECSConfig;

// a project's own sniff, outside the "Symplify\" namespace
require_once __DIR__ . '/LocalSniff.php';

return ECSConfig::configure()
    ->withRules([LocalSniff::class]);
