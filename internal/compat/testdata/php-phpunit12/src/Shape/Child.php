<?php

namespace Calc\Shape;

class Child extends Base
{
    public function f(): int
    {
        return parent::f() + 1;
    }
}
