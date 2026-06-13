<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveDeadVarThisFixer\Fixture;

function anotherFunction()
{
    /**
     * @var SomeType $this
     */
    $this->run();
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveDeadVarThisFixer\Fixture;

function anotherFunction()
{
    $this->run();
}

?>
