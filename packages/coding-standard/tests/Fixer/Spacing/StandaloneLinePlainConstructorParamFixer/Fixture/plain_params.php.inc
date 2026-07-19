<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer\Fixture;

final class PlainParams
{
    public function __construct(CorePermissions $security, Translator $translator, RouterInterface $router)
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer\Fixture;

final class PlainParams
{
    public function __construct(
        CorePermissions $security,
        Translator $translator,
        RouterInterface $router
    )
    {
    }
}

?>
