<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer\Fixture;

final class SkipThreeParams
{
    public function __construct(CorePermissions $security, Translator $translator, RouterInterface $router)
    {
    }
}
