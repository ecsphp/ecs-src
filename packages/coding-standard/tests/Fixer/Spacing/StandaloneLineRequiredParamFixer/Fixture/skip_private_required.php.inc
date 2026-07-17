<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class SkipPrivateRequired
{
    #[Required]
    private function autowireDependencies(FormModel $formModel, SubmissionModel $submissionModel): void
    {
    }
}

?>
