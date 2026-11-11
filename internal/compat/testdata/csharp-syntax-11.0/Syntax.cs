namespace Syntax;

public record Cat(string Name);

public record Dog(string Name);

// A union type (C# 15).
public union Pet(Cat, Dog);

// A closed hierarchy (C# 15).
public closed record class Shape;

public class Loops
{
    // A labelled break (C# 15).
    public int First(int[][] rows)
    {
        outer:
        foreach (var row in rows)
        {
            foreach (var cell in row)
            {
                if (cell > 0)
                {
                    break outer;
                }
            }
        }
        return 0;
    }

    public List<int> Sized(int[] xs) => Make(xs);

    // Arguments to a collection expression (C# 15).
    private static List<int> Make(int[] xs) => [with(capacity: 8), .. xs];
}

public static class Strings
{
    // An extension indexer (C# 15).
    extension(string s)
    {
        public char this[Index i] => At(s, i);
    }

    private static char At(string s, Index i) => s[i];
}
