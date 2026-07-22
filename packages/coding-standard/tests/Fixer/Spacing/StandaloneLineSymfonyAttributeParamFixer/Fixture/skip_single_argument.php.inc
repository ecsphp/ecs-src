<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\Routing\Attribute\Route;

final class SkipSingleArgument
{
    #[Route(path: '/some')]
    public function __invoke()
    {
    }
}
