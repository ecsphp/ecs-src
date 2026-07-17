<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class DocblockRequired
{
    /**
     * @required
     */
    public function autowireDependencies(FormModel $formModel, SubmissionModel $submissionModel): void
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class DocblockRequired
{
    /**
     * @required
     */
    public function autowireDependencies(
        FormModel $formModel,
        SubmissionModel $submissionModel
    ): void
    {
    }
}

?>
