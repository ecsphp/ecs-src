<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class SkipNoRequired
{
    public function autowireDependencies(FormModel $formModel, SubmissionModel $submissionModel): void
    {
    }
}

?>
