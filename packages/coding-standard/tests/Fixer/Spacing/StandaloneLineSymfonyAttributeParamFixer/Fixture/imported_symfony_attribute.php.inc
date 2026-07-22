<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\Console\Attribute\AsCommand;

#[AsCommand(name: 'app:some', description: 'Some description')]
final class ImportedSymfonyAttribute
{
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\Console\Attribute\AsCommand;

#[AsCommand(
    name: 'app:some',
    description: 'Some description'
)]
final class ImportedSymfonyAttribute
{
}

?>
