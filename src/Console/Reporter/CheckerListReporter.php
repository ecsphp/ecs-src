<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Console\Reporter;

use Symplify\EasyCodingStandard\Console\Style\EasyCodingStandardStyle;

final readonly class CheckerListReporter
{
    public function __construct(
        private EasyCodingStandardStyle $easyCodingStandardStyle
    ) {
    }

    /**
     * @param string[] $checkerClasses
     */
    public function report(array $checkerClasses, string $type): void
    {
        if ($checkerClasses === []) {
            return;
        }

        $sectionMessage = sprintf(
            '%d checker%s %s:',
            count($checkerClasses),
            count($checkerClasses) === 1 ? '' : 's',
            $type
        );
        $this->easyCodingStandardStyle->section($sectionMessage);
        $this->easyCodingStandardStyle->listing($checkerClasses);
    }
}
