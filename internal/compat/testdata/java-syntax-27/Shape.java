// Java 25: a statement before super(...) in a constructor.
class Shape {
    final int size;

    Shape(int size) {
        this.size = size;
    }
}

class Square extends Shape {
    Square(int size) {
        if (size < 0) {
            throw new IllegalArgumentException("size");
        }
        super(size);
    }

    int area() {
        return size * size;
    }

    int twice() {
        return area() * 2;
    }
}
