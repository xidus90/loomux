<?php

namespace Syntax;

class Temperature
{
    public private(set) int $reads = 0;

    public int $celsius = 0 {
        set(int $value) {
            $this->reads++;
            $this->celsius = $value;
        }
    }

    public int $fahrenheit {
        get => Scale::convert($this->celsius);
    }
}

class Scale
{
    public static function convert(int $celsius): int
    {
        return intdiv($celsius * 9, 5) + 32;
    }

    public function read(): int
    {
        return new Temperature()->fahrenheit;
    }
}
