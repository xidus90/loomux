"""A base and a child class: the shapes the graph is checked on."""


class Base:
    def f(self) -> int:
        return 1


class Child(Base):
    def f(self) -> int:
        return super().f() + 1


def double(value: int) -> int:
    return Child().f() * value
