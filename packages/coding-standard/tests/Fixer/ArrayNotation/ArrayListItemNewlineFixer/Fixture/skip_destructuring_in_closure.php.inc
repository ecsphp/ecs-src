<?php

namespace Symplify\CodingStandard\Tests\Fixer\ArrayNotation\ArrayListItemNewlineFixer\Fixture;

final class SkipDestructuringInClosure
{
    public function run()
    {
        $resolver->setDefaults([
            'special' => function ($options) {
                [$first, $second] = $options->getPair();

                return $first;
            },
        ]);
    }
}
