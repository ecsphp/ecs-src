<?php

function intersectionTypeWithDocBlockBelow()
{
    /** @var MockObject&AbstractIntegration */
    /** @phpstan-ignore classConstant.deprecatedClass */
    $integration = create();
}

?>
-----
<?php

function intersectionTypeWithDocBlockBelow()
{
    /** @var MockObject&AbstractIntegration $integration */
    /** @phpstan-ignore classConstant.deprecatedClass */
    $integration = create();
}

?>
