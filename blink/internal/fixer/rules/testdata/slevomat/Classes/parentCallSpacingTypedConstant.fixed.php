<?php

class Foo extends Bar
{
    public const string SELF = 'self';

    public const string PARENT = 'parent';

    public function run()
    {
        $a = 1;

        parent::run();

        return $a;
    }
}
