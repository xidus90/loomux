package calc;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

class CalcTest {
    @Test
    void doublesAValue() {
        assertEquals(4, new Calc().twice(2));
    }
}
