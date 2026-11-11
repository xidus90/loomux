using Xunit;

namespace Lib.Tests;

public class CalcTests
{
    [Fact]
    public void DoublesAValue() => Assert.Equal(4, Calc.Double(2));
}
