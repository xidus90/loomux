from urllib.parse import urlsplit, unquote

cases = [
    "a.md", "topics/b.md?x=1#frag", "/abs/p.md", "../up.md", "https://ex.invalid/x",
    "mailto:a@b", "c:/x/y.md", "C:" + chr(92) + "x", "brain://project/space/topics/y",
    "BRAIN://Project/x", "//host/path?q", "  " + chr(9) + " sp.md", "a" + chr(9) + "b" + chr(10) + "c.md",
    "1abc:rest", "a+b-c.d:rest", "ab_c:rest", ":nope", "#only", "?q#f", "x%20y.md",
    "%zz%2", "%e2%82", "%C3%A4.md", "%", "a%2Fb", "", " ", chr(0) + chr(31) + " x.md",
    "%41%4a%4A", "%ff",
]
for c in cases:
    s = urlsplit(c)
    print(repr(c), "|", repr(s.scheme), repr(s.netloc), repr(s.path), "|", repr(unquote(c)))
