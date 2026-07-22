<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

#[\Symfony\Component\Console\Attribute\AsCommand]
final class SkipNoArguments
{
    #[\Override]
    public function run(): void
    {
    }
}
