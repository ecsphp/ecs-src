<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

#[\Symfony\Component\Console\Attribute\AsCommand(name: 'app:some')]
final class AsCommandSingleArgument
{
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

#[\Symfony\Component\Console\Attribute\AsCommand(
    name: 'app:some'
)]
final class AsCommandSingleArgument
{
}

?>
