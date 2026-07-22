<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

final class FqnRouteAttribute
{
    #[\Symfony\Component\Routing\Attribute\Route(path: '/some', name: 'some_route')]
    public function __invoke()
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

final class FqnRouteAttribute
{
    #[\Symfony\Component\Routing\Attribute\Route(
        path: '/some',
        name: 'some_route'
    )]
    public function __invoke()
    {
    }
}

?>
