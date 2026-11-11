// Java 21: record patterns and a pattern switch with a guard.
sealed interface Shape permits Circle, Square {
}

record Circle(int r) implements Shape {
}

record Square(int side) implements Shape {
}

class Areas {
    static int area(Shape s) {
        return switch (s) {
            case Circle(int r) when r > 10 -> big(r);
            case Circle(int r) -> 3 * r * r;
            case Square(int side) -> side * side;
        };
    }

    static int big(int r) {
        return 3 * r * r;
    }

    static boolean unit(Object o) {
        return o instanceof Square(int side) && side == 1;
    }
}
