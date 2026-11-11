import java.lang.annotation.ElementType;
import java.lang.annotation.Target;
import java.util.function.IntUnaryOperator;

// Java 8: a lambda, a default method and a type annotation.
@Target(ElementType.TYPE_USE)
@interface NonNull {
}

interface Greeter {
    String name();

    default String greet() {
        return "hello " + name();
    }
}

class Syntax implements Greeter {
    public String name() {
        return "world";
    }

    int twice(int value) {
        IntUnaryOperator twice = x -> x * 2;
        return twice.applyAsInt(value);
    }

    @NonNull String shout() {
        return twice(1) + name();
    }
}
