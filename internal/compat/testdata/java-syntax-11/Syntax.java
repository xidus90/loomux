import java.util.function.BinaryOperator;

// Java 11: var in the parameters of a lambda.
class Syntax {
    int add(int a, int b) {
        BinaryOperator<Integer> sum = (var x, var y) -> x + y;
        return sum.apply(a, b);
    }

    int twice(int value) {
        return add(value, value);
    }
}
