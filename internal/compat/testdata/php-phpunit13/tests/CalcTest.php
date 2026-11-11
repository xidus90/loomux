<?php

namespace Calc\Tests;

use Calc\Calc;
use PHPUnit\Framework\TestCase;

final class CalcTest extends TestCase
{
    public function testDoublesAValue(): void
    {
        self::assertSame(4, (new Calc())->twice(2));
    }
}
