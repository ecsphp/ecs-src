<?php

/**
 * @param $x array<int, string>|null
 * @phpstan-param $y int
 * @psalm-var $c list<int>
 * @var $z Foo\Bar
 */
function test($x, $y): void
{
}

?>
-----
<?php

/**
 * @param array<int, string>|null $x
 * @phpstan-param int $y
 * @psalm-var list<int> $c
 * @var Foo\Bar $z
 */
function test($x, $y): void
{
}

?>
