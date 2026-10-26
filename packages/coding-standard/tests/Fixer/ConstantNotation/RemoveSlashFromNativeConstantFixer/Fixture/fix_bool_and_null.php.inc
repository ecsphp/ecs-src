<?php

namespace Symplify\CodingStandard\Tests\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer\Fixture;

function fixBoolAndNull()
{
    if (\true) {
        return \false;
    }

    return \null;
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer\Fixture;

function fixBoolAndNull()
{
    if (true) {
        return false;
    }

    return null;
}

?>
