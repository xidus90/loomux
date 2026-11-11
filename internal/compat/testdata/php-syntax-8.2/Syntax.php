<?php

namespace Syntax;

interface Shape
{
    public function area(): int;
}

interface Named
{
}

trait Sized
{
    public const UNIT = 1;

    public function unit(): int
    {
        return self::UNIT;
    }
}

readonly class Square implements Shape
{
    use Sized;

    public function __construct(public int $side)
    {
    }

    public function area(): int
    {
        return $this->side * $this->side * $this->unit();
    }
}

function describe((Shape&Named)|null $shape): string
{
    return $shape === null ? 'none' : 'shape';
}

function main(): int
{
    describe(null);
    return (new Square(2))->area();
}
