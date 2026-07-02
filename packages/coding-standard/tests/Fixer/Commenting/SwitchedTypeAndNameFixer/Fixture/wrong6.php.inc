<?php

/**
 * @param $a string the input value
 * @param $b array<int, string> mapped items
 */
function test($a, $b): void
{
}

?>
-----
<?php

/**
 * @param string $a the input value
 * @param array<int, string> $b mapped items
 */
function test($a, $b): void
{
}

?>
