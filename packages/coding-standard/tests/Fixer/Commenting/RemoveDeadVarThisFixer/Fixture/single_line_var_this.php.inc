<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveDeadVarThisFixer\Fixture;

function someFunction()
{
    /** @var SomeType $this */
    $this->run();
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveDeadVarThisFixer\Fixture;

function someFunction()
{
    $this->run();
}

?>
