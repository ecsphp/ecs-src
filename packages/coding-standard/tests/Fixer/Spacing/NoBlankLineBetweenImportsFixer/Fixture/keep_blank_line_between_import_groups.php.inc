<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\NoBlankLineBetweenImportsFixer\Fixture;

use Foo\Bar;

use Foo\Baz;

use function Foo\first;

use function Foo\second;

use const Foo\FIRST;

use Foo\Qux;

class KeepBlankLineBetweenImportGroups
{
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\NoBlankLineBetweenImportsFixer\Fixture;

use Foo\Bar;
use Foo\Baz;

use function Foo\first;
use function Foo\second;

use const Foo\FIRST;

use Foo\Qux;

class KeepBlankLineBetweenImportGroups
{
}

?>
