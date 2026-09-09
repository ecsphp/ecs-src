<?php
declare(strict_types=1);
namespace App;
class Sample{
    function run($items=array())
    {
        $name = "value";
        return $name . "!";
    }
}
-----
<?php

declare(strict_types=1);

namespace App;

class Sample
{
    public function run($items = [])
    {
        $name = 'value';

        return $name.'!';
    }
}
