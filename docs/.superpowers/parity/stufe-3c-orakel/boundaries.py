"""The boundaries the mutation round asked about: urlsplit at the edges of a
scheme name and of an empty network location, and the decoder at the first and
last lead byte of every range."""

from urllib.parse import unquote, urlsplit

for case in ["///x", "z:r", "Z:r", "a0:r", "a9:r", "a/:r"]:
    s = urlsplit(case)
    print(repr(case), repr(s.scheme), repr(s.netloc), repr(s.path))

for case in [
    "%C2%80", "%DF%BF", "%E0%A0%80", "%E1%80%80", "%EF%BF%BF", "%ED%9F%BF",
    "%F0%90%80%80", "%F1%80%80%80", "%F3%BF%BF%BF", "%F4%8F%BF%BF",
    "%C1%BF", "%F5%80", "%EF%BF", "%F3%BF%BF", "%C2", "%E0%9F%80",
]:
    print(case, ascii(unquote(case)))
