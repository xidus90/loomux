<?php

namespace Syntax;

class Shape
{
    public const FORMAT = static function (int $size): string {
        return 'size ' . $size;
    };

    public function __construct(public int $size = 1)
    {
    }

    public function grow(): static
    {
        return clone($this, ['size' => $this->size + 1]);
    }

    public function twice(): int
    {
        return $this->size |> double(...);
    }
}

function double(int $value): int
{
    return $value * 2;
}

function main(): int
{
    (void) double(1);
    return (new Shape())->twice();
}
