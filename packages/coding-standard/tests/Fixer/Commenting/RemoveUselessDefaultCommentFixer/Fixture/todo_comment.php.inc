<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveUselessDefaultCommentFixer\Fixture;

class PackagesData
{
    public function getIterator()
    {
        // TODO: Implement getIterator() method.
        $value = 100;
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveUselessDefaultCommentFixer\Fixture;

class PackagesData
{
    public function getIterator()
    {
        $value = 100;
    }
}

?>
