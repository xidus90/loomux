"""Services over the models."""
from . import models
from .models import User
from lib import util


def logged(tag):
    def wrap(fn):
        return fn

    return wrap


@logged("make")
def make_user(name):
    user = User(name)
    admin = models.Base(name)
    return user, admin


class Service:
    def __init__(self, prefix):
        self.prefix = util.clean(prefix)

    def run(self, names):
        def label(name):
            return self.prefix + util.clean(name)

        users = [make_user(label(n)) for n in names]
        self.report(users)
        return users

    def report(self, users):
        for user in users + [User.guest()]:
            print(user.describe())
