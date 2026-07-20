<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class ThirdPersonVerb
{
    /**
     * Contacts a user.
     *
     * @return object
     */
    public function contactUser()
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer\Fixture;

final class ThirdPersonVerb
{
    /**
     *
     * @return object
     */
    public function contactUser()
    {
    }
}

?>
