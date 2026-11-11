namespace Lib;

public class Base
{
    public virtual int F() => 1;
}

public class Child : Base
{
    public override int F() => base.F() + 1;
}

public static class Calc
{
    public static int Double(int value) => new Child().F() * value;
}
