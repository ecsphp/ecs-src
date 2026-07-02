<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class TrailingPeriod
{
    /**
     * Set name.
     */
    public function setName()
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class TrailingPeriod
{
    /**
     */
    public function setName()
    {
    }
}

?>
