<?php

/**
 * @return ($arrayHydration is true ? array<int, array<string, mixed>> : array<int, LeadList>)
 */
function getLists(bool $arrayHydration = false): array
{
}

/**
 * @phpstan-return ($isPublic is true ? string : int)
 */
function conditionalWithIs(bool $isPublic): string|int
{
}

/**
 * @return $arrayHydration ? array<int, mixed> : array<int, LeadList>
 */
function bareConditional(bool $arrayHydration = false): array
{
}

?>
-----
<?php

/**
 * @return ($arrayHydration is true ? array<int, array<string, mixed>> : array<int, LeadList>)
 */
function getLists(bool $arrayHydration = false): array
{
}

/**
 * @phpstan-return ($isPublic is true ? string : int)
 */
function conditionalWithIs(bool $isPublic): string|int
{
}

/**
 * @return $arrayHydration ? array<int, mixed> : array<int, LeadList>
 */
function bareConditional(bool $arrayHydration = false): array
{
}

?>
