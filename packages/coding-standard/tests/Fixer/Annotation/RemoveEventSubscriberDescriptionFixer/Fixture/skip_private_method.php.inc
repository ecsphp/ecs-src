<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class SkipPrivateMethod
{
    /**
     * Delete lead event
     */
    private function onLeadDelete(LeadEvent $event): bool
    {
    }
}
