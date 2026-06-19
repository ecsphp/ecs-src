<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Console\Style;

use Entropy\Console\Output\OutputColorizer;
use Entropy\Console\Output\OutputPrinter;
use Entropy\Console\Output\ProgressBar;

/**
 * @api
 */
final readonly class EasyCodingStandardStyleFactory
{
    /**
     * @api
     */
    public function create(): EasyCodingStandardStyle
    {
        $outputPrinter = new OutputPrinter(new OutputColorizer());
        $progressBar = new ProgressBar();

        $isDebug = in_array('--debug', $_SERVER['argv'] ?? [], true);

        return new EasyCodingStandardStyle($outputPrinter, $progressBar, $isDebug);
    }
}
