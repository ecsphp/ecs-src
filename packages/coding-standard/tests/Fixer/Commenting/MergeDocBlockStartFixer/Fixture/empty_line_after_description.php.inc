<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\MergeDocBlockStartFixer\Fixture;

class EmptyLineAfterDescription
{
    /*
     * Cleans query params saving url values.
     *
     * @param string $url
     */
    public function clean($url)
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\MergeDocBlockStartFixer\Fixture;

class EmptyLineAfterDescription
{
    /**
     * Cleans query params saving url values.
     * @param string $url
     */
    public function clean($url)
    {
    }
}

?>
