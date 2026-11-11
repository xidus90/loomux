// Java 17: a sealed interface and the types it permits.
sealed interface Shape permits Circle, Square {
    int area();
}

record Circle(int r) implements Shape {
    public int area() {
        return 3 * r * r;
    }
}

final class Square implements Shape {
    private final int side;

    Square(int side) {
        this.side = side;
    }

    public int area() {
        return side * side;
    }

    int twice() {
        return area() * 2;
    }
}
