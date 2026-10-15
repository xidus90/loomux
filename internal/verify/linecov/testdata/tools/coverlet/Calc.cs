public static class Calc
{
    public static int One() => 1;
    [ExcludeFromCodeCoverage(Justification = "debug helper")]
    public static int Two() => 2;
    [ExcludeFromCodeCoverage]
    public static int Three() => 3;
}
