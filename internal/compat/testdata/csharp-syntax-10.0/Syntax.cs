namespace Syntax;

public class Counter
{
    // The field keyword in an accessor (C# 14).
    public int Value
    {
        get => field;
        set => field = Clamp(value);
    }

    private static int Clamp(int value) => Math.Max(0, value);
}

public class Link
{
    public Link? Next { get; set; }

    public int Label { get; set; }

    // A null-conditional assignment (C# 14).
    public void Relabel(int label)
    {
        Next?.Label = label;
        Touch();
    }

    private void Touch() { }
}

public static class Enumerables
{
    // An extension block (C# 14).
    extension<T>(IEnumerable<T> source)
    {
        public bool IsEmpty => !source.Any();

        public int Size() => Count(source);
    }

    private static int Count<T>(IEnumerable<T> source) => source.Count();
}

public static class Parsers
{
    private delegate bool TryParse(string text, out int result);

    // Lambda parameter modifiers without types (C# 14).
    public static bool Parse(string text) => Try(text, (s, out n) => int.TryParse(s, out n));

    private static bool Try(string text, TryParse parse) => parse(text, out _);
}
