// Java 25: a module import and a compact source file with an instance main.
import module java.base;

void main() {
    IO.println(greet("world"));
}

String greet(String name) {
    return "hello " + name;
}
