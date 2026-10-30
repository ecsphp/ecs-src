<?php

namespace App;

class Loader
{
    public function load(string $ns): void
    {
        eval(<<<EOPHP
namespace {$ns};

function loaded(): bool
{
    return true;
}
EOPHP
        );
    }
}
