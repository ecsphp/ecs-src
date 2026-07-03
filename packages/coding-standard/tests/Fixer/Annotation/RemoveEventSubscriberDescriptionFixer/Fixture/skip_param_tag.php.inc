<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class SkipParamTag
{
    /**
     * Delete lead event
     *
     * @param LeadEvent $event
     */
    public function onLeadDelete(LeadEvent $event): bool
    {
    }
}
