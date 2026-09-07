<?php
declare(strict_types=1);
namespace App;
class Sample{
    public function run(null|int $a,array $items=array()):bool{
        $result = $a
            && count($items) > 0;
        return $result;
    }
}
-----
<?php

declare(strict_types=1);

namespace App;

class Sample
{
    public function run(?int $a, array $items = []): bool
    {
        $result = $a
            && count($items) > 0;
        return $result;
    }
}
