<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer\Fixture;

final class SkipPromotedProperty
{
    public function __construct(private readonly Translator $translator, RouterInterface $router)
    {
    }
}
