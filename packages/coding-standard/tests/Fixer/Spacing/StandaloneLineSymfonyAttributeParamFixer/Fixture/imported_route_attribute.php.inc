<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\Routing\Attribute\Route;

final class ImportedRouteAttribute
{
    #[Route(path: '/some', name: 'some_route')]
    public function __invoke()
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\Routing\Attribute\Route;

final class ImportedRouteAttribute
{
    #[Route(
        path: '/some',
        name: 'some_route'
    )]
    public function __invoke()
    {
    }
}

?>
