<?php

namespace Calc;

use Calc\Shape\Child;

class Calc
{
    public function twice(int $value): int
    {
        $child = new Child();
        return $child->f() * $value;
    }
}
