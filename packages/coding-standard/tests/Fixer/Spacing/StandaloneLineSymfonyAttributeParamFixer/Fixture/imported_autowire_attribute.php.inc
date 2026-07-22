<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\DependencyInjection\Attribute\Autowire;

final class ImportedAutowireAttribute
{
    public function __construct(
        #[Autowire(service: 'some.service', lazy: true)]
        private readonly object $service
    ) {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\DependencyInjection\Attribute\Autowire;

final class ImportedAutowireAttribute
{
    public function __construct(
        #[Autowire(
            service: 'some.service',
            lazy: true
        )]
        private readonly object $service
    ) {
    }
}

?>
