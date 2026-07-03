<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class ParamTagKept
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

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class ParamTagKept
{
    /**
     *
     * @param LeadEvent $event
     */
    public function onLeadDelete(LeadEvent $event): bool
    {
    }
}

?>
