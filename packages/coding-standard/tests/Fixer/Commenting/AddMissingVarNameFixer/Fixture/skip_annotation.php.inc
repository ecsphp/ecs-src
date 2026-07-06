<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\AddMissingVarNameFixer\Fixture;

class SkipAnnotation
{
    /** @Entity */
    private $first;

    /** @\Doctrine\ORM\Mapping\Entity */
    private $second;
}
