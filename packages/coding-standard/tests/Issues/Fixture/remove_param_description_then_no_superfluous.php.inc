<?php

final class SomeClass
{
    /**
     * @param string $name Name
     */
    public function foo(string $name): void
    {
    }
}
-----
<?php

final class SomeClass
{
    public function foo(string $name): void
    {
    }
}
