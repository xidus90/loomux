from pkg.core import Base, Child, double


def test_base() -> None:
    assert Base().f() == 1


def test_child_adds_one() -> None:
    assert Child().f() == 2


def test_double() -> None:
    assert double(3) == 6
