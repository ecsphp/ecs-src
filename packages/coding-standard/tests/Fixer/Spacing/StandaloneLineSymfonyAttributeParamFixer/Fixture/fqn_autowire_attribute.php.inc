<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

final class FqnAutowireAttribute
{
    public function __construct(
        #[\Symfony\Component\DependencyInjection\Attribute\Autowire(service: 'some.service', lazy: true)]
        private readonly object $service
    ) {
    }
}

?>
-----
<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

final class FqnAutowireAttribute
{
    public function __construct(
        #[\Symfony\Component\DependencyInjection\Attribute\Autowire(
            service: 'some.service',
            lazy: true
        )]
        private readonly object $service
    ) {
    }
}

?>
