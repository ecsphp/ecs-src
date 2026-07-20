<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePromotedPropertyFixer\Fixture;

final class ReadonlyPromotedProperties
{
    public function __construct(private readonly EntityManager $entityManager, private readonly SchedulerPlanner $schedulerPlanner, private readonly SchedulerRepository $schedulerRepository)
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePromotedPropertyFixer\Fixture;

final class ReadonlyPromotedProperties
{
    public function __construct(
        private readonly EntityManager $entityManager,
        private readonly SchedulerPlanner $schedulerPlanner,
        private readonly SchedulerRepository $schedulerRepository
    )
    {
    }
}

?>
