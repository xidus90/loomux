from app.models import User


def test_guest():
    assert User.guest()
