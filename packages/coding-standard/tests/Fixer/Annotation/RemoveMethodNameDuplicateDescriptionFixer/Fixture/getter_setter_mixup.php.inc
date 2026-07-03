<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class GetterSetterMixup
{
    /**
     * Get results.
     */
    public function setResults()
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class GetterSetterMixup
{
    /**
     */
    public function setResults()
    {
    }
}

?>
