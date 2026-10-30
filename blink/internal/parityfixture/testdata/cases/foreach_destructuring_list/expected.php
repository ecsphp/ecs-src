<?php

class C
{
    public function run()
    {
        foreach ($this->handlers as ['threshold' => $threshold, 'handler' => $handler]) {
            $handler($threshold);
        }
    }
}
