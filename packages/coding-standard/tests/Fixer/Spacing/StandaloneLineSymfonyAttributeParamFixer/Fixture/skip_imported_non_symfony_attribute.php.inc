<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Doctrine\ORM\Mapping\Table;

#[Table(name: 'some_table', schema: 'public')]
final class SkipImportedNonSymfonyAttribute
{
}
