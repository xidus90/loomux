package calc;

import calc.shape.Child;

public class Calc {
    public int twice(int value) {
        Child child = new Child();
        return child.f() * value;
    }
}
