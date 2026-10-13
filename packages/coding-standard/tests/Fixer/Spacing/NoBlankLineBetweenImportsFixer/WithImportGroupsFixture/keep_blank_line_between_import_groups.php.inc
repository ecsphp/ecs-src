<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\NoBlankLineBetweenImportsFixer\WithImportGroupsFixture;

use Foo\Bar;

use Foo\Baz;

use function Foo\baz;

class KeepBlankLineBetweenImportGroups
{
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\NoBlankLineBetweenImportsFixer\WithImportGroupsFixture;

use Foo\Bar;
use Foo\Baz;

use function Foo\baz;

class KeepBlankLineBetweenImportGroups
{
}

?>
