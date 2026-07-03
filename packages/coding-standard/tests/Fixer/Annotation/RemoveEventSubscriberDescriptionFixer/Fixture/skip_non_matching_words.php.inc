<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class SkipNonMatchingWords
{
    /**
     * Send welcome event
     */
    public function onLeadDelete(LeadEvent $event): bool
    {
    }
}
