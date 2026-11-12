<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class SSuffixMethodName
{
    /**
     * has
     */
    public function has(): void
    {
    }

    /**
     * Is alive
     */
    public function isAlive(): void
    {
    }

    /**
     * fetchStats
     */
    public function fetchStats(): void
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class SSuffixMethodName
{
    /**
     */
    public function has(): void
    {
    }

    /**
     */
    public function isAlive(): void
    {
    }

    /**
     */
    public function fetchStats(): void
    {
    }
}

?>
