"""The models of the application."""


def _helper(value):
    return value.strip()


class Base:
    def __init__(self, name):
        self.name = _helper(name)

    def describe(self):
        return self.name


class User(Base):
    def __eq__(self, other):
        return self.describe() == other.describe()

    @staticmethod
    def guest():
        return User("guest")
