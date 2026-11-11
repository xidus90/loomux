<?php

namespace Syntax;

class Base
{
    public function size(): int
    {
        return 1;
    }
}

class Box extends Base
{
    public const int LIMIT = 10;

    #[\Override]
    public function size(): int
    {
        $name = 'LIMIT';
        return static::{$name} + parent::size();
    }
}
