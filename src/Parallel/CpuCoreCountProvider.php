<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Parallel;

use Fidry\CpuCoreCounter\CpuCoreCounter;
use Fidry\CpuCoreCounter\NumberOfCpuCoreNotFound;

final class CpuCoreCountProvider
{
    private const int DEFAULT_CORE_COUNT = 2;

    public function provide(): int
    {
        try {
            $coreCount = new CpuCoreCounter()
                ->getCount();
        } catch (NumberOfCpuCoreNotFound) {
            return self::DEFAULT_CORE_COUNT;
        }

        // leave one core free, to avoid maxing out the CPU
        return max(1, $coreCount - 1);
    }
}
