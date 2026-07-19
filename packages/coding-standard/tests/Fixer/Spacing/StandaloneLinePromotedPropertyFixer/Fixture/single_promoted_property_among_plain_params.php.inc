<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePromotedPropertyFixer\Fixture;

final class SinglePromotedPropertyAmongPlainParams
{
    public function __construct(
        CorePermissions $security, Translator $translator, RouterInterface $router,
        private LeadModel $leadModel,
    ) {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePromotedPropertyFixer\Fixture;

final class SinglePromotedPropertyAmongPlainParams
{
    public function __construct(
        CorePermissions $security,
        Translator $translator,
        RouterInterface $router,
        private LeadModel $leadModel,
    ) {
    }
}

?>
