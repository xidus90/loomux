from app import service
from app.service import *


def test_run():
    svc = service.Service("x-")
    assert svc.run(["a"]) == [make_user("x-a")]


def test_fixture():
    with open("fixture.txt") as f:
        assert f.read()
