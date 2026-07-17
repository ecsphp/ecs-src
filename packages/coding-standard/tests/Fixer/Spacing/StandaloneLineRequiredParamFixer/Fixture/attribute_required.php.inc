<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class AttributeRequired
{
    #[Required]
    public function autowireDependencies(FormModel $formModel, SubmissionModel $submissionModel): void
    {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineRequiredParamFixer\Fixture;

final class AttributeRequired
{
    #[Required]
    public function autowireDependencies(
        FormModel $formModel,
        SubmissionModel $submissionModel
    ): void
    {
    }
}

?>
