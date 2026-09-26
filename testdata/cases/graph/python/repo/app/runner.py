"""Uses pkg through the names its __init__.py passes on."""
import pkg
from pkg import Engine


class Turbo(Engine):
    def go(self):
        self.spin()
        pkg.start()
