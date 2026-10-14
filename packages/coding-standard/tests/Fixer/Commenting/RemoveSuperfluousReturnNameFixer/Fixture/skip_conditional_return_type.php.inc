<?php

/**
 * @return $arrayHydration ? array<int, array<string, mixed>> : array<int, LeadList>
 */
function getLists(bool $arrayHydration = false): array
{
}

/**
 * @phpstan-return $isPublic is true ? string : int
 */
function conditionalWithIs(bool $isPublic): string|int
{
}

?>
-----
<?php

/**
 * @return $arrayHydration ? array<int, array<string, mixed>> : array<int, LeadList>
 */
function getLists(bool $arrayHydration = false): array
{
}

/**
 * @phpstan-return $isPublic is true ? string : int
 */
function conditionalWithIs(bool $isPublic): string|int
{
}

?>
