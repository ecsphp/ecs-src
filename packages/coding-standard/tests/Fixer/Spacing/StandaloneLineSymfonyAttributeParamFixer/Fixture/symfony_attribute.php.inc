<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

#[\Symfony\Component\Console\Attribute\AsCommand(name: 'mautic:entity:import', description: 'Import entity data from a ZIP file.')]
final class SymfonyAttribute
{
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

#[\Symfony\Component\Console\Attribute\AsCommand(
    name: 'mautic:entity:import',
    description: 'Import entity data from a ZIP file.'
)]
final class SymfonyAttribute
{
}

?>
