<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\AddMissingVarNameFixer\Fixture;

final class SkipPropertyAssign
{
    private $value;

    public function run()
    {
        /** @var int */
        $this->value = 1000;
    }
}
