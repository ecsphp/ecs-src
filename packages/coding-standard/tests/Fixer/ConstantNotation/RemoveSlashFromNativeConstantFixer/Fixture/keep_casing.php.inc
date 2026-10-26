<?php

namespace Symplify\CodingStandard\Tests\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer\Fixture;

$value = \TRUE;
$other = \Null;

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer\Fixture;

$value = TRUE;
$other = Null;

?>
