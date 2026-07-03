<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\MergeDocBlockStartFixer\Fixture;

class LeadingEmptyLine
{
    /*
     *
     * @return $this
     */
    public function setEvent($event)
    {
        return $this;
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\MergeDocBlockStartFixer\Fixture;

class LeadingEmptyLine
{
    /**
     * @return $this
     */
    public function setEvent($event)
    {
        return $this;
    }
}

?>
