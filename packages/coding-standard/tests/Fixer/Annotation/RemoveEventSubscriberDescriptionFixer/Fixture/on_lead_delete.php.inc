<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class OnLeadDelete
{
    /**
     * Delete lead event
     */
    public function onLeadDelete(LeadEvent $event): bool
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\Fixture;

final class OnLeadDelete
{
    
    public function onLeadDelete(LeadEvent $event): bool
    {
    }
}

?>
